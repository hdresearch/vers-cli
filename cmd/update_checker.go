package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/hdresearch/vers-cli/internal/config"
)

// checkForUpdates checks for available updates and updates the CLI config
func checkForUpdates() error {
	DebugPrint("Checking for updates...\n")

	// Get current version
	currentVersion := strings.TrimPrefix(Version, "v")
	if currentVersion == "dev" || currentVersion == "unknown" {
		DebugPrint("Skipping update check for development version\n")
		return nil
	}

	// Check for latest release
	latest, err := getLatestRelease()
	if err != nil {
		DebugPrint("Failed to check for updates: %v\n", err)
		return nil // Don't error out - just skip the check
	}

	latestVersion := strings.TrimPrefix(latest.TagName, "v")
	DebugPrint("Current: %s, Latest: %s\n", currentVersion, latestVersion)

	// Load CLI config
	cliConfig, err := config.LoadCLIConfig()
	if err != nil {
		DebugPrint("Failed to load CLI config: %v\n", err)
		return nil
	}

	// Update next check time regardless
	cliConfig.SetNextCheckTime()

	// Check if there's an update
	if currentVersion != latestVersion {
		cliConfig.SetAvailableVersion(latest.TagName)
		DebugPrint("Update available: %s\n", latest.TagName)
	} else {
		cliConfig.ClearUpdateState()
		DebugPrint("Already on latest version\n")
	}

	// Save config
	if err := config.SaveCLIConfig(cliConfig); err != nil {
		DebugPrint("Failed to save CLI config: %v\n", err)
	}

	return nil
}

// promptForUpdate shows an update notification to the user
func promptForUpdate(availableVersion string) {
	fmt.Printf("\n🚀 A new version of vers is available: %s -> %s\n", Version, availableVersion)
	fmt.Printf("Run 'vers upgrade' to update, or 'vers upgrade --check-only' to see what's new.\n")

	// Ask if user wants to be reminded later or skip this version
	fmt.Printf("\nOptions:\n")
	fmt.Printf("  [Enter] Continue and remind me later\n")
	fmt.Printf("  s      Skip this version\n")
	fmt.Printf("  n      Don't remind me for 24 hours\n")
	fmt.Printf("Choice: ")

	var choice string
	fmt.Scanln(&choice)
	choice = strings.TrimSpace(strings.ToLower(choice))

	cliConfig, err := config.LoadCLIConfig()
	if err != nil {
		return
	}

	switch choice {
	case "s":
		cliConfig.SkipVersion(availableVersion)
		fmt.Println("✓ Skipping version " + availableVersion)
	case "n":
		// Set next check to 24 hours from now
		cliConfig.UpdateCheck.NextCheck = time.Now().Add(24 * time.Hour)
		fmt.Println("✓ Will remind you tomorrow")
	default:
		// Keep the update available but don't change timing
		fmt.Println("✓ Will remind you next time")
	}

	config.SaveCLIConfig(cliConfig)
	fmt.Println()
}

// showUpdateNotification shows a brief update notification
func showUpdateNotification(availableVersion string) {
	fmt.Printf("💡 Update available: %s -> %s (run 'vers upgrade')\n\n", Version, availableVersion)
}

// handleUpdateCheck performs the update check logic
func handleUpdateCheck() {
	cliConfig, err := config.LoadCLIConfig()
	if err != nil {
		DebugPrint("Failed to load CLI config: %v\n", err)
		return
	}

	// Check if we have an available update to show
	if cliConfig.HasAvailableUpdate() {
		DebugPrint("Showing available update: %s\n", cliConfig.UpdateCheck.AvailableVersion)
		showUpdateNotification(cliConfig.UpdateCheck.AvailableVersion)
		return
	}

	// Check if it's time to check for updates
	if !cliConfig.ShouldCheckForUpdate() {
		DebugPrint("Not time to check for updates yet\n")
		return
	}

	DebugPrint("Time to check for updates\n")
	go func() {
		// Run update check in background to avoid blocking
		if err := checkForUpdates(); err != nil {
			DebugPrint("Background update check failed: %v\n", err)
		}
	}()
}
