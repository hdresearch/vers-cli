package cmd

import (
	"fmt"
	"os"

	"github.com/hdresearch/vers-cli/internal/config"
	vers "github.com/hdresearch/vers-sdk-go" // Import godotenv
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
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		versURL := os.Getenv("VERS_URL")
		fmt.Println("Overriding with versURL: ", versURL)
		client = vers.NewClient() // Initialize the global client

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
	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
} 