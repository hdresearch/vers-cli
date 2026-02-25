package docker

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseComposeFile(t *testing.T) {
	// Create a temporary compose file
	tmpDir := t.TempDir()
	composePath := filepath.Join(tmpDir, "docker-compose.yml")

	composeContent := `
version: "3.8"
services:
  web:
    build:
      context: ./web
      dockerfile: Dockerfile.prod
    ports:
      - "3000:3000"
    environment:
      - NODE_ENV=production
      - API_URL=http://api:8080
    depends_on:
      - api
      - db
  api:
    build: ./api
    ports:
      - "8080:8080"
    environment:
      DEBUG: "true"
    depends_on:
      - db
  db:
    image: postgres:15
    environment:
      POSTGRES_PASSWORD: secret
      POSTGRES_DB: myapp
`
	if err := os.WriteFile(composePath, []byte(composeContent), 0644); err != nil {
		t.Fatalf("Failed to write compose file: %v", err)
	}

	// Parse the file
	compose, err := ParseComposeFile(composePath)
	if err != nil {
		t.Fatalf("Failed to parse compose file: %v", err)
	}

	// Verify version
	if compose.Version != "3.8" {
		t.Errorf("Expected version 3.8, got %s", compose.Version)
	}

	// Verify services
	if len(compose.Services) != 3 {
		t.Errorf("Expected 3 services, got %d", len(compose.Services))
	}

	// Check web service
	web, ok := compose.Services["web"]
	if !ok {
		t.Fatal("web service not found")
	}
	if web.Build == nil {
		t.Fatal("web service should have build config")
	}
	if web.Build.Context != "./web" {
		t.Errorf("Expected build context ./web, got %s", web.Build.Context)
	}
	if web.Build.Dockerfile != "Dockerfile.prod" {
		t.Errorf("Expected Dockerfile.prod, got %s", web.Build.Dockerfile)
	}
	if len(web.Ports) != 1 || web.Ports[0] != "3000:3000" {
		t.Errorf("Unexpected ports: %v", web.Ports)
	}

	// Check api service (short build format)
	api, ok := compose.Services["api"]
	if !ok {
		t.Fatal("api service not found")
	}
	if api.Build == nil {
		t.Fatal("api service should have build config")
	}
	if api.Build.Context != "./api" {
		t.Errorf("Expected build context ./api, got %s", api.Build.Context)
	}
	if api.Build.Dockerfile != "Dockerfile" {
		t.Errorf("Expected default Dockerfile, got %s", api.Build.Dockerfile)
	}

	// Check db service (image-based)
	db, ok := compose.Services["db"]
	if !ok {
		t.Fatal("db service not found")
	}
	if db.Image != "postgres:15" {
		t.Errorf("Expected postgres:15, got %s", db.Image)
	}
}

func TestParseServices(t *testing.T) {
	tmpDir := t.TempDir()
	composePath := filepath.Join(tmpDir, "docker-compose.yml")

	composeContent := `
version: "3"
services:
  frontend:
    build: ./frontend
    depends_on:
      - backend
  backend:
    build: ./backend
    depends_on:
      - database
  database:
    image: mysql:8
`
	if err := os.WriteFile(composePath, []byte(composeContent), 0644); err != nil {
		t.Fatalf("Failed to write compose file: %v", err)
	}

	compose, err := ParseComposeFile(composePath)
	if err != nil {
		t.Fatalf("Failed to parse compose file: %v", err)
	}

	services, err := compose.ParseServices(tmpDir)
	if err != nil {
		t.Fatalf("Failed to parse services: %v", err)
	}

	// Verify topological order: database -> backend -> frontend
	if len(services) != 3 {
		t.Fatalf("Expected 3 services, got %d", len(services))
	}

	if services[0].Name != "database" {
		t.Errorf("Expected database first, got %s", services[0].Name)
	}
	if services[1].Name != "backend" {
		t.Errorf("Expected backend second, got %s", services[1].Name)
	}
	if services[2].Name != "frontend" {
		t.Errorf("Expected frontend third, got %s", services[2].Name)
	}
}

func TestCircularDependency(t *testing.T) {
	tmpDir := t.TempDir()
	composePath := filepath.Join(tmpDir, "docker-compose.yml")

	composeContent := `
version: "3"
services:
  a:
    image: alpine
    depends_on:
      - b
  b:
    image: alpine
    depends_on:
      - c
  c:
    image: alpine
    depends_on:
      - a
`
	if err := os.WriteFile(composePath, []byte(composeContent), 0644); err != nil {
		t.Fatalf("Failed to write compose file: %v", err)
	}

	compose, err := ParseComposeFile(composePath)
	if err != nil {
		t.Fatalf("Failed to parse compose file: %v", err)
	}

	_, err = compose.ParseServices(tmpDir)
	if err == nil {
		t.Fatal("Expected circular dependency error")
	}
}

func TestParseEnvironment(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected map[string]string
	}{
		{
			name:  "list format",
			input: []interface{}{"FOO=bar", "BAZ=qux"},
			expected: map[string]string{
				"FOO": "bar",
				"BAZ": "qux",
			},
		},
		{
			name: "map format",
			input: map[string]interface{}{
				"FOO": "bar",
				"BAZ": "qux",
			},
			expected: map[string]string{
				"FOO": "bar",
				"BAZ": "qux",
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := parseEnvironment(tc.input)
			for k, v := range tc.expected {
				if result[k] != v {
					t.Errorf("Expected %s=%s, got %s=%s", k, v, k, result[k])
				}
			}
		})
	}
}

func TestParseMemoryLimit(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"512m", 512},
		{"1g", 1024},
		{"2G", 2048},
		{"256M", 256},
		{"100", 256}, // Minimum
		{"", 1024},   // Default
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			result := parseMemoryLimit(tc.input)
			if result != tc.expected {
				t.Errorf("parseMemoryLimit(%q) = %d, expected %d", tc.input, result, tc.expected)
			}
		})
	}
}
