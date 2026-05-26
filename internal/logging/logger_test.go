package logging

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestLogger_FileOutput(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.log")
	l, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	l.Info("Hello %s", "world")
	l.Warn("This is a warning")
	l.Error("Something went wrong")

	// Read the file.
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)

	if !strings.Contains(content, "Hello world") {
		t.Fatalf("missing 'Hello world' in:\n%s", content)
	}
	if !strings.Contains(content, "[INFO]") {
		t.Fatalf("missing [INFO] in:\n%s", content)
	}
	if !strings.Contains(content, "[WARN]") {
		t.Fatalf("missing [WARN] in:\n%s", content)
	}
	if !strings.Contains(content, "[ERROR]") {
		t.Fatalf("missing [ERROR] in:\n%s", content)
	}
}

func TestLogger_EmptyPath(t *testing.T) {
	// Should not crash, just not write to file.
	l, err := New("")
	if err != nil {
		t.Fatal(err)
	}
	l.Info("no file output")
	l.Close()
}

func TestLogger_LevelFilter(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "filter.log")
	l, err := New(path)
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	l.SetLevel(LevelWarn)
	l.Debug("should be filtered")
	l.Info("should be filtered")
	l.Warn("should appear")
	l.Error("should appear")

	data, _ := os.ReadFile(path)
	content := string(data)

	if strings.Contains(content, "filtered") {
		t.Fatalf("debug/info should be filtered, got:\n%s", content)
	}
	if !strings.Contains(content, "should appear") {
		t.Fatalf("warn/error should appear")
	}
}

func TestLogger_Callback(t *testing.T) {
	var received []string
	cb := func(level Level, ts time.Time, msg string) {
		received = append(received, level.String()+":"+msg)
	}

	l, err := New("")
	if err != nil {
		t.Fatal(err)
	}
	l.SetCallback(cb)

	l.Info("callback test")
	l.Warn("warning test")

	if len(received) != 2 {
		t.Fatalf("expected 2 callbacks, got %d: %v", len(received), received)
	}
	if received[0] != "INFO:callback test" {
		t.Fatalf("unexpected: %q", received[0])
	}
	if received[1] != "WARN:warning test" {
		t.Fatalf("unexpected: %q", received[1])
	}
}

func TestResolveDatePlaceholder(t *testing.T) {
	input := "C:\\logs\\<yyyy-mm-dd>_app.log"
	result := resolveDatePlaceholder(input)
	expected := "C:\\logs\\" + time.Now().Format("2006-01-02") + "_app.log"
	if result != expected {
		t.Fatalf("expected %q, got %q", expected, result)
	}
}

func TestNopLogger(t *testing.T) {
	l := NopLogger()
	l.Debug("should not panic")
	l.Info("should not panic")
	l.Error("should not panic")
}
