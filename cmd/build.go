package cmd

import (
	"archive/tar"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hdresearch/vers-sdk-go"
	"github.com/hdresearch/vers-sdk-go/option"
	"github.com/spf13/cobra"
)

// buildCmd represents the build command
var buildCmd = &cobra.Command{
	Use:   "build",
	Short: "Build a rootfs image",
	Long:  `Build a rootfs image according to the configuration in vers.toml and the Dockerfile in the current directory.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load configuration from vers.toml
		config, err := loadConfig()
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		// Apply flag overrides
		applyFlagOverrides(cmd, config)

		return BuildRootfs(config)
	},
}

// BuildRootfs builds a rootfs image according to the provided configuration
func BuildRootfs(config *Config) error {
	// Validate builder value
	if config.Builder.Name == "none" {
		fmt.Printf("Builder is set to 'none'; skipping")
		return nil
	} else if config.Builder.Name != "docker" {
		return fmt.Errorf("unsupported builder: %s (only 'docker' is currently supported)", config.Builder.Name)
	}

	// Check for Dockerfile
	if _, err := os.Stat(config.Builder.Dockerfile); os.IsNotExist(err) {
		return fmt.Errorf("Dockerfile '%s' not found in current directory", config.Builder.Dockerfile)
	}

	// Create temporary tar archive
	tempFile, err := os.CreateTemp("", "vers-rootfs-*.tar")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer os.Remove(tempFile.Name()) // Clean up the temp file when done

	fmt.Println("Creating tar archive of working directory...")
	if err := createTarArchive(tempFile); err != nil {
		return fmt.Errorf("failed to create tar archive: %w", err)
	}

	// Reset file pointer to beginning for reading
	if _, err := tempFile.Seek(0, 0); err != nil {
		return fmt.Errorf("failed to reset file pointer: %w", err)
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Prepare for upload
	fmt.Printf("Uploading rootfs archive as '%s'...\n", config.Rootfs.Name)

	// Reading the file into memory for the request
	fileContent, err := os.ReadFile(tempFile.Name())
	if err != nil {
		return fmt.Errorf("failed to read tar file: %w", err)
	}

	// Create upload request option with the file content as the body
	fileOption := option.WithRequestBody("application/x-tar", fileContent)

	// Upload with the file content
	body := vers.APIRootfUploadParams{
		Dockerfile: vers.F(config.Builder.Dockerfile),
	}
	res, err := client.API.Rootfs.Upload(ctx, config.Rootfs.Name, body, fileOption)
	if err != nil {
		return err
	}

	fmt.Printf("Successfully uploaded rootfs: %s\n", res.RootfsName)
	return nil
}

// createTarArchive creates a tar archive of the current directory, excluding .vers and vers.toml
func createTarArchive(tarFile *os.File) error {
	tw := tar.NewWriter(tarFile)
	defer tw.Close()

	// Get the current working directory
	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// Walk through the directory
	return filepath.Walk(workDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip .vers directory and vers.toml file
		relPath, err := filepath.Rel(workDir, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}

		if relPath == ".vers" ||
			strings.HasPrefix(relPath, ".vers"+string(os.PathSeparator)) ||
			relPath == "vers.toml" ||
			relPath == tarFile.Name() { // Skip the tar file itself
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip if it's the current directory
		if relPath == "." {
			return nil
		}

		// Create header
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return fmt.Errorf("failed to create tar header: %w", err)
		}

		// Set proper name with relative path
		header.Name = relPath

		// Write header
		if err := tw.WriteHeader(header); err != nil {
			return fmt.Errorf("failed to write tar header: %w", err)
		}

		// If it's a regular file, write its contents
		if info.Mode().IsRegular() {
			file, err := os.Open(path)
			if err != nil {
				return fmt.Errorf("failed to open file %s: %w", path, err)
			}
			defer file.Close()

			if _, err := io.Copy(tw, file); err != nil {
				return fmt.Errorf("failed to copy file contents: %w", err)
			}
		}

		return nil
	})
}

func init() {
	rootCmd.AddCommand(buildCmd)

	// Add flags to override toml configuration
	buildCmd.Flags().String("rootfs", "", "Override rootfs name")
	buildCmd.Flags().String("dockerfile", "", "Dockerfile path")
}
