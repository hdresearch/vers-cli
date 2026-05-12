package cmd

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/hdresearch/vers-cli/internal/auth"
	pres "github.com/hdresearch/vers-cli/internal/presenters"
	"github.com/spf13/cobra"
)

var (
	ghMintOrg          string
	ghMintRepo         string
	ghMintFormat       string
	ghMintRepositories []string
	ghMintPermissions  []string
)

// mintTokenRequest is the POST body for the vers-landing installation-token endpoint.
type mintTokenRequest struct {
	Repositories []string          `json:"repositories,omitempty"`
	Permissions  map[string]string `json:"permissions,omitempty"`
}

// mintTokenResponse is the response from the vers-landing installation-token endpoint.
type mintTokenResponse struct {
	Token               string          `json:"token"`
	ExpiresAt           string          `json:"expires_at"`
	Permissions         json.RawMessage `json:"permissions,omitempty"`
	Repositories        json.RawMessage `json:"repositories,omitempty"`
	RepositorySelection string          `json:"repository_selection,omitempty"`
	InstallationID      int64           `json:"installation_id,omitempty"`
	OrgID               string          `json:"org_id,omitempty"`
}

var githubMintTokenCmd = &cobra.Command{
	Use:   "mint-token",
	Short: "Mint a short-lived GitHub installation access token",
	Long: `Mint a short-lived GitHub App installation access token via the
Vers platform. The token is printed to stdout for piping into other
commands (e.g. git clone, gh CLI, curl).

The token is scoped to the GitHub App installation on your
organization. You can optionally restrict it to specific
repositories or permissions.

Tokens typically expire in ~1 hour (GitHub-side policy). Mint a
fresh token per task rather than caching.

Exit codes:
  0  Token minted successfully
  1  No GitHub App installation found (install first with "vers github install")
  2  API or network error

Examples:
  vers github mint-token
  vers github mint-token --format json
  vers github mint-token --repo my-repo
  vers github mint-token --repo my-repo --repo another-repo
  vers github mint-token --permission contents=read --permission pull_requests=write
  TOKEN=$(vers github mint-token)
  git clone https://x-access-token:${TOKEN}@github.com/myorg/myrepo.git`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		apiCtx, cancel := context.WithTimeout(context.Background(), application.Timeouts.APIMedium)
		defer cancel()

		// Build request body.
		reqBody := mintTokenRequest{}

		// --repo flags: extract repo name (strip org/ prefix if present).
		if ghMintRepo != "" {
			// Legacy single --repo flag support.
			ghMintRepositories = append(ghMintRepositories, ghMintRepo)
		}
		for _, r := range ghMintRepositories {
			// Strip "org/" prefix if present; the API wants bare repo names.
			if idx := strings.Index(r, "/"); idx >= 0 {
				r = r[idx+1:]
			}
			reqBody.Repositories = append(reqBody.Repositories, r)
		}

		// --permission flags: "key=value" pairs.
		if len(ghMintPermissions) > 0 {
			reqBody.Permissions = make(map[string]string, len(ghMintPermissions))
			for _, p := range ghMintPermissions {
				parts := strings.SplitN(p, "=", 2)
				if len(parts) != 2 {
					return fmt.Errorf("invalid --permission %q: expected key=value (e.g. contents=read)", p)
				}
				reqBody.Permissions[parts[0]] = parts[1]
			}
		}

		resp, err := doMintToken(apiCtx, reqBody)
		if err != nil {
			cmd.SilenceUsage = true
			return err
		}

		format := pres.ParseFormat(false, ghMintFormat)
		switch format {
		case pres.FormatJSON:
			pres.PrintJSON(resp)
		default:
			// Raw token to stdout for piping.
			fmt.Fprint(application.IO.Out, resp.Token)
			// Add newline only if stdout is a TTY.
			if f, ok := application.IO.Out.(*os.File); ok {
				if fi, err := f.Stat(); err == nil && fi.Mode()&os.ModeCharDevice != 0 {
					fmt.Fprintln(application.IO.Out)
				}
			}
		}
		return nil
	},
}

// doMintToken calls the vers-landing installation-token endpoint directly.
// This endpoint lives on the landing/dashboard app (vers.sh by default),
// NOT on api.vers.sh. The landing base URL is resolved via
// auth.GetVersLandingURL() (VERS_LANDING_URL env override, otherwise
// DEFAULT_VERS_LANDING_URL_STR).
func doMintToken(ctx context.Context, req mintTokenRequest) (*mintTokenResponse, error) {
	// Get the API key (same key works for both api.vers.sh and vers.sh).
	apiKey, err := auth.GetAPIKey()
	if err != nil || apiKey == "" {
		return nil, fmt.Errorf("authentication required: run vers login first")
	}

	// Resolve the vers-landing base URL (VERS_LANDING_URL or default).
	landingURL, err := auth.GetVersLandingURL()
	if err != nil {
		return nil, fmt.Errorf("failed to resolve landing URL: %w", err)
	}

	endpoint := strings.TrimRight(landingURL.String(), "/") + "/api/github/installation-token"

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+apiKey)

	httpResp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("API request failed: %w", err)
	}
	defer httpResp.Body.Close()

	respBody, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	switch {
	case httpResp.StatusCode == 401 || httpResp.StatusCode == 403:
		return nil, fmt.Errorf("authentication failed (HTTP %d): check your API key", httpResp.StatusCode)
	case httpResp.StatusCode == 404:
		return nil, fmt.Errorf("no GitHub App installation found for your organization: run vers github install first")
	case httpResp.StatusCode == 503:
		return nil, fmt.Errorf("GitHub App not configured on the Vers server (HTTP 503)")
	case httpResp.StatusCode >= 400:
		return nil, fmt.Errorf("API error (HTTP %d): %s", httpResp.StatusCode, string(respBody))
	}

	var tokenResp mintTokenResponse
	if err := json.Unmarshal(respBody, &tokenResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if tokenResp.Token == "" {
		return nil, fmt.Errorf("API returned empty token")
	}

	return &tokenResp, nil
}

func init() {
	githubCmd.AddCommand(githubMintTokenCmd)

	githubMintTokenCmd.Flags().StringVar(&ghMintOrg, "org", "", "Organization name (usually auto-detected from API key)")
	githubMintTokenCmd.Flags().StringVar(&ghMintRepo, "repo", "", "Restrict token to a specific repository (owner/repo or repo)")
	githubMintTokenCmd.Flags().StringArrayVar(&ghMintRepositories, "repositories", nil, "Restrict token to specific repositories (bare names)")
	githubMintTokenCmd.Flags().StringArrayVar(&ghMintPermissions, "permission", nil, "Permission in key=value format (e.g. contents=read)")
	githubMintTokenCmd.Flags().StringVar(&ghMintFormat, "format", "", "Output format (json)")
}
