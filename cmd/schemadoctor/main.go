package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/alexandrmotologa/schemadoctor/internal/ast"
	"github.com/alexandrmotologa/schemadoctor/internal/config"
	"github.com/alexandrmotologa/schemadoctor/internal/proxy"
	"github.com/alexandrmotologa/schemadoctor/internal/schema"
	"github.com/alexandrmotologa/schemadoctor/internal/server"
	"github.com/alexandrmotologa/schemadoctor/internal/telemetry"
)

var (
	Version   = "1.0.0"
	BuildTime = "2026-11-15"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "serve":
		runServe(os.Args[2:])
	case "fix":
		runFix(os.Args[2:])
	case "bench":
		runBench(os.Args[2:])
	case "version", "--version", "-v":
		fmt.Printf("schemadoctor v%s (%s)\n", Version, BuildTime)
	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`schemadoctor - Zero-Latency Streaming JSON Self-Healing & AST Repair Proxy

Usage:
  schemadoctor <command> [options]

Commands:
  serve      Start the reverse proxy and embedded studio web dashboard
  fix        Repair malformed JSON from a file or standard input
  bench      Benchmark the AST repair engine throughput and latency
  version    Show version and build information

Use "schemadoctor <command> --help" for more information about a command.`)
}

func runServe(args []string) {
	fs := flag.NewFlagSet("serve", flag.ExitOnError)
	configPath := fs.String("config", "", "Path to YAML configuration file")
	proxyPort := fs.Int("port", 8080, "Port for the LLM reverse proxy")
	studioPort := fs.Int("studio-port", 4141, "Port for the Studio UI and management API")
	upstreamURL := fs.String("upstream", "", "Upstream LLM target URL (e.g. https://api.openai.com)")
	mockUpstream := fs.Bool("mock-upstream", false, "Start built-in simulated upstream LLM for offline testing")
	dbPath := fs.String("db", "data/schemadoctor.db", "Path to embedded SQLite database")
	schemasDir := fs.String("schemas", "schemas", "Directory for registered JSON schema files")

	_ = fs.Parse(args)

	cfg, err := config.LoadFromFile(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if *proxyPort != 8080 || cfg.Server.ProxyPort == 0 {
		cfg.Server.ProxyPort = *proxyPort
	}
	if *studioPort != 4141 || cfg.Server.StudioPort == 0 {
		cfg.Server.StudioPort = *studioPort
	}
	if *upstreamURL != "" {
		cfg.Upstream.URL = *upstreamURL
	}
	if *mockUpstream {
		cfg.Upstream.MockUpstream = true
	}
	if *dbPath != "data/schemadoctor.db" || cfg.Database.Path == "" {
		cfg.Database.Path = *dbPath
	}
	if *schemasDir != "schemas" || cfg.Schemas.Directory == "" {
		cfg.Schemas.Directory = *schemasDir
	}

	// Start Mock LLM server if requested
	var mockServer *proxy.MockLLMServer
	if cfg.Upstream.MockUpstream {
		mockServer = proxy.NewMockLLMServer()
		defer mockServer.Close()
		cfg.Upstream.URL = mockServer.URL
		log.Printf("[mock-upstream] Started mock LLM server at %s", mockServer.URL)
	}

	// Initialize subsystems
	reg := schema.NewRegistry()
	_ = reg.LoadFromDir(cfg.Schemas.Directory)

	store, err := telemetry.NewStore(cfg.Database.Path)
	if err != nil {
		log.Fatalf("Failed to initialize SQLite store: %v", err)
	}
	defer store.Close()

	metrics := telemetry.NewMetrics()
	bus := telemetry.NewEventBus()
	interceptor := proxy.NewInterceptor(reg, store, metrics, bus)

	revProxy, err := proxy.NewReverseProxy(cfg.Upstream.URL, interceptor)
	if err != nil {
		log.Fatalf("Failed to initialize reverse proxy: %v", err)
	}

	apiServer := server.NewAPIServer(reg, store, metrics, bus, interceptor)

	// Setup Studio & API HTTP server
	studioMux := http.NewServeMux()
	apiServer.RegisterRoutes(studioMux)
	server.RegisterStaticFiles(studioMux)

	studioAddr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.StudioPort)
	go func() {
		log.Printf("[studio] React 19 Studio & API listening on http://localhost:%d", cfg.Server.StudioPort)
		if err := http.ListenAndServe(studioAddr, studioMux); err != nil {
			log.Fatalf("Studio server error: %v", err)
		}
	}()

	// Setup Proxy HTTP server
	proxyAddr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.ProxyPort)
	log.Printf("[proxy] Reverse proxy listening on http://localhost:%d -> %s", cfg.Server.ProxyPort, cfg.Upstream.URL)
	if err := http.ListenAndServe(proxyAddr, revProxy); err != nil {
		log.Fatalf("Proxy server error: %v", err)
	}
}

