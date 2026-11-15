package ast

import (
	"encoding/json"
	"testing"
)

func TestRepairTruncatedArray(t *testing.T) {
	input := []byte(`{"items": [1, 2, 3`)
	res := Repair(input)
	if !res.Valid {
		t.Fatalf("expected valid JSON, got: %s", string(res.Repaired))
	}
	if !res.Flags.BalancedDelimiters {
		t.Errorf("expected BalancedDelimiters flag")
	}

	var data struct {
		Items []int `json:"items"`
	}
	if err := json.Unmarshal(res.Repaired, &data); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if len(data.Items) != 3 {
		t.Errorf("expected 3 items, got %d", len(data.Items))
	}
}

func TestRepairTruncatedObject(t *testing.T) {
	input := []byte(`{"user": {"name": "Alex", "age": 30`)
	res := Repair(input)
	if !res.Valid {
		t.Fatalf("expected valid JSON, got: %s", string(res.Repaired))
	}

	var data struct {
		User struct {
			Name string `json:"name"`
			Age  int    `json:"age"`
		} `json:"user"`
	}
	if err := json.Unmarshal(res.Repaired, &data); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if data.User.Name != "Alex" || data.User.Age != 30 {
		t.Errorf("unexpected content: %+v", data)
	}
}

func TestRepairUnterminatedString(t *testing.T) {
	input := []byte(`{"status": "in_progres`)
	res := Repair(input)
	if !res.Valid {
		t.Fatalf("expected valid JSON, got: %s", string(res.Repaired))
	}
	if !res.Flags.ClosedOpenString {
		t.Errorf("expected ClosedOpenString flag")
	}

	var data map[string]string
	if err := json.Unmarshal(res.Repaired, &data); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if data["status"] != "in_progres" {
		t.Errorf("expected in_progres, got %s", data["status"])
	}
}

func TestRepairMarkdownFencesAndPreamble(t *testing.T) {
	input := []byte("Sure! Here is your result:\n```json\n{\"id\": 101, \"ready\": true}\n```\nLet me know if you need more.")
	res := Repair(input)
	if !res.Valid {
		t.Fatalf("expected valid JSON, got: %s", string(res.Repaired))
	}
	if !res.Flags.StrippedMarkdown {
		t.Errorf("expected StrippedMarkdown flag")
	}

	var data struct {
		ID    int  `json:"id"`
		Ready bool `json:"ready"`
	}
	if err := json.Unmarshal(res.Repaired, &data); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if data.ID != 101 || !data.Ready {
		t.Errorf("unexpected data: %+v", data)
	}
}

func TestRepairTrailingCommas(t *testing.T) {
	input := []byte(`{"items": [1, 2, ], "count": 2, }`)
	res := Repair(input)
	if !res.Valid {
		t.Fatalf("expected valid JSON, got: %s", string(res.Repaired))
	}
	if !res.Flags.RemovedTrailingComma {
		t.Errorf("expected RemovedTrailingComma flag")
	}

	var data struct {
		Items []int `json:"items"`
		Count int   `json:"count"`
	}
	if err := json.Unmarshal(res.Repaired, &data); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if len(data.Items) != 2 || data.Count != 2 {
		t.Errorf("unexpected data: %+v", data)
	}
}

func TestRepairDanglingKey(t *testing.T) {
	input := []byte(`{"valid": true, "extra": }`)
	res := Repair(input)
	if !res.Valid {
		t.Fatalf("expected valid JSON, got: %s", string(res.Repaired))
	}
	if !res.Flags.CompletedDanglingKey {
		t.Errorf("expected CompletedDanglingKey flag")
	}

	var data map[string]interface{}
	if err := json.Unmarshal(res.Repaired, &data); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if data["extra"] != nil {
		t.Errorf("expected nil for dangling key, got %v", data["extra"])
	}
}

func TestRepairPythonLiterals(t *testing.T) {
	input := []byte(`{"active": True, "disabled": False, "meta": None}`)
	res := Repair(input)
	if !res.Valid {
		t.Fatalf("expected valid JSON, got: %s", string(res.Repaired))
	}

	var data struct {
		Active   bool        `json:"active"`
		Disabled bool        `json:"disabled"`
		Meta     interface{} `json:"meta"`
	}
	if err := json.Unmarshal(res.Repaired, &data); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if !data.Active || data.Disabled || data.Meta != nil {
		t.Errorf("unexpected values: %+v", data)
	}
}

func TestRepairSingleQuotes(t *testing.T) {
	input := []byte(`{'name': 'SchemaDoctor', 'version': '1.0'}`)
	res := Repair(input)
	if !res.Valid {
		t.Fatalf("expected valid JSON, got: %s", string(res.Repaired))
	}

	var data map[string]string
	if err := json.Unmarshal(res.Repaired, &data); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if data["name"] != "SchemaDoctor" || data["version"] != "1.0" {
		t.Errorf("unexpected values: %+v", data)
	}
}

func TestRepairUnescapedQuotes(t *testing.T) {
	input := []byte(`{"comment": "User said "hello" to me", "rating": 5}`)
	res := Repair(input)
	if !res.Valid {
		t.Fatalf("expected valid JSON, got: %s", string(res.Repaired))
	}

	var data struct {
		Comment string `json:"comment"`
		Rating  int    `json:"rating"`
	}
	if err := json.Unmarshal(res.Repaired, &data); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if data.Rating != 5 {
		t.Errorf("expected rating 5, got %d", data.Rating)
	}
}

func TestAlreadyValidJSON(t *testing.T) {
	input := []byte(`{"status":"ok","code":200}`)
	res := Repair(input)
	if !res.Valid {
		t.Fatalf("expected valid JSON")
	}
	if res.Flags.Any() {
		t.Errorf("expected no flags for valid JSON")
	}
}


func BenchmarkRepairTruncated(b *testing.B) {
	input := []byte(`{"user": {"name": "Alex", "age": 30, "hobbies": ["coding", "ast", `)
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = Repair(input)
	}
}

func BenchmarkRepairMarkdown(b *testing.B) {
	input := []byte("```json\n{\"id\": 123, \"tags\": [\"ai\", \"llm\"], \"active\": True, }\n```")
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = Repair(input)
	}
}
