# F1: --format json → --json (completed)

Implemented finding F1 from `AGENT_NATIVE_AUDIT.md`. vers-cli now standardizes on `--json` across the entire command surface, with `--format` retained as a hidden, deprecated alias.

## Changed files

```
 README.md                          | 12 +++---
 cmd/branch.go                      | 12 ++++--
 cmd/commit.go                      | 36 ++++++++++++----
 cmd/deploy.go                      | 12 ++++--
 cmd/env.go                         | 10 +++--
 cmd/info.go                        | 12 ++++--
 cmd/pause.go                       | 12 ++++--
 cmd/repo.go                        | 48 ++++++++++++++++-----
 cmd/resize.go                      | 12 ++++--
 cmd/resume.go                      | 12 ++++--
 cmd/run.go                         | 12 ++++--
 cmd/run_commit.go                  | 12 ++++--
 cmd/status.go                      | 12 ++++--
 cmd/tag.go                         | 24 +++++++----
 internal/presenters/format.go      | 24 +++++++++--
 internal/presenters/format_test.go | 37 +++++++++++++----
 16 files changed, 224 insertions(+), 75 deletions(-)
```

19 cobra `--format` registrations now have a sibling `--json` BoolVar; 19 `pres.ParseFormat` call sites now pass the JSON flag and propagate the validation error.

Commit: `6744100 feat: standardize on --json flag (deprecate --format json)` on branch `pi-parallel-d95f3d3f-0`.

## Design choices

1. **Centralized validation in `presenters.ParseFormat`.** New signature: `ParseFormat(quiet, jsonFlag bool, formatStr string) (OutputFormat, error)`. Precedence is `quiet > json > format > default`. Invalid `--format` values return an enumerated error rather than silently falling through to default-format (which was the previous behavior — a P3 violation).
2. **Cobra's `MarkDeprecated` handles the warning.** It prints `Flag --format has been deprecated, use --json instead` to stderr automatically when the flag is used, and hides the flag from `--help`. No custom warning code needed.
3. **Stdout/stderr separation preserved.** Deprecation warnings go to stderr; JSON data continues to land cleanly on stdout. `vers status --format json | jq` still works without contamination.

## Validation

- `go build ./...` — clean
- `go vet ./...` — clean
- `go test ./...` — all packages pass (including expanded `format_test.go` with 9 cases covering the new precedence and error semantics)

Smoke tests against built binary:

```
$ vers status --json
[]                                            # exit 0, clean stdout

$ vers status --format json 2>/tmp/err
[]                                            # exit 0, JSON on stdout
$ cat /tmp/err
Flag --format has been deprecated, use --json instead

$ vers status --format yaml
Flag --format has been deprecated, use --json instead
Error: --format must be "json" (got: "yaml"). Note: --format is deprecated, use --json instead
                                              # exit 1
```

`vers status --help` no longer lists `--format`; long descriptions everywhere now read "Use --json for machine-readable output."

## Risks / open questions

- **Usage banner printed on RunE error.** When `--format yaml` is rejected, cobra prints the command's usage block after the error. This is pre-existing behavior — `SilenceUsage = true` is only set in the auth path in `cmd/root.go:210`. Out of scope for F1, but worth a follow-up if cleaner agent-facing errors are wanted (set `SilenceUsage = true` on the root command).
- **`alias` command is still missing JSON output entirely.** This is finding F2, separate task.
- **No tests assert deprecation warning emission to stderr.** The cobra `MarkDeprecated` behavior is library-provided and stable, so I didn't add an integration test capturing stderr; the unit tests exercise only the validation path. If desired, a small `cmd/`-level test could capture stderr from a `--format json` invocation.

## Recommended next step

Hand off to F2 (info → get) and F3 (pagination) as planned. After both land, audit the few commands that have a `--quiet` flag but no `--json` (`alias`, possibly `head`, `checkout`) and unify those — the `ParseFormat` machinery is already in place to extend them cheaply.

---

Implemented F1 (canonical `--json` flag).
Changed files: 16 (13 cmd files, presenters package, README).
Validation: `go build`, `go vet`, `go test ./...` all clean; manual smoke tests confirm `--json` works, legacy `--format json` works with stderr deprecation, invalid `--format` values error with the enumerated message.
Open risks/questions: usage banner on error is pre-existing; no integration test for deprecation warning.
Recommended next step: proceed with F2 and F3 as planned.
