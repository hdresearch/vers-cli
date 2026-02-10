package utils

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/hdresearch/vers-cli/internal/auth"
	"github.com/hdresearch/vers-sdk-go"
)

// Mock server setup helpers
func setupMockServer(tb testing.TB, nodeIP string, statusCode int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request has proper authorization
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// New SDK uses /api/v1/vms path
		if !strings.Contains(r.URL.Path, "/vms") {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		// Set the node IP header if provided
		if nodeIP != "" && nodeIP != "unknown" {
			w.Header().Set("X-Node-IP", nodeIP)
		}

		// Set content-type for JSON response
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(statusCode)
		// Return proper VM list response structure for new SDK
		w.Write([]byte(`[{
			"vm_id": "test-vm-123",
			"ip": "192.168.1.100",
			"parent": "",
			"owner_id": "owner-123",
			"created_at": "2024-01-01T00:00:00Z"
		}]`))
	}))
}

// setupTestEnvironment sets up a clean test environment
func setupTestEnvironment(serverURL string) (restore func()) {
	// Save original environment
	originalVals := map[string]string{
		"VERS_URL":      os.Getenv("VERS_URL"),
		"VERS_API_KEY":  os.Getenv("VERS_API_KEY"),
		"VERS_BASE_URL": os.Getenv("VERS_BASE_URL"),
	}

	// Set test environment
	os.Setenv("VERS_URL", serverURL)
	os.Setenv("VERS_API_KEY", "test-api-key")
	os.Setenv("VERS_BASE_URL", serverURL)

	return func() {
		for key, val := range originalVals {
			if val != "" {
				os.Setenv(key, val)
			} else {
				os.Unsetenv(key)
			}
		}
	}
}

// setupTestClient creates a test client using the auth package (which respects env vars)
func setupTestClient() (*vers.Client, error) {
	clientOptions, err := auth.GetClientOptions()
	if err != nil {
		return vers.NewClient(), nil
	}
	return vers.NewClient(clientOptions...), nil
}

func TestGetVmAndNodeIP_WithAlias(t *testing.T) {
	// Setup mock server
	server := setupMockServer(t, "192.168.1.100", http.StatusOK)
	defer server.Close()

	// Setup test environment
	restore := setupTestEnvironment(server.URL)
	defer restore()

	// Setup a temporary alias mapping "my-server" -> "test-vm-123"
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get home dir: %v", err)
	}
	aliasDir := homeDir + "/.vers"
	aliasFile := aliasDir + "/aliases.json"

	// Backup existing aliases file if present
	existingAliases, hadExisting := backupFile(t, aliasFile)
	defer restoreFile(t, aliasFile, existingAliases, hadExisting)

	// Write test alias
	if err := os.MkdirAll(aliasDir, 0755); err != nil {
		t.Fatalf("failed to create alias dir: %v", err)
	}
	if err := os.WriteFile(aliasFile, []byte(`{"my-server": "test-vm-123"}`), 0644); err != nil {
		t.Fatalf("failed to write alias file: %v", err)
	}

	// Create test client
	client, err := setupTestClient()
	if err != nil {
		t.Fatalf("Failed to create test client: %v", err)
	}

	ctx := context.Background()

	// Test that alias "my-server" resolves to VM "test-vm-123"
	vm, nodeIP, err := GetVmAndNodeIP(ctx, client, "my-server")

	if err != nil {
		t.Fatalf("Expected no error when using alias, got: %v", err)
	}
	if nodeIP == "" {
		t.Error("Expected non-empty node IP")
	}
	if vm.VmID != "test-vm-123" {
		t.Errorf("Expected VM ID test-vm-123, got %s", vm.VmID)
	}
}

