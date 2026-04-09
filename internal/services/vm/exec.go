package vm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/hdresearch/vers-cli/internal/auth"
)

// ExecRequest matches the orchestrator's VmExecRequest.
type ExecRequest struct {
	Command    []string          `json:"command"`
	Env        map[string]string `json:"env,omitempty"`
	WorkingDir string            `json:"working_dir,omitempty"`
	Stdin      string            `json:"stdin,omitempty"`
	TimeoutSec uint64            `json:"timeout_secs,omitempty"`
}

// ExecResponse matches the orchestrator's VmExecResponse.
type ExecResponse struct {
	ExitCode int    `json:"exit_code"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
}

// ExecStreamChunk is a single line from the NDJSON exec stream.
type ExecStreamChunk struct {
	Type     string `json:"type"`                // "chunk" or "exit"
	Stream   string `json:"stream,omitempty"`    // "stdout" or "stderr"
	Data     string `json:"data,omitempty"`      // base64-encoded bytes
	ExitCode *int   `json:"exit_code,omitempty"` // only on type=="exit"
}

// Exec runs a command on a VM via the orchestrator API (non-streaming).
func Exec(ctx context.Context, vmID string, req ExecRequest) (*ExecResponse, error) {
	apiKey, err := auth.GetAPIKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}

	baseURL, err := auth.GetVersUrl()
	if err != nil {
		return nil, fmt.Errorf("failed to get API URL: %w", err)
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/vm/%s/exec", baseURL.String(), vmID)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		errBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(errBody))
	}

	var result ExecResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result, nil
}

// ExecStream runs a command on a VM via the orchestrator streaming API.
// It returns the response body for the caller to consume as NDJSON.
func ExecStream(ctx context.Context, vmID string, req ExecRequest) (io.ReadCloser, error) {
	apiKey, err := auth.GetAPIKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}

	baseURL, err := auth.GetVersUrl()
	if err != nil {
		return nil, fmt.Errorf("failed to get API URL: %w", err)
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/vm/%s/exec/stream", baseURL.String(), vmID)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Authorization", "Bearer "+apiKey)
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		defer resp.Body.Close()
		errBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(errBody))
	}

	return resp.Body, nil
}
