package presenters_test

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/hdresearch/vers-cli/internal/presenters"
)

func TestParseFormat(t *testing.T) {
	tests := []struct {
		quiet    bool
		format   string
		expected presenters.OutputFormat
	}{
		{false, "", presenters.FormatDefault},
		{true, "", presenters.FormatQuiet},
		{false, "json", presenters.FormatJSON},
		{true, "json", presenters.FormatQuiet}, // quiet takes precedence
	}

	for _, tt := range tests {
		got := presenters.ParseFormat(tt.quiet, tt.format)
		if got != tt.expected {
			t.Errorf("ParseFormat(quiet=%v, format=%q) = %v, want %v", tt.quiet, tt.format, got, tt.expected)
		}
	}
}

func TestPrintQuiet(t *testing.T) {
	out := captureStdout(t, func() {
		presenters.PrintQuiet([]string{"abc-123", "def-456", "ghi-789"})
	})

	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d: %q", len(lines), out)
	}
	if lines[0] != "abc-123" {
		t.Errorf("line 0 = %q, want abc-123", lines[0])
	}
	if lines[1] != "def-456" {
		t.Errorf("line 1 = %q, want def-456", lines[1])
	}
	if lines[2] != "ghi-789" {
		t.Errorf("line 2 = %q, want ghi-789", lines[2])
	}
}

func TestPrintQuietEmpty(t *testing.T) {
	out := captureStdout(t, func() {
		presenters.PrintQuiet([]string{})
	})
	if strings.TrimSpace(out) != "" {
		t.Errorf("expected empty output, got %q", out)
	}
}

func TestPrintJSON(t *testing.T) {
	data := map[string]string{"id": "abc-123", "name": "test"}
	out := captureStdout(t, func() {
		presenters.PrintJSON(data)
	})

	var parsed map[string]string
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, out)
	}
	if parsed["id"] != "abc-123" {
		t.Errorf("expected id=abc-123, got %s", parsed["id"])
	}
}

func TestPrintJSONArray(t *testing.T) {
	data := []string{"vm-1", "vm-2"}
	out := captureStdout(t, func() {
		presenters.PrintJSON(data)
	})

	var parsed []string
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v\nOutput: %s", err, out)
	}
	if len(parsed) != 2 || parsed[0] != "vm-1" {
		t.Errorf("unexpected parsed result: %v", parsed)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	fn()
	_ = w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	_ = r.Close()
	return buf.String()
}
