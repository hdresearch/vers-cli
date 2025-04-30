package cmd

import (
	"archive/tar"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hdresearch/vers-cli/styles"
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
	if config.Rootfs.Builder == "none" {
		fmt.Printf("Builder is set to 'none'; skipping")
		return nil
	} else if config.Rootfs.Builder != "docker" {
		return fmt.Errorf("unsupported builder: %s (only 'docker' is currently supported)", config.Rootfs.Builder)
	}

	// Check for Dockerfile
	if _, err := os.Stat("Dockerfile"); os.IsNotExist(err) {
		return fmt.Errorf("Dockerfile not found in current directory")
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
	res, err := client.API.Rootfs.Upload(ctx, config.Rootfs.Name, fileOption)
	if err != nil {
		// Parse and handle the error based on the error type
		return handleBuildError(err, config.Rootfs.Name)
	}

	fmt.Printf("Successfully uploaded rootfs: %s\n", res.RootfsName)
	return nil
}

// handleBuildError parses the error from the API and returns a user-friendly error message
func handleBuildError(err error, rootfsName string) error {
	// Extract status code and body information from the error
	statusCode, errorBody := extractErrorInfo(err)

	// If we have a valid status code, process based on that
	if statusCode > 0 {
		// First try to parse as JSON (in case it is JSON)
		var errorResponse struct {
			Error string `json:"error"`
		}

		errorMessage := ""
		if errorBody != "" {
			if err := json.Unmarshal([]byte(errorBody), &errorResponse); err == nil && errorResponse.Error != "" {
				// Successfully parsed JSON
				errorMessage = errorResponse.Error
			} else {
				// Treat as plain text
				errorMessage = errorBody
			}
		}

		// Handle based on status code
		switch statusCode {
		case http.StatusConflict: // 409
			return fmt.Errorf(styles.ErrorTextStyle.Render("rootfs '%s' already exists"), rootfsName)
		case http.StatusUnauthorized: // 401
			if errorMessage != "" {
				return fmt.Errorf(styles.ErrorTextStyle.Render("unauthorized: %s"), errorMessage)
			}
			return fmt.Errorf(styles.ErrorTextStyle.Render("missing or invalid API key"))
		case http.StatusForbidden: // 403
			if errorMessage != "" {
				return fmt.Errorf(styles.ErrorTextStyle.Render("access denied: %s"), errorMessage)
			}
			return fmt.Errorf(styles.ErrorTextStyle.Render("access denied for rootfs '%s'"), rootfsName)
		case http.StatusInternalServerError: // 500
			if errorMessage != "" {
				return fmt.Errorf(styles.ErrorTextStyle.Render("server error: %s"), errorMessage)
			}
			return fmt.Errorf(styles.ErrorTextStyle.Render("server error occurred"))
		default:
			if errorMessage != "" {
				return fmt.Errorf(styles.ErrorTextStyle.Render("error (%d): %s"), statusCode, errorMessage)
			}
			return fmt.Errorf(styles.ErrorTextStyle.Render("request failed with status code %d"), statusCode)
		}
	}

	// If we couldn't extract HTTP status code, look for specific error messages
	errMsg := err.Error()

	// Check for specific error phrases
	if strings.Contains(strings.ToLower(errMsg), "already exists") {
		return fmt.Errorf(styles.ErrorTextStyle.Render("rootfs '%s' already exists"), rootfsName)
	}

	if strings.Contains(strings.ToLower(errMsg), "unauthorized") ||
		strings.Contains(strings.ToLower(errMsg), "authentication") {
		return fmt.Errorf(styles.ErrorTextStyle.Render("authentication failed, please run 'vers login'"))
	}

	// Default to original error
	return fmt.Errorf(styles.ErrorTextStyle.Render("failed to upload rootfs: %v"), err)
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
}
