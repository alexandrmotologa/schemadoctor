package schema

import (
	"encoding/json"
	"fmt"
	"sort"
)

// InferSchema analyzes an arbitrary JSON payload and generates a compliant JSON Schema definition.
func InferSchema(jsonBytes []byte, title string) (map[string]interface{}, error) {
	if len(jsonBytes) == 0 {
		return nil, fmt.Errorf("cannot infer schema from empty input")
	}

	var data interface{}
	if err := json.Unmarshal(jsonBytes, &data); err != nil {
		return nil, fmt.Errorf("invalid json payload for schema inference: %w", err)
	}

	if title == "" {
		title = "InferredSchema"
	}

	root := inferNode(data)
	if root == nil {
		root = map[string]interface{}{"type": "object"}
	}

	root["$schema"] = "https://json-schema.org/draft/2020-12/schema"
	root["title"] = title

	return root, nil
}

func inferNode(v interface{}) map[string]interface{} {
	if v == nil {
		return map[string]interface{}{"type": "null"}
	}

	switch val := v.(type) {
	case bool:
		return map[string]interface{}{"type": "boolean"}
	case float64:
		if val == float64(int64(val)) {
			return map[string]interface{}{"type": "integer"}
		}
		return map[string]interface{}{"type": "number"}
	case string:
		return map[string]interface{}{"type": "string"}
	case []interface{}:
		node := map[string]interface{}{
			"type": "array",
		}
		if len(val) > 0 {
			// Infer from the first item
			node["items"] = inferNode(val[0])
		} else {
			node["items"] = map[string]interface{}{}
		}
		return node
	case map[string]interface{}:
		props := make(map[string]interface{})
		var required []string

		// Sort keys for deterministic output
		keys := make([]string, 0, len(val))
		for k := range val {
			keys = append(keys, k)
		}
		sort.Strings(keys)

		for _, k := range keys {
			props[k] = inferNode(val[k])
			if val[k] != nil {
				required = append(required, k)
			}
		}

		node := map[string]interface{}{
			"type":       "object",
			"properties": props,
		}
		if len(required) > 0 {
			node["required"] = required
		}
		return node
	default:
		return map[string]interface{}{"type": "string"}
	}
}
