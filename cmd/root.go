package cmd

import (
	"fmt"
	"os"

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
		// Skip config loading and client init for commands that don't require it (optional)
		// if cmd.Name() == "init" || cmd.Name() == "login" || cmd.Name() == "help" {
		// 	return nil
		// }

		// Add other options if needed, e.g., API key
		// apiKey := os.Getenv("VERS_API_KEY")
		// if apiKey != "" {
		//  clientOptions = append(clientOptions, vers.WithAPIKey(apiKey))
		// }

		err := godotenv.Load()
    	if err != nil {
    		// Log only if you want to be strict about the .env file existing
    		// log.Println("Warning: Could not load .env file:", err)
    	}

		versURL := os.Getenv("VERS_URL")
		fmt.Println("Overriding with versURL: ", versURL)
		client = vers.NewClient() // Initialize the global client

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