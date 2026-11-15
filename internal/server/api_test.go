package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexandrmotologa/schemadoctor/internal/proxy"
	"github.com/alexandrmotologa/schemadoctor/internal/schema"
	"github.com/alexandrmotologa/schemadoctor/internal/telemetry"
)

func TestAPIServerEndpoints(t *testing.T) {
	reg := schema.NewRegistry()
	store, err := telemetry.NewStore(":memory:")
	if err != nil {
		t.Fatalf("failed to open memory sqlite: %v", err)
	}
	defer store.Close()

	metrics := telemetry.NewMetrics()
	bus := telemetry.NewEventBus()
	interceptor := proxy.NewInterceptor(reg, store, metrics, bus)

	api := NewAPIServer(reg, store, metrics, bus, interceptor)
	mux := http.NewServeMux()
	api.RegisterRoutes(mux)

	ts := httptest.NewServer(mux)
	defer ts.Close()

	// 1. Test /health
	resp, err := http.Get(ts.URL + "/health")
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("health check failed: %v, status: %d", err, resp.StatusCode)
	}

	// 2. Test /api/stats
	resp, err = http.Get(ts.URL + "/api/stats")
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("stats check failed: %v", err)
	}

	// 3. Test /api/repair
	repairReq := []byte(`{"input": "{\"user\": \"Alex\", \"roles\": [\"dev\""}`)
	resp, err = http.Post(ts.URL+"/api/repair", "application/json", bytes.NewReader(repairReq))
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("repair check failed: %v", err)
	}

	var repairData struct {
		Valid    bool   `json:"valid"`
		Repaired string `json:"repaired"`
	}
	_ = json.NewDecoder(resp.Body).Decode(&repairData)
	if !repairData.Valid {
		t.Errorf("expected valid JSON from /api/repair, got: %s", repairData.Repaired)
	}

	// 4. Test /api/schemas POST and GET
	schemaPayload := []byte(`{
		"name": "test_schema",
		"schema": {
			"type": "object",
			"properties": {"id": {"type": "integer"}}
		}
	}`)
	resp, err = http.Post(ts.URL+"/api/schemas", "application/json", bytes.NewReader(schemaPayload))
	if err != nil || resp.StatusCode != http.StatusCreated {
		t.Fatalf("schema registration failed: %v, status: %d", err, resp.StatusCode)
	}

	resp, err = http.Get(ts.URL + "/api/schemas")
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("schema list failed: %v", err)
	}

	// 5. Test /api/logs
	resp, err = http.Get(ts.URL + "/api/logs")
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("logs query failed: %v", err)
	}

	// 6. Test /metrics Prometheus exposition
	resp, err = http.Get(ts.URL + "/metrics")
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("metrics endpoint failed: %v", err)
	}
	metricsBody, _ := io.ReadAll(resp.Body)
	if !bytes.Contains(metricsBody, []byte("schemadoctor_requests_total")) {
		t.Errorf("expected schemadoctor_requests_total in /metrics")
	}

	// 7. Test /api/schemas/infer
	inferReq := []byte(`{"input": "{\"title\": \"Task A\", \"done\": false, \"priority\": 2}", "title": "TaskSchema"}`)
	resp, err = http.Post(ts.URL+"/api/schemas/infer", "application/json", bytes.NewReader(inferReq))
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("schema infer endpoint failed: %v, status: %d", err, resp.StatusCode)
	}
}
