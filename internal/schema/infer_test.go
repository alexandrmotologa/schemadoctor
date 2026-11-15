package schema

import (
	"encoding/json"
	"testing"
)

func TestInferSchema(t *testing.T) {
	input := []byte(`{
		"user_id": 105,
		"username": "alex",
		"is_active": true,
		"score": 98.75,
		"roles": ["admin", "editor"],
		"address": {
			"city": "San Francisco",
			"zip": 94103
		}
	}`)

	inferred, err := InferSchema(input, "UserAccount")
	if err != nil {
		t.Fatalf("failed to infer schema: %v", err)
	}

	rawSchema, err := json.Marshal(inferred)
	if err != nil {
		t.Fatalf("failed to marshal inferred schema: %v", err)
	}

	// Verify that the inferred schema compiles and validates against the original payload!
	reg := NewRegistry()
	if err := reg.Register("UserAccount", rawSchema); err != nil {
		t.Fatalf("inferred schema failed compilation: %v", err)
	}

	meta, ok := reg.Get("UserAccount")
	if !ok {
		t.Fatalf("expected UserAccount in registry")
	}

	if err := Validate(meta, input); err != nil {
		t.Errorf("original input failed validation against its own inferred schema: %v", err)
	}

	// Verify property types
	props := inferred["properties"].(map[string]interface{})
	if props["user_id"].(map[string]interface{})["type"] != "integer" {
		t.Errorf("expected user_id to be integer")
	}
	if props["score"].(map[string]interface{})["type"] != "number" {
		t.Errorf("expected score to be number")
	}
	if props["is_active"].(map[string]interface{})["type"] != "boolean" {
		t.Errorf("expected is_active to be boolean")
	}
	if props["roles"].(map[string]interface{})["type"] != "array" {
		t.Errorf("expected roles to be array")
	}
}
