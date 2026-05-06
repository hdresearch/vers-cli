package update

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
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

// IsDevVersion reports whether a version string represents a local/dev build
// for which we should skip update checks entirely.
//
// This catches:
//   - the literal sentinels "dev" / "unknown" / "" set when ldflags weren't applied
//   - "dev-<sha>" / "*-dirty" produced by our init() fallback against `git describe`
//   - Go module *pseudo-versions* of the form "v0.0.0-<timestamp>-<commit>",
//     which are what `go install` and `go run` produce against an untagged
//     commit. Without this, integration test runs that build via the module
//     graph end up with a perfectly valid-looking semver and try to "upgrade"
//     to whatever real release is currently latest, contaminating test output.
func IsDevVersion(v string) bool {
	stripped := strings.TrimPrefix(v, "v")
	if stripped == "" || stripped == "dev" || stripped == "unknown" {
		return true
	}
	if strings.HasPrefix(stripped, "dev-") || strings.Contains(stripped, "-dirty") {
		return true
	}
	// Go pseudo-versions all start with "0.0.0-" (untagged repo) or
	// "X.Y.Z-0.<timestamp>-<commit>" (post-tag). The first form is the
	// only one we hit in practice (the release pipeline sets a real
	// version via ldflags), so checking that prefix is sufficient.
	if strings.HasPrefix(stripped, "0.0.0-") {
		return true
	}
	return false
}

// CheckForUpdates checks if there's a new version available.
// This is a thin wrapper around CheckForUpdatesContext that uses a default
// 5s timeout for backward compatibility with callers that don't manage
// their own context (e.g. `vers upgrade`).
func CheckForUpdates(currentVersion, repository string, verbose bool) (bool, string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return CheckForUpdatesContext(ctx, currentVersion, repository, verbose)
}

// CheckForUpdatesContext checks if there's a new version available, honoring
// the supplied context's deadline/cancellation.
func CheckForUpdatesContext(ctx context.Context, currentVersion, repository string, verbose bool) (bool, string, error) {
	if IsDevVersion(currentVersion) {
		if verbose {
			fmt.Printf("[DEBUG] Skipping update check for development version %q\n", currentVersion)
		}
		return false, "", nil
	}

	latest, err := GetLatestReleaseContext(ctx, repository, false, verbose)
	if err != nil {
		if verbose {
			fmt.Printf("[DEBUG] Failed to check for updates: %v\n", err)
		}
		return false, "", err
	}

	current := strings.TrimPrefix(currentVersion, "v")
	latestVersion := strings.TrimPrefix(latest.TagName, "v")
	if verbose {
		fmt.Printf("[DEBUG] Current: %s, Latest: %s\n", current, latestVersion)
	}

	return current != latestVersion, latest.TagName, nil
}

// GetLatestRelease fetches the latest release from GitHub.
// If includePrerelease is true, it will return the latest release including
// prereleases. Uses a default 5s timeout.
func GetLatestRelease(repository string, includePrerelease bool, verbose bool) (*GitHubRelease, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return GetLatestReleaseContext(ctx, repository, includePrerelease, verbose)
}

