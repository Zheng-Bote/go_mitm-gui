package input

import (
	"os"
	"testing"
)

const sampleCSV = `EmployeeNumber,FirstName,LastName,Email
100001,John,Doe,john.doe@example.com
100002,Jane,Smith,jane.smith@example.com
100003,Bob,Johnson,bob.j@example.com`

func TestCSVSource_Load(t *testing.T) {
	// Write sample CSV to temp file.
	tmpFile, err := os.CreateTemp(t.TempDir(), "*.csv")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tmpFile.WriteString(sampleCSV); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	source := &CSVSource{}
	data, err := source.Load(tmpFile.Name())
	if err != nil {
		t.Fatalf("CSVSource.Load failed: %v", err)
	}

	if data.Format != "CSV" {
		t.Fatalf("expected format CSV, got %q", data.Format)
	}
	if data.RowCount != 3 {
		t.Fatalf("expected 3 rows, got %d", data.RowCount)
	}
	if len(data.Fields) != 4 {
		t.Fatalf("expected 4 fields, got %d: %v", len(data.Fields), data.Fields)
	}
	if len(data.JSONPayload) == 0 {
		t.Fatal("expected non-empty JSON payload")
	}
}

func TestCSVSource_NoDataRows(t *testing.T) {
	tmpFile, err := os.CreateTemp(t.TempDir(), "*.csv")
	if err != nil {
		t.Fatal(err)
	}
	tmpFile.WriteString("header1,header2\n")
	tmpFile.Close()

	source := &CSVSource{}
	_, err = source.Load(tmpFile.Name())
	if err == nil {
		t.Fatal("expected error for no data rows, got nil")
	}
}

func TestNewDataSource(t *testing.T) {
	if src := NewDataSource("CSV", nil); src == nil {
		t.Fatal("expected CSV source, got nil")
	}
	if src := NewDataSource("Excel", nil); src == nil {
		t.Fatal("expected Excel source, got nil")
	}
	if src := NewDataSource("Oracle DB", nil); src != nil {
		t.Fatal("expected nil for unsupported format")
	}
}