func runFix(args []string) {
	fs := flag.NewFlagSet("fix", flag.ExitOnError)
	schemaName := fs.String("schema", "", "Optional target schema name or file path to coerce against")
	indent := fs.Bool("indent", true, "Format output with 2-space indentation")
	inPlace := fs.Bool("in-place", false, "Overwrite file in place (only when a file argument is provided)")

	_ = fs.Parse(args)

	var inputBytes []byte
	var filePath string

	remaining := fs.Args()
	if len(remaining) > 0 {
		filePath = remaining[0]
		data, err := os.ReadFile(filePath)
		if err != nil {
			log.Fatalf("Failed to read file %s: %v", filePath, err)
		}
		inputBytes = data
	} else {
		// Read from stdin
		stat, _ := os.Stdin.Stat()
		if (stat.Mode() & os.ModeCharDevice) != 0 {
			fmt.Println("Reading from standard input. Paste JSON and press Ctrl+D (or Ctrl+Z on Windows):")
		}
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			log.Fatalf("Failed to read stdin: %v", err)
		}
		inputBytes = data
	}

	res := ast.Repair(inputBytes)
	out := res.Repaired

	if *schemaName != "" {
		reg := schema.NewRegistry()
		// Check if schemaName is a file
		if strings.HasSuffix(*schemaName, ".json") && fileExists(*schemaName) {
			sData, err := os.ReadFile(*schemaName)
			if err == nil {
				baseName := filepath.Base(*schemaName)
				_ = reg.Register(baseName, sData)
				*schemaName = baseName
			}
		}
		if meta, ok := reg.Get(*schemaName); ok {
			cRes := schema.Coerce(meta, out)
			if cRes.Success {
				out = cRes.Repaired
			}
		}
	}

	if *indent && json.Valid(out) {
		var v interface{}
		if err := json.Unmarshal(out, &v); err == nil {
			formatted, err := json.MarshalIndent(v, "", "  ")
			if err == nil {
				out = formatted
			}
		}
	}

	if *inPlace && filePath != "" {
		if err := os.WriteFile(filePath, out, 0644); err != nil {
			log.Fatalf("Failed to write to %s: %v", filePath, err)
		}
		fmt.Printf("Successfully repaired %s\n", filePath)
		return
	}

	fmt.Println(string(out))
}

func runBench(args []string) {
	fs := flag.NewFlagSet("bench", flag.ExitOnError)
	iterations := fs.Int("n", 100000, "Number of repair iterations to execute")
	_ = fs.Parse(args)

	samples := []struct {
		name string
		raw  []byte
	}{
		{"Truncated Array", []byte(`{"status": "success", "results": [{"id": 1}, {"id": 2`)},
		{"Markdown Fences", []byte("```json\n{\"id\": 101, \"ready\": True, \"tags\": [1, 2, ], }\n```")},
		{"Unescaped Quotes", []byte(`{"comment": "Customer said "Hello" today", "rating": 5}`)},
		{"Dangling Key", []byte(`{"status": 200, "data": }`)},
		{"Python Literals", []byte(`{"enabled": True, "archived": False, "meta": None}`)},
	}

	fmt.Printf("Running schemadoctor benchmark (%d iterations per sample)...\n", *iterations)
	fmt.Printf("%-24s | %-12s | %-14s | %-12s\n", "Sample", "Throughput", "Avg Latency", "Valid JSON")
	fmt.Println(strings.Repeat("-", 70))

	for _, s := range samples {
		start := time.Now()
		var lastValid bool
		for i := 0; i < *iterations; i++ {
			res := ast.Repair(s.raw)
			lastValid = res.Valid
		}
		elapsed := time.Since(start)
		opsPerSec := float64(*iterations) / elapsed.Seconds()
		avgLatencyUs := float64(elapsed.Microseconds()) / float64(*iterations)

		validMark := "YES"
		if !lastValid {
			validMark = "NO"
		}

		fmt.Printf("%-24s | %10.0f/s | %10.3f us | %-10s\n", s.name, opsPerSec, avgLatencyUs, validMark)
	}

	fmt.Println(strings.Repeat("-", 70))
	fmt.Println("All benchmarks completed successfully.")
}

func fileExists(p string) bool {
	info, err := os.Stat(p)
	return err == nil && !info.IsDir()
}
