package handlers

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"runtime"
	"strings"

	"github.com/hdresearch/vers-cli/internal/app"
	update "github.com/hdresearch/vers-cli/internal/update"
)

type UpgradeReq struct {
	CurrentVersion string
	Repository     string
	CheckOnly      bool
	Prerelease     bool
	SkipChecksum   bool
}

func HandleUpgrade(a *app.App, r UpgradeReq) error {
	DebugPrint := func(format string, args ...any) {
		if a.Verbose {
			fmt.Printf("[DEBUG] "+format, args...)
		}
	}

	current := strings.TrimPrefix(r.CurrentVersion, "v")
	if current == "dev" || current == "unknown" {
		return fmt.Errorf("cannot upgrade development or unknown versions")
	}
	fmt.Printf("Current version: %s\n", r.CurrentVersion)

	latest, err := update.GetLatestRelease(r.Repository, r.Prerelease, a.Verbose)
	if err != nil {
		return fmt.Errorf("failed to check for updates: %w", err)
	}
	latestVersion := strings.TrimPrefix(latest.TagName, "v")
	fmt.Printf("Latest version: %s\n", latest.TagName)
	if current == latestVersion {
		fmt.Println("You are already running the latest version!")
		update.UpdateCheckTime()
		return nil
	}
	if r.CheckOnly {
		fmt.Printf("A new version is available: %s -> %s\n", r.CurrentVersion, latest.TagName)
		fmt.Println("Run 'vers upgrade' to install the update.")
		return nil
	}

	fmt.Printf("\nUpgrade from %s to %s?\n", r.CurrentVersion, latest.TagName)
	if !AskConfirmation(a) {
		fmt.Println("Upgrade cancelled.")
		return nil
	}

	if err := performUpgrade(DebugPrint, latest, r.SkipChecksum); err != nil {
		return err
	}
	update.UpdateCheckTime()
	return nil
}

// AskConfirmation uses the app prompter to confirm (fallback to default prompt if unavailable).
func AskConfirmation(a *app.App) bool {
	if a.Prompter != nil {
		ok, _ := a.Prompter.YesNo("Proceed")
		return ok
	}
	return false
}

func performUpgrade(DebugPrint func(string, ...any), release *update.GitHubRelease, skipChecksum bool) error {
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

	tempFile, err := downloadFile(binaryURL, binarySize)
	if err != nil {
		return fmt.Errorf("failed to download update: %w", err)
	}
	defer os.Remove(tempFile)

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

	currentExe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("failed to get current executable path: %w", err)
	}
	backupPath := currentExe + ".backup"
	DebugPrint("Creating backup at: %s\n", backupPath)
	if err := copyFile(currentExe, backupPath); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}
	if err := installNewBinary(tempFile, currentExe); err != nil {
		os.Rename(backupPath, currentExe)
		return fmt.Errorf("failed to install update: %w", err)
	}
	os.Remove(backupPath)
	fmt.Printf("✓ Successfully upgraded to version %s!\n", release.TagName)
	fmt.Println("Please restart any running vers processes to use the new version.")
	return nil
}

func getBinaryName() string {
	goos := runtime.GOOS
	goarch := runtime.GOARCH
	switch goarch {
	case "amd64":
		goarch = "amd64"
	case "arm64":
		goarch = "arm64"
	case "386":
		goarch = "386"
	}
	name := fmt.Sprintf("vers-%s-%s", goos, goarch)
	if goos == "windows" {
		name += ".exe"
	}
	return name
}

func verifyChecksum(filePath, checksumURL string) error {
	resp, err := http.Get(checksumURL)
	if err != nil {
		return fmt.Errorf("failed to download checksum: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to download checksum: HTTP %d", resp.StatusCode)
	}
	expectedBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read checksum: %w", err)
	}
	expected := strings.TrimSpace(string(expectedBytes))
	if len(expected) > 64 {
		expected = expected[:64]
	}
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return err
	}
	actual := hex.EncodeToString(h.Sum(nil))
	if actual != expected {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expected, actual)
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
	tmp, err := os.CreateTemp("", "vers-upgrade-*")
	if err != nil {
		return "", err
	}
	defer tmp.Close()
	var downloaded int64
	buf := make([]byte, 32*1024)
	for {
		n, err := resp.Body.Read(buf)
		if n > 0 {
			if _, werr := tmp.Write(buf[:n]); werr != nil {
				return "", werr
			}
			downloaded += int64(n)
			if expectedSize > 0 {
				pct := float64(downloaded) / float64(expectedSize) * 100
				fmt.Printf("\rProgress: %.1f%% (%d/%d bytes)", pct, downloaded, expectedSize)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
	}
	fmt.Println()
	return tmp.Name(), nil
}

func installNewBinary(src, dst string) error {
	if err := copyFile(src, dst); err != nil {
		return err
	}
	return os.Chmod(dst, 0755)
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	if fi, err := in.Stat(); err == nil {
		_ = os.Chmod(dst, fi.Mode())
	}
	return nil
}
