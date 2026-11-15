package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/alexandrmotologa/schemadoctor/internal/ast"
	"github.com/alexandrmotologa/schemadoctor/internal/schema"
	"github.com/alexandrmotologa/schemadoctor/internal/telemetry"
	"github.com/google/uuid"
)

// Interceptor handles response inspection, AST repair, and telemetry recording.
type Interceptor struct {
	Registry *schema.Registry
	Store    *telemetry.Store
	Metrics  *telemetry.Metrics
	EventBus *telemetry.EventBus
}

// NewInterceptor initializes a new response interceptor.
func NewInterceptor(reg *schema.Registry, store *telemetry.Store, metrics *telemetry.Metrics, bus *telemetry.EventBus) *Interceptor {
	return &Interceptor{
		Registry: reg,
		Store:    store,
		Metrics:  metrics,
		EventBus: bus,
	}
}

// RepairRawPayload directly heals a raw JSON string or bytes, optionally coercing against a named schema.
func (h *Interceptor) RepairRawPayload(raw []byte, schemaName string) ([]byte, ast.RepairFlags, bool, int, []ast.DiagnosticRecord) {
	repairRes := ast.Repair(raw)

	coercedCount := 0
	out := repairRes.Repaired

	if schemaName != "" && h.Registry != nil {
		if meta, ok := h.Registry.Get(schemaName); ok {
			coerceRes := schema.Coerce(meta, out)
			if coerceRes.Success {
				out = coerceRes.Repaired
				coercedCount = coerceRes.CoercedCount
			}
		}
	}

	return out, repairRes.Flags, repairRes.Valid, coercedCount, repairRes.Diagnostics
}

