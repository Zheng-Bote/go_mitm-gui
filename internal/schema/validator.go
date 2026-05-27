// Package schema provides JSON Schema validation for loaded data.
package schema

import (
	"encoding/json"
	"fmt"
	"runtime"
	"sync"

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
		if len(d) == 0 {
			result.Warnings = append(result.Warnings, "JSON array is empty — nothing to validate")
			break
		}

		numWorkers := runtime.NumCPU()
		if numWorkers > len(d) {
			numWorkers = len(d)
		}

		jobs := make(chan struct {
			index int
			item  interface{}
		}, len(d))
		
		var mu sync.Mutex
		var wg sync.WaitGroup

		for w := 0; w < numWorkers; w++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				for job := range jobs {
					if err := sch.Validate(job.item); err != nil {
						mu.Lock()
						result.Valid = false
						result.Errors = append(result.Errors, formatValidationErrors(err, job.index)...)
						mu.Unlock()
					}
				}
			}()
		}

		for i, item := range d {
			jobs <- struct {
				index int
				item  interface{}
			}{i, item}
		}
		close(jobs)
		wg.Wait()

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
