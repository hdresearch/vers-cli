package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var projectName string

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new vers project",
	Long:  `Initialize a new vers project with a vers.toml configuration file.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// // If no project name is provided, use the current directory name
		// if projectName == "" {
		// 	dir, err := os.Getwd()
		// 	if err != nil {
		// 		return fmt.Errorf("error getting current directory: %w", err)
		// 	}
		// 	projectName = filepath.Base(dir)
		// }

		// // Check if vers.toml already exists
		// configFile := "vers.toml"
		// if _, err := os.Stat(configFile); err == nil {
		// 	overwrite := false
		// 	prompt := &survey.Confirm{
		// 		Message: "vers.toml already exists. Overwrite?",
		// 		Default: false,
		// 	}
		// 	survey.AskOne(prompt, &overwrite)
		// 	if !overwrite {
		// 		return fmt.Errorf("operation canceled")
		// 	}
		// }

		// // Prompt for project type
		// projectType := ""
		// typePrompt := &survey.Select{
		// 	Message: "Choose project type:",
		// 	Options: []string{"python", "typescript", "javascript", "go", "rust", "desktop"},
		// }
		// err := survey.AskOne(typePrompt, &projectType, survey.WithValidator(survey.Required))
		// if err != nil {
		// 	return fmt.Errorf("error getting project type: %w", err)
		// }

		// // Create a basic config
		// cfg := &config.Config{
		// 	Meta: config.MetaConfig{
		// 		Project: projectName,
		// 		Type:    projectType,
		// 	},
		// 	Build: config.BuildConfig{
		// 		Builder: "default",
		// 	},
		// 	Run: config.RunConfig{
		// 		Command: "",
		// 	},
		// 	Env: make(map[string]string),
		// 	Machine: map[string]config.MachineConfig{
		// 		"default": {
		// 			Name:  "default",
		// 			Image: "ubuntu:latest",
		// 		},
		// 	},
		// }

		// // Set build command based on project type
		// switch projectType {
		// case "python":
		// 	cfg.Build.BuildCommand = "pip install -r requirements.txt"
		// 	cfg.Run.Command = "python main.py"
		// case "typescript":
		// 	cfg.Build.BuildCommand = "npm install && npm run build"
		// 	cfg.Run.Command = "npm start"
		// case "javascript":
		// 	cfg.Build.BuildCommand = "npm install"
		// 	cfg.Run.Command = "npm start"
		// case "go":
		// 	cfg.Build.BuildCommand = "go build -o app"
		// 	cfg.Run.Command = "./app"
		// case "rust":
		// 	cfg.Build.BuildCommand = "cargo build --release"
		// 	cfg.Run.Command = "./target/release/app"
		// }

		// // Write to vers.toml
		// f, err := os.Create(configFile)
		// if err != nil {
		// 	return fmt.Errorf("error creating config file: %w", err)
		// }
		// defer f.Close()

		// encoder := toml.NewEncoder(f)
		// if err := encoder.Encode(cfg); err != nil {
		// 	return fmt.Errorf("error writing config: %w", err)
		// }

		// fmt.Printf("Initialized %s project '%s'\n", projectType, projectName)
		// fmt.Println("Configuration written to vers.toml")

		// // Call the SDK for any additional setup
		// client = vers.NewClient()

		// // Using the SDK to initialize the project
		// // This is a stub - you'll need to implement this based on the SDK's actual capabilities
		// fmt.Printf("Initializing project of type: %s\n", projectType)

		// Create a hidden .vers directory
		versDir := ".vers"
		if err := os.MkdirAll(versDir, 0755); err != nil {
			return fmt.Errorf("error creating .vers directory: %w", err)
		}

		// Create .vers/refs directory
		refsDir := filepath.Join(versDir, "refs")
		if err := os.MkdirAll(refsDir, 0755); err != nil {
			return fmt.Errorf("error creating .vers/refs directory: %w", err)
		}

		// Create refs subdirectories
		if err := os.MkdirAll(filepath.Join(refsDir, "heads"), 0755); err != nil {
			return fmt.Errorf("error creating refs/heads directory: %w", err)
		}
		if err := os.MkdirAll(filepath.Join(refsDir, "tags"), 0755); err != nil {
			return fmt.Errorf("error creating refs/tags directory: %w", err)
		}

		// Create .vers/objects directory
		objectsDir := filepath.Join(versDir, "objects")
		if err := os.MkdirAll(objectsDir, 0755); err != nil {
			return fmt.Errorf("error creating objects directory: %w", err)
		}

		// Create objects subdirectories (info and pack)
		if err := os.MkdirAll(filepath.Join(objectsDir, "info"), 0755); err != nil {
			return fmt.Errorf("error creating objects/info directory: %w", err)
		}
		if err := os.MkdirAll(filepath.Join(objectsDir, "pack"), 0755); err != nil {
			return fmt.Errorf("error creating objects/pack directory: %w", err)
		}

		// Create .vers/info directory
		infoDir := filepath.Join(versDir, "info")
		if err := os.MkdirAll(infoDir, 0755); err != nil {
			return fmt.Errorf("error creating info directory: %w", err)
		}

		// Create .vers/hooks directory
		hooksDir := filepath.Join(versDir, "hooks")
		if err := os.MkdirAll(hooksDir, 0755); err != nil {
			return fmt.Errorf("error creating hooks directory: %w", err)
		}

		// Create .vers/logs directory
		logsDir := filepath.Join(versDir, "logs")
		if err := os.MkdirAll(logsDir, 0755); err != nil {
			return fmt.Errorf("error creating logs directory: %w", err)
		}
		if err := os.MkdirAll(filepath.Join(logsDir, "refs", "heads"), 0755); err != nil {
			return fmt.Errorf("error creating logs/refs/heads directory: %w", err)
		}

		// Create .vers/HEAD file
		headFile := filepath.Join(versDir, "HEAD")
		if err := os.WriteFile(headFile, []byte("ref: refs/heads/main\n"), 0644); err != nil {
			return fmt.Errorf("error creating .vers/HEAD file: %w", err)
		}

		// Create .vers/config file
		configFile := filepath.Join(versDir, "config")
		defaultConfig := `[core]
	repositoryformatversion = 0
	filemode = true
	bare = false
[vers]
	version = 1
`
		if err := os.WriteFile(configFile, []byte(defaultConfig), 0644); err != nil {
			return fmt.Errorf("error creating config file: %w", err)
		}

		// Create .vers/description file
		descFile := filepath.Join(versDir, "description")
		if err := os.WriteFile(descFile, []byte("Unnamed repository; edit this file to name the repository.\n"), 0644); err != nil {
			return fmt.Errorf("error creating description file: %w", err)
		}

		// Create an empty index file
		indexFile := filepath.Join(versDir, "index")
		if err := os.WriteFile(indexFile, []byte{}, 0644); err != nil {
			return fmt.Errorf("error creating index file: %w", err)
		}

		fmt.Printf("Initialized vers repository in %s directory\n", versDir)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)

	// Define flags for the init command
	initCmd.Flags().StringVarP(&projectName, "name", "n", "", "Project name (defaults to directory name)")
}
