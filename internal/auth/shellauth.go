package auth

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ShellAuthInitRequest is the request body for POST /api/shell-auth.
type ShellAuthInitRequest struct {
	Email        string `json:"email"`
	SSHPublicKey string `json:"ssh_public_key"`
	Body         string `json:"body,omitempty"`
}

// ShellAuthInitResponse is the response from POST /api/shell-auth.
type ShellAuthInitResponse struct {
	Success         bool   `json:"success"`
	UserID          string `json:"user_id"`
	Email           string `json:"email"`
	IsNewUser       bool   `json:"is_new_user"`
	Nonce           string `json:"nonce"`
	SSHPublicKey    string `json:"ssh_public_key"`
	AlreadyVerified bool   `json:"already_verified"`
}

// ShellAuthVerifyRequest is the request body for POST /api/shell-auth/verify-key.
type ShellAuthVerifyRequest struct {
	Email        string `json:"email"`
	SSHPublicKey string `json:"ssh_public_key"`
}

// ShellAuthOrg represents an organization in the verify response.
type ShellAuthOrg struct {
	OrgID string `json:"org_id"`
	Name  string `json:"name"`
	Role  string `json:"role"`
}

// ShellAuthVerifyResponse is the response from POST /api/shell-auth/verify-key.
type ShellAuthVerifyResponse struct {
	Verified bool           `json:"verified"`
	IsActive bool           `json:"is_active"`
	UserID   string         `json:"user_id"`
	KeyID    string         `json:"key_id"`
	Orgs     []ShellAuthOrg `json:"orgs"`
	Reason   string         `json:"reason,omitempty"`
}

// ShellAuthCreateKeyRequest is the request body for POST /api/shell-auth/api-keys.
type ShellAuthCreateKeyRequest struct {
	Email        string `json:"email"`
	SSHPublicKey string `json:"ssh_public_key"`
	Label        string `json:"label"`
	OrgName      string `json:"org_name,omitempty"`
}

// ShellAuthCreateKeyResponse is the response from POST /api/shell-auth/api-keys.
type ShellAuthCreateKeyResponse struct {
	Success  bool   `json:"success"`
	APIKey   string `json:"api_key"`
	APIKeyID string `json:"api_key_id"`
	OrgID    string `json:"org_id"`
	OrgName  string `json:"org_name"`
}

// ShellAuthErrorResponse captures error details from the API.
type ShellAuthErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// GetGitEmail returns the user's git config email.
func GetGitEmail() (string, error) {
	out, err := exec.Command("git", "config", "user.email").Output()
	if err != nil {
		return "", fmt.Errorf("could not get git email (is git configured?): %w", err)
	}
	email := strings.TrimSpace(string(out))
	if email == "" {
		return "", fmt.Errorf("git user.email is not set — run: git config --global user.email \"you@example.com\"")
	}
	return email, nil
}

// FindSSHPublicKey finds the user's SSH public key, checking common locations.
// Returns the key contents (not the path).
func FindSSHPublicKey() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("could not find home directory: %w", err)
	}

	// Check common SSH key paths in preference order
	candidates := []string{
		filepath.Join(home, ".ssh", "id_ed25519.pub"),
		filepath.Join(home, ".ssh", "id_ecdsa.pub"),
		filepath.Join(home, ".ssh", "id_rsa.pub"),
	}

	for _, path := range candidates {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		key := strings.TrimSpace(string(data))
		if key != "" {
			return key, nil
		}
	}

	return "", fmt.Errorf("no SSH public key found — checked: %s\nGenerate one with: ssh-keygen -t ed25519", strings.Join(candidates, ", "))
}

// validSSHKeyTypes are the accepted SSH public key type prefixes.
var validSSHKeyTypes = map[string]bool{
	"ssh-ed25519":                        true,
	"ssh-rsa":                            true,
	"ecdsa-sha2-nistp256":                true,
	"ecdsa-sha2-nistp384":                true,
	"ecdsa-sha2-nistp521":                true,
	"sk-ssh-ed25519@openssh.com":         true,
	"sk-ecdsa-sha2-nistp256@openssh.com": true,
}

// ReadAndValidateSSHPublicKey reads an SSH public key from a file path and validates it.
// Returns the key contents on success.
func ReadAndValidateSSHPublicKey(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("SSH key file not found: %s", path)
	}
	if info.IsDir() {
		return "", fmt.Errorf("SSH key path is a directory, not a file: %s", path)
	}
	if info.Size() > 16*1024 {
		return "", fmt.Errorf("SSH key file too large (%d bytes) — expected a public key", info.Size())
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("failed to read SSH key file: %w", err)
	}

	key := strings.TrimSpace(string(data))
	if key == "" {
		return "", fmt.Errorf("SSH key file is empty: %s", path)
	}

	// Validate format: should be "<type> <base64-data> [comment]"
	parts := strings.Fields(key)
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid SSH public key format in %s — expected \"<type> <base64-data> [comment]\"", path)
	}

	if !validSSHKeyTypes[parts[0]] {
		return "", fmt.Errorf("unrecognized SSH key type %q in %s — expected one of: ssh-ed25519, ssh-rsa, ecdsa-sha2-*", parts[0], path)
	}

	return key, nil
}

