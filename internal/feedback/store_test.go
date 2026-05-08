package feedback

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewID(t *testing.T) {
	id1, err := NewID()
	if err != nil {
		t.Fatalf("NewID: %v", err)
	}
	if got := len(id1); got < 8 || got > 12 {
		t.Errorf("expected id length in 8..12, got %d (%q)", got, id1)
	}
	if id1 != strings.ToLower(id1) {
		t.Errorf("expected lowercase id, got %q", id1)
	}
	id2, err := NewID()
	if err != nil {
		t.Fatalf("NewID: %v", err)
	}
	if id1 == id2 {
		t.Errorf("expected unique ids, got duplicate %q", id1)
	}
}

func TestAppendAndReadAll(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "nested", "feedback.jsonl")

	entries := []Entry{
		{ID: "a1", Timestamp: "2026-05-07T00:00:00Z", VersVersion: "v1", Message: "hi", SentUpstream: false},
		{ID: "b2", Timestamp: "2026-05-07T00:00:01Z", VersVersion: "v1", Message: "two\nlines", SentUpstream: true},
	}
	for _, e := range entries {
		if err := Append(path, e); err != nil {
			t.Fatalf("Append: %v", err)
		}
	}

	got, err := ReadAll(path)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if len(got) != len(entries) {
		t.Fatalf("expected %d entries, got %d", len(entries), len(got))
	}
	for i, e := range entries {
		if got[i] != e {
			t.Errorf("entry %d: want %+v, got %+v", i, e, got[i])
		}
	}

	// Each line in the file must be a single valid JSON object.
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read raw: %v", err)
	}
	for i, line := range strings.Split(strings.TrimRight(string(raw), "\n"), "\n") {
		var e Entry
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			t.Errorf("line %d not valid json: %v", i, err)
		}
	}
}

func TestReadAllMissingFile(t *testing.T) {
	dir := t.TempDir()
	got, err := ReadAll(filepath.Join(dir, "nope.jsonl"))
	if err != nil {
		t.Fatalf("ReadAll on missing file: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty slice, got %d entries", len(got))
	}
}

func TestDefaultPathHonorsEnv(t *testing.T) {
	t.Setenv(PathEnvVar, "/tmp/custom-feedback.jsonl")
	p, err := DefaultPath()
	if err != nil {
		t.Fatalf("DefaultPath: %v", err)
	}
	if p != "/tmp/custom-feedback.jsonl" {
		t.Errorf("expected env override, got %q", p)
	}
}

func TestDefaultPathFallsBackToHome(t *testing.T) {
	t.Setenv(PathEnvVar, "")
	p, err := DefaultPath()
	if err != nil {
		t.Fatalf("DefaultPath: %v", err)
	}
	if !strings.HasSuffix(p, filepath.Join(".vers", "feedback.jsonl")) {
		t.Errorf("expected path to end with .vers/feedback.jsonl, got %q", p)
	}
}

func TestPostUpstreamSuccess(t *testing.T) {
	var gotBody []byte
	var gotCT string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotCT = r.Header.Get("Content-Type")
		gotBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	e := Entry{ID: "x", Timestamp: "t", VersVersion: "v", Message: "m", SentUpstream: false}
	status, err := PostUpstream(context.Background(), srv.URL, e)
	if err != nil {
		t.Fatalf("PostUpstream: %v", err)
	}
	if status != 200 {
		t.Errorf("expected 200, got %d", status)
	}
	if gotCT != "application/json" {
		t.Errorf("expected application/json content-type, got %q", gotCT)
	}

	// Body must not include sent_upstream.
	var payload map[string]interface{}
	if err := json.Unmarshal(gotBody, &payload); err != nil {
		t.Fatalf("unmarshal posted body: %v", err)
	}
	if _, ok := payload["sent_upstream"]; ok {
		t.Errorf("sent_upstream must not be POSTed; payload=%v", payload)
	}
	for _, k := range []string{"id", "timestamp", "vers_version", "message"} {
		if _, ok := payload[k]; !ok {
			t.Errorf("missing %q in payload: %v", k, payload)
		}
	}
}

func TestPostUpstreamNon2xx(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	status, err := PostUpstream(context.Background(), srv.URL, Entry{ID: "x"})
	if err == nil {
		t.Errorf("expected error on 500, got nil")
	}
	if status != 500 {
		t.Errorf("expected status 500, got %d", status)
	}
}

func TestPostUpstreamTransportError(t *testing.T) {
	// Non-routable address should fail fast.
	status, err := PostUpstream(context.Background(), "http://127.0.0.1:1/", Entry{ID: "x"})
	if err == nil {
		t.Errorf("expected transport error, got nil")
	}
	if status != 0 {
		t.Errorf("expected status 0 on transport error, got %d", status)
	}
}
