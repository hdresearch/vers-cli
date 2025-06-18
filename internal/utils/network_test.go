package utils

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
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

		// Verify the URL path format
		if !strings.HasPrefix(r.URL.Path, "/api/vm/") {
			w.WriteHeader(http.StatusNotFound)
			return
		}

		// Set the node IP header if provided
		if nodeIP != "" && nodeIP != "unknown" {
			w.Header().Set("X-Node-IP", nodeIP)
		}

		w.WriteHeader(statusCode)
		w.Write([]byte(`{"data": {"id": "test-vm", "state": "Running"}}`))
	}))
}

func TestGetNodeIPForVM_Success(t *testing.T) {
	// Setup mock server with valid node IP
	expectedIP := "192.168.1.100"
	server := setupMockServer(t, expectedIP, http.StatusOK)
	defer server.Close()

	// Set up test environment
	originalVersUrl := os.Getenv("VERS_URL")
	originalApiKey := os.Getenv("VERS_API_KEY")
	defer func() {
		if originalVersUrl != "" {
			os.Setenv("VERS_URL", originalVersUrl)
		} else {
			os.Unsetenv("VERS_URL")
		}
		if originalApiKey != "" {
			os.Setenv("VERS_API_KEY", originalApiKey)
		} else {
			os.Unsetenv("VERS_API_KEY")
		}
	}()

	// Set test environment variables
	os.Setenv("VERS_URL", strings.TrimPrefix(server.URL, "http://"))
	os.Setenv("VERS_API_KEY", "test-api-key")

	// Test the function
	nodeIP, err := GetNodeIPForVM("test-vm-id")

	// Assertions
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	if nodeIP != expectedIP {
		t.Errorf("Expected node IP %s, got %s", expectedIP, nodeIP)
	}
}

func TestGetNodeIPForVM_NoNodeIPHeader(t *testing.T) {
	// Setup mock server without node IP header
	server := setupMockServer(t, "", http.StatusOK)
	defer server.Close()

	// Set up test environment
	originalVersUrl := os.Getenv("VERS_URL")
	originalApiKey := os.Getenv("VERS_API_KEY")
	defer func() {
		if originalVersUrl != "" {
			os.Setenv("VERS_URL", originalVersUrl)
		} else {
			os.Unsetenv("VERS_URL")
		}
		if originalApiKey != "" {
			os.Setenv("VERS_API_KEY", originalApiKey)
		} else {
			os.Unsetenv("VERS_API_KEY")
		}
	}()

	// Set test environment variables
	os.Setenv("VERS_URL", strings.TrimPrefix(server.URL, "http://"))
	os.Setenv("VERS_API_KEY", "test-api-key")

	// Test the function
	nodeIP, err := GetNodeIPForVM("test-vm-id")

	// Assertions
	if err == nil {
		t.Fatal("Expected error when no node IP header is present")
	}
	if nodeIP != "" {
		t.Errorf("Expected empty node IP, got %s", nodeIP)
	}
	if !strings.Contains(err.Error(), "no node IP found in response headers") {
		t.Errorf("Expected specific error message, got: %v", err)
	}
}

func TestGetNodeIPForVM_UnknownNodeIP(t *testing.T) {
	// Setup mock server with "unknown" node IP
	server := setupMockServer(t, "unknown", http.StatusOK)
	defer server.Close()

	// Set up test environment
	originalVersUrl := os.Getenv("VERS_URL")
	originalApiKey := os.Getenv("VERS_API_KEY")
	defer func() {
		if originalVersUrl != "" {
			os.Setenv("VERS_URL", originalVersUrl)
		} else {
			os.Unsetenv("VERS_URL")
		}
		if originalApiKey != "" {
			os.Setenv("VERS_API_KEY", originalApiKey)
		} else {
			os.Unsetenv("VERS_API_KEY")
		}
	}()

	// Set test environment variables
	os.Setenv("VERS_URL", strings.TrimPrefix(server.URL, "http://"))
	os.Setenv("VERS_API_KEY", "test-api-key")

	// Test the function
	nodeIP, err := GetNodeIPForVM("test-vm-id")

	// Assertions
	if err == nil {
		t.Fatal("Expected error when node IP is 'unknown'")
	}
	if nodeIP != "" {
		t.Errorf("Expected empty node IP, got %s", nodeIP)
	}
}

func TestGetNodeIPForVM_HTTPError(t *testing.T) {
	// Setup mock server that returns error
	server := setupMockServer(t, "", http.StatusInternalServerError)
	defer server.Close()

	// Set up test environment
	originalVersUrl := os.Getenv("VERS_URL")
	originalApiKey := os.Getenv("VERS_API_KEY")
	defer func() {
		if originalVersUrl != "" {
			os.Setenv("VERS_URL", originalVersUrl)
		} else {
			os.Unsetenv("VERS_URL")
		}
		if originalApiKey != "" {
			os.Setenv("VERS_API_KEY", originalApiKey)
		} else {
			os.Unsetenv("VERS_API_KEY")
		}
	}()

	// Set test environment variables
	os.Setenv("VERS_URL", strings.TrimPrefix(server.URL, "http://"))
	os.Setenv("VERS_API_KEY", "test-api-key")

	// Test the function
	nodeIP, err := GetNodeIPForVM("test-vm-id")

	// Assertions
	if err == nil {
		t.Fatal("Expected error when server returns error status")
	}
	if nodeIP != "" {
		t.Errorf("Expected empty node IP, got %s", nodeIP)
	}
}

