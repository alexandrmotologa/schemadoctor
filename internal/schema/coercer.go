package schema

import (
	"encoding/json"
	"strconv"
	"strings"
)

// CoerceResult holds the output of the type coercion step.
type CoerceResult struct {
	Repaired     []byte
	CoercedCount int
	Success      bool
}

// Coerce traverses the parsed JSON object and coerces types to conform to the schema.
func Coerce(meta *SchemaMeta, input []byte) CoerceResult {
	if meta == nil || len(meta.Raw) == 0 {
		return CoerceResult{Repaired: input, CoercedCount: 0, Success: true}
	}

	var schemaObj map[string]interface{}
	if err := json.Unmarshal(meta.Raw, &schemaObj); err != nil {
		return CoerceResult{Repaired: input, CoercedCount: 0, Success: false}
	}

	var data interface{}
	if err := json.Unmarshal(input, &data); err != nil {
		return CoerceResult{Repaired: input, CoercedCount: 0, Success: false}
	}

	coercedCount := 0
	data = coerceNode(data, schemaObj, &coercedCount)

	out, err := json.Marshal(data)
	if err != nil {
		return CoerceResult{Repaired: input, CoercedCount: 0, Success: false}
	}

	return CoerceResult{
		Repaired:     out,
		CoercedCount: coercedCount,
		Success:      true,
	}
}

func coerceNode(val interface{}, schema map[string]interface{}, count *int) interface{} {
	if schema == nil || val == nil {
		return val
	}

	expectedTypes := getExpectedTypes(schema)

	// Object handling
	if obj, ok := val.(map[string]interface{}); ok {
		props, _ := schema["properties"].(map[string]interface{})
		if props != nil {
			for k, v := range obj {
				if propSchema, exists := props[k].(map[string]interface{}); exists {
					obj[k] = coerceNode(v, propSchema, count)
				}
			}
		}
		return obj
	}

	// Array handling
	if arr, ok := val.([]interface{}); ok {
		itemSchema, _ := schema["items"].(map[string]interface{})
		if itemSchema != nil {
			for i, v := range arr {
				arr[i] = coerceNode(v, itemSchema, count)
			}
		}
		return arr
	}

	// Singleton primitive to Array coercion
	if containsType(expectedTypes, "array") {
		itemSchema, _ := schema["items"].(map[string]interface{})
		elem := val
		if itemSchema != nil {
			elem = coerceNode(val, itemSchema, count)
		}
		*count++
		return []interface{}{elem}
	}

	// String to Number / Integer coercion
	if str, ok := val.(string); ok {
		trimmed := strings.TrimSpace(str)

		if containsType(expectedTypes, "integer") {
			if n, err := strconv.ParseInt(trimmed, 10, 64); err == nil {
				*count++
				return n
			}
		}

		if containsType(expectedTypes, "number") {
			if f, err := strconv.ParseFloat(trimmed, 64); err == nil {
				*count++
				return f
			}
		}

		// String to Boolean coercion
		if containsType(expectedTypes, "boolean") {
			lower := strings.ToLower(trimmed)
			if lower == "true" || lower == "yes" || lower == "1" {
				*count++
				return true
			}
			if lower == "false" || lower == "no" || lower == "0" {
				*count++
				return false
			}
		}

		// Empty string to null
		if containsType(expectedTypes, "null") && trimmed == "" {
			*count++
			return nil
		}
	}

	// Number to Boolean coercion (e.g. 1 -> true, 0 -> false)
	if num, ok := val.(float64); ok && containsType(expectedTypes, "boolean") {
		if num == 1 {
			*count++
			return true
		}
		if num == 0 {
			*count++
			return false
		}
	}

	return val
}

func getExpectedTypes(schema map[string]interface{}) []string {
	var types []string
	if t, ok := schema["type"].(string); ok {
		types = append(types, t)
	} else if arr, ok := schema["type"].([]interface{}); ok {
		for _, item := range arr {
			if s, ok := item.(string); ok {
				types = append(types, s)
			}
		}
	}
	return types
}

func containsType(types []string, target string) bool {
	for _, t := range types {
		if t == target {
			return true
		}
	}
	return false
}