func TestGetVmAndNodeIP_AliasNotFound(t *testing.T) {
	// Setup mock server that returns a VM list NOT containing the aliased ID
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if !strings.Contains(r.URL.Path, "/vms") {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`[{
			"vm_id": "other-vm-456",
			"ip": "192.168.1.101",
			"parent": "",
			"owner_id": "owner-123",
			"created_at": "2024-01-01T00:00:00Z"
		}]`))
	}))
	defer server.Close()

	restore := setupTestEnvironment(server.URL)
	defer restore()

	// Setup alias pointing to a VM not in the list
	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("failed to get home dir: %v", err)
	}
	aliasFile := homeDir + "/.vers/aliases.json"
	existingAliases, hadExisting := backupFile(t, aliasFile)
	defer restoreFile(t, aliasFile, existingAliases, hadExisting)

	if err := os.MkdirAll(homeDir+"/.vers", 0755); err != nil {
		t.Fatalf("failed to create alias dir: %v", err)
	}
	if err := os.WriteFile(aliasFile, []byte(`{"my-alias": "nonexistent-vm"}`), 0644); err != nil {
		t.Fatalf("failed to write alias file: %v", err)
	}

	client, err := setupTestClient()
	if err != nil {
		t.Fatalf("Failed to create test client: %v", err)
	}

	ctx := context.Background()

	// Should fail with descriptive error mentioning both alias and resolved ID
	_, _, err = GetVmAndNodeIP(ctx, client, "my-alias")
	if err == nil {
		t.Fatal("Expected error when alias resolves to non-existent VM")
	}
	if !strings.Contains(err.Error(), "my-alias") {
		t.Errorf("Expected error to mention alias 'my-alias', got: %v", err)
	}
	if !strings.Contains(err.Error(), "nonexistent-vm") {
		t.Errorf("Expected error to mention resolved ID 'nonexistent-vm', got: %v", err)
	}
}

// backupFile reads and returns a file's contents, and whether it existed
func backupFile(t *testing.T, path string) ([]byte, bool) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}
	return data, true
}

// restoreFile restores a file to its previous state
func restoreFile(t *testing.T, path string, data []byte, existed bool) {
	t.Helper()
	if existed {
		os.WriteFile(path, data, 0644)
	} else {
		os.Remove(path)
	}
}

func TestGetVmAndNodeIP_Success(t *testing.T) {
	// Setup mock server
	server := setupMockServer(t, "192.168.1.100", http.StatusOK)
	defer server.Close()

	// Setup test environment
	restore := setupTestEnvironment(server.URL)
	defer restore()

	// Create test client
	client, err := setupTestClient()
	if err != nil {
		t.Fatalf("Failed to create test client: %v", err)
	}

	ctx := context.Background()

	// Test the function
	vm, nodeIP, err := GetVmAndNodeIP(ctx, client, "test-vm-123")

	// Assertions
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	// The function returns the hostname from VERS_URL, not the X-Node-IP header
	if nodeIP == "" {
		t.Error("Expected non-empty node IP")
	}
	if vm.VmID != "test-vm-123" {
		t.Errorf("Expected VM ID test-vm-123, got %s", vm.VmID)
	}
}

func TestGetVmAndNodeIP_NoNodeIPHeader(t *testing.T) {
	// Setup mock server without node IP header
	server := setupMockServer(t, "", http.StatusOK)
	defer server.Close()

	// Setup test environment
	restore := setupTestEnvironment(server.URL)
	defer restore()

	// Create test client
	client, err := setupTestClient()
	if err != nil {
		t.Fatalf("Failed to create test client: %v", err)
	}

	ctx := context.Background()

	// Test the function
	vm, nodeIP, err := GetVmAndNodeIP(ctx, client, "test-vm-123")

	// Assertions - should succeed with fallback IP
	if err != nil {
		t.Fatalf("Expected no error with fallback, got: %v", err)
	}
	if nodeIP == "" {
		t.Error("Expected fallback node IP, got empty string")
	}
	if vm.VmID != "test-vm-123" {
		t.Errorf("Expected VM ID test-vm-123, got %s", vm.VmID)
	}
	// Should use fallback host from auth.GetVersUrlHost()
	t.Logf("Fallback node IP used: %s", nodeIP)
}

