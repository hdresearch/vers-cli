# Release & Versioning

Build metadata
- Injected via `-ldflags` in `make build` (version, commit, build date, etc.).
- Binary outputs to `./bin/vers`.

Local install
- `make build-and-install` or `go install ./cmd/vers` to add `vers` to PATH.

Publishing
- Tag versions in Git and produce platform builds via CI (future).

