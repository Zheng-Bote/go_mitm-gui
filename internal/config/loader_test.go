package config

import (
	"os"
	"testing"

	"github.com/zheng-bote/go_mitm-gui/internal/crypto"
)

// sampleINI is a minimal INI that exercises all major fields
// (based on the PRD format).
const sampleINI = `[GLOBAL]
Admins = "q225265", "zb_bamboo"
log_level = "DEBUG"
log_file_path = "C:\logs\<yyyy-mm-dd>_app.log"

[HR]
INPUT_EXCEL = true
INPUT_CSV = true
INPUT_ORA-DB = false
INPUT_PG-DB = false
INPUT_KAFKA = false
audit_log_enabled = true
audit_log_file_path = "C:\logs\<yyyy-mm-dd>_hr_audit.log"
JSON_SCHEMA_PATH = "C:\schemas\hr_schema_v1.json"
UPLOAD_URL = "http://localhost:8000/api/hr/upload"

[HR-INPUT-EXCEL]
DEFAULT_FILE_PATH = "C:\input\hr_data.xlsx"

[HR-INPUT-CSV]
DEFAULT_FILE_PATH = "C:\input\hr_data.csv"

[HR-INPUT-ORA-DB]
HOST = "dbhost"
PORT = 1521
SERVICE_NAME = "orcl"
USERNAME = "dbuser"
PASSWORD = "dbpass"

[HR-INPUT-KAFKA]
BOOTSTRAP_SERVERS = "localhost:9092"
TOPIC = "hr-topic"
GROUP_ID = "hr-group"
KEY = "hr-key"
SECRET = "hr-secret"
security_protocol = "SASL_SSL"
sasl_mechanism = "PLAIN"
auto_offset_reset = "earliest"

[ORG]
INPUT_EXCEL = true
INPUT_CSV = true
INPUT_ORA-DB = false
INPUT_PG-DB = false
INPUT_KAFKA = false
audit_log_enabled = true
audit_log_file_path = "C:\logs\<yyyy-mm-dd>_org_audit.log"
JSON_SCHEMA_PATH = "C:\schemas\org_schema.json"
UPLOAD_URL = "http://localhost:8000/api/org/upload"

[ORG-INPUT-EXCEL]
DEFAULT_FILE_PATH = "C:\input\org_data.xlsx"

[ORG-INPUT-CSV]
DEFAULT_FILE_PATH = "C:\input\org_data.csv"
`

func TestParsePlainConfig(t *testing.T) {
	loader := New()
	cfg, err := loader.LoadPlainConfig([]byte(sampleINI))
	if err != nil {
		t.Fatalf("LoadPlainConfig failed: %v", err)
	}

	// Verify GLOBAL.
	if len(cfg.Global.Admins) != 2 {
		t.Fatalf("expected 2 admins, got %d: %v", len(cfg.Global.Admins), cfg.Global.Admins)
	}
	if cfg.Global.Admins[0] != "q225265" {
		t.Fatalf("expected admin[0]=q225265, got %q", cfg.Global.Admins[0])
	}
	if cfg.Global.Admins[1] != "zb_bamboo" {
		t.Fatalf("expected admin[1]=zb_bamboo, got %q", cfg.Global.Admins[1])
	}
	if cfg.Global.LogLevel != "DEBUG" {
		t.Fatalf("expected log_level=DEBUG, got %q", cfg.Global.LogLevel)
	}

	// Verify HR topic exists.
	hr, ok := cfg.Topics["HR"]
	if !ok {
		t.Fatal("expected HR topic in config")
	}

	if !hr.InputExcel {
		t.Fatal("expected HR.InputExcel=true")
	}
	if !hr.InputCSV {
		t.Fatal("expected HR.InputCSV=true")
	}
	if hr.InputOraDB {
		t.Fatal("expected HR.InputOraDB=false")
	}
	if !hr.AuditLogEnabled {
		t.Fatal("expected HR.AuditLogEnabled=true")
	}
	if hr.JSONSchemaPath != "C:\\schemas\\hr_schema_v1.json" {
		t.Fatalf("expected JSONSchemaPath, got %q", hr.JSONSchemaPath)
	}
	if hr.UploadURL != "http://localhost:8000/api/hr/upload" {
		t.Fatalf("expected UploadURL, got %q", hr.UploadURL)
	}
	if hr.ExcelInput == nil {
		t.Fatal("expected HR.ExcelInput to be parsed")
	} else if hr.ExcelInput.DefaultFilePath != "C:\\input\\hr_data.xlsx" {
		t.Fatalf("unexpected Excel path: %q", hr.ExcelInput.DefaultFilePath)
	}
	if hr.CSVInput == nil {
		t.Fatal("expected HR.CSVInput to be parsed")
	}

	// DB/Kafka input configs should be nil because flags are false.
	if hr.OraDBInput != nil {
		t.Fatal("expected HR.OraDBInput=nil since INPUT_ORA-DB=false")
	}
	if hr.PGDBInput != nil {
		t.Fatal("expected HR.PGDBInput=nil")
	}
	if hr.KafkaInput != nil {
		t.Fatal("expected HR.KafkaInput=nil since INPUT_KAFKA=false")
	}

	// Verify ORG topic exists.
	org, ok := cfg.Topics["ORG"]
	if !ok {
		t.Fatal("expected ORG topic in config")
	}
	if !org.InputExcel {
		t.Fatal("expected ORG.InputExcel=true")
	}
	if org.UploadURL != "http://localhost:8000/api/org/upload" {
		t.Fatalf("unexpected ORG UploadURL: %q", org.UploadURL)
	}
}

func TestLoadConfig_EncryptedRoundTrip(t *testing.T) {
	loader := New()
	masterPassword := "test-master-pwd"

	// Encrypt the sample INI.
	encrypted, err := crypto.Encrypt(masterPassword, []byte(sampleINI))
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	// Write to temp file.
	tmpFile, err := os.CreateTemp(t.TempDir(), "*.enc")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tmpFile.Write(encrypted); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	// Load via LoadConfig.
	cfg, err := loader.LoadConfig(tmpFile.Name(), masterPassword)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Basic verification.
	if len(cfg.Topics) != 2 {
		t.Fatalf("expected 2 topics (HR, ORG), got %d", len(cfg.Topics))
	}
	if cfg.Topics["HR"].UploadURL != "http://localhost:8000/api/hr/upload" {
		t.Fatalf("unexpected HR UploadURL after encrypted round-trip: %q",
			cfg.Topics["HR"].UploadURL)
	}
}

func TestLoadConfig_WrongPassword(t *testing.T) {
	loader := New()
	masterPassword := "correct-password"

	encrypted, err := crypto.Encrypt(masterPassword, []byte("[GLOBAL]\nlog_level = INFO\n"))
	if err != nil {
		t.Fatal(err)
	}

	tmpFile, err := os.CreateTemp(t.TempDir(), "*.enc")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := tmpFile.Write(encrypted); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	_, err = loader.LoadConfig(tmpFile.Name(), "wrong-password")
	if err == nil {
		t.Fatal("expected error for wrong password, got nil")
	}
}

func TestLoadConfig_EmptyFile(t *testing.T) {
	loader := New()
	_, err := loader.LoadPlainConfig([]byte{})
	if err == nil {
		t.Fatal("expected error for empty INI, got nil")
	}
}
