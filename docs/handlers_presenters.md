# Handlers & Presenters

Handlers
- Validate, resolve IDs/aliases, and orchestrate calls to services/SDK.
- Map errors to user-facing messages; return a small view struct for presenters.
- Examples: pause.go, resume.go, branch.go, rename.go.

Presenters
- Print concise, styled output for CLI commands (success, warnings, errors).
- Reuse `styles` to keep a consistent look.

Example (rename)
1) `cmd/rename.go` parses flags/args and calls `HandleRename`.
2) Handler resolves cluster/VM ID, sends `Update` to SDK.
3) Presenter prints ✓ and new alias.