// HandleBufferedResponse inspects a buffered HTTP response from an upstream LLM,
// locates JSON within completion content, applies AST repair, and replaces it.
func (h *Interceptor) HandleBufferedResponse(resp *http.Response, targetSchema string) error {
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	resp.Body.Close()

	start := time.Now()
	traceID := fmt.Sprintf("trc-%s", uuid.New().String()[:8])

	// Attempt to parse as standard OpenAI chat completion response
	var chatResp map[string]interface{}
	if err := json.Unmarshal(bodyBytes, &chatResp); err != nil {
		// If not JSON at all, restore body and return
		resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		return nil
	}

	// Check if this is an OpenAI response or Anthropic response
	choices, isChoices := chatResp["choices"].([]interface{})
	anthropicContent, isAnthropic := chatResp["content"].([]interface{})

	var rawTarget string
	var repairLocation string // "content", "tool_arguments", "anthropic_text", "anthropic_tool_input"
	var targetToolIndex = -1

	if isChoices && len(choices) > 0 {
		if firstChoice, ok := choices[0].(map[string]interface{}); ok {
			if msg, ok := firstChoice["message"].(map[string]interface{}); ok {
				// Check for tool_calls first
				if toolCalls, ok := msg["tool_calls"].([]interface{}); ok && len(toolCalls) > 0 {
					for idx, tc := range toolCalls {
						if tcMap, ok := tc.(map[string]interface{}); ok {
							if fn, ok := tcMap["function"].(map[string]interface{}); ok {
								if args, ok := fn["arguments"].(string); ok && looksLikeJSON(strings.TrimSpace(args)) {
									rawTarget = args
									repairLocation = "tool_arguments"
									targetToolIndex = idx
									break
								}
							}
						}
					}
				}

				// If no tool_calls need repair, check message.content
				if rawTarget == "" {
					if content, ok := msg["content"].(string); ok {
						rawTarget = content
						repairLocation = "content"
					}
				}
			} else if text, ok := firstChoice["text"].(string); ok {
				rawTarget = text
				repairLocation = "text"
			}
		}
	} else if isAnthropic && len(anthropicContent) > 0 {
		// Inspect Anthropic content blocks
		for idx, block := range anthropicContent {
			if bMap, ok := block.(map[string]interface{}); ok {
				bType, _ := bMap["type"].(string)
				if bType == "text" {
					if txt, ok := bMap["text"].(string); ok && looksLikeJSON(strings.TrimSpace(txt)) {
						rawTarget = txt
						repairLocation = "anthropic_text"
						targetToolIndex = idx
						break
					}
				} else if bType == "tool_use" {
					if inputStr, ok := bMap["input"].(string); ok && looksLikeJSON(strings.TrimSpace(inputStr)) {
						rawTarget = inputStr
						repairLocation = "anthropic_tool_input"
						targetToolIndex = idx
						break
					}
				}
			}
		}
	}

	trimmed := strings.TrimSpace(rawTarget)
	if trimmed == "" || !looksLikeJSON(trimmed) {
		resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		return nil
	}

	// Apply AST self-healing repair
	healedBytes, flags, valid, coercedCount, _ := h.RepairRawPayload([]byte(trimmed), targetSchema)
	latencyUs := time.Since(start).Microseconds()
	repaired := flags.Any() || coercedCount > 0

	model, _ := chatResp["model"].(string)
	if model == "" {
		model = "llm-upstream"
	}

	// Record telemetry
	status := "valid"
	tokensSaved := int64(0)
	if repaired {
		status = "healed"
		tokensSaved = int64(len(trimmed)/4 + 100) // estimated retry tokens avoided
	}
	if !valid {
		status = "failed"
	}

	evt := &telemetry.RepairEvent{
		TraceID:        traceID,
		Timestamp:      time.Now(),
		Provider:       "llm-proxy",
		Model:          model,
		Status:         status,
		LatencyUs:      latencyUs,
		TokensSaved:    tokensSaved,
		OriginalSize:   len(trimmed),
		RepairedSize:   len(healedBytes),
		Flags:          flags.Summary(),
		RawInput:       trimmed,
		RepairedOutput: string(healedBytes),
	}

	if h.Store != nil {
		_ = h.Store.InsertEvent(evt)
	}
	if h.Metrics != nil {
		h.Metrics.Record(repaired, valid, latencyUs, tokensSaved,
			flags.ClosedOpenString || flags.BalancedDelimiters,
			flags.StrippedMarkdown,
			flags.FixedUnescapedQuote,
			flags.RemovedTrailingComma,
			flags.CompletedDanglingKey,
			coercedCount > 0,
		)
	}
	if h.EventBus != nil {
		h.EventBus.Publish(evt)
	}

	// If repaired, update the response body payload in the exact target location
	if repaired && valid {
		switch repairLocation {
		case "tool_arguments":
			firstChoice := choices[0].(map[string]interface{})
			msg := firstChoice["message"].(map[string]interface{})
			toolCalls := msg["tool_calls"].([]interface{})
			targetCall := toolCalls[targetToolIndex].(map[string]interface{})
			targetCall["function"].(map[string]interface{})["arguments"] = string(healedBytes)

		case "content":
			firstChoice := choices[0].(map[string]interface{})
			msg := firstChoice["message"].(map[string]interface{})
			msg["content"] = string(healedBytes)

		case "text":
			firstChoice := choices[0].(map[string]interface{})
			firstChoice["text"] = string(healedBytes)

		case "anthropic_text":
			targetBlock := anthropicContent[targetToolIndex].(map[string]interface{})
			targetBlock["text"] = string(healedBytes)

		case "anthropic_tool_input":
			targetBlock := anthropicContent[targetToolIndex].(map[string]interface{})
			targetBlock["input"] = string(healedBytes)
		}

		newBodyBytes, err := json.Marshal(chatResp)
		if err == nil {
			resp.Body = io.NopCloser(bytes.NewReader(newBodyBytes))
			resp.ContentLength = int64(len(newBodyBytes))
			resp.Header.Set("Content-Length", fmt.Sprintf("%d", len(newBodyBytes)))
			resp.Header.Set("X-SchemaDoctor-Healed", "true")
			resp.Header.Set("X-SchemaDoctor-Location", repairLocation)
			resp.Header.Set("X-SchemaDoctor-Latency-Us", fmt.Sprintf("%d", latencyUs))
			return nil
		}
	}

	resp.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	return nil
}

func looksLikeJSON(s string) bool {
	return strings.HasPrefix(s, "{") ||
		strings.HasPrefix(s, "[") ||
		strings.HasPrefix(s, "```") ||
		strings.Contains(s, "{\n") ||
		strings.Contains(s, "{\r\n") ||
		strings.Contains(s, "```json")
}