func TestGetVmAndNodeIP_UnknownNodeIP(t *testing.T) {
	// Setup mock server with "unknown" node IP
	server := setupMockServer(t, "unknown", http.StatusOK)
	defer server.Close()

	// Setup test environment
	restore := setupTestEnvironment(server.URL)
	defer restore()

	// Create test client
	client, err := setupTestClient()
	if err != nil {
		t.Fatalf("Failed to create test client: %v", err)
	}

	ctx := context.Background()

	// Test the function
	vm, nodeIP, err := GetVmAndNodeIP(ctx, client, "test-vm-123")

	// Assertions - should succeed with fallback IP since "unknown" is treated as empty
	if err != nil {
		t.Fatalf("Expected no error with fallback, got: %v", err)
	}
	if nodeIP == "" {
		t.Error("Expected fallback node IP, got empty string")
	}
	if vm.VmID != "test-vm-123" {
		t.Errorf("Expected VM ID test-vm-123, got %s", vm.VmID)
	}
}

func TestGetVmAndNodeIP_HTTPError(t *testing.T) {
	// Setup mock server that returns error
	server := setupMockServer(t, "", http.StatusInternalServerError)
	defer server.Close()

	// Setup test environment
	restore := setupTestEnvironment(server.URL)
	defer restore()

	// Create test client
	client, err := setupTestClient()
	if err != nil {
		t.Fatalf("Failed to create test client: %v", err)
	}

	ctx := context.Background()

	// Test the function
	vm, nodeIP, err := GetVmAndNodeIP(ctx, client, "test-vm-123")

	// Assertions
	if err == nil {
		t.Fatal("Expected error when server returns error status")
	}
	if nodeIP != "" {
		t.Errorf("Expected empty node IP, got %s", nodeIP)
	}
	if vm != nil {
		t.Errorf("Expected nil VM, got %+v", vm)
	}
}

func TestGetVmAndNodeIP_InvalidJSON(t *testing.T) {
	// Setup mock server that returns invalid JSON
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request has proper authorization
		authHeader := r.Header.Get("Authorization")
		if !strings.HasPrefix(authHeader, "Bearer ") {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		w.Header().Set("X-Node-IP", "192.168.1.100")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`invalid json response`))
	}))
	defer server.Close()

	// Setup test environment
	restore := setupTestEnvironment(server.URL)
	defer restore()

	// Create test client
	client, err := setupTestClient()
	if err != nil {
		t.Fatalf("Failed to create test client: %v", err)
	}

	ctx := context.Background()

	// Test the function
	vm, nodeIP, err := GetVmAndNodeIP(ctx, client, "test-vm-123")

	// Assertions
	if err == nil {
		t.Fatal("Expected error when response contains invalid JSON")
	}
	if !strings.Contains(err.Error(), "error parsing") && !strings.Contains(err.Error(), "invalid character") {
		t.Errorf("Expected JSON parsing error, got: %v", err)
	}
	if nodeIP != "" {
		t.Errorf("Expected empty node IP, got %s", nodeIP)
	}
	if vm != nil {
		t.Errorf("Expected nil VM, got %+v", vm)
	}
}

func TestGetVmAndNodeIP_ContextTimeout(t *testing.T) {
	// Setup a server that delays response beyond timeout
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Delay longer than our context timeout
		time.Sleep(2 * time.Second)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": {"id": "test-vm"}}`))
	}))
	defer server.Close()

	// Setup test environment
	restore := setupTestEnvironment(server.URL)
	defer restore()

	// Create test client
	client, err := setupTestClient()
	if err != nil {
		t.Fatalf("Failed to create test client: %v", err)
	}

	// Create context with short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Test the function
	start := time.Now()
	vm, nodeIP, err := GetVmAndNodeIP(ctx, client, "test-vm-123")
	duration := time.Since(start)

	// Assertions
	if err == nil {
		t.Fatal("Expected timeout error")
	}
	if nodeIP != "" {
		t.Errorf("Expected empty node IP, got %s", nodeIP)
	}
	if vm != nil {
		t.Errorf("Expected nil VM, got %+v", vm)
	}
	if duration > 1*time.Second {
		t.Error("Request should have timed out quickly")
	}
}

