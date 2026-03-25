package env

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/hdresearch/vers-cli/internal/auth"
	vers "github.com/hdresearch/vers-sdk-go"
)

// EnvVarsResponse matches the API response structure
type EnvVarsResponse struct {
	Vars map[string]string `json:"vars"`
}

// SetEnvVarsRequest matches the API request structure
type SetEnvVarsRequest struct {
	Vars    map[string]string `json:"vars"`
	Replace bool              `json:"replace"`
}

// makeRequest is a helper to make authenticated HTTP requests
func makeRequest(ctx context.Context, method, path string, body io.Reader) (*http.Response, error) {
	apiKey, err := auth.GetAPIKey()
	if err != nil {
		return nil, fmt.Errorf("failed to get API key: %w", err)
	}

	baseURL, err := auth.GetVersUrl()
	if err != nil {
		return nil, fmt.Errorf("failed to get API URL: %w", err)
	}

	u, err := url.Parse(baseURL.String() + path)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, method, u.String(), body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if resp.StatusCode >= 400 {
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("API error %d: %s", resp.StatusCode, string(body))
	}

	return resp, nil
}

// ListEnvVars retrieves all environment variables for the current user
func ListEnvVars(ctx context.Context, client *vers.Client) (map[string]string, error) {
	resp, err := makeRequest(ctx, "GET", "/api/v1/env_vars", nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var envResp EnvVarsResponse
	if err := json.NewDecoder(resp.Body).Decode(&envResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return envResp.Vars, nil
}

// SetEnvVar sets a single environment variable
func SetEnvVar(ctx context.Context, client *vers.Client, key, value string) error {
	reqBody := SetEnvVarsRequest{
		Vars:    map[string]string{key: value},
		Replace: false, // upsert mode
	}

	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := makeRequest(ctx, "PUT", "/api/v1/env_vars", bytes.NewReader(reqBytes))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// DeleteEnvVar deletes an environment variable by key
func DeleteEnvVar(ctx context.Context, client *vers.Client, key string) error {
	resp, err := makeRequest(ctx, "DELETE", fmt.Sprintf("/api/v1/env_vars/%s", key), nil)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}

// ReplaceAllEnvVars replaces all environment variables with the provided set
func ReplaceAllEnvVars(ctx context.Context, client *vers.Client, vars map[string]string) error {
	reqBody := SetEnvVarsRequest{
		Vars:    vars,
		Replace: true, // replace mode
	}

	reqBytes, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	resp, err := makeRequest(ctx, "PUT", "/api/v1/env_vars", bytes.NewReader(reqBytes))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	return nil
}
