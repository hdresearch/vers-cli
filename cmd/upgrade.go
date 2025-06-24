package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	confirmation "github.com/hdresearch/vers-cli/internal/utils"
	"github.com/spf13/cobra"
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

var upgradeCmd = &cobra.Command{
	Use:   "upgrade",
	Short: "Upgrade vers CLI to the latest version",
	Long: `Check for and install the latest version of the vers CLI from GitHub releases.

This command will:
- Check the current version against the latest GitHub release
- Download and install the new version if available
- Verify the download using SHA256 checksums
- Preserve the current binary as a backup

Examples:
  vers upgrade                    # Check and upgrade to latest version
  vers upgrade --check-only       # Only check for updates, don't install
  vers upgrade --prerelease       # Include pre-release versions
  vers upgrade --skip-checksum    # Skip checksum verification (not recommended)`,
	RunE: runUpgrade,
}

var (
	checkOnly    bool
	prerelease   bool
	skipChecksum bool
)

func init() {
	rootCmd.AddCommand(upgradeCmd)

	upgradeCmd.Flags().BoolVar(&checkOnly, "check-only", false, "Only check for updates without installing")
	upgradeCmd.Flags().BoolVar(&prerelease, "prerelease", false, "Include pre-release versions")
	upgradeCmd.Flags().BoolVar(&skipChecksum, "skip-checksum", false, "Skip SHA256 checksum verification (not recommended)")
}

func runUpgrade(cmd *cobra.Command, args []string) error {
	DebugPrint("Starting upgrade process\n")

	// Get current version
	currentVersion := strings.TrimPrefix(Version, "v")
	if currentVersion == "dev" || currentVersion == "unknown" {
		return fmt.Errorf("cannot upgrade development or unknown versions")
	}

	fmt.Printf("Current version: %s\n", Version)

	// Check for latest release
	latest, err := getLatestRelease()
	if err != nil {
		return fmt.Errorf("failed to check for updates: %w", err)
	}

	latestVersion := strings.TrimPrefix(latest.TagName, "v")
	fmt.Printf("Latest version: %s\n", latest.TagName)

	// Compare versions
	if currentVersion == latestVersion {
		fmt.Println("You are already running the latest version!")
		return nil
	}

	if checkOnly {
		fmt.Printf("A new version is available: %s -> %s\n", Version, latest.TagName)
		fmt.Println("Run 'vers upgrade' to install the update.")
		return nil
	}

	// Confirm upgrade using shared confirmation utility
	fmt.Printf("\nUpgrade from %s to %s?\n", Version, latest.TagName)
	if !confirmation.AskConfirmation() {
		fmt.Println("Upgrade cancelled.")
		return nil
	}

	// Perform upgrade
	return performUpgrade(latest)
}

