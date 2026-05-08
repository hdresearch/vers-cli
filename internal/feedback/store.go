// Package feedback implements local journaling and opt-in upstream POST for
// the `vers feedback` command (F18 Phase A).
//
// Storage is an append-only JSONL file at ~/.vers/feedback.jsonl by default.
// Upstream delivery is opt-in via the VERS_FEEDBACK_ENDPOINT env var; when
// set, each new entry is POSTed best-effort with a 5s timeout.
package feedback

import (
	"bufio"
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base32"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// EndpointEnvVar is the env var used to enable upstream POST.
const EndpointEnvVar = "VERS_FEEDBACK_ENDPOINT"

// PathEnvVar overrides the default journal path. Primarily for tests.
const PathEnvVar = "VERS_FEEDBACK_PATH"

// Entry is one record in the local JSONL journal.
type Entry struct {
	ID           string `json:"id"`
	Timestamp    string `json:"timestamp"`
	VersVersion  string `json:"vers_version"`
	Message      string `json:"message"`
	SentUpstream bool   `json:"sent_upstream"`
}

// upstreamPayload is the shape POSTed to VERS_FEEDBACK_ENDPOINT. The
// sent_upstream flag is intentionally journal-only.
type upstreamPayload struct {
	ID          string `json:"id"`
	Timestamp   string `json:"timestamp"`
	VersVersion string `json:"vers_version"`
	Message     string `json:"message"`
}

// DefaultPath returns the default journal path (~/.vers/feedback.jsonl),
// honoring VERS_FEEDBACK_PATH when set.
func DefaultPath() (string, error) {
	if p := os.Getenv(PathEnvVar); p != "" {
		return p, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	return filepath.Join(home, ".vers", "feedback.jsonl"), nil
}

// NewID returns a short random base32 identifier (10 chars, lowercase, no
// padding). Used as the entry id.
func NewID() (string, error) {
	var buf [6]byte
	if _, err := rand.Read(buf[:]); err != nil {
		return "", fmt.Errorf("read random: %w", err)
	}
	enc := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(buf[:])
	// 6 random bytes -> 10 base32 chars.
	return strings.ToLower(enc), nil
}

// Append writes an entry as a single JSON line to path, creating parent
// directories as needed.
func Append(path string, e Entry) error {
	if path == "" {
		return fmt.Errorf("feedback: empty journal path")
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create journal dir: %w", err)
	}
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open journal: %w", err)
	}
	defer f.Close()
	b, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("marshal entry: %w", err)
	}
	b = append(b, '\n')
	if _, err := f.Write(b); err != nil {
		return fmt.Errorf("write journal: %w", err)
	}
	return nil
}

// ReadAll loads every entry from the journal, oldest first. A missing file
// returns an empty slice and no error.
func ReadAll(path string) ([]Entry, error) {
	if path == "" {
		return nil, fmt.Errorf("feedback: empty journal path")
	}
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("open journal: %w", err)
	}
	defer f.Close()

	var out []Entry
	scanner := bufio.NewScanner(f)
	// Allow generously-sized lines (long messages).
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		var e Entry
		if err := json.Unmarshal(line, &e); err != nil {
			return nil, fmt.Errorf("parse journal line %d: %w", lineNum, err)
		}
		out = append(out, e)
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan journal: %w", err)
	}
	return out, nil
}

// PostUpstream POSTs the entry payload to endpoint as application/json with a
// 5s timeout. It returns the HTTP status code on success. On any failure
// (transport, non-2xx) it returns a non-nil error along with the status code
// (0 if the request never completed).
func PostUpstream(ctx context.Context, endpoint string, e Entry) (int, error) {
	payload := upstreamPayload{
		ID:          e.ID,
		Timestamp:   e.Timestamp,
		VersVersion: e.VersVersion,
		Message:     e.Message,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return 0, fmt.Errorf("marshal payload: %w", err)
	}

	reqCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return 0, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()
	// Drain so connections can be reused.
	_, _ = io.Copy(io.Discard, resp.Body)
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return resp.StatusCode, fmt.Errorf("upstream returned status %d", resp.StatusCode)
	}
	return resp.StatusCode, nil
}

// Now returns the current time as an RFC3339 string in UTC.
func Now() string {
	return time.Now().UTC().Format(time.RFC3339)
}
