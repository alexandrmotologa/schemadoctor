package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/alexandrmotologa/schemadoctor/internal/ast"
	"github.com/alexandrmotologa/schemadoctor/internal/proxy"
	"github.com/alexandrmotologa/schemadoctor/internal/schema"
	"github.com/alexandrmotologa/schemadoctor/internal/telemetry"
)

// APIServer provides management, playground, and telemetry REST endpoints.
type APIServer struct {
	StartTime   time.Time
	Registry    *schema.Registry
	Store       *telemetry.Store
	Metrics     *telemetry.Metrics
	EventBus    *telemetry.EventBus
	Interceptor *proxy.Interceptor
}

// NewAPIServer creates an APIServer instance.
func NewAPIServer(reg *schema.Registry, store *telemetry.Store, metrics *telemetry.Metrics, bus *telemetry.EventBus, interceptor *proxy.Interceptor) *APIServer {
	return &APIServer{
		StartTime:   time.Now(),
		Registry:    reg,
		Store:       store,
		Metrics:     metrics,
		EventBus:    bus,
		Interceptor: interceptor,
	}
}

// RegisterRoutes mounts all API routes onto the provided ServeMux.
func (s *APIServer) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/api/stats", s.handleStats)
	mux.HandleFunc("/api/logs", s.handleLogs)
	mux.HandleFunc("/api/repair", s.handleRepair)
	mux.HandleFunc("/api/schemas", s.handleSchemas)
	mux.HandleFunc("/api/stream", s.handleStream)
}

func (s *APIServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":         "ok",
		"uptime_seconds": time.Since(s.StartTime).Seconds(),
		"version":        "1.0.0",
	})
}

func (s *APIServer) handleStats(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	snap := s.Metrics.Snapshot()
	json.NewEncoder(w).Encode(snap)
}

func (s *APIServer) handleLogs(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	limit := 50
	offset := 0

	if lStr := r.URL.Query().Get("limit"); lStr != "" {
		if l, err := strconv.Atoi(lStr); err == nil && l > 0 {
			limit = l
		}
	}
	if oStr := r.URL.Query().Get("offset"); oStr != "" {
		if o, err := strconv.Atoi(oStr); err == nil && o >= 0 {
			offset = o
		}
	}

	events, count, err := s.Store.GetEvents(limit, offset)
	if err != nil {
		http.Error(w, fmt.Sprintf("database query error: %v", err), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(map[string]interface{}{
		"events": events,
		"total":  count,
		"limit":  limit,
		"offset": offset,
	})
}

func (s *APIServer) handleRepair(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-SchemaDoctor-Target-Schema")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "failed to read body", http.StatusBadRequest)
		return
	}

	var req struct {
		Input  string `json:"input"`
		Schema string `json:"schema"`
	}

	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		// Treat entire body as raw string if JSON parsing failed
		req.Input = string(bodyBytes)
	}

	start := time.Now()
	repairedBytes, flags, valid, coercedCount := s.Interceptor.RepairRawPayload([]byte(req.Input), req.Schema)
	latencyUs := time.Since(start).Microseconds()

	tokensSaved := int64(0)
	if flags.Any() || coercedCount > 0 {
		tokensSaved = int64(len(req.Input)/4 + 100)
	}

	// Also record event to store and bus
	evt := &telemetry.RepairEvent{
		TraceID:        fmt.Sprintf("trc-play-%d", time.Now().UnixNano()%100000),
		Timestamp:      time.Now(),
		Provider:       "studio-playground",
		Model:          "manual-test",
		Status:         "healed",
		LatencyUs:      latencyUs,
		TokensSaved:    tokensSaved,
		OriginalSize:   len(req.Input),
		RepairedSize:   len(repairedBytes),
		Flags:          flags.Summary(),
		RawInput:       req.Input,
		RepairedOutput: string(repairedBytes),
	}
	if !flags.Any() && coercedCount == 0 {
		evt.Status = "valid"
	}
	if !valid {
		evt.Status = "failed"
	}

	s.Store.InsertEvent(evt)
	s.Metrics.Record(flags.Any() || coercedCount > 0, valid, latencyUs, tokensSaved,
		flags.ClosedOpenString || flags.BalancedDelimiters,
		flags.StrippedMarkdown,
		flags.FixedUnescapedQuote,
		flags.RemovedTrailingComma,
		flags.CompletedDanglingKey,
		coercedCount > 0,
	)
	s.EventBus.Publish(evt)

	json.NewEncoder(w).Encode(map[string]interface{}{
		"valid":         valid,
		"repaired":      string(repairedBytes),
		"flags":         flags,
		"coerced_count": coercedCount,
		"latency_us":    latencyUs,
		"tokens_saved":  tokensSaved,
	})
}

func (s *APIServer) handleSchemas(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusOK)
		return
	}

	if r.Method == http.MethodGet {
		schemas := s.Registry.List()
		json.NewEncoder(w).Encode(schemas)
		return
	}

	if r.Method == http.MethodPost {
		var req struct {
			Name   string          `json:"name"`
			Schema json.RawMessage `json:"schema"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "invalid request body", http.StatusBadRequest)
			return
		}

		if req.Name == "" || len(req.Schema) == 0 {
			http.Error(w, "name and schema are required", http.StatusBadRequest)
			return
		}

		if err := s.Registry.Register(req.Name, req.Schema); err != nil {
			http.Error(w, fmt.Sprintf("failed to compile schema: %v", err), http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]string{
			"status": "registered",
			"name":   req.Name,
		})
		return
	}

	http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
}

func (s *APIServer) handleStream(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	sub := s.EventBus.Subscribe()
	defer s.EventBus.Unsubscribe(sub)

	// Send initial ping
	fmt.Fprintf(w, "event: connected\ndata: {}\n\n")
	flusher.Flush()

	notify := r.Context().Done()
	for {
		select {
		case <-notify:
			return
		case data, ok := <-sub:
			if !ok {
				return
			}
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}
}

// Dummy reference to avoid unused import
var _ = ast.TokenEOF
