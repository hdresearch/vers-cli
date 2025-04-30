package cmd

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	vers "github.com/hdresearch/vers-sdk-go"
	"github.com/spf13/cobra"
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run [cluster]",
	Short: "Start a development environment",
	Long:  `Start a Vers development environment according to the configuration in vers.toml.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load configuration from vers.toml
		config, err := loadConfig()
		if err != nil {
			return fmt.Errorf("failed to load configuration: %w", err)
		}

		// Override with flags if provided
		applyFlagOverrides(cmd, config)

		return StartCluster(config, args)
	},
}

// StartCluster starts a development environment according to the provided configuration
func StartCluster(config *Config, args []string) error {
	baseCtx := context.Background()
	apiCtx, cancel := context.WithTimeout(baseCtx, 30*time.Second)
	defer cancel()

	// Create cluster parameters based on config
	clusterParams := vers.APIClusterNewParams{
		Create: vers.CreateParam{
			MemSizeMib: vers.F(config.Machine.MemSizeMib),
			VcpuCount:  vers.F(config.Machine.VcpuCount),
			RootfsName: vers.F(config.Rootfs.Name),
			KernelName: vers.F(config.Kernel.Name),
		},
	}

	fmt.Println("Sending request to start cluster...")
	clusterInfo, err := client.API.Cluster.New(apiCtx, clusterParams)
	if err != nil {
		return handleRunError(err, config.Rootfs.Name, config.Kernel.Name)
	}

	// Use information from the response
	fmt.Printf("Cluster (ID: %s) started successfully with root vm '%s'.\n",
		clusterInfo.ID,
		clusterInfo.RootVmID,
	)

	// Store VM ID in version control system
	vmID := clusterInfo.RootVmID
	if vmID != "" {
		// Check if .vers directory exists
		versDir := ".vers"
		if _, err := os.Stat(versDir); os.IsNotExist(err) {
			fmt.Println("Warning: .vers directory not found. Run 'vers init' first.")
		} else {
			// Update refs/heads/main with VM ID
			mainRefPath := filepath.Join(versDir, "refs", "heads", "main")
			if err := os.WriteFile(mainRefPath, []byte(vmID+"\n"), 0644); err != nil {
				fmt.Printf("Warning: Failed to update refs: %v\n", err)
			} else {
				fmt.Printf("Updated VM reference: %s -> %s\n", "refs/heads/main", vmID)
			}

			// HEAD already points to refs/heads/main from init, so we don't need to update it
			fmt.Println("HEAD is now pointing to the new VM")
		}
	}

	return nil
}

// handleRunError parses the error from the API and returns a user-friendly error message
func handleRunError(err error, rootfsName, kernelName string) error {
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
		case http.StatusBadRequest: // 400
			if errorMessage != "" {
				if strings.Contains(strings.ToLower(errorMessage), "config") {
					return fmt.Errorf("invalid configuration: %s", errorMessage)
				}
				return fmt.Errorf("bad request: %s", errorMessage)
			}
			return fmt.Errorf("invalid configuration (check memory size, CPU count, rootfs, and kernel names)")

		case http.StatusUnauthorized: // 401
			if errorMessage != "" {
				return fmt.Errorf("unauthorized: %s", errorMessage)
			}
			return fmt.Errorf("authentication failed, please run 'vers login'")

		case http.StatusForbidden: // 403
			if errorMessage != "" {
				return fmt.Errorf("access denied: %s", errorMessage)
			}
			return fmt.Errorf("access denied for this operation")

		case http.StatusNotFound: // 404
			if errorMessage != "" {
				if strings.Contains(strings.ToLower(errorMessage), "kernel") {
					return fmt.Errorf("kernel '%s' not found", kernelName)
				} else if strings.Contains(strings.ToLower(errorMessage), "rootfs") {
					return fmt.Errorf("rootfs '%s' not found", rootfsName)
				}
				return fmt.Errorf("resource not found: %s", errorMessage)
			}
			return fmt.Errorf("resource not found (check if rootfs '%s' and kernel '%s' exist)", rootfsName, kernelName)

		case http.StatusInternalServerError: // 500
			if errorMessage != "" {
				return fmt.Errorf("server error: %s", errorMessage)
			}
			return fmt.Errorf("server error occurred")

		default:
			if errorMessage != "" {
				return fmt.Errorf("error (%d): %s", statusCode, errorMessage)
			}
			return fmt.Errorf("request failed with status code %d", statusCode)
		}
	}

	// If we couldn't extract HTTP status code, look for specific error patterns in the message
	errMsg := strings.ToLower(err.Error())

	if strings.Contains(errMsg, "invalid config") || strings.Contains(errMsg, "bad config") {
		return fmt.Errorf("invalid configuration")
	}

	if strings.Contains(errMsg, "not found") {
		if strings.Contains(errMsg, "kernel") {
			return fmt.Errorf("kernel '%s' not found", kernelName)
		} else if strings.Contains(errMsg, "rootfs") {
			return fmt.Errorf("rootfs '%s' not found", rootfsName)
		}
		return fmt.Errorf("resource not found")
	}

	if strings.Contains(errMsg, "unauthorized") || strings.Contains(errMsg, "authentication") {
		return fmt.Errorf("authentication failed, please run 'vers login'")
	}

	// Default to original error
	return fmt.Errorf("failed to start cluster: %v", err)
}

func init() {
	rootCmd.AddCommand(runCmd)

	// Add flags to override toml configuration
	runCmd.Flags().Int64("mem-size", 0, "Override memory size (MiB)")
	runCmd.Flags().Int64("vcpu-count", 0, "Override number of virtual CPUs")
	runCmd.Flags().String("rootfs", "", "Override rootfs name")
	runCmd.Flags().String("kernel", "", "Override kernel name")
}
