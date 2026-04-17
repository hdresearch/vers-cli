# Vers CLI

A command-line interface for managing virtual machine/container-based development environments.


## Installation

### Quick Install (Recommended)

```bash
curl -fsSL https://raw.githubusercontent.com/hdresearch/vers-cli/main/install.sh | sh
```

This script will:
- Detect your OS and architecture automatically
- Download the appropriate prebuilt binary
- Verify the checksum for security
- Install to `~/.local/bin` (or use `INSTALL_DIR` to customize)
- Make the binary executable

**Custom installation directory:**
```bash
INSTALL_DIR=/usr/local/bin curl -fsSL https://raw.githubusercontent.com/hdresearch/vers-cli/main/install.sh | sh
```

**Install a specific version:**
```bash
VERS_VERSION=v0.5.0 curl -fsSL https://raw.githubusercontent.com/hdresearch/vers-cli/main/install.sh | sh
```

### Install from Source

```bash
go install github.com/hdresearch/vers-cli/cmd/vers@latest
```

### Manual Installation

Download prebuilt binaries from the [releases page](https://github.com/hdresearch/vers-cli/releases).


## Usage

### VMs

```bash
# Start a new VM
vers run

# List all VMs
vers status

# List just VM IDs (for scripting)
vers status -q

# Full JSON output
vers status --format json

# Detailed metadata for a VM (IP, lineage, timestamps)
vers info <vm-id>
vers info --format json

# Execute a command on a VM
vers execute <vm-id> <command> [args...]

# Create a branch from a VM
vers branch <vm-id> [--count N] [--checkout]

# Pause / resume a VM
vers pause <vm-id>
vers resume <vm-id>

# Resize a VM's disk
vers resize <vm-id> --size <mib>

# Delete VMs
vers kill <vm-id>
vers kill <vm-1> <vm-2> <vm-3>
vers kill -r <vm-id>              # recursive (include children)
```

### Build from a Dockerfile

`vers build` turns a literal Dockerfile into a sequence of actions on a
throwaway VM, committing a "layer" after each step and caching them in
`.vers/buildcache.json`.

```bash
# FROM scratch — sizing is explicit
vers build --mem-size 2048 --vcpu-count 2 --fs-size-vm-mib 4096 .

# FROM <tag-or-commit-id> — no sizing flags needed
vers build -t myapp:prod .
vers build -f build.Dockerfile --build-arg VERSION=1.2.3 .

# Scripting: print just the final commit id
COMMIT=$(vers build -q .)
vers run-commit "$COMMIT"
```

Supported instructions: `FROM`, `RUN`, `COPY`, `ADD` (local only), `ENV`,
`ARG`, `WORKDIR`, `USER`, `LABEL`, `CMD`, `ENTRYPOINT`, `EXPOSE`.
Multi-stage builds and `COPY --from=` are not yet supported.

`FROM` resolves as follows:
- `FROM scratch` — fresh VM; requires `--mem-size`, `--vcpu-count`, `--fs-size-vm-mib`
- `FROM <name>` — looked up as a vers tag first, falling back to a commit id

### Commits

```bash
# Commit the current HEAD VM
vers commit

# Commit a specific VM
vers commit <vm-id>

# List your commits
vers commit list
vers commit list -q               # just IDs
vers commit list --format json
vers commit list --public         # public commits

# View commit history (parent chain)
vers commit history <commit-id>

# Make a commit public/private
vers commit publish <commit-id>
vers commit unpublish <commit-id>

# Delete commits
vers commit delete <commit-id>
vers commit delete <id-1> <id-2>
```

### Tags

Named pointers to commits — like git tags.

```bash
# Create a tag
vers tag create <name> <commit-id>
vers tag create production abc-123 -d "stable release"

# List all tags
vers tag list
vers tag list -q                  # just names
vers tag list --format json

# Get tag details
vers tag get <name>

# Move a tag to a different commit
vers tag update <name> --commit <new-id>
vers tag update <name> --description "updated desc"

# Delete tags
vers tag delete <name>
vers tag delete <name-1> <name-2>
```

### Shell Composition

Commands with `-q` output are designed to compose with standard Unix tools:

```bash
# Kill all VMs
vers kill $(vers status -q)

# Delete all commits
vers commit delete $(vers commit list -q)

# Delete all tags
vers tag delete $(vers tag list -q)

# Get info on the first VM
vers info $(vers status -q | head -1)

# JSON piped to jq
vers status --format json | jq '.[].vm_id'
vers info <vm-id> --format json | jq '.ip'
```

`ps` is an alias for `status`:
```bash
vers ps -q
```


## Configuration

Vers CLI uses a `vers.toml` configuration file to define your environment.

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


## Development

### Architecture

Commands under `cmd/` are intentionally thin: they parse flags/args and delegate to handlers in `internal/handlers/`, which coordinate services in `internal/services/` and render results via `internal/presenters/`. A shared `App` container (in `internal/app/`) wires common deps (SDK client, IO, prompter, exec runner, timeouts) in `cmd/root.go`.

When adding a new command:
- Add a handler `internal/handlers/<command>.go` with `Handle(ctx, app, Req) (View, error)`.
- Add a presenter `internal/presenters/<command>_presenter.go` to render `View`.
- Keep the Cobra file minimal: parse → build `Req` → call handler → render.

### SDK Requests

If a request field needs a `param.Field[T]`, wrap with `vers.F(value)`. See the [Go SDK Readme](https://github.com/hdresearch/vers-sdk-go) and existing handlers for examples.

### Building

```bash
go build -o bin/vers ./cmd/vers
```

This repository uses [Air](https://github.com/air-verse/air) for hot reloading during development:
```bash
air status
```

### Testing

Unit tests:
```bash
make test        # or: make test-unit
```

Integration tests (require `VERS_URL` and `VERS_API_KEY`):
```bash
VERS_URL=https://... VERS_API_KEY=... make test-integration

# Run a specific test
VERS_URL=... VERS_API_KEY=... make test-integration ARGS='-run TagLifecycle -v'
```

### MCP Server (experimental)

Built-in MCP server to expose Vers operations as tools for agent clients (Claude Desktop/Code, etc.).

```bash
# stdio transport (local agents)
VERS_URL=https://<url> VERS_API_KEY=<token> vers mcp serve --transport stdio

# HTTP/SSE transport
VERS_MCP_HTTP_TOKEN=<secret> VERS_URL=... VERS_API_KEY=... vers mcp serve --transport http --addr :3920
```

Tools: `vers.status`, `vers.run`, `vers.execute`, `vers.branch`, `vers.kill`, `vers.version`, `vers.capabilities`

Resources: `vers://status`
