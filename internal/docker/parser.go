package docker

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// Instruction represents a single Dockerfile instruction
type Instruction struct {
	Command string   // FROM, RUN, COPY, WORKDIR, ENV, EXPOSE, CMD, ENTRYPOINT, etc.
	Args    []string // Arguments to the instruction
	Raw     string   // Original line (for debugging)
}

// Dockerfile represents a parsed Dockerfile
type Dockerfile struct {
	Instructions []Instruction
	BaseImage    string            // FROM image
	WorkDir      string            // Current WORKDIR
	Env          map[string]string // Environment variables
	ExposePorts  []string          // EXPOSE ports
	Cmd          []string          // CMD instruction
	Entrypoint   []string          // ENTRYPOINT instruction
}

// ParseDockerfile reads and parses a Dockerfile from the given path
func ParseDockerfile(path string) (*Dockerfile, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open Dockerfile: %w", err)
	}
	defer file.Close()

	df := &Dockerfile{
		Instructions: []Instruction{},
		Env:          make(map[string]string),
		ExposePorts:  []string{},
	}

	scanner := bufio.NewScanner(file)
	var currentLine string

	for scanner.Scan() {
		line := scanner.Text()

		// Handle line continuations
		if strings.HasSuffix(line, "\\") {
			currentLine += strings.TrimSuffix(line, "\\") + " "
			continue
		}
		currentLine += line

		// Skip empty lines and comments
		trimmed := strings.TrimSpace(currentLine)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			currentLine = ""
			continue
		}

		instr, err := parseInstruction(trimmed)
		if err != nil {
			currentLine = ""
			continue // Skip invalid instructions
		}

		df.Instructions = append(df.Instructions, instr)

		// Extract key information
		switch instr.Command {
		case "FROM":
			if len(instr.Args) > 0 {
				df.BaseImage = instr.Args[0]
			}
		case "WORKDIR":
			if len(instr.Args) > 0 {
				df.WorkDir = instr.Args[0]
			}
		case "ENV":
			if len(instr.Args) >= 2 {
				df.Env[instr.Args[0]] = strings.Join(instr.Args[1:], " ")
			} else if len(instr.Args) == 1 && strings.Contains(instr.Args[0], "=") {
				parts := strings.SplitN(instr.Args[0], "=", 2)
				df.Env[parts[0]] = parts[1]
			}
		case "EXPOSE":
			df.ExposePorts = append(df.ExposePorts, instr.Args...)
		case "CMD":
			df.Cmd = instr.Args
		case "ENTRYPOINT":
			df.Entrypoint = instr.Args
		}

		currentLine = ""
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading Dockerfile: %w", err)
	}

	if df.BaseImage == "" {
		return nil, fmt.Errorf("Dockerfile must have a FROM instruction")
	}

	return df, nil
}

// parseInstruction parses a single Dockerfile instruction line
func parseInstruction(line string) (Instruction, error) {
	// Match instruction pattern: INSTRUCTION args...
	parts := strings.SplitN(line, " ", 2)
	if len(parts) == 0 {
		return Instruction{}, fmt.Errorf("empty instruction")
	}

	cmd := strings.ToUpper(parts[0])
	var args []string

	if len(parts) > 1 {
		argStr := strings.TrimSpace(parts[1])

		// Handle JSON array format: ["arg1", "arg2"]
		if strings.HasPrefix(argStr, "[") && strings.HasSuffix(argStr, "]") {
			args = parseJSONArray(argStr)
		} else {
			// Handle shell format
			args = parseShellArgs(argStr)
		}
	}

	return Instruction{
		Command: cmd,
		Args:    args,
		Raw:     line,
	}, nil
}

// parseJSONArray parses JSON array format arguments
func parseJSONArray(s string) []string {
	// Remove brackets
	s = strings.TrimPrefix(s, "[")
	s = strings.TrimSuffix(s, "]")

	// Match quoted strings
	re := regexp.MustCompile(`"([^"]*)"`)
	matches := re.FindAllStringSubmatch(s, -1)

	var args []string
	for _, match := range matches {
		if len(match) > 1 {
			args = append(args, match[1])
		}
	}
	return args
}

// parseShellArgs parses shell-style arguments
func parseShellArgs(s string) []string {
	// Simple split by spaces, but respect quotes
	var args []string
	var current strings.Builder
	inQuote := false
	quoteChar := rune(0)

	for _, ch := range s {
		switch {
		case (ch == '"' || ch == '\'') && !inQuote:
			inQuote = true
			quoteChar = ch
		case ch == quoteChar && inQuote:
			inQuote = false
			quoteChar = 0
		case ch == ' ' && !inQuote:
			if current.Len() > 0 {
				args = append(args, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(ch)
		}
	}
	if current.Len() > 0 {
		args = append(args, current.String())
	}

	return args
}

// GetRunCommands returns all RUN instruction commands
func (df *Dockerfile) GetRunCommands() []string {
	var commands []string
	for _, instr := range df.Instructions {
		if instr.Command == "RUN" {
			commands = append(commands, strings.Join(instr.Args, " "))
		}
	}
	return commands
}

// GetCopyInstructions returns all COPY/ADD instructions
func (df *Dockerfile) GetCopyInstructions() []Instruction {
	var copies []Instruction
	for _, instr := range df.Instructions {
		if instr.Command == "COPY" || instr.Command == "ADD" {
			copies = append(copies, instr)
		}
	}
	return copies
}

// GetStartCommand returns the command to run (CMD or ENTRYPOINT)
func (df *Dockerfile) GetStartCommand() []string {
	if len(df.Entrypoint) > 0 && len(df.Cmd) > 0 {
		// Combine ENTRYPOINT with CMD as arguments
		return append(df.Entrypoint, df.Cmd...)
	}
	if len(df.Entrypoint) > 0 {
		return df.Entrypoint
	}
	return df.Cmd
}

// ToSetupScript converts Dockerfile instructions to a bash setup script
func (df *Dockerfile) ToSetupScript() string {
	var script strings.Builder

	script.WriteString("#!/bin/bash\nset -e\n\n")

	// Set environment variables
	for key, value := range df.Env {
		script.WriteString(fmt.Sprintf("export %s=%q\n", key, value))
	}
	if len(df.Env) > 0 {
		script.WriteString("\n")
	}

	// Create and change to workdir
	if df.WorkDir != "" {
		script.WriteString(fmt.Sprintf("mkdir -p %s\n", df.WorkDir))
		script.WriteString(fmt.Sprintf("cd %s\n\n", df.WorkDir))
	}

	// Run commands
	for _, instr := range df.Instructions {
		if instr.Command == "RUN" {
			script.WriteString(strings.Join(instr.Args, " "))
			script.WriteString("\n")
		}
	}

	return script.String()
}
