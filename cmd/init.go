package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/hdresearch/vers-cli/internal/assets"
	"github.com/hdresearch/vers-cli/internal/auth"
	"github.com/hdresearch/vers-cli/internal/runconfig"
	"github.com/hdresearch/vers-cli/styles"
	"github.com/spf13/cobra"
)

var (
	projectName      string
	memSize          int64
	vcpuCount        int64
	rootfsName       string
	kernelName       string
	dockerfileName   string
	fsSizeClusterMib int64
	fsSizeVmMib      int64
)

// initCmd represents the init command
var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize a new vers project",
	Long:  `Initialize a new vers project with a vers.toml configuration file.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Check if API key exists, prompt for login if not
		hasAPIKey, err := auth.HasAPIKey()
		if err != nil {
			return fmt.Errorf("error checking API key: %w", err)
		}
		if !hasAPIKey {
			return auth.PromptForLogin()
		}

		// Create a hidden .vers directory
		versDir := ".vers"
		if err := os.MkdirAll(versDir, 0755); err != nil {
			return fmt.Errorf("error creating .vers directory: %w", err)
		}

		// Create a .gitignore file if it doesn't exist
		gitignorePath := ".gitignore"
		if _, err := os.Stat(gitignorePath); os.IsNotExist(err) {
			gitignoreContent := assets.GitIgnoreContent
			if err := os.WriteFile(gitignorePath, []byte(gitignoreContent), 0644); err != nil {
				return fmt.Errorf("error creating .gitignore file: %w", err)
			}
		}

		// Create .vers/logs directory for commit history
		logsDir := filepath.Join(versDir, "logs")
		if err := os.MkdirAll(logsDir, 0755); err != nil {
			return fmt.Errorf("error creating logs directory: %w", err)
		}
		if err := os.MkdirAll(filepath.Join(logsDir, "commits"), 0755); err != nil {
			return fmt.Errorf("error creating logs/commits directory: %w", err)
		}

		// Create .vers/HEAD file (initially empty - will be set when first VM is created)
		headFile := filepath.Join(versDir, "HEAD")
		if err := os.WriteFile(headFile, []byte(""), 0644); err != nil {
			return fmt.Errorf("error creating .vers/HEAD file: %w", err)
		}

		// Create .vers/config file
		configFile := filepath.Join(versDir, "config")
		defaultConfig := `[vers]
	version = 1
`
		if err := os.WriteFile(configFile, []byte(defaultConfig), 0644); err != nil {
			return fmt.Errorf("error creating config file: %w", err)
		}

		// Create vers.toml file if it doesn't exist
		versTomlPath := "vers.toml"
		if _, err := os.Stat(versTomlPath); os.IsNotExist(err) {
			// Use provided flag values or defaults
			if projectName == "" {
				// Default to current directory name if not specified
				currentDir, err := os.Getwd()
				if err == nil {
					projectName = filepath.Base(currentDir)
				} else {
					projectName = "unnamed-project"
				}
			}

			if rootfsName == "" {
				rootfsName = "default"
			}

			// Create the vers.toml content
			config := &runconfig.Config{
				Machine: runconfig.MachineConfig{
					MemSizeMib:       memSize,
					VcpuCount:        vcpuCount,
					FsSizeClusterMib: fsSizeClusterMib,
					FsSizeVmMib:      fsSizeVmMib,
				},
				Rootfs: runconfig.RootfsConfig{
					Name: rootfsName,
				},
				Builder: runconfig.BuilderConfig{
					Name:       "none",
					Dockerfile: "Dockerfile",
				},
				Kernel: runconfig.KernelConfig{
					Name: kernelName,
				},
			}

			var buf bytes.Buffer
			encoder := toml.NewEncoder(&buf)
			if err := encoder.Encode(config); err != nil {
				return fmt.Errorf("error encoding config: %w", err)
			}
			versTomlContent := buf.String()

			if err := os.WriteFile(versTomlPath, []byte(versTomlContent), 0644); err != nil {
				return fmt.Errorf("error creating vers.toml file: %w", err)
			}
			fmt.Print(styles.MutedTextStyle.Render("Created vers.toml with default configuration\n"))
		} else {
			fmt.Print(styles.MutedTextStyle.Render("vers.toml already exists, skipping\n"))
		}

		logoStyle := styles.AppStyle.Foreground(styles.TerminalMagenta)
		fmt.Println(logoStyle.Render(`	
		▗▖  ▗▖▗▄▄▄▖▗▄▄▖  ▗▄▄▖
		▐▌  ▐▌▐▌   ▐▌ ▐▌▐▌   
		▐▌  ▐▌▐▛▀▀▘▐▛▀▚▖ ▝▀▚▖
		 ▝▚▞▘ ▐▙▄▄▖▐▌ ▐▌▗▄▄▞▘						 
   `))

		fmt.Printf("%s", styles.MutedTextStyle.Render(fmt.Sprintf("Initialized vers repository in %s directory\n", versDir)))

		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)

	// Define flags for the init command
	initCmd.Flags().StringVarP(&projectName, "name", "n", "", "Project name (defaults to directory name)")

	// Add flags for vers.toml configuration
	initCmd.Flags().Int64Var(&memSize, "mem-size", 512, "Memory size in MiB")
	initCmd.Flags().Int64Var(&vcpuCount, "vcpu-count", 1, "Number of virtual CPUs")
	initCmd.Flags().StringVar(&rootfsName, "rootfs", "", "Name of the rootfs image (defaults to project name)")
	initCmd.Flags().StringVar(&kernelName, "kernel", "default.bin", "Name of the kernel image")
	initCmd.Flags().StringVar(&dockerfileName, "dockerfile", "Dockerfile", "Name of the Docker file")
	initCmd.Flags().Int64Var(&fsSizeClusterMib, "fs-size-cluster", 1024, "Total cluster filesystem size in MiB")
	initCmd.Flags().Int64Var(&fsSizeVmMib, "fs-size-vm", 512, "VM filesystem size in MiB")
}
