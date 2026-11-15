package schema

import (
	"encoding/json"
	"fmt"
)

// ValidationError describes a schema constraint failure.
type ValidationError struct {
	InstanceLocation string `json:"instance_location"`
	KeywordLocation  string `json:"keyword_location"`
	Message          string `json:"message"`
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("schema validation failed at %s: %s", e.InstanceLocation, e.Message)
}

// Validate checks whether jsonBytes conforms to the compiled schema.
func Validate(meta *SchemaMeta, jsonBytes []byte) error {
	if meta == nil || meta.Compiled == nil {
		return nil
	}

	var v interface{}
	if err := json.Unmarshal(jsonBytes, &v); err != nil {
		return fmt.Errorf("invalid json payload: %w", err)
	}

	if err := meta.Compiled.Validate(v); err != nil {
		return err
	}

	return nil
}