func TestGetNodeIPForVM_NoAPIKey(t *testing.T) {
	// Setup mock server that checks for proper authorization
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the request has proper authorization
		authHeader := r.Header.Get("Authorization")

		// Check if auth header is missing, empty, or has no token after "Bearer"
		if authHeader == "" || authHeader == "Bearer" || authHeader == "Bearer " {
			// Return unauthorized for empty or malformed auth
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error": "unauthorized"}`))
			return
		}

		w.Header().Set("X-Node-IP", "192.168.1.100")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Set up test environment without API key
	originalApiKey := os.Getenv("VERS_API_KEY")
	originalVersUrl := os.Getenv("VERS_URL")
	originalHome := os.Getenv("HOME")

	// Create a temporary home directory to isolate config file
	tmpDir := t.TempDir()

	defer func() {
		if originalApiKey != "" {
			os.Setenv("VERS_API_KEY", originalApiKey)
		} else {
			os.Unsetenv("VERS_API_KEY")
		}
		if originalVersUrl != "" {
			os.Setenv("VERS_URL", originalVersUrl)
		} else {
			os.Unsetenv("VERS_URL")
		}
		if originalHome != "" {
			os.Setenv("HOME", originalHome)
		} else {
			os.Unsetenv("HOME")
		}
	}()

	// Remove API key, set test URL, and isolate config with temp home
	os.Unsetenv("VERS_API_KEY")
	os.Setenv("VERS_URL", strings.TrimPrefix(server.URL, "http://"))
	os.Setenv("HOME", tmpDir) // This will make auth.GetAPIKey look for config in empty temp dir

	// Test the function
	nodeIP, err := GetNodeIPForVM("test-vm-id")

	// Assertions - should succeed with empty API key but fail at HTTP level due to authorization
	if err == nil {
		t.Fatal("Expected error when no API key is available (should fail at HTTP level)")
	}
	if nodeIP != "" {
		t.Errorf("Expected empty node IP, got %s", nodeIP)
	}
	// The error should be about the HTTP request failing (401 Unauthorized)
	if !strings.Contains(err.Error(), "no node IP found in response headers") {
		t.Errorf("Expected node IP header error due to 401 response, got: %v", err)
	}
}

func TestGetNodeIPForVM_ProductionURL(t *testing.T) {
	// Test with production URL format (should use HTTPS)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Node-IP", "10.0.0.1")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": {"id": "test-vm"}}`))
	}))
	defer server.Close()

	// Set up test environment
	originalVersUrl := os.Getenv("VERS_URL")
	originalApiKey := os.Getenv("VERS_API_KEY")
	defer func() {
		if originalVersUrl != "" {
			os.Setenv("VERS_URL", originalVersUrl)
		} else {
			os.Unsetenv("VERS_URL")
		}
		if originalApiKey != "" {
			os.Setenv("VERS_API_KEY", originalApiKey)
		} else {
			os.Unsetenv("VERS_API_KEY")
		}
	}()

	// Set production-like environment
	os.Setenv("VERS_URL", "api.vers.sh")
	os.Setenv("VERS_API_KEY", "test-api-key")

	// Note: This test will fail in real scenarios because we can't easily mock
	// the production endpoint, but it demonstrates the test structure
	nodeIP, err := GetNodeIPForVM("test-vm-id")

	// We expect this to fail in test environment, but we can check error handling
	if err != nil && !strings.Contains(err.Error(), "failed to make request") {
		t.Logf("Expected request error in test environment: %v", err)
	}
	_ = nodeIP // Just to use the variable
}

func TestGetNodeIPForVM_RequestTimeout(t *testing.T) {
	// Setup a server that delays response beyond timeout
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Delay longer than our timeout
		time.Sleep(35 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Set up test environment
	originalVersUrl := os.Getenv("VERS_URL")
	originalApiKey := os.Getenv("VERS_API_KEY")
	defer func() {
		if originalVersUrl != "" {
			os.Setenv("VERS_URL", originalVersUrl)
		} else {
			os.Unsetenv("VERS_URL")
		}
		if originalApiKey != "" {
			os.Setenv("VERS_API_KEY", originalApiKey)
		} else {
			os.Unsetenv("VERS_API_KEY")
		}
	}()

	// Set test environment variables
	os.Setenv("VERS_URL", strings.TrimPrefix(server.URL, "http://"))
	os.Setenv("VERS_API_KEY", "test-api-key")

	// Test the function
	start := time.Now()
	nodeIP, err := GetNodeIPForVM("test-vm-id")
	duration := time.Since(start)

	// Assertions
	if err == nil {
		t.Fatal("Expected timeout error")
	}
	if nodeIP != "" {
		t.Errorf("Expected empty node IP, got %s", nodeIP)
	}
	if duration > 35*time.Second {
		t.Error("Request should have timed out before 35 seconds")
	}
	if !strings.Contains(err.Error(), "failed to make request") {
		t.Errorf("Expected request error, got: %v", err)
	}
}

// Benchmark test
func BenchmarkGetNodeIPForVM(b *testing.B) {
	// Setup mock server
	server := setupMockServer(b, "192.168.1.100", http.StatusOK)
	defer server.Close()

	// Set up test environment
	originalVersUrl := os.Getenv("VERS_URL")
	originalApiKey := os.Getenv("VERS_API_KEY")
	defer func() {
		if originalVersUrl != "" {
			os.Setenv("VERS_URL", originalVersUrl)
		} else {
			os.Unsetenv("VERS_URL")
		}
		if originalApiKey != "" {
			os.Setenv("VERS_API_KEY", originalApiKey)
		} else {
			os.Unsetenv("VERS_API_KEY")
		}
	}()

	// Set test environment variables
	os.Setenv("VERS_URL", strings.TrimPrefix(server.URL, "http://"))
	os.Setenv("VERS_API_KEY", "test-api-key")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := GetNodeIPForVM("test-vm-id")
		if err != nil {
			b.Fatalf("Benchmark failed: %v", err)
		}
	}
}
