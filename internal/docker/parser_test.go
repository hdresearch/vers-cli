package docker

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseInstruction(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		wantCmd  string
		wantArgs []string
	}{
		{
			name:     "FROM instruction",
			line:     "FROM node:18-alpine",
			wantCmd:  "FROM",
			wantArgs: []string{"node:18-alpine"},
		},
		{
			name:     "WORKDIR instruction",
			line:     "WORKDIR /app",
			wantCmd:  "WORKDIR",
			wantArgs: []string{"/app"},
		},
		{
			name:     "RUN instruction with shell command",
			line:     "RUN npm install",
			wantCmd:  "RUN",
			wantArgs: []string{"npm", "install"},
		},
		{
			name:     "RUN instruction with complex command",
			line:     "RUN apt-get update && apt-get install -y curl",
			wantCmd:  "RUN",
			wantArgs: []string{"apt-get", "update", "&&", "apt-get", "install", "-y", "curl"},
		},
		{
			name:     "COPY instruction",
			line:     "COPY package*.json ./",
			wantCmd:  "COPY",
			wantArgs: []string{"package*.json", "./"},
		},
		{
			name:     "EXPOSE instruction",
			line:     "EXPOSE 3000",
			wantCmd:  "EXPOSE",
			wantArgs: []string{"3000"},
		},
		{
			name:     "ENV instruction with equals",
			line:     "ENV NODE_ENV=production",
			wantCmd:  "ENV",
			wantArgs: []string{"NODE_ENV=production"},
		},
		{
			name:     "ENV instruction with space",
			line:     "ENV NODE_ENV production",
			wantCmd:  "ENV",
			wantArgs: []string{"NODE_ENV", "production"},
		},
		{
			name:     "CMD instruction JSON format",
			line:     `CMD ["npm", "start"]`,
			wantCmd:  "CMD",
			wantArgs: []string{"npm", "start"},
		},
		{
			name:     "CMD instruction shell format",
			line:     "CMD npm start",
			wantCmd:  "CMD",
			wantArgs: []string{"npm", "start"},
		},
		{
			name:     "ENTRYPOINT instruction JSON format",
			line:     `ENTRYPOINT ["node", "server.js"]`,
			wantCmd:  "ENTRYPOINT",
			wantArgs: []string{"node", "server.js"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			instr, err := parseInstruction(tt.line)
			if err != nil {
				t.Fatalf("parseInstruction() error = %v", err)
			}
			if instr.Command != tt.wantCmd {
				t.Errorf("parseInstruction() command = %v, want %v", instr.Command, tt.wantCmd)
			}
			if len(instr.Args) != len(tt.wantArgs) {
				t.Errorf("parseInstruction() args = %v, want %v", instr.Args, tt.wantArgs)
				return
			}
			for i, arg := range instr.Args {
				if arg != tt.wantArgs[i] {
					t.Errorf("parseInstruction() args[%d] = %v, want %v", i, arg, tt.wantArgs[i])
				}
			}
		})
	}
}

func TestParseDockerfile(t *testing.T) {
	// Create a temporary Dockerfile
	content := `# Test Dockerfile
FROM node:18-alpine

WORKDIR /app

ENV NODE_ENV=production

COPY package*.json ./
RUN npm install

COPY . .

EXPOSE 3000
EXPOSE 8080

CMD ["npm", "start"]
`
	tmpDir := t.TempDir()
	dockerfilePath := filepath.Join(tmpDir, "Dockerfile")
	if err := os.WriteFile(dockerfilePath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write temp Dockerfile: %v", err)
	}

	df, err := ParseDockerfile(dockerfilePath)
	if err != nil {
		t.Fatalf("ParseDockerfile() error = %v", err)
	}

	// Check base image
	if df.BaseImage != "node:18-alpine" {
		t.Errorf("BaseImage = %v, want node:18-alpine", df.BaseImage)
	}

	// Check workdir
	if df.WorkDir != "/app" {
		t.Errorf("WorkDir = %v, want /app", df.WorkDir)
	}

	// Check environment variables
	if df.Env["NODE_ENV"] != "production" {
		t.Errorf("Env[NODE_ENV] = %v, want production", df.Env["NODE_ENV"])
	}

	// Check exposed ports
	if len(df.ExposePorts) != 2 {
		t.Errorf("len(ExposePorts) = %v, want 2", len(df.ExposePorts))
	}

	// Check CMD
	if len(df.Cmd) != 2 || df.Cmd[0] != "npm" || df.Cmd[1] != "start" {
		t.Errorf("Cmd = %v, want [npm start]", df.Cmd)
	}

	// Check run commands
	runCmds := df.GetRunCommands()
	if len(runCmds) != 1 {
		t.Errorf("len(GetRunCommands()) = %v, want 1", len(runCmds))
	}

	// Check copy instructions
	copies := df.GetCopyInstructions()
	if len(copies) != 2 {
		t.Errorf("len(GetCopyInstructions()) = %v, want 2", len(copies))
	}
}

