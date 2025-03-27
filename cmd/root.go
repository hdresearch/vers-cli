package cmd

import (
	"fmt"
	"os"

	"github.com/hdresearch/vers-cli/internal/config"
	vers "github.com/hdresearch/vers-sdk-go"
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
		// Skip config loading for commands that don't require it
		if cmd.Name() == "init" || cmd.Name() == "login" || cmd.Name() == "help" {
			return nil
		}

		// Find and load the configuration
		var err error
		configPath, cfg, err = config.FindConfig()
		if err != nil {
			return fmt.Errorf("error finding config: %w", err)
		}

		// Create the SDK client
		client = vers.NewClient()
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