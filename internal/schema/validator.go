// Package schema provides JSON Schema validation for loaded data.
package schema

import (
	"encoding/json"
	"fmt"

	"github.com/santhosh-tekuri/jsonschema/v5"

	"github.com/zheng-bote/go_mitm-gui/internal/model"
)

type Validator struct{}

func New() *Validator { return &Validator{} }

func (v *Validator) Validate(jsonData []byte, schemaPath string) (*model.ValidationResult, error) {
	sch, err := jsonschema.Compile(schemaPath)
	if err != nil {
		return nil, fmt.Errorf("schema: failed to compile %q: %w", schemaPath, err)
	}

	var data interface{}
	if err := json.Unmarshal(jsonData, &data); err != nil {
		return nil, fmt.Errorf("schema: failed to parse JSON data: %w", err)
	}

	result := &model.ValidationResult{
		Valid:    true,
		Errors:   []string{},
		Warnings: []string{},
	}

	switch d := data.(type) {
	case []interface{}:
		for i, item := range d {
			if err := sch.Validate(item); err != nil {
				result.Valid = false
				result.Errors = append(result.Errors, formatValidationErrors(err, i)...)
			}
		}
		if len(d) == 0 {
			result.Warnings = append(result.Warnings, "JSON array is empty — nothing to validate")
		}
	default:
		if err := sch.Validate(data); err != nil {
			result.Valid = false
			result.Errors = append(result.Errors, formatValidationErrors(err, -1)...)
		}
	}

	if result.Valid {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("All records passed validation against %q", schemaPath))
	}

	return result, nil
}

func (v *Validator) ValidateLoaded(data *model.LoadedData, schemaPath string) (*model.ValidationResult, error) {
	return v.Validate(data.JSONPayload, schemaPath)
}

func formatValidationErrors(err error, rowIndex int) []string {
	var msgs []string
	prefix := ""
	if rowIndex >= 0 {
		prefix = fmt.Sprintf("Row %d: ", rowIndex+1)
	}

	switch e := err.(type) {
	case *jsonschema.ValidationError:
		msgs = append(msgs, prefix+e.Error())
	case interface{ Unwrap() []error }:
		for _, sub := range e.Unwrap() {
			msgs = append(msgs, formatValidationErrors(sub, rowIndex)...)
		}
	default:
		msgs = append(msgs, prefix+err.Error())
	}
	return msgs
}
