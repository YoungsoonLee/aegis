package audit

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/YoungsoonLee/aegis/internal/config"
)

func TestLogger_Disabled(t *testing.T) {
	l, err := NewLogger(config.AuditConfig{Enabled: false})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer l.Close()

	// Should not panic or error on disabled logger
	l.Log(&Event{ID: "test", Timestamp: time.Now()})
}

func TestLogger_StdoutSink(t *testing.T) {
	l, err := NewLogger(config.AuditConfig{
		Enabled: true,
		Sinks:   []config.SinkConfig{{Type: "stdout"}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer l.Close()

	if len(l.sinks) != 1 {
		t.Errorf("got %d sinks, want 1", len(l.sinks))
	}
}

func TestLogger_FileSink(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.jsonl")

	l, err := NewLogger(config.AuditConfig{
		Enabled: true,
		Sinks:   []config.SinkConfig{{Type: "file", Path: path}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	event := &Event{
		ID:        "evt-001",
		Timestamp: time.Now(),
		Direction: "inbound",
		Target:    "openai",
		Method:    "POST",
		Path:      "/v1/chat/completions",
		Guards: []GuardResult{
			{Name: "pii", Action: "pass"},
			{Name: "injection", Action: "block", Details: "injection detected"},
		},
		Blocked:  true,
		Duration: 5 * time.Millisecond,
	}

	l.Log(event)
	l.Close()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read audit file: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "evt-001") {
		t.Error("audit file should contain event ID")
	}
	if !strings.Contains(content, "injection") {
		t.Error("audit file should contain guard result")
	}

	var logged Event
	if err := json.Unmarshal([]byte(strings.TrimSpace(content)), &logged); err != nil {
		t.Fatalf("audit output should be valid JSON: %v", err)
	}
	if logged.ID != "evt-001" {
		t.Errorf("logged ID = %q, want %q", logged.ID, "evt-001")
	}
	if !logged.Blocked {
		t.Error("logged event should be blocked")
	}
	if len(logged.Guards) != 2 {
		t.Errorf("logged guards count = %d, want 2", len(logged.Guards))
	}
}

func TestLogger_MultipleEvents(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "multi.jsonl")

	l, err := NewLogger(config.AuditConfig{
		Enabled: true,
		Sinks:   []config.SinkConfig{{Type: "file", Path: path}},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for i := 0; i < 5; i++ {
		l.Log(&Event{
			ID:        "evt",
			Timestamp: time.Now(),
			Path:      "/test",
		})
	}
	l.Close()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 5 {
		t.Errorf("got %d lines, want 5", len(lines))
	}
}

func TestLogger_UnknownSinkType(t *testing.T) {
	_, err := NewLogger(config.AuditConfig{
		Enabled: true,
		Sinks:   []config.SinkConfig{{Type: "elasticsearch"}},
	})
	if err == nil {
		t.Fatal("expected error for unknown sink type")
	}
}

func TestLogger_MultipleSinks(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "multi-sink.jsonl")

	l, err := NewLogger(config.AuditConfig{
		Enabled: true,
		Sinks: []config.SinkConfig{
			{Type: "file", Path: path},
			{Type: "stdout"},
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer l.Close()

	if len(l.sinks) != 2 {
		t.Errorf("got %d sinks, want 2", len(l.sinks))
	}
}

func TestWriterSink_Write(t *testing.T) {
	var buf bytes.Buffer
	s := &writerSink{w: &buf}

	event := &Event{
		ID:        "test-123",
		Timestamp: time.Now(),
		Blocked:   false,
	}

	if err := s.Write(event); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "test-123") {
		t.Error("output should contain event ID")
	}
	if !strings.HasSuffix(output, "\n") {
		t.Error("output should end with newline")
	}
}

func TestWriterSink_Close_NoCloser(t *testing.T) {
	s := &writerSink{w: &bytes.Buffer{}}
	if err := s.Close(); err != nil {
		t.Errorf("close without closer should return nil, got %v", err)
	}
}
