# Services

Purpose
- Small domain-specific helpers that call the Vers SDK and return DTOs tailored to the handlers/TUI.

Key services
- status: list clusters, get cluster by ID/alias.
- vm: connection info, metadata helpers.
- history: commit log for a VM.
- tree: resolve cluster for a head VM, render tree structure.
- deletion: delete VM/cluster; maps API errors to typed errors.

Patterns
- Context timeouts: use `APIMedium` / `APILong` from `app.Timeouts`.
- Keep services thin — most behavior lives in handlers.

