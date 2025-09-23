# Testing

Integration tests
- Live under `test/`; they install and invoke the local CLI via `os/exec`.
- Require API credentials from `.env` at repo root: `VERS_URL`, `VERS_API_KEY`.

Run
- `cd test && go test -v` (optionally `-cover`).

Tips
- Keep tests deterministic; avoid shared mutable state between tests.
- Use separate aliases/names per test run to avoid collisions.