func getLatestRelease() (*GitHubRelease, error) {
	// Extract owner/repo from Repository constant
	repoURL := strings.TrimPrefix(Repository, "https://github.com/")

	apiURL := fmt.Sprintf("https://api.github.com/repos/%s/releases", repoURL)
	if !prerelease {
		apiURL += "/latest"
	}

	DebugPrint("Fetching release info from: %s\n", apiURL)

	resp, err := http.Get(apiURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch release info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("GitHub API returned status: %d", resp.StatusCode)
	}

	if prerelease {
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

func performUpgrade(release *GitHubRelease) error {
	// Find the appropriate binary for current platform
	binaryName := getBinaryName()
	var binaryURL, checksumURL string
	var binarySize int64

	for _, asset := range release.Assets {
		if asset.Name == binaryName {
			binaryURL = asset.BrowserDownloadURL
			binarySize = asset.Size
		}
		if asset.Name == binaryName+".sha256" {
			checksumURL = asset.BrowserDownloadURL
		}
	}

	if binaryURL == "" {
		return fmt.Errorf("no compatible binary found for %s-%s", runtime.GOOS, runtime.GOARCH)
	}

	DebugPrint("Downloading binary from: %s\n", binaryURL)
	fmt.Printf("Downloading %s...\n", binaryName)

	// Download the new binary
	tempFile, err := downloadFile(binaryURL, binarySize)
	if err != nil {
		return fmt.Errorf("failed to download update: %w", err)
	}
	defer os.Remove(tempFile)

	// Verify checksum if available and not skipped
	if checksumURL != "" && !skipChecksum {
		DebugPrint("Verifying checksum from: %s\n", checksumURL)
		fmt.Println("Verifying download integrity...")

		if err := verifyChecksum(tempFile, checksumURL); err != nil {
			return fmt.Errorf("checksum verification failed: %w", err)
		}
		fmt.Println("✓ Checksum verification passed")
	} else if skipChecksum {
		fmt.Println("⚠️  Skipping checksum verification (not recommended)")
	}

	// Get current executable path
	currentExe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get current executable path: %w", err)
	}

	// Create backup of current version
	backupPath := currentExe + ".backup"
	DebugPrint("Creating backup at: %s\n", backupPath)

	if err := copyFile(currentExe, backupPath); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	// Install new binary
	if err := installNewBinary(tempFile, currentExe); err != nil {
		// Restore backup on failure
		DebugPrint("Installation failed, restoring backup\n")
		os.Rename(backupPath, currentExe)
		return fmt.Errorf("failed to install update: %w", err)
	}

	// Clean up backup on success
	os.Remove(backupPath)

	fmt.Printf("✓ Successfully upgraded to version %s!\n", release.TagName)
	fmt.Println("Please restart any running vers processes to use the new version.")

	return nil
}

func getBinaryName() string {
	goos := runtime.GOOS
	goarch := runtime.GOARCH

	// Convert Go arch names to match your release naming convention
	switch goarch {
	case "amd64":
		goarch = "amd64" // Your releases use amd64, not x86_64
	case "arm64":
		goarch = "arm64"
	case "386":
		goarch = "386"
	}

	// Build binary name to match your release pattern
	binaryName := fmt.Sprintf("vers-%s-%s", goos, goarch)

	// Add .exe extension for Windows
	if goos == "windows" {
		binaryName += ".exe"
	}

	return binaryName
}

func verifyChecksum(filePath, checksumURL string) error {
	// Download checksum file
	checksumResp, err := http.Get(checksumURL)
	if err != nil {
		return fmt.Errorf("failed to download checksum: %w", err)
	}
	defer checksumResp.Body.Close()

	if checksumResp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download checksum: HTTP %d", checksumResp.StatusCode)
	}

	expectedChecksumBytes, err := io.ReadAll(checksumResp.Body)
	if err != nil {
		return fmt.Errorf("failed to read checksum: %w", err)
	}

	// Extract just the checksum (first 64 characters)
	expectedChecksum := strings.TrimSpace(string(expectedChecksumBytes))
	if len(expectedChecksum) >= 64 {
		expectedChecksum = expectedChecksum[:64]
	}

	// Calculate actual checksum
	file, err := os.Open(filePath)
	if err != nil {
		return fmt.Errorf("failed to open file for checksum: %w", err)
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return fmt.Errorf("failed to calculate checksum: %w", err)
	}

	actualChecksum := hex.EncodeToString(hash.Sum(nil))

	if actualChecksum != expectedChecksum {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedChecksum, actualChecksum)
	}

	return nil
}

func downloadFile(url string, expectedSize int64) (string, error) {
	resp, err := http.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed with status: %d", resp.StatusCode)
	}

	// Create temporary file
	tempFile, err := os.CreateTemp("", "vers-upgrade-*")
	if err != nil {
		return "", err
	}
	defer tempFile.Close()

	// Download with progress
	var downloaded int64
	buffer := make([]byte, 32*1024) // 32KB buffer

	for {
		n, err := resp.Body.Read(buffer)
		if n > 0 {
			tempFile.Write(buffer[:n])
			downloaded += int64(n)

			if expectedSize > 0 {
				progress := float64(downloaded) / float64(expectedSize) * 100
				fmt.Printf("\rProgress: %.1f%% (%d/%d bytes)", progress, downloaded, expectedSize)
			}
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
	}

	fmt.Println() // New line after progress
	return tempFile.Name(), nil
}

func installNewBinary(sourcePath, targetPath string) error {
	// Copy the binary and set executable permissions
	if err := copyFile(sourcePath, targetPath); err != nil {
		return err
	}

	// Make sure it's executable
	return os.Chmod(targetPath, 0755)
}

func copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return err
	}

	// Copy permissions
	srcInfo, err := srcFile.Stat()
	if err != nil {
		return err
	}

	return os.Chmod(dst, srcInfo.Mode())
}
