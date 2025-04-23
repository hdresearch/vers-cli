package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/hdresearch/vers-cli/internal/auth"
	"github.com/hdresearch/vers-cli/internal/config"
	vers "github.com/hdresearch/vers-sdk-go"
	"github.com/joho/godotenv" // Import godotenv
	"github.com/spf13/cobra"
)

// Global vars for configuration and SDK client
var (
	configPath string
	cfg        *config.Config
	client     *vers.Client
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "vers",
	Short: "vers is a CLI tool for managing virtual development environments",
	Long: `Vers CLI provides a command-line interface for managing virtual machine/container-based 
development environments. It offers lifecycle management, state management, 
interaction capabilities, and more.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	// Run: func(cmd *cobra.Command, args []string) { },
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
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
			return fmt.Errorf("authentication required")
		}

		versURL := os.Getenv("VERS_URL")
		if versURL != "" {
			fmt.Println("Overriding with versURL: ", versURL)
		}
		// Set the API key in the environment for the SDK
		os.Setenv("VERS_API_KEY", apiKey)

		// Initialize the client *only* if we have an API key
		client = vers.NewClient()

		// // Configuration loading (keep if needed, but separate from client init)
		// var err error
		// configPath, cfg, err = config.FindConfig()
		// if err != nil {
		// 	return fmt.Errorf("error finding config: %w", err)
		// }

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
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.vers.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
} 