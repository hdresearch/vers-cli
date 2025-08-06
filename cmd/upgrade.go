package cmd

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"

	"github.com/hdresearch/vers-cli/internal/output"
	update "github.com/hdresearch/vers-cli/internal/update"
	confirmation "github.com/hdresearch/vers-cli/internal/utils"
	"github.com/spf13/cobra"
)

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

	// Version comparison phase
	versionCheck := output.New()
	versionCheck.WriteLinef("Current version: %s", Version)

	// Check for latest release using shared update package
	latest, err := update.GetLatestRelease(Repository, prerelease, verbose)
	if err != nil {
		return fmt.Errorf("failed to check for updates: %w", err)
	}

	latestVersion := strings.TrimPrefix(latest.TagName, "v")
	versionCheck.WriteLinef("Latest version: %s", latest.TagName)

	// Compare versions
	if currentVersion == latestVersion {
		versionCheck.WriteLine("You are already running the latest version!").Print()

		// Reset the update check timer since we manually checked
		update.UpdateCheckTime()

		return nil
	}

	if checkOnly {
		versionCheck.WriteLinef("A new version is available: %s -> %s", Version, latest.TagName).
			WriteLine("Run 'vers upgrade' to install the update.").
			Print()
		return nil
	}

	// Print version info first
	versionCheck.Print()

	// Confirmation phase
	fmt.Printf("\nUpgrade from %s to %s?\n", Version, latest.TagName)

	if !confirmation.AskConfirmation() {
		output.ImmediateLine("Upgrade cancelled.")
		return nil
	}

	// Perform upgrade
	err = performUpgrade(latest)
	if err == nil {
		// Reset the update check timer after successful upgrade
		update.UpdateCheckTime()
	}

	return err
}

func performUpgrade(release *update.GitHubRelease) error {
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

	// Download phase
	fmt.Printf("Downloading %s...\n", binaryName)

	DebugPrint("Downloading binary from: %s\n", binaryURL)

	// Download the new binary
	tempFile, err := downloadFile(binaryURL, binarySize)
	if err != nil {
		return fmt.Errorf("failed to download update: %w", err)
	}
	defer os.Remove(tempFile)

	// Verification and installation phase
	install := output.New()

	// Verify checksum if available and not skipped
	if checksumURL != "" && !skipChecksum {
		DebugPrint("Verifying checksum from: %s\n", checksumURL)
		install.WriteLine("Verifying download integrity...")

		if err := verifyChecksum(tempFile, checksumURL); err != nil {
			return fmt.Errorf("checksum verification failed: %w", err)
		}
		install.WriteLine("✓ Checksum verification passed")
	} else if skipChecksum {
		install.WriteLine("Skipping checksum verification (not recommended)")
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

	install.WriteLinef("✓ Successfully upgraded to version %s!", release.TagName).
		WriteLine("Please restart any running vers processes to use the new version.").
		Print()

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

	fmt.Print("\n") // New line after progress
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
