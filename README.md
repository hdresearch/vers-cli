# Vers CLI

A command-line interface for managing virtual machine/container-based development environments.


## Development

### Thin Command Architecture

Commands under `cmd/` are intentionally thin: they parse flags/args and delegate to handlers in `internal/handlers/`, which coordinate services in `internal/services/` and render results via `internal/presenters/`. A shared `App` container (in `internal/app/`) wires common deps (SDK client, IO, prompter, exec runner, timeouts) in `cmd/root.go`.

When adding a new command:
- Add a handler `internal/handlers/<command>.go` with `Handle(ctx, app, Req) (View, error)`.
- Add a presenter `internal/presenters/<command>_presenter.go` to render `View`.
- Keep the Cobra file minimal: parse → build `Req` → call handler → render.

### SDK Requests

If a request field needs a `param.Field[T]`, wrap with `vers.F(value)`. See the [Go SDK Readme](https://github.com/hdresearch/vers-sdk-go) and existing handlers (e.g., run/run-commit/rename) for examples.


## Features

- **Environment Management**: Start environments with `run` command
- **State Inspection**: Check environment status
- **Command Execution**: Run commands within environments
- **Branching**: Create branches from existing environments

## Installation

```bash
go install github.com/hdresearch/vers-cli/cmd/vers@latest
```


## Usage

### Available Commands

```bash
# Check the status of all clusters
vers status

# Check the status of a specific cluster
vers status -c <cluster-id>

# Start a development environment (creates a new cluster)
vers run [cluster-name]

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

### UI (experimental)

The `vers ui` command launches an interactive TUI that is currently EXPERIMENTAL. It may change or break; not recommended for production use. Expect rough edges and please report issues.


## Development

To build the binary locally, run:
```bash
go build -o bin/vers ./cmd/vers
```

This repository uses [Air](https://github.com/air-verse/air?tab=readme-ov-file) for development with hot reloading. You can run 
```
air
```
which will take the place of running the binary. So to develop on e.g. `vers status` you would run

```
air status
```

### Testing

- Unit tests (includes MCP tests):
  ```bash
  make test        # or: make test-unit
  ```
- Integration tests (require env: VERS_URL, VERS_API_KEY):
  ```bash
  VERS_URL=https://... VERS_API_KEY=... make test-integration
  # Optional args passthrough
  VERS_URL=... VERS_API_KEY=... make test-integration ARGS='-run Copy -v'
  ```
### MCP Server (built-in, experimental)

This repo includes an MCP server to expose Vers operations as tools and resources for agent clients (Claude Desktop/Code, etc.).

- Build:
  - `make build` (MCP server is included in the binary)
- Run (stdio transport for local agents):
  - `VERS_URL=https://<vers-url> VERS_API_KEY=<token> ./bin/vers mcp serve --transport stdio`
- Run over HTTP/SSE (for MCP connector):
  - `export VERS_MCP_HTTP_TOKEN=<secret>` (optional but recommended)
  - `VERS_URL=... VERS_API_KEY=... ./bin/vers mcp serve --transport http --addr :3920`
  - `curl http://localhost:3920/healthz` → `ok`

Tools exposed
- `vers.status` — snapshot of clusters/VMs (inputs: cluster?, target?)
- `vers.run` — start a cluster (inputs: memSizeMib?, vcpuCount?, rootfsName?, kernelName?, fsSizeClusterMib?, fsSizeVmMib?, clusterAlias?, vmAlias?)
- `vers.execute` — run a command in a VM (inputs: target?, command [required], timeoutSeconds?)
- `vers.branch` — create a VM from existing/HEAD (inputs: target?, alias?, checkout?)
- `vers.kill` — delete VMs/clusters (inputs: targets?, skipConfirmation [required], recursive?, isCluster?, killAll?)
- `vers.version` — server info (no backend calls)
- `vers.capabilities` — server settings/tool list

Resources
- `vers://status` — global status as JSON
- `vers://status/{cluster}` — cluster-specific status
- `vers://cluster/{id}/tree` — VM tree and HEAD for a cluster

Notes
- Execute streams stdout/stderr via MCP logging messages; final summary + structured output returned.
- Destructive tools require `skipConfirmation=true` in MCP mode.
- Basic rate limits per minute are enforced per tool; hitting limits returns a coded MCP error.