// shellAuthBaseURL returns the base URL for shell auth endpoints.
func shellAuthBaseURL() (string, error) {
	versURL, err := GetVersUrl()
	if err != nil {
		return "", err
	}
	// The shell-auth endpoints are on the main site, not the API subdomain.
	// e.g. https://api.vers.sh -> https://vers.sh
	host := versURL.Hostname()
	scheme := versURL.Scheme
	if strings.HasPrefix(host, "api.") {
		host = strings.TrimPrefix(host, "api.")
	}
	return fmt.Sprintf("%s://%s", scheme, host), nil
}

// shellAuthPost makes a POST request to a shell-auth endpoint.
func shellAuthPost(path string, body interface{}, result interface{}) error {
	baseURL, err := shellAuthBaseURL()
	if err != nil {
		return err
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}

	url := baseURL + path
	if os.Getenv("VERS_VERBOSE") == "true" {
		fmt.Printf("[DEBUG] POST %s\n", url)
	}

	resp, err := http.Post(url, "application/json", bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		var errResp ShellAuthErrorResponse
		if json.Unmarshal(respBody, &errResp) == nil && (errResp.Error != "" || errResp.Message != "") {
			msg := errResp.Error
			if msg == "" {
				msg = errResp.Message
			}
			return fmt.Errorf("API error (%d): %s", resp.StatusCode, msg)
		}
		return fmt.Errorf("API error (%d): %s", resp.StatusCode, string(respBody))
	}

	if err := json.Unmarshal(respBody, result); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}

	return nil
}

// ShellAuthInitiate starts the shell auth flow.
func ShellAuthInitiate(email, sshPubKey string) (*ShellAuthInitResponse, error) {
	hostname, _ := os.Hostname()
	req := ShellAuthInitRequest{
		Email:        email,
		SSHPublicKey: sshPubKey,
		Body:         fmt.Sprintf("vers login --git from %s", hostname),
	}
	var resp ShellAuthInitResponse
	if err := shellAuthPost("/api/shell-auth", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}

// ShellAuthCheckVerification does a single verification check (no polling).
// Use when you already know the key is verified and just need the org list.
func ShellAuthCheckVerification(email, sshPubKey string) (*ShellAuthVerifyResponse, error) {
	req := ShellAuthVerifyRequest{
		Email:        email,
		SSHPublicKey: sshPubKey,
	}
	var resp ShellAuthVerifyResponse
	if err := shellAuthPost("/api/shell-auth/verify-key", req, &resp); err != nil {
		return nil, err
	}
	if !resp.Verified {
		return nil, fmt.Errorf("expected key to be verified but it was not")
	}
	return &resp, nil
}

// ShellAuthPollVerification polls until the email is verified or timeout.
// Returns the verify response with orgs on success.
func ShellAuthPollVerification(email, sshPubKey string, timeout time.Duration) (*ShellAuthVerifyResponse, error) {
	deadline := time.Now().Add(timeout)
	req := ShellAuthVerifyRequest{
		Email:        email,
		SSHPublicKey: sshPubKey,
	}

	for time.Now().Before(deadline) {
		var resp ShellAuthVerifyResponse
		if err := shellAuthPost("/api/shell-auth/verify-key", req, &resp); err != nil {
			// 401 means not yet verified — keep polling
			if !strings.Contains(err.Error(), "401") {
				return nil, err
			}
		}
		if resp.Verified {
			return &resp, nil
		}
		time.Sleep(3 * time.Second)
	}

	return nil, fmt.Errorf("verification timed out after %s — check your email and try again", timeout)
}

// ShellAuthCreateAPIKey creates an API key for the verified user.
func ShellAuthCreateAPIKey(email, sshPubKey, label, orgName string) (*ShellAuthCreateKeyResponse, error) {
	req := ShellAuthCreateKeyRequest{
		Email:        email,
		SSHPublicKey: sshPubKey,
		Label:        label,
		OrgName:      orgName,
	}
	var resp ShellAuthCreateKeyResponse
	if err := shellAuthPost("/api/shell-auth/api-keys", req, &resp); err != nil {
		return nil, err
	}
	return &resp, nil
}
