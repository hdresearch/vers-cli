package builder

import (
	"bufio"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
)

// streamResponse mirrors the orchestrator's NDJSON exec stream format.
type streamResponse struct {
	Type     string `json:"type"`
	Stream   string `json:"stream,omitempty"`
	DataB64  string `json:"data_b64,omitempty"`
	ExitCode *int   `json:"exit_code,omitempty"`
	Code     string `json:"code,omitempty"`
	Message  string `json:"message,omitempty"`
}

// streamOutput consumes an exec NDJSON stream, writing stdout/stderr and
// returning the exit code. Identical in semantics to the handler-local
// version in internal/handlers/execute.go; duplicated here to keep the
// builder package dependency-free of the handler tree.
func streamOutput(body io.Reader, stdout, stderr io.Writer) (int, error) {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)

	exitCode := 0
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var r streamResponse
		if err := json.Unmarshal(line, &r); err != nil {
			continue
		}
		switch r.Type {
		case "chunk":
			data, err := base64.StdEncoding.DecodeString(r.DataB64)
			if err != nil {
				continue
			}
			switch r.Stream {
			case "stdout":
				_, _ = stdout.Write(data)
			case "stderr":
				_, _ = stderr.Write(data)
			}
		case "exit":
			if r.ExitCode != nil {
				exitCode = *r.ExitCode
			}
			return exitCode, nil
		case "error":
			return 1, fmt.Errorf("exec error [%s]: %s", r.Code, r.Message)
		}
	}
	if err := scanner.Err(); err != nil {
		return 1, fmt.Errorf("stream read: %w", err)
	}
	return exitCode, nil
}
