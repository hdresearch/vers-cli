# Vers CLI

A command-line interface for managing virtual machine/container-based development environments.

## Features

- **Lifecycle Management**: Initialize, start, stop, and kill environments
- **State Management**: Commit, branch, checkout, and pick environment states
- **Interaction**: SSH into environments and run commands
- **Configuration**: Simple configuration via vers.toml

## Installation

```bash
go install github.com/hdresearch/vers-cli@latest
```

## Usage

### Basic Commands

```bash
# Initialize a new project
vers init

# Start a development environment
vers up

# Connect to an environment
vers ssh default

# Run a command within the environment
vers run ls -la

# Stop the environment
vers stop
```

### State Management

```bash
# Commit the current state
vers commit -m "Initial working state"

# Create a new branch
vers branch experimental

# Switch to a branch
vers checkout experimental

# Keep a specific branch
vers pick main
```

## Configuration

Vers CLI uses a `vers.toml` configuration file to define your environment. 
The file is created when you run `vers init` and can be customized for your specific needs.

Example:

```toml
[meta]
project = "myapp"
type = "python"

[build]
builder = "docker"
build_command = "pip install -r requirements.txt"

[run]
command = "python main.py"

[env]
DATABASE_URL = "postgres://localhost:5432/mydb"
```