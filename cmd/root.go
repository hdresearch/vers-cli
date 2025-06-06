package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/hdresearch/vers-cli/internal/auth"
	vers "github.com/hdresearch/vers-sdk-go"
	"github.com/joho/godotenv" // Import godotenv
	"github.com/spf13/cobra"
)

// These variables are set at build time using ldflags
var (
	// Core version info
	Version   = "dev"
	GitCommit = "unknown"
	BuildDate = "unknown"

	// Manifest info
	Name        = "vers-cli"
	Description = "A CLI tool for version management"
	Author      = "the VERS team"
	Repository  = "https://github.com/tynandaly/vers-cli"
	License     = "MIT"
)

// Global vars for configuration and SDK client
var (
	client  *vers.Client
	verbose bool
)

// MetadataInfo represents the complete version information
type MetadataInfo struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
	GitCommit   string `json:"gitCommit"`
	BuildDate   string `json:"buildDate"`
	Author      string `json:"author"`
	Repository  string `json:"repository"`
	License     string `json:"license"`
	GoVersion   string `json:"goVersion"`
	Platform    string `json:"platform"`
	Arch        string `json:"arch"`
	Timestamp   string `json:"timestamp"`
}

// getVersionInfo returns the complete version information
func getVersionInfo() *MetadataInfo {
	return &MetadataInfo{
		Name:        Name,
		Version:     Version,
		Description: Description,
		GitCommit:   GitCommit,
		BuildDate:   BuildDate,
		Author:      Author,
		Repository:  Repository,
		License:     License,
		GoVersion:   runtime.Version(),
		Platform:    runtime.GOOS,
		Arch:        runtime.GOARCH,
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
	}
}

// DebugPrint prints debug information only when verbose mode is enabled
func DebugPrint(format string, args ...interface{}) {
	if verbose {
		fmt.Printf("[DEBUG] "+format, args...)
	}
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "vers",
	Short: "vers is a CLI tool for managing virtual development environments",
	Long: `Vers CLI provides a command-line interface for managing virtual machine/container-based 
development environments. It offers lifecycle management, state management, 
interaction capabilities, and more.`,
	Run: func(cmd *cobra.Command, args []string) {
		// Handle --VVersion flag
		if vversion, _ := cmd.Flags().GetBool("VVersion"); vversion {
			versionInfo := getVersionInfo()
			jsonOutput, err := json.MarshalIndent(versionInfo, "", "  ")
			if err != nil {
				fmt.Printf("Error marshalling version info: %v\n", err)
				os.Exit(1)
			}
			fmt.Println(string(jsonOutput))
			return
		}

		// If no command specified, show help
		cmd.Help()
	},
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// Handle version flags before any authentication
		if version, _ := cmd.Flags().GetBool("version"); version {
			fmt.Println(Version)
			os.Exit(0)
		}

		if vversion, _ := cmd.Flags().GetBool("VVersion"); vversion {
			// This will be handled in the Run function
			return nil
		}

		// Set verbose environment variable for other packages to use
		if verbose {
			os.Setenv("VERS_VERBOSE", "true")
		}

		if cmd.Name() == "login" || cmd.Name() == "help" || cmd.CalledAs() == "help" {
			return nil
		}

		// Load .env for the VERS_URL
		godotenv.Load()

		// Initialize the client with API key if available
		apiKey, err := auth.GetAPIKey()
		if err != nil {
			return fmt.Errorf("failed to load API key: %w", err)
		}

		if apiKey == "" {
			auth.PromptForLogin()
			cmd.SilenceUsage = true
			return fmt.Errorf("authentication required")
		}

		// Set the API key in the environment for the SDK
		os.Setenv("VERS_API_KEY", apiKey)

		clientOptions := auth.GetClientOptions()

		// Initialize the client *only* if we have an API key
		client = vers.NewClient(clientOptions...)

		return nil
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		// Check if the error is a 401 Unauthorized
		if strings.Contains(err.Error(), "401") ||
			strings.Contains(strings.ToLower(err.Error()), "unauthorized") {
			fmt.Println("Authentication failed. Please run 'vers login' to re-authenticate with a valid API token.")
			os.Exit(1)
		}
		os.Exit(1)
	}
}

func init() {
	// Add global persistent flags
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "Enable verbose output")

	// Add version flags
	rootCmd.Flags().Bool("version", false, "Show version information")
	rootCmd.Flags().Bool("VVersion", false, "Show detailed version and build information")

	// Handle version flag (simple version)
	rootCmd.SetVersionTemplate("{{.Version}}\n")
	rootCmd.Version = Version

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
}
