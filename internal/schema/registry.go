package schema

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/santhosh-tekuri/jsonschema/v5"
)

// SchemaMeta stores information and parsed compiler instance for a schema.
type SchemaMeta struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Raw         json.RawMessage `json:"raw"`
	Compiled    *jsonschema.Schema
}

// Registry manages thread-safe schema storage and compilation.
type Registry struct {
	mu      sync.RWMutex
	schemas map[string]*SchemaMeta
}

// NewRegistry initializes an empty schema registry.
func NewRegistry() *Registry {
	return &Registry{
		schemas: make(map[string]*SchemaMeta),
	}
}

// Register compiles and registers a JSON Schema under a given identifier.
func (r *Registry) Register(name string, raw []byte) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var js map[string]interface{}
	if err := json.Unmarshal(raw, &js); err != nil {
		return fmt.Errorf("invalid json schema payload: %w", err)
	}

	compiler := jsonschema.NewCompiler()
	url := fmt.Sprintf("schema://%s.json", name)
	if err := compiler.AddResource(url, bytes.NewReader(raw)); err != nil {
		return fmt.Errorf("failed to add schema resource: %w", err)
	}

	compiled, err := compiler.Compile(url)
	if err != nil {
		return fmt.Errorf("failed to compile schema %s: %w", name, err)
	}

	desc, _ := js["description"].(string)

	r.schemas[name] = &SchemaMeta{
		Name:        name,
		Description: desc,
		Raw:         raw,
		Compiled:    compiled,
	}

	return nil
}

// Get returns the registered schema metadata, if found.
func (r *Registry) Get(name string) (*SchemaMeta, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	meta, ok := r.schemas[name]
	return meta, ok
}

// List returns a list of all registered schemas.
func (r *Registry) List() []*SchemaMeta {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*SchemaMeta, 0, len(r.schemas))
	for _, meta := range r.schemas {
		result = append(result, meta)
	}
	return result
}

// LoadFromDir scans a directory for *.json files and registers them.
func (r *Registry) LoadFromDir(dir string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}

		name := stringsTrimExt(entry.Name())
		fullPath := filepath.Join(dir, entry.Name())
		content, err := os.ReadFile(fullPath)
		if err != nil {
			return fmt.Errorf("failed to read %s: %w", fullPath, err)
		}

		if err := r.Register(name, content); err != nil {
			// Continue or log error
			continue
		}
	}
	return nil
}

func stringsTrimExt(fileName string) string {
	ext := filepath.Ext(fileName)
	return fileName[:len(fileName)-len(ext)]
}
