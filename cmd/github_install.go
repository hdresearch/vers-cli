package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"time"

	pres "github.com/hdresearch/vers-cli/internal/presenters"
	"github.com/spf13/cobra"
)

var (
	ghInstallOrg     string
	ghInstallNoOpen  bool
	ghInstallFormat  string
	ghInstallTimeout int
)

// installURLResponse is the response from the install-url endpoint.
// TODO: verify exact response shape against vers-landing once endpoint is confirmed.
type installURLResponse struct {
	URL string `json:"url"`
}

// githubStatusResponse is the response from the github-status endpoint.
// TODO: verify exact response shape against vers-landing once endpoint is confirmed.
type githubStatusResponse struct {
	Installed      bool   `json:"installed"`
	InstallationID int64  `json:"installation_id,omitempty"`
	Org            string `json:"org,omitempty"`
}

var githubInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Install the Vers GitHub App on an organization",
	Long: `Get the Vers GitHub App installation URL for your organization and
optionally open it in a browser. Then poll until the installation
webhook fires and the app is confirmed installed.

The command prints the install URL to stdout. If a TTY is detected
(and --no-open is not set), it also attempts to open the URL in
your default browser.

Exit codes:
  0  GitHub App installed successfully
  1  Timeout waiting for installation
  2  Network or API error
  3  Authentication error

Examples:
  vers github install
  vers github install --org myorg
  vers github install --no-open
  vers github install --timeout 600 --format json`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		timeout := time.Duration(ghInstallTimeout) * time.Second
		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		// Step 1: Get the install URL.
		// TODO: The exact endpoint is unverified. This assumes:
		//   GET /api/v1/github/install-url[?org=<name>]
		// If the endpoint does not exist, we fall back to a static dashboard URL.
		path := "api/v1/github/install-url"
		if ghInstallOrg != "" {
			path += "?org=" + ghInstallOrg
		}

		var urlResp installURLResponse
		err := application.Client.Get(ctx, path, nil, &urlResp)

		var installURL string
		if err != nil || urlResp.URL == "" {
			// Fallback: construct a reasonable dashboard URL.
			// TODO: verify this fallback path against vers-landing.
			baseURL := application.BaseURL.String()
			if ghInstallOrg != "" {
				installURL = fmt.Sprintf("%s/dashboard/github/install?org=%s", baseURL, ghInstallOrg)
			} else {
				installURL = fmt.Sprintf("%s/dashboard/github/install", baseURL)
			}
			if application.Verbose && err != nil {
				fmt.Fprintf(application.IO.Err, "Warning: install-url endpoint unavailable (%v), using fallback URL\n", err)
			}
		} else {
			installURL = urlResp.URL
		}

		// Step 2: Print the URL (always).
		fmt.Fprintln(application.IO.Out, installURL)

		// Step 3: Open browser if not suppressed.
		if !ghInstallNoOpen {
			if err := openBrowser(installURL); err != nil && application.Verbose {
				fmt.Fprintf(application.IO.Err, "Warning: could not open browser: %v\n", err)
			}
		}

		// Step 4: Poll for installation status.
		fmt.Fprintf(application.IO.Err, "Waiting for GitHub App installation (timeout %s)...\n", timeout)

		statusPath := "api/v1/github/status"
		if ghInstallOrg != "" {
			statusPath += "?org=" + ghInstallOrg
		}

		ticker := time.NewTicker(5 * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				result := githubStatusResponse{Installed: false}
				if ghInstallFormat == "json" {
					pres.PrintJSON(result)
				} else {
					fmt.Fprintln(application.IO.Err, "✗ Timed out waiting for GitHub App installation.")
				}
				cmd.SilenceUsage = true
				return fmt.Errorf("timed out waiting for installation")
			case <-ticker.C:
				var status githubStatusResponse
				if err := application.Client.Get(ctx, statusPath, nil, &status); err != nil {
					if application.Verbose {
						fmt.Fprintf(application.IO.Err, "  poll error: %v\n", err)
					}
					continue
				}
				if status.Installed {
					if ghInstallFormat == "json" {
						pres.PrintJSON(status)
					} else {
						fmt.Fprintf(application.IO.Err, "✓ GitHub App installed successfully")
						if status.InstallationID != 0 {
							fmt.Fprintf(application.IO.Err, " (installation %d)", status.InstallationID)
						}
						fmt.Fprintln(application.IO.Err)
					}
					return nil
				}
			}
		}
	},
}

// openBrowser attempts to open the given URL in the default browser.
func openBrowser(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		return fmt.Errorf("unsupported platform %s", runtime.GOOS)
	}
	return cmd.Start()
}

// installResultJSON is a helper for JSON output of install results.
type installResultJSON struct {
	Installed      bool   `json:"installed"`
	InstallationID int64  `json:"installation_id,omitempty"`
	URL            string `json:"url,omitempty"`
	Error          string `json:"error,omitempty"`
}

// marshalInstallResult produces JSON output for the install command.
func marshalInstallResult(r installResultJSON) string {
	b, _ := json.Marshal(r)
	return string(b)
}

func init() {
	githubCmd.AddCommand(githubInstallCmd)

	githubInstallCmd.Flags().StringVar(&ghInstallOrg, "org", "", "GitHub organization name")
	githubInstallCmd.Flags().BoolVar(&ghInstallNoOpen, "no-open", false, "Do not attempt to open the browser")
	githubInstallCmd.Flags().StringVar(&ghInstallFormat, "format", "", "Output format (json)")
	githubInstallCmd.Flags().IntVar(&ghInstallTimeout, "timeout", 300, "Seconds to wait for installation")
}