// GetLatestReleaseContext is like GetLatestRelease but honors the supplied context.
func GetLatestReleaseContext(ctx context.Context, repository string, includePrerelease bool, verbose bool) (*GitHubRelease, error) {
	// Extract owner/repo from Repository constant
	repoURL := strings.TrimPrefix(repository, "https://github.com/")

	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/releases", repoURL)
	if !includePrerelease {
		apiURL += "/latest"
	}

	if verbose {
		fmt.Printf("[DEBUG] Fetching release info from: %s\n", apiURL)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build release request: %w", err)
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req)
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

// isNewerSemver returns true if `latest` is a strictly higher version than
// `current` using a tolerant lexical comparison after stripping a leading
// "v". Falls back to plain string inequality if either side fails to parse
// as dotted integers, which preserves the old behavior.
func isNewerSemver(current, latest string) bool {
	c := strings.TrimPrefix(current, "v")
	l := strings.TrimPrefix(latest, "v")
	if c == "" || l == "" || c == l {
		return false
	}

	// Strip pre-release/build metadata for the numeric comparison.
	stripMeta := func(s string) string {
		if i := strings.IndexAny(s, "-+"); i >= 0 {
			return s[:i]
		}
		return s
	}

	cp := strings.Split(stripMeta(c), ".")
	lp := strings.Split(stripMeta(l), ".")
	parseInt := func(s string) (int, bool) {
		n := 0
		if s == "" {
			return 0, false
		}
		for _, r := range s {
			if r < '0' || r > '9' {
				return 0, false
			}
			n = n*10 + int(r-'0')
		}
		return n, true
	}

	for i := 0; i < len(cp) || i < len(lp); i++ {
		var ci, li int
		var ok bool
		if i < len(cp) {
			if ci, ok = parseInt(cp[i]); !ok {
				return c != l // unparseable -> fall back
			}
		}
		if i < len(lp) {
			if li, ok = parseInt(lp[i]); !ok {
				return c != l
			}
		}
		if li > ci {
			return true
		}
		if li < ci {
			return false
		}
	}
	return false
}

// MaybeNotifyUpdate prints an "update available" message to the given writer
// when a newer release is known. It is designed to be called once during CLI
// startup and is cheap on the hot path:
//
//   - If a previously-cached LatestVersion is newer than `current`, the
//     message prints synchronously with no network I/O.
//   - Otherwise, if it's been longer than the configured check interval since
//     the last successful check, it performs a single bounded HTTP request
//     (capped at `timeout`) to refresh the cache. The cache (and NextCheck
//     timestamp) are only advanced on a successful response, so transient
//     network failures don't suppress the nag for an hour.
//
// Errors are intentionally swallowed — the update check must never break a
// real command. When verbose is true, debug output is written to stderr.
func MaybeNotifyUpdate(ctx context.Context, current, repository string, timeout time.Duration, verbose bool) {
	// Escape hatches for CI / scripted use. Either of these silences the
	// nag and skips all network I/O. Mirrors NO_UPDATE_NOTIFIER (npm) and
	// HOMEBREW_NO_AUTO_UPDATE (brew), which are well-known patterns.
	if envFlagSet("VERS_NO_UPDATE_CHECK") || envFlagSet("NO_UPDATE_NOTIFIER") {
		if verbose {
			fmt.Fprintf(os.Stderr, "[DEBUG] update: skipped (VERS_NO_UPDATE_CHECK / NO_UPDATE_NOTIFIER set)\n")
		}
		return
	}
	if IsDevVersion(current) {
		if verbose {
			fmt.Fprintf(os.Stderr, "[DEBUG] update: skipped (dev/pseudo version %q)\n", current)
		}
		return
	}

	cliConfig, err := config.LoadCLIConfig()
	if err != nil {
		if verbose {
			fmt.Fprintf(os.Stderr, "[DEBUG] update: failed to load CLI config: %v\n", err)
		}
		return
	}

	// 1. Fast path: print from cache if we already know about a newer release.
	cached := cliConfig.UpdateCheck.LatestVersion
	printedCached := false
	if cached != "" && isNewerSemver(current, cached) {
		printUpdateBanner(current, cached)
		printedCached = true
	}

	if !cliConfig.ShouldCheckForUpdate() {
		return
	}

	// 2. Slow path: refresh from GitHub with a tight timeout.
	fetchCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	latest, err := GetLatestReleaseContext(fetchCtx, repository, false, verbose)
	if err != nil {
		if verbose {
			fmt.Fprintf(os.Stderr, "[DEBUG] update: refresh failed: %v\n", err)
		}
		// Don't fully bump NextCheck on failure — try again soon (5min backoff)
		// so a flaky network at the moment of first launch doesn't suppress
		// the nag for the full check interval.
		cliConfig.UpdateCheck.NextCheck = time.Now().Add(5 * time.Minute)
		_ = config.SaveCLIConfig(cliConfig)
		return
	}

	cliConfig.UpdateCheck.LatestVersion = latest.TagName
	cliConfig.SetNextCheckTime()
	if err := config.SaveCLIConfig(cliConfig); err != nil && verbose {
		fmt.Fprintf(os.Stderr, "[DEBUG] update: failed to save CLI config: %v\n", err)
	}

	// If the refresh revealed a newer version that the fast path didn't
	// already print, print now.
	if !printedCached && isNewerSemver(current, latest.TagName) {
		printUpdateBanner(current, latest.TagName)
	}
}

func printUpdateBanner(current, latest string) {
	fmt.Fprintf(os.Stderr, "💡 vers update available: %s -> %s (run 'vers upgrade')\n\n", current, latest)
}

// envFlagSet returns true if the env var is set to a "truthy" value.
// Treats unset, empty string, "0", "false", "no", "off" (case-insensitive)
// as false and anything else as true. This matches the convention used by
// most CLI tools and avoids surprises like VERS_NO_UPDATE_CHECK=0 being
// interpreted as "yes, suppress".
func envFlagSet(name string) bool {
	v := strings.TrimSpace(os.Getenv(name))
	if v == "" {
		return false
	}
	switch strings.ToLower(v) {
	case "0", "false", "no", "off":
		return false
	}
	return true
}
