# Development

Requirements
- Go 1.23+
- make (optional)

Build & Run
- Build: `make build` or `go build -o bin/vers ./cmd/vers`
- Run: `./bin/vers status` or `./bin/vers ui`
- Install: `make build-and-install` or `go install ./cmd/vers`

Hot reload (optional)
- `air` rebuilds on changes; run subcommands like `air status` during dev.

Coding style
- Run `gofmt` and `go vet ./...` before pushing.
- Keep packages short and lower-case; tests in `*_test.go`.

Configuration
- `vers.toml` holds user config (e.g., defaults, feature flags).
- Credentials in `.env` (gitignored): `VERS_URL`, `VERS_API_KEY`, `GO_INSTALL_PATH`, `GO_PATH`.

