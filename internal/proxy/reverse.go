package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"
)

// ReverseProxy forwards requests to upstream LLMs while transparently repairing responses.
type ReverseProxy struct {
	TargetURL   *url.URL
	Proxy       *httputil.ReverseProxy
	Interceptor *Interceptor
	Transport   *http.Transport
}

// NewReverseProxy creates a new proxy instance pointing to upstream LLM URL.
func NewReverseProxy(upstreamURL string, interceptor *Interceptor) (*ReverseProxy, error) {
	target, err := url.Parse(upstreamURL)
	if err != nil {
		return nil, fmt.Errorf("invalid upstream url %s: %w", upstreamURL, err)
	}

	transport := &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 20,
		IdleConnTimeout:     90 * time.Second,
	}

	rp := &ReverseProxy{
		TargetURL:   target,
		Interceptor: interceptor,
		Transport:   transport,
	}

	proxy := &httputil.ReverseProxy{
		Director: func(req *http.Request) {
			req.URL.Scheme = target.Scheme
			req.URL.Host = target.Host
			req.Host = target.Host
		},
		Transport: transport,
		ModifyResponse: func(resp *http.Response) error {
			contentType := resp.Header.Get("Content-Type")
			if strings.Contains(contentType, "text/event-stream") {
				// Streaming is handled separately in ServeHTTP
				return nil
			}

			targetSchema := resp.Request.Header.Get("X-SchemaDoctor-Target-Schema")
			return interceptor.HandleBufferedResponse(resp, targetSchema)
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			http.Error(w, fmt.Sprintf("upstream proxy error: %v", err), http.StatusBadGateway)
		},
	}

	rp.Proxy = proxy
	return rp, nil
}

// ServeHTTP inspects whether request is streaming SSE or buffered JSON and dispatches appropriately.
func (rp *ReverseProxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	targetSchema := r.Header.Get("X-SchemaDoctor-Target-Schema")
	if targetSchema == "" {
		targetSchema = r.URL.Query().Get("schema")
	}

	// Check if the request body specifies "stream": true
	isStream := false
	var bodyBytes []byte

	if r.Body != nil && (r.Method == http.MethodPost || r.Method == http.MethodPut) {
		var err error
		bodyBytes, err = io.ReadAll(r.Body)
		if err == nil {
			r.Body = io.NopCloser(bytes.NewReader(bodyBytes))

			var reqMap map[string]interface{}
			if err := json.Unmarshal(bodyBytes, &reqMap); err == nil {
				if streamVal, ok := reqMap["stream"].(bool); ok && streamVal {
					isStream = true
				}
			}
		}
	}

	if isStream {
		// Forward request to upstream directly via Transport to handle streaming chunks
		outReq, err := http.NewRequestWithContext(r.Context(), r.Method, rp.TargetURL.ResolveReference(r.URL).String(), bytes.NewReader(bodyBytes))
		if err != nil {
			http.Error(w, fmt.Sprintf("failed to create upstream request: %v", err), http.StatusInternalServerError)
			return
		}

		// Copy request headers
		for k, vv := range r.Header {
			for _, v := range vv {
				outReq.Header.Add(k, v)
			}
		}
		outReq.Host = rp.TargetURL.Host

		resp, err := rp.Transport.RoundTrip(outReq)
		if err != nil {
			http.Error(w, fmt.Sprintf("upstream stream connection failed: %v", err), http.StatusBadGateway)
			return
		}

		if strings.Contains(resp.Header.Get("Content-Type"), "text/event-stream") {
			if err := rp.Interceptor.HandleStreamingResponse(w, resp, targetSchema); err != nil {
				// Streaming error
				return
			}
			return
		}

		// If upstream returned non-streaming response despite stream: true (e.g. error JSON)
		if err := rp.Interceptor.HandleBufferedResponse(resp, targetSchema); err != nil {
			http.Error(w, fmt.Sprintf("failed to process response: %v", err), http.StatusInternalServerError)
			return
		}

		for k, vv := range resp.Header {
			for _, v := range vv {
				w.Header().Add(k, v)
			}
		}
		w.WriteHeader(resp.StatusCode)
		io.Copy(w, resp.Body)
		resp.Body.Close()
		return
	}

	// Standard buffered request
	rp.Proxy.ServeHTTP(w, r)
}
