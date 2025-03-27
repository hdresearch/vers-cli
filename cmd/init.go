package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/AlecAivazis/survey/v2"
	"github.com/BurntSushi/toml"
	"github.com/hdresearch/vers-cli/internal/config"
	"github.com/hdresearch/vers-cli/internal/sdk"
	"github.com/spf13/cobra"
)

var projectName string

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new vers project",
	Long:  `Initialize a new vers project with a vers.toml configuration file.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// If no project name is provided, use the current directory name
		if projectName == "" {
			dir, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("error getting current directory: %w", err)
			}
			projectName = filepath.Base(dir)
		}

		// Check if vers.toml already exists
		configFile := "vers.toml"
		if _, err := os.Stat(configFile); err == nil {
			overwrite := false
			prompt := &survey.Confirm{
				Message: "vers.toml already exists. Overwrite?",
				Default: false,
			}
			survey.AskOne(prompt, &overwrite)
			if !overwrite {
				return fmt.Errorf("operation canceled")
			}
		}

		// Prompt for project type
		projectType := ""
		typePrompt := &survey.Select{
			Message: "Choose project type:",
			Options: []string{"python", "typescript", "javascript", "go", "rust", "desktop"},
		}
		err := survey.AskOne(typePrompt, &projectType, survey.WithValidator(survey.Required))
		if err != nil {
			return fmt.Errorf("error getting project type: %w", err)
		}

		// Create a basic config
		cfg := &config.Config{
			Meta: config.MetaConfig{
				Project: projectName,
				Type:    projectType,
			},
			Build: config.BuildConfig{
				Builder: "default",
			},
			Run: config.RunConfig{
				Command: "",
			},
			Env: make(map[string]string),
			Machine: map[string]config.MachineConfig{
				"default": {
					Name:  "default",
					Image: "ubuntu:latest",
				},
			},
		}

		// Set build command based on project type
		switch projectType {
		case "python":
			cfg.Build.BuildCommand = "pip install -r requirements.txt"
			cfg.Run.Command = "python main.py"
		case "typescript":
			cfg.Build.BuildCommand = "npm install && npm run build"
			cfg.Run.Command = "npm start"
		case "javascript":
			cfg.Build.BuildCommand = "npm install"
			cfg.Run.Command = "npm start"
		case "go":
			cfg.Build.BuildCommand = "go build -o app"
			cfg.Run.Command = "./app"
		case "rust":
			cfg.Build.BuildCommand = "cargo build --release"
			cfg.Run.Command = "./target/release/app"
		}

		// Write to vers.toml
		f, err := os.Create(configFile)
		if err != nil {
			return fmt.Errorf("error creating config file: %w", err)
		}
		defer f.Close()

		encoder := toml.NewEncoder(f)
		if err := encoder.Encode(cfg); err != nil {
			return fmt.Errorf("error writing config: %w", err)
		}

		fmt.Printf("Initialized %s project '%s'\n", projectType, projectName)
		fmt.Println("Configuration written to vers.toml")

		// Call the SDK for any additional setup
		client = sdk.NewStubClient(nil)
		return client.InitProject(projectType)
	},
}

func init() {
	rootCmd.AddCommand(initCmd)

	// Define flags for the init command
	initCmd.Flags().StringVarP(&projectName, "name", "n", "", "Project name (defaults to directory name)")
} 