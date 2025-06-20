package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hdresearch/vers-cli/internal/auth"
	vers "github.com/hdresearch/vers-sdk-go"
)

// TestUpCommandIntegration tests the up command end-to-end with the actual service
func TestUpCommandIntegration(t *testing.T) {
	// Skip if we're in a unit test environment (no API key available)
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Check if we have authentication available
	hasAPIKey, err := auth.HasAPIKey()
	if err != nil || !hasAPIKey {
		t.Skip("Skipping integration test: No API key available. Run 'vers login' first.")
	}

	tests := []struct {
		name         string
		setupConfig  func(t *testing.T, tempDir string)
		expectBuild  bool
		expectError  bool
		errorMessage string
	}{
		{
			name: "up with docker builder",
			setupConfig: func(t *testing.T, tempDir string) {
				// Create a vers.toml with docker builder
				configContent := `
[machine]
mem_size_mib = 512
vcpu_count = 1
fs_size_cluster_mib = 1024
fs_size_vm_mib = 512

[rootfs]
name = "test-rootfs-` + fmt.Sprintf("%d", time.Now().Unix()) + `"

[builder]
name = "docker"
dockerfile = "Dockerfile"

[kernel]
name = "default.bin"
`
				err := os.WriteFile(filepath.Join(tempDir, "vers.toml"), []byte(configContent), 0644)
				if err != nil {
					t.Fatalf("Failed to create vers.toml: %v", err)
				}

				// Create a simple Dockerfile
				dockerfileContent := `FROM alpine:latest
CMD ["sleep", "infinity"]
`
				err = os.WriteFile(filepath.Join(tempDir, "Dockerfile"), []byte(dockerfileContent), 0644)
				if err != nil {
					t.Fatalf("Failed to create Dockerfile: %v", err)
				}
			},
			expectBuild: true,
			expectError: false,
		},
		{
			name: "up with none builder",
			setupConfig: func(t *testing.T, tempDir string) {
				// Create a vers.toml with none builder (should skip build)
				configContent := `
[machine]
mem_size_mib = 512
vcpu_count = 1
fs_size_cluster_mib = 1024
fs_size_vm_mib = 512

[rootfs]
name = "default"

[builder]
name = "none"

[kernel]
name = "default.bin"
`
				err := os.WriteFile(filepath.Join(tempDir, "vers.toml"), []byte(configContent), 0644)
				if err != nil {
					t.Fatalf("Failed to create vers.toml: %v", err)
				}
			},
			expectBuild: false,
			expectError: false,
		},
		{
			name: "up with missing dockerfile",
			setupConfig: func(t *testing.T, tempDir string) {
				// Create a vers.toml with docker builder but no Dockerfile
				configContent := `
[machine]
mem_size_mib = 512
vcpu_count = 1

[rootfs]
name = "test-rootfs-missing-dockerfile"

[builder]
name = "docker"
dockerfile = "Dockerfile"

[kernel]
name = "default.bin"
`
				err := os.WriteFile(filepath.Join(tempDir, "vers.toml"), []byte(configContent), 0644)
				if err != nil {
					t.Fatalf("Failed to create vers.toml: %v", err)
				}
				// Don't create Dockerfile - this should cause an error
			},
			expectBuild:  true,
			expectError:  true,
			errorMessage: "Dockerfile 'Dockerfile' not found",
		},
		{
			name: "up with default name should fail when building",
			setupConfig: func(t *testing.T, tempDir string) {
				// Create a vers.toml with default rootfs name and docker builder
				configContent := `
[machine]
mem_size_mib = 512
vcpu_count = 1

[rootfs]
name = "default"

[builder]
name = "docker"
dockerfile = "Dockerfile"

[kernel]
name = "default.bin"
`
				err := os.WriteFile(filepath.Join(tempDir, "vers.toml"), []byte(configContent), 0644)
				if err != nil {
					t.Fatalf("Failed to create vers.toml: %v", err)
				}

				// Create a Dockerfile
				dockerfileContent := `FROM alpine:latest
CMD ["sleep", "infinity"]
`
				err = os.WriteFile(filepath.Join(tempDir, "Dockerfile"), []byte(dockerfileContent), 0644)
				if err != nil {
					t.Fatalf("Failed to create Dockerfile: %v", err)
				}
			},
			expectBuild:  true,
			expectError:  true,
			errorMessage: "please specify a new name for rootfs.name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory for test
			tempDir, err := ioutil.TempDir("", "vers-up-test-")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tempDir)

			// Change to temp directory
			originalDir, err := os.Getwd()
			if err != nil {
				t.Fatalf("Failed to get current directory: %v", err)
			}
			defer os.Chdir(originalDir)

			err = os.Chdir(tempDir)
			if err != nil {
				t.Fatalf("Failed to change to temp directory: %v", err)
			}

			// Setup test configuration
			tt.setupConfig(t, tempDir)

			// Initialize client for integration test
			setupIntegrationClient(t)

			// Load the configuration
			config, err := loadConfig()
			if err != nil {
				t.Fatalf("Failed to load config: %v", err)
			}

			// Test the build phase if expected
			if tt.expectBuild && config.Builder.Name != "none" {
				err = BuildRootfs(config)
				if tt.expectError {
					if err == nil {
						t.Errorf("Expected build error but got none")
						return
					}
					if !strings.Contains(err.Error(), tt.errorMessage) {
						t.Errorf("Expected error containing '%s', got: %v", tt.errorMessage, err)
						return
					}
					// If we expected an error in build phase, don't continue to StartCluster
					return
				} else if err != nil {
					// Check if it's a service error vs a client error
					if strings.Contains(err.Error(), "500 Internal Server Error") {
						t.Skipf("Skipping test due to service error (this may be expected in test environment): %v", err)
						return
					}
					t.Errorf("Unexpected build error: %v", err)
					return
				}
			}

			// Test the cluster start phase
			err = StartCluster(config, []string{})
			if tt.expectError && !tt.expectBuild {
				if err == nil {
					t.Errorf("Expected cluster start error but got none")
					return
				}
				if !strings.Contains(err.Error(), tt.errorMessage) {
					t.Errorf("Expected error containing '%s', got: %v", tt.errorMessage, err)
					return
				}
			} else if err != nil {
				// Check if it's a service error vs a client error
				if strings.Contains(err.Error(), "500 Internal Server Error") {
					t.Skipf("Skipping test due to service error (this may be expected in test environment): %v", err)
					return
				}
				t.Errorf("Unexpected cluster start error: %v", err)
				return
			}

			// If we successfully started a cluster, we should clean it up
			// Note: In a real integration test, you might want to add cleanup
			// by calling the appropriate API to stop/destroy the cluster
			if !tt.expectError {
				t.Logf("Integration test '%s' completed successfully", tt.name)
			}
		})
	}
}

