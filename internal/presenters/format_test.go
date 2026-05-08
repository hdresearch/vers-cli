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
		name     string
		quiet    bool
		jsonFlag bool
		format   string
		expected presenters.OutputFormat
		wantErr  bool
	}{
		{"default", false, false, "", presenters.FormatDefault, false},
		{"quiet only", true, false, "", presenters.FormatQuiet, false},
		{"json flag", false, true, "", presenters.FormatJSON, false},
		{"legacy format=json", false, false, "json", presenters.FormatJSON, false},
		{"quiet beats json flag", true, true, "", presenters.FormatQuiet, false},
		{"quiet beats format=json", true, false, "json", presenters.FormatQuiet, false},
		{"json flag + format=json (both ok)", false, true, "json", presenters.FormatJSON, false},
		{"invalid format value", false, false, "yaml", presenters.FormatDefault, true},
		{"invalid format value with json flag also set", false, true, "yaml", presenters.FormatDefault, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := presenters.ParseFormat(tt.quiet, tt.jsonFlag, tt.format)
			if (err != nil) != tt.wantErr {
				t.Fatalf("ParseFormat err=%v, wantErr=%v", err, tt.wantErr)
			}
			if !tt.wantErr && got != tt.expected {
				t.Errorf("ParseFormat(quiet=%v, json=%v, format=%q) = %v, want %v",
					tt.quiet, tt.jsonFlag, tt.format, got, tt.expected)
			}
			if tt.wantErr && err != nil {
				// error message should enumerate the valid value and mention deprecation
				msg := err.Error()
				if !strings.Contains(msg, `"json"`) || !strings.Contains(msg, "deprecated") {
					t.Errorf("error message missing valid set or deprecation note: %q", msg)
				}
			}
		})
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
