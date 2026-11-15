package schema

import (
	"encoding/json"
	"testing"
)

func TestSchemaRegistryAndValidate(t *testing.T) {
	reg := NewRegistry()
	schemaRaw := []byte(`{
		"$schema": "https://json-schema.org/draft/2020-12/schema",
		"type": "object",
		"properties": {
			"name": {"type": "string"},
			"age": {"type": "integer"}
		},
		"required": ["name", "age"]
	}`)

	if err := reg.Register("user_profile", schemaRaw); err != nil {
		t.Fatalf("failed to register schema: %v", err)
	}

	meta, ok := reg.Get("user_profile")
	if !ok {
		t.Fatalf("expected user_profile in registry")
	}

	validJSON := []byte(`{"name": "Alice", "age": 25}`)
	if err := Validate(meta, validJSON); err != nil {
		t.Errorf("expected valid, got: %v", err)
	}

	invalidJSON := []byte(`{"name": "Alice"}`)
	if err := Validate(meta, invalidJSON); err == nil {
		t.Errorf("expected error for missing required field")
	}
}

func TestTypeCoercion(t *testing.T) {
	reg := NewRegistry()
	schemaRaw := []byte(`{
		"type": "object",
		"properties": {
			"age": {"type": "integer"},
			"score": {"type": "number"},
			"active": {"type": "boolean"},
			"tags": {
				"type": "array",
				"items": {"type": "string"}
			}
		}
	}`)

	if err := reg.Register("coercion_test", schemaRaw); err != nil {
		t.Fatalf("register error: %v", err)
	}

	meta, _ := reg.Get("coercion_test")

	// Payload where LLM encoded numbers and booleans as strings, and singleton as string
	malformed := []byte(`{
		"age": "42",
		"score": "98.5",
		"active": "true",
		"tags": "golang"
	}`)

	res := Coerce(meta, malformed)
	if !res.Success {
		t.Fatalf("coercion failed")
	}
	if res.CoercedCount != 4 {
		t.Errorf("expected 4 coerced fields, got %d", res.CoercedCount)
	}

	var parsed struct {
		Age    int      `json:"age"`
		Score  float64  `json:"score"`
		Active bool     `json:"active"`
		Tags   []string `json:"tags"`
	}

	if err := json.Unmarshal(res.Repaired, &parsed); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if parsed.Age != 42 || parsed.Score != 98.5 || !parsed.Active || len(parsed.Tags) != 1 || parsed.Tags[0] != "golang" {
		t.Errorf("unexpected coerced result: %+v", parsed)
	}

	// Validate against compiled schema
	if err := Validate(meta, res.Repaired); err != nil {
		t.Errorf("repaired json failed schema validation: %v", err)
	}
}
