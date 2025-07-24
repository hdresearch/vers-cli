package update

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hdresearch/vers-cli/internal/config"
)

// GitHubRelease represents a GitHub release
type GitHubRelease struct {
	TagName    string `json:"tag_name"`
	Name       string `json:"name"`
	Body       string `json:"body"`
	Draft      bool   `json:"draft"`
	Prerelease bool   `json:"prerelease"`
	Assets     []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
		Size               int64  `json:"size"`
	} `json:"assets"`
	PublishedAt time.Time `json:"published_at"`
}

// CheckForUpdates checks if there's a new version available
func CheckForUpdates(currentVersion, repository string, verbose bool) (bool, string, error) {
	// Skip check for dev versions
	currentVersion = strings.TrimPrefix(currentVersion, "v")
	if currentVersion == "dev" || currentVersion == "unknown" {
		if verbose {
			fmt.Printf("[DEBUG] Skipping update check for development version\n")
		}
		return false, "", nil
	}

	// Get latest release
	latest, err := GetLatestRelease(repository, false, verbose)
	if err != nil {
		if verbose {
			fmt.Printf("[DEBUG] Failed to check for updates: %v\n", err)
		}
		return false, "", nil // Don't error out - just skip the check
	}

	latestVersion := strings.TrimPrefix(latest.TagName, "v")
	if verbose {
		fmt.Printf("[DEBUG] Current: %s, Latest: %s\n", currentVersion, latestVersion)
	}

	// Check if there's an update available
	hasUpdate := currentVersion != latestVersion
	return hasUpdate, latest.TagName, nil
}

// GetLatestRelease fetches the latest release from GitHub
// If includePrerelease is true, it will return the latest release including prereleases
func GetLatestRelease(repository string, includePrerelease bool, verbose bool) (*GitHubRelease, error) {
	// Extract owner/repo from Repository constant
	repoURL := strings.TrimPrefix(repository, "https://github.com/")

	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/releases", repoURL)
	if !includePrerelease {
		apiURL += "/latest"
	}

	if verbose {
		fmt.Printf("[DEBUG] Fetching release info from: %s\n", apiURL)
	}

	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch release info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status: %d", resp.StatusCode)
	}

	if includePrerelease {
		var releases []GitHubRelease
		if err := json.NewDecoder(resp.Body).Decode(&releases); err != nil {
			return nil, fmt.Errorf("failed to decode release info: %w", err)
		}

		// Find the latest release (including prereleases)
		for _, release := range releases {
			if !release.Draft {
				return &release, nil
			}
		}
		return nil, fmt.Errorf("no releases found")
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, fmt.Errorf("failed to decode release info: %w", err)
	}

	return &release, nil
}

// ShouldCheckForUpdate determines if it's time to check for updates
func ShouldCheckForUpdate() bool {
	cliConfig, err := config.LoadCLIConfig()
	if err != nil {
		return false
	}

	return cliConfig.ShouldCheckForUpdate()
}

// UpdateCheckTime updates the next check time
func UpdateCheckTime() {
	cliConfig, err := config.LoadCLIConfig()
	if err != nil {
		return
	}

	cliConfig.SetNextCheckTime()
	config.SaveCLIConfig(cliConfig)
}