func TestParseDockerfile_MultiStage(t *testing.T) {
	content := `FROM node:18 AS builder
WORKDIR /build
COPY . .
RUN npm run build

FROM node:18-alpine
WORKDIR /app
COPY --from=builder /build/dist ./dist
CMD ["node", "dist/index.js"]
`
	tmpDir := t.TempDir()
	dockerfilePath := filepath.Join(tmpDir, "Dockerfile")
	if err := os.WriteFile(dockerfilePath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write temp Dockerfile: %v", err)
	}

	df, err := ParseDockerfile(dockerfilePath)
	if err != nil {
		t.Fatalf("ParseDockerfile() error = %v", err)
	}

	// For multi-stage builds, we use the last FROM (the final stage)
	// since that's what represents the runtime image
	if df.BaseImage != "node:18-alpine" {
		t.Errorf("BaseImage = %v, want node:18-alpine", df.BaseImage)
	}

	// Final workdir should be /app from the final stage
	if df.WorkDir != "/app" {
		t.Errorf("WorkDir = %v, want /app", df.WorkDir)
	}
}

func TestParseDockerfile_NoFROM(t *testing.T) {
	content := `WORKDIR /app
RUN echo hello
`
	tmpDir := t.TempDir()
	dockerfilePath := filepath.Join(tmpDir, "Dockerfile")
	if err := os.WriteFile(dockerfilePath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write temp Dockerfile: %v", err)
	}

	_, err := ParseDockerfile(dockerfilePath)
	if err == nil {
		t.Error("ParseDockerfile() expected error for missing FROM, got nil")
	}
}

func TestParseDockerfile_LineContinuation(t *testing.T) {
	content := `FROM ubuntu:22.04
RUN apt-get update && \
    apt-get install -y curl && \
    rm -rf /var/lib/apt/lists/*
`
	tmpDir := t.TempDir()
	dockerfilePath := filepath.Join(tmpDir, "Dockerfile")
	if err := os.WriteFile(dockerfilePath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write temp Dockerfile: %v", err)
	}

	df, err := ParseDockerfile(dockerfilePath)
	if err != nil {
		t.Fatalf("ParseDockerfile() error = %v", err)
	}

	runCmds := df.GetRunCommands()
	if len(runCmds) != 1 {
		t.Errorf("len(GetRunCommands()) = %v, want 1 (continuation should be joined)", len(runCmds))
	}
}

func TestToSetupScript(t *testing.T) {
	content := `FROM node:18
WORKDIR /app
ENV NODE_ENV=production
RUN npm install
RUN npm run build
`
	tmpDir := t.TempDir()
	dockerfilePath := filepath.Join(tmpDir, "Dockerfile")
	if err := os.WriteFile(dockerfilePath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to write temp Dockerfile: %v", err)
	}

	df, err := ParseDockerfile(dockerfilePath)
	if err != nil {
		t.Fatalf("ParseDockerfile() error = %v", err)
	}

	script := df.ToSetupScript()

	// Check script contains expected content
	if !contains(script, "#!/bin/bash") {
		t.Error("Script should start with shebang")
	}
	if !contains(script, "set -e") {
		t.Error("Script should have set -e")
	}
	if !contains(script, "NODE_ENV") {
		t.Error("Script should set NODE_ENV")
	}
	if !contains(script, "mkdir -p /app") {
		t.Error("Script should create workdir")
	}
	if !contains(script, "npm install") {
		t.Error("Script should contain npm install")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
