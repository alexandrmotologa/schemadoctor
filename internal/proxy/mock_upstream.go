package proxy

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"
)

// MockLLMServer provides a self-contained mock upstream LLM endpoint for local testing and demos.
type MockLLMServer struct {
	server *httptest.Server
	URL    string
}

// NewMockLLMServer initializes and starts a local mock upstream server.
func NewMockLLMServer() *MockLLMServer {
	mux := http.NewServeMux()

	mux.HandleFunc("/v1/models", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		resp := map[string]interface{}{
			"object": "list",
			"data": []map[string]interface{}{
				{"id": "gpt-4o-mini", "object": "model"},
				{"id": "mock-truncated", "object": "model"},
				{"id": "mock-markdown", "object": "model"},
				{"id": "mock-quotes", "object": "model"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})

	mux.HandleFunc("/v1/chat/completions", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Model    string `json:"model"`
			Stream   bool   `json:"stream"`
			Messages []struct {
				Role    string `json:"role"`
				Content string `json:"content"`
			} `json:"messages"`
		}

		_ = json.NewDecoder(r.Body).Decode(&req)

		// Determine mock broken payload based on model or prompt
		var brokenContent string
		var finishReason = "stop"

		switch {
		case strings.Contains(req.Model, "truncated") || containsPrompt(req.Messages, "truncate"):
			brokenContent = `{"status": "success", "results": [{"id": 1, "title": "First"}, {"id": 2, "title": "Second"`
			finishReason = "length"
		case strings.Contains(req.Model, "markdown") || containsPrompt(req.Messages, "markdown"):
			brokenContent = "```json\n{\"id\": 404, \"message\": \"Resource not found\", \"retry\": False, }\n```"
		case strings.Contains(req.Model, "quotes") || containsPrompt(req.Messages, "quote"):
			brokenContent = `{"user": "Alex", "note": "Said "hello world" earlier", "items": [1, 2, ]}`
		case strings.Contains(req.Model, "coercion") || containsPrompt(req.Messages, "coerce"):
			brokenContent = `{"name": "Developer", "age": "29", "score": "95.5", "active": "true", "tags": "go"}`
		default:
			// Default broken sample: truncated array
			brokenContent = `{"user": {"name": "Alice", "roles": ["admin", "developer"`
			finishReason = "length"
		}

		if req.Stream {
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Cache-Control", "no-cache")
			w.Header().Set("Connection", "keep-alive")

			flusher, ok := w.(http.Flusher)
			if !ok {
				http.Error(w, "streaming unsupported", http.StatusInternalServerError)
				return
			}

			// Chunk the broken content into 3 pieces
			chunkLen := (len(brokenContent) + 2) / 3
			for i := 0; i < len(brokenContent); i += chunkLen {
				end := i + chunkLen
				if end > len(brokenContent) {
					end = len(brokenContent)
				}
				chunkPart := brokenContent[i:end]

				chunkObj := map[string]interface{}{
					"id":      "chatcmpl-mock",
					"object":  "chat.completion.chunk",
					"created": time.Now().Unix(),
					"choices": []map[string]interface{}{
						{
							"index": 0,
							"delta": map[string]string{
								"content": chunkPart,
							},
						},
					},
				}
				data, _ := json.Marshal(chunkObj)
				fmt.Fprintf(w, "data: %s\n\n", data)
				flusher.Flush()
				time.Sleep(5 * time.Millisecond)
			}

			// Final chunk with finish_reason
			finalChunk := map[string]interface{}{
				"id":      "chatcmpl-mock",
				"object":  "chat.completion.chunk",
				"created": time.Now().Unix(),
				"choices": []map[string]interface{}{
					{
						"index":         0,
						"delta":         map[string]string{},
						"finish_reason": finishReason,
					},
				},
			}
			finalData, _ := json.Marshal(finalChunk)
			fmt.Fprintf(w, "data: %s\n\n", finalData)
			fmt.Fprintf(w, "data: [DONE]\n\n")
			flusher.Flush()
			return
		}

		// Non-streaming response
		w.Header().Set("Content-Type", "application/json")
		resp := map[string]interface{}{
			"id":      "chatcmpl-mock-123",
			"object":  "chat.completion",
			"created": time.Now().Unix(),
			"model":   req.Model,
			"choices": []map[string]interface{}{
				{
					"index": 0,
					"message": map[string]string{
						"role":    "assistant",
						"content": brokenContent,
					},
					"finish_reason": finishReason,
				},
			},
			"usage": map[string]int{
				"prompt_tokens":     45,
				"completion_tokens": 60,
				"total_tokens":      105,
			},
		}

		json.NewEncoder(w).Encode(resp)
	})

	server := httptest.NewServer(mux)
	return &MockLLMServer{
		server: server,
		URL:    server.URL,
	}
}

// Close stops the mock upstream server.
func (m *MockLLMServer) Close() {
	m.server.Close()
}

func containsPrompt(messages []struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}, term string) bool {
	for _, m := range messages {
		if strings.Contains(strings.ToLower(m.Content), strings.ToLower(term)) {
			return true
		}
	}
	return false
}
