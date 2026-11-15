package proxy

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alexandrmotologa/schemadoctor/internal/schema"
	"github.com/alexandrmotologa/schemadoctor/internal/telemetry"
)

func TestProxyNonStreamingRepair(t *testing.T) {
	mockUpstream := NewMockLLMServer()
	defer mockUpstream.Close()

	reg := schema.NewRegistry()
	store, _ := telemetry.NewStore(":memory:")
	defer store.Close()
	metrics := telemetry.NewMetrics()
	bus := telemetry.NewEventBus()

	interceptor := NewInterceptor(reg, store, metrics, bus)
	proxy, err := NewReverseProxy(mockUpstream.URL, interceptor)
	if err != nil {
		t.Fatalf("failed to create proxy: %v", err)
	}

	proxyServer := httptest.NewServer(proxy)
	defer proxyServer.Close()

	// Request with truncated array model
	reqBody := []byte(`{
		"model": "mock-truncated",
		"messages": [{"role": "user", "content": "List items"}]
	}`)

	resp, err := http.Post(proxyServer.URL+"/v1/chat/completions", "application/json", bytes.NewReader(reqBody))
	if err != nil {
		t.Fatalf("post failed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", resp.StatusCode)
	}

	if resp.Header.Get("X-SchemaDoctor-Healed") != "true" {
		t.Errorf("expected X-SchemaDoctor-Healed header")
	}

	var chatResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		t.Fatalf("failed to decode chat completion response: %v", err)
	}

	content := chatResp.Choices[0].Message.Content
	if !json.Valid([]byte(content)) {
		t.Fatalf("repaired content is not valid JSON: %s", content)
	}

	// Verify metrics
	snap := metrics.Snapshot()
	if snap.TotalRequests != 1 || snap.RepairedRequests != 1 {
		t.Errorf("expected 1 request and 1 repair, got: %+v", snap)
	}

	// Verify SQLite logs
	events, count, err := store.GetEvents(10, 0)
	if err != nil || count != 1 {
		t.Errorf("expected 1 event in store, got %d: %v", count, err)
	}
	if len(events) > 0 && events[0].Status != "healed" {
		t.Errorf("expected healed status in store, got %s", events[0].Status)
	}
}

func TestProxyStreamingSSE(t *testing.T) {
	mockUpstream := NewMockLLMServer()
	defer mockUpstream.Close()

	reg := schema.NewRegistry()
	store, _ := telemetry.NewStore(":memory:")
	defer store.Close()
	metrics := telemetry.NewMetrics()
	bus := telemetry.NewEventBus()

	interceptor := NewInterceptor(reg, store, metrics, bus)
	proxy, err := NewReverseProxy(mockUpstream.URL, interceptor)
	if err != nil {
		t.Fatalf("failed to create proxy: %v", err)
	}

	proxyServer := httptest.NewServer(proxy)
	defer proxyServer.Close()

	reqBody := []byte(`{
		"model": "mock-truncated",
		"stream": true,
		"messages": [{"role": "user", "content": "List items"}]
	}`)

	resp, err := http.Post(proxyServer.URL+"/v1/chat/completions", "application/json", bytes.NewReader(reqBody))
	if err != nil {
		t.Fatalf("streaming post failed: %v", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read stream body: %v", err)
	}

	streamStr := string(bodyBytes)
	if !strings.Contains(streamStr, "data: [DONE]") {
		t.Errorf("expected [DONE] marker in stream")
	}

	// Reconstruct accumulated delta content
	var fullContent strings.Builder
	lines := strings.Split(streamStr, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "data: ") {
			payload := strings.TrimPrefix(line, "data: ")
			if payload == "[DONE]" {
				continue
			}
			var chunk struct {
				Choices []struct {
					Delta struct {
						Content string `json:"content"`
					} `json:"delta"`
				} `json:"choices"`
			}
			if err := json.Unmarshal([]byte(payload), &chunk); err == nil && len(chunk.Choices) > 0 {
				fullContent.WriteString(chunk.Choices[0].Delta.Content)
			}
		}
	}

	finalJSON := fullContent.String()
	if !json.Valid([]byte(finalJSON)) {
		t.Fatalf("accumulated SSE content is not valid JSON: %s", finalJSON)
	}
}

func TestProxySchemaCoercion(t *testing.T) {
	mockUpstream := NewMockLLMServer()
	defer mockUpstream.Close()

	reg := schema.NewRegistry()
	schemaRaw := []byte(`{
		"type": "object",
		"properties": {
			"name": {"type": "string"},
			"age": {"type": "integer"},
			"score": {"type": "number"},
			"active": {"type": "boolean"},
			"tags": {"type": "array", "items": {"type": "string"}}
		}
	}`)
	_ = reg.Register("person_schema", schemaRaw)

	store, _ := telemetry.NewStore(":memory:")
	defer store.Close()
	metrics := telemetry.NewMetrics()
	bus := telemetry.NewEventBus()

	interceptor := NewInterceptor(reg, store, metrics, bus)
	proxy, _ := NewReverseProxy(mockUpstream.URL, interceptor)

	proxyServer := httptest.NewServer(proxy)
	defer proxyServer.Close()

	reqBody := []byte(`{
		"model": "mock-coercion",
		"messages": [{"role": "user", "content": "Return person"}]
	}`)

	req, _ := http.NewRequest(http.MethodPost, proxyServer.URL+"/v1/chat/completions", bytes.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-SchemaDoctor-Target-Schema", "person_schema")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()

	var chatResp struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	_ = json.NewDecoder(resp.Body).Decode(&chatResp)
	content := chatResp.Choices[0].Message.Content

	var person struct {
		Name   string   `json:"name"`
		Age    int      `json:"age"`
		Score  float64  `json:"score"`
		Active bool     `json:"active"`
		Tags   []string `json:"tags"`
	}

	if err := json.Unmarshal([]byte(content), &person); err != nil {
		t.Fatalf("failed to unmarshal coerced person: %v, content: %s", err, content)
	}

	if person.Age != 29 || person.Score != 95.5 || !person.Active || len(person.Tags) != 1 || person.Tags[0] != "go" {
		t.Errorf("coercion values mismatch: %+v", person)
	}
}
