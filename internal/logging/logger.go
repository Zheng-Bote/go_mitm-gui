// Package logging provides dual logging: file + optional GUI callback.
package logging

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Level int

const (
	LevelDebug Level = iota
	LevelInfo
	LevelWarn
	LevelError
)

func (l Level) String() string {
	switch l {
	case LevelDebug:
		return "DEBUG"
	case LevelInfo:
		return "INFO"
	case LevelWarn:
		return "WARN"
	case LevelError:
		return "ERROR"
	}
	return "UNKNOWN"
}

func ParseLevel(s string) Level {
	switch strings.ToUpper(strings.TrimSpace(s)) {
	case "DEBUG":
		return LevelDebug
	case "INFO":
		return LevelInfo
	case "WARN", "WARNING":
		return LevelWarn
	case "ERROR":
		return LevelError
	}
	return LevelInfo
}

type LogCallback func(level Level, timestamp time.Time, message string)

type Logger struct {
	mu          sync.RWMutex
	file        *os.File
	fileLog     *log.Logger
	callback    LogCallback
	level       Level
	currentPath string
}

func New(path string) (*Logger, error) {
	l := &Logger{level: LevelInfo, currentPath: path}
	if path != "" {
		if err := l.openFile(path); err != nil {
			return nil, err
		}
	}
	return l, nil
}

func (l *Logger) openFile(path string) error {
	resolved := resolveDatePlaceholder(path)
	dir := filepath.Dir(resolved)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("logging: cannot create log dir %q: %w", dir, err)
	}
	f, err := os.OpenFile(resolved, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return fmt.Errorf("logging: cannot open log file %q: %w", resolved, err)
	}
	l.file = f
	l.fileLog = log.New(f, "", 0)
	l.currentPath = path
	return nil
}

// Reconfigure switches the log file path and level. The old file is closed.
func (l *Logger) Reconfigure(path string, level Level) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.file != nil {
		l.file.Close()
		l.file = nil
		l.fileLog = nil
	}
	l.level = level

	if path != "" {
		resolved := resolveDatePlaceholder(path)
		if l.currentPath != "" && resolveDatePlaceholder(l.currentPath) == resolved {
			return nil
		}
		if err := l.openFile(path); err != nil {
			l.level = LevelError
			l.currentPath = ""
			return err
		}
		l.logLocked(LevelInfo, "Log file switched to: %s", resolved)
	}
	return nil
}

func resolveDatePlaceholder(path string) string {
	return strings.ReplaceAll(path, "<yyyy-mm-dd>", time.Now().Format("2006-01-02"))
}

func (l *Logger) SetLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

func (l *Logger) SetCallback(cb LogCallback) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.callback = cb
}

func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	if l.file != nil {
		return l.file.Close()
	}
	return nil
}

func (l *Logger) Debug(format string, args ...interface{}) { l.log(LevelDebug, format, args...) }
func (l *Logger) Info(format string, args ...interface{})  { l.log(LevelInfo, format, args...) }
func (l *Logger) Warn(format string, args ...interface{})  { l.log(LevelWarn, format, args...) }
func (l *Logger) Error(format string, args ...interface{}) { l.log(LevelError, format, args...) }

func (l *Logger) log(level Level, format string, args ...interface{}) {
	l.mu.RLock()
	minLevel := l.level
	fl := l.fileLog
	cb := l.callback
	l.mu.RUnlock()

	if level < minLevel {
		return
	}
	now := time.Now()
	msg := fmt.Sprintf(format, args...)
	line := fmt.Sprintf("%s [%s] %s\n", now.Format("2006-01-02 15:04:05.000"), level.String(), msg)
	if fl != nil {
		fl.Print(line)
	}
	if cb != nil {
		cb(level, now, msg)
	}
}

func (l *Logger) logLocked(level Level, format string, args ...interface{}) {
	now := time.Now()
	msg := fmt.Sprintf(format, args...)
	line := fmt.Sprintf("%s [%s] %s\n", now.Format("2006-01-02 15:04:05.000"), level.String(), msg)
	if l.fileLog != nil {
		l.fileLog.Print(line)
	}
	if l.callback != nil {
		l.callback(level, now, msg)
	}
}

func NopLogger() *Logger {
	return &Logger{level: LevelError}
}

func (l *Logger) Writer(level Level) io.Writer {
	return &logWriter{logger: l, level: level}
}

type logWriter struct {
	logger *Logger
	level  Level
}

func (w *logWriter) Write(p []byte) (int, error) {
	msg := strings.TrimRight(string(p), "\n\r")
	if msg != "" {
		w.logger.log(w.level, "%s", msg)
	}
	return len(p), nil
}

// AuditLogger creates a simple per-topic audit log file writer.
type AuditLogger struct {
	path    string
	enabled bool
	mu      sync.Mutex
}

func NewAuditLogger(path string, enabled bool) *AuditLogger {
	return &AuditLogger{path: path, enabled: enabled}
}

func (a *AuditLogger) Log(event, details string) {
	if !a.enabled || a.path == "" {
		return
	}
	a.mu.Lock()
	defer a.mu.Unlock()

	resolved := resolveDatePlaceholder(a.path)
	dir := filepath.Dir(resolved)
	os.MkdirAll(dir, 0755)

	f, err := os.OpenFile(resolved, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()

	line := fmt.Sprintf("%s [%s] %s\n", time.Now().Format("2006-01-02 15:04:05.000"), event, details)
	f.WriteString(line)
}

func (a *AuditLogger) Close() {}