// setupIntegrationClient initializes the global client variable for integration tests
func setupIntegrationClient(t *testing.T) {
	// Get API key
	apiKey, err := auth.GetAPIKey()
	if err != nil {
		t.Fatalf("Failed to get API key: %v", err)
	}

	if apiKey == "" {
		t.Skip("No API key available for integration test")
	}

	// Set API key in environment
	os.Setenv("VERS_API_KEY", apiKey)

	// Get client options
	clientOptions, err := auth.GetClientOptions()
	if err != nil {
		t.Fatalf("Failed to get client options: %v", err)
	}
	if clientOptions == nil {
		t.Fatalf("Failed to get client options")
	}

	// Initialize global client
	client = vers.NewClient(clientOptions...)
}

// TestUpCommandUnit tests the up command configuration loading without calling the service
func TestUpCommandUnit(t *testing.T) {
	tests := []struct {
		name           string
		configContent  string
		expectError    bool
		expectedConfig *Config
	}{
		{
			name: "valid config",
			configContent: `
[machine]
mem_size_mib = 1024
vcpu_count = 2

[rootfs]
name = "test-rootfs"

[builder]
name = "docker"
dockerfile = "Dockerfile"

[kernel]
name = "test.bin"
`,
			expectError: false,
			expectedConfig: &Config{
				Machine: MachineConfig{
					MemSizeMib: 1024,
					VcpuCount:  2,
				},
				Rootfs: RootfsConfig{
					Name: "test-rootfs",
				},
				Builder: BuilderConfig{
					Name:       "docker",
					Dockerfile: "Dockerfile",
				},
				Kernel: KernelConfig{
					Name: "test.bin",
				},
			},
		},
		{
			name:          "defaults when no config file",
			configContent: "", // No file will be created
			expectError:   false,
			expectedConfig: &Config{
				Machine: MachineConfig{
					MemSizeMib: 512,
					VcpuCount:  1,
				},
				Rootfs: RootfsConfig{
					Name: "default",
				},
				Builder: BuilderConfig{
					Name:       "docker",
					Dockerfile: "Dockerfile",
				},
				Kernel: KernelConfig{
					Name: "default.bin",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary directory
			tempDir, err := ioutil.TempDir("", "vers-config-test-")
			if err != nil {
				t.Fatalf("Failed to create temp dir: %v", err)
			}
			defer os.RemoveAll(tempDir)

			// Change to temp directory
			originalDir, err := os.Getwd()
			if err != nil {
				t.Fatalf("Failed to get current directory: %v", err)
			}
			defer os.Chdir(originalDir)

			err = os.Chdir(tempDir)
			if err != nil {
				t.Fatalf("Failed to change to temp directory: %v", err)
			}

			// Create config file if content provided
			if tt.configContent != "" {
				err = os.WriteFile("vers.toml", []byte(tt.configContent), 0644)
				if err != nil {
					t.Fatalf("Failed to create vers.toml: %v", err)
				}
			}

			// Test loadConfig
			config, err := loadConfig()
			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Verify configuration values
			if config.Machine.MemSizeMib != tt.expectedConfig.Machine.MemSizeMib {
				t.Errorf("Expected MemSizeMib %d, got %d", tt.expectedConfig.Machine.MemSizeMib, config.Machine.MemSizeMib)
			}

			if config.Machine.VcpuCount != tt.expectedConfig.Machine.VcpuCount {
				t.Errorf("Expected VcpuCount %d, got %d", tt.expectedConfig.Machine.VcpuCount, config.Machine.VcpuCount)
			}

			if config.Rootfs.Name != tt.expectedConfig.Rootfs.Name {
				t.Errorf("Expected Rootfs.Name %s, got %s", tt.expectedConfig.Rootfs.Name, config.Rootfs.Name)
			}

			if config.Builder.Name != tt.expectedConfig.Builder.Name {
				t.Errorf("Expected Builder.Name %s, got %s", tt.expectedConfig.Builder.Name, config.Builder.Name)
			}

			if config.Kernel.Name != tt.expectedConfig.Kernel.Name {
				t.Errorf("Expected Kernel.Name %s, got %s", tt.expectedConfig.Kernel.Name, config.Kernel.Name)
			}
		})
	}
}
