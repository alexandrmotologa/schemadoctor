package proxy

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/alexandrmotologa/schemadoctor/internal/ast"
	"github.com/alexandrmotologa/schemadoctor/internal/telemetry"
	"github.com/google/uuid"
)

// HandleStreamingResponse intercepts Server-Sent Events (SSE) from upstream LLMs,
// balances unclosed delimiters on premature stream cutoffs or max_tokens limits,
// and streams clean tokens directly to the client.
func (h *Interceptor) HandleStreamingResponse(w http.ResponseWriter, upstreamResp *http.Response, targetSchema string) error {
	defer upstreamResp.Body.Close()

	// Copy upstream headers to client
	for k, v := range upstreamResp.Header {
		w.Header()[k] = v
	}
	w.WriteHeader(upstreamResp.StatusCode)

	flusher, ok := w.(http.Flusher)
	if !ok {
		// Fallback to direct copy if flushing is not supported
		_, err := io.Copy(w, upstreamResp.Body)
		return err
	}

	reader := bufio.NewReader(upstreamResp.Body)
	start := time.Now()
	traceID := fmt.Sprintf("trc-sse-%s", uuid.New().String()[:8])

	var accumulatedContent strings.Builder
	var lastChunkID = "chatcmpl-stream"
	var lastModel = "llm-streaming"
	var finishReason = ""

	for {
		line, err := reader.ReadBytes('\n')
		if len(line) > 0 {
			trimmedLine := bytes.TrimSpace(line)

			if bytes.HasPrefix(trimmedLine, []byte("data: ")) {
				payload := bytes.TrimPrefix(trimmedLine, []byte("data: "))

				if bytes.Equal(payload, []byte("[DONE]")) {
					// Stream completion reached. Check if accumulated content needs delimiter closure!
					raw := accumulatedContent.String()
					if len(raw) > 0 && looksLikeJSON(raw) {
						suffix := ast.CalculateClosingSuffix([]byte(raw))
						if len(suffix) > 0 {
							// Inject closing delta chunk before [DONE]
							injectChunk := map[string]interface{}{
								"id":      lastChunkID,
								"object":  "chat.completion.chunk",
								"created": time.Now().Unix(),
								"model":   lastModel,
								"choices": []map[string]interface{}{
									{
										"index": 0,
										"delta": map[string]string{
											"content": string(suffix),
										},
										"finish_reason": "length_healed",
									},
								},
							}
							injectedData, _ := json.Marshal(injectChunk)
							fmt.Fprintf(w, "data: %s\n\n", injectedData)
							flusher.Flush()
						}

						healedBytes := append([]byte(raw), suffix...)
						valid := json.Valid(healedBytes)
						latencyUs := time.Since(start).Microseconds()
						repaired := len(suffix) > 0
						tokensSaved := int64(0)
						if repaired {
							tokensSaved = int64(len(raw)/4 + 100)
						}

						evt := &telemetry.RepairEvent{
							TraceID:        traceID,
							Timestamp:      time.Now(),
							Provider:       "llm-proxy-sse",
							Model:          lastModel,
							Status:         "healed",
							LatencyUs:      latencyUs,
							TokensSaved:    tokensSaved,
							OriginalSize:   len(raw),
							RepairedSize:   len(healedBytes),
							Flags:          "balanced_delimiters",
							RawInput:       raw,
							RepairedOutput: string(healedBytes),
						}

						if h.Store != nil {
							_ = h.Store.InsertEvent(evt)
						}
						if h.Metrics != nil {
							h.Metrics.Record(repaired, valid, latencyUs, tokensSaved,
								repaired, false, false, false, false, false,
							)
						}
						if h.EventBus != nil {
							h.EventBus.Publish(evt)
						}
					}

					// Forward the [DONE] marker
					w.Write(line)
					flusher.Flush()
					break
				}

				// Inspect standard chunk
				var chunk struct {
					ID      string `json:"id"`
					Model   string `json:"model"`
					Choices []struct {
						Delta struct {
							Content string `json:"content"`
						} `json:"delta"`
						FinishReason string `json:"finish_reason"`
					} `json:"choices"`
				}

				if err := json.Unmarshal(payload, &chunk); err == nil {
					if chunk.ID != "" {
						lastChunkID = chunk.ID
					}
					if chunk.Model != "" {
						lastModel = chunk.Model
					}
					if len(chunk.Choices) > 0 {
						accumulatedContent.WriteString(chunk.Choices[0].Delta.Content)
						if chunk.Choices[0].FinishReason != "" {
							finishReason = chunk.Choices[0].FinishReason
						}
					}
				}
			}

			w.Write(line)
			flusher.Flush()
		}

		if err != nil {
			if err != io.EOF {
				return err
			}
			break
		}
	}

	_ = finishReason
	return nil
}
