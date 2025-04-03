# Vers CLI

A command-line interface for managing virtual machine/container-based development environments.


## Development

The scripts you should use as models are `status.go`, `execute.go`, `up.go`, `branch.go`. 

You can largely have AI generate new command scripts with those previous scripts as a model. You'll have to manually adjust the SDK calls, though, since the AI won't have access to the details of the SDK. 

If a request specifies a parameter you'll see this type `Command param.Field[string] \`json:"command,required"`\`, make sure that you prepare the parameter as follows: `vers.F(commandStr)`. See the "Request Fields" section of the [Go SDK Readme](https://github.com/hdresearch/vers-sdk-go) for more details. You can also look at the example of `execute.go`. 


## Features

- **Environment Management**: Start environments with `up` command
- **State Inspection**: Check environment status
- **Command Execution**: Run commands within environments
- **Branching**: Create branches from existing environments

## Installation

```bash
go install github.com/hdresearch/vers-cli@latest
```

## Usage

### Available Commands

```bash
# Check the status of all clusters
vers status

# Check the status of a specific cluster
vers status -c <cluster-id>

# Start a development environment (creates a new cluster)
vers up [cluster-name]

# Execute a command on a VM
vers execute <vm-id> <command> [args...]

# Create a new branch from a VM
vers branch <vm-id>
```

## Configuration

Vers CLI uses a `vers.toml` configuration file to define your environment. 
The file should be created manually and can be customized for your specific needs.

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