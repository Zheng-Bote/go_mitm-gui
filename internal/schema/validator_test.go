package schema

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zheng-bote/go_mitm-gui/internal/model"
)

const sampleSchema = `{
  "$schema": "https://json-schema.org/draft/2020-12/schema",
  "type": "object",
  "properties": {
    "EmployeeNumber": { "type": "string", "pattern": "^[0-9]{6,8}$" },
    "FirstName": { "type": "string", "minLength": 2, "maxLength": 255 },
    "LastName": { "type": "string", "minLength": 2, "maxLength": 255 },
    "Email": { "type": "string", "format": "email" }
  },
  "required": ["EmployeeNumber", "FirstName", "LastName"]
}`

func writeSchema(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "schema.json")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestValidate_ValidData(t *testing.T) {
	schemaPath := writeSchema(t, sampleSchema)
	v := New()
	jsonData := []byte(`[
		{"EmployeeNumber":"100001","FirstName":"John","LastName":"Doe","Email":"john@example.com"},
		{"EmployeeNumber":"100002","FirstName":"Jane","LastName":"Smith"}
	]`)
	result, err := v.Validate(jsonData, schemaPath)
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}
	if !result.Valid {
		t.Fatalf("expected valid, got errors: %v", result.Errors)
	}
}

func TestValidate_InvalidData(t *testing.T) {
	schemaPath := writeSchema(t, sampleSchema)
	v := New()
	jsonData := []byte(`[
		{"EmployeeNumber":"12","LastName":"Doe"},
		{"EmployeeNumber":"100002","FirstName":"J","LastName":"Smith"}
	]`)
	result, err := v.Validate(jsonData, schemaPath)
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}
	if result.Valid {
		t.Fatal("expected invalid, got valid")
	}
	if len(result.Errors) == 0 {
		t.Fatal("expected validation errors, got none")
	}
}

func TestValidate_EmptyArray(t *testing.T) {
	schemaPath := writeSchema(t, sampleSchema)
	v := New()
	result, err := v.Validate([]byte(`[]`), schemaPath)
	if err != nil {
		t.Fatalf("Validate failed: %v", err)
	}
	if !result.Valid {
		t.Fatalf("expected valid empty array: %v", result.Errors)
	}
}

func TestValidate_InvalidJSON(t *testing.T) {
	schemaPath := writeSchema(t, sampleSchema)
	v := New()
	_, err := v.Validate([]byte(`not valid json`), schemaPath)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestValidate_MissingSchema(t *testing.T) {
	v := New()
	_, err := v.Validate([]byte(`{}`), "/nonexistent/schema.json")
	if err == nil {
		t.Fatal("expected error for missing schema")
	}
}

func TestValidate_LoadedData(t *testing.T) {
	schemaPath := writeSchema(t, sampleSchema)
	v := New()
	data := &model.LoadedData{
		JSONPayload: []byte(`[{"EmployeeNumber":"100001","FirstName":"John","LastName":"Doe"}]`),
	}
	result, err := v.ValidateLoaded(data, schemaPath)
	if err != nil {
		t.Fatalf("ValidateLoaded failed: %v", err)
	}
	if !result.Valid {
		t.Fatalf("expected valid: %v", result.Errors)
	}
}