func TestGetVmAndNodeIP_FallbackBehavior(t *testing.T) {
	// Setup mock server without node IP header to test fallback
	server := setupMockServer(t, "", http.StatusOK)
	defer server.Close()

	// Setup test environment with debug mode
	restore := setupTestEnvironment(server.URL)
	defer restore()

	// Enable debug mode
	originalDebug := os.Getenv("VERS_DEBUG")
	os.Setenv("VERS_DEBUG", "true")
	defer func() {
		if originalDebug != "" {
			os.Setenv("VERS_DEBUG", originalDebug)
		} else {
			os.Unsetenv("VERS_DEBUG")
		}
	}()

	// Create test client
	client, err := setupTestClient()
	if err != nil {
		t.Fatalf("Failed to create test client: %v", err)
	}

	ctx := context.Background()

	// Test the function
	vm, nodeIP, err := GetVmAndNodeIP(ctx, client, "test-vm-123")

	// Assertions
	if err != nil {
		t.Fatalf("Expected no error with fallback, got: %v", err)
	}
	if nodeIP == "" {
		t.Error("Expected fallback node IP, got empty string")
	}
	if vm.VmID != "test-vm-123" {
		t.Errorf("Expected VM ID test-vm-123, got %s", vm.VmID)
	}
	// In debug mode, a debug message should be printed
	t.Logf("Fallback node IP used: %s", nodeIP)
}

func TestGetVmAndNodeIP_AuthFailure(t *testing.T) {
	// Setup mock server that checks for proper authorization
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Always return unauthorized
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "unauthorized"}`))
	}))
	defer server.Close()

	// Setup test environment
	restore := setupTestEnvironment(server.URL)
	defer restore()

	// Create test client
	client, err := setupTestClient()
	if err != nil {
		t.Fatalf("Failed to create test client: %v", err)
	}

	ctx := context.Background()

	// Test the function
	vm, nodeIP, err := GetVmAndNodeIP(ctx, client, "test-vm-123")

	// Assertions
	if err == nil {
		t.Fatal("Expected error when authentication fails")
	}
	if nodeIP != "" {
		t.Errorf("Expected empty node IP, got %s", nodeIP)
	}
	if vm != nil {
		t.Errorf("Expected nil VM, got %+v", vm)
	}
}

// Integration test with real API (only runs when explicitly requested)
func TestGetVmAndNodeIP_Integration(t *testing.T) {
	// Skip if we're in short test mode
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Check if we have real authentication
	hasAPIKey, err := auth.HasAPIKey()
	if err != nil || !hasAPIKey {
		t.Skip("Skipping integration test: No API key available")
	}

	// Use real configuration
	clientOptions, err := auth.GetClientOptions()
	if err != nil {
		t.Fatalf("Failed to get client options: %v", err)
	}

	client := vers.NewClient(clientOptions...)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Try with a non-existent VM ID - we expect this to fail gracefully
	_, _, err = GetVmAndNodeIP(ctx, client, "nonexistent-vm-id-12345")

	// We expect an error (VM not found), but not a panic or crash
	if err == nil {
		t.Log("Unexpected success with nonexistent VM ID")
	} else {
		t.Logf("Expected error with nonexistent VM: %v", err)
	}
}

// Benchmark test
func BenchmarkGetVmAndNodeIP(b *testing.B) {
	// Setup mock server
	server := setupMockServer(b, "192.168.1.100", http.StatusOK)
	defer server.Close()

	// Setup test environment
	restore := setupTestEnvironment(server.URL)
	defer restore()

	// Create test client
	client, err := setupTestClient()
	if err != nil {
		b.Fatalf("Failed to create test client: %v", err)
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := GetVmAndNodeIP(ctx, client, "test-vm-123")
		if err != nil {
			b.Fatalf("Benchmark failed: %v", err)
		}
	}
}
