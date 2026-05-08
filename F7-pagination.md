# F7: Pagination + Truncation Hints

Branch: `pi-parallel-d95f3d3f-2`
Commit: `bc31992 feat: paginate list commands with --limit and truncation hints`

## Implemented

Added `--limit N` (default 50; `0` = unbounded) and `--offset N` to every list-style command, plus a uniform truncation hint surface:

| Command | File | Notes |
|---|---|---|
| `vers status` | `cmd/status.go` | List mode only. Single-VM mode unchanged. |
| `vers commit list` | `cmd/commit.go` | |
| `vers repo list` | `cmd/repo.go` | |
| `vers repo tag list` | `cmd/repo.go` | Included for surface consistency (was implicit in scope). |
| `vers tag list` | `cmd/tag.go` | |
| `vers env list` | `cmd/env.go` | Sorted by key for stable pagination. JSON wraps as `[{key,value}]` envelope when truncated to preserve order; bare object map when not truncated. |
| `vers alias` (no arg) | `cmd/alias.go` | Sorted by name. |

## Output shapes

**JSON, not truncated** — bare items array (backwards-compat, identical to pre-change shape):
```json
[ {...}, {...} ]
```

**JSON, truncated** — envelope with hint and next_offset:
```json
{
  "items": [ {...} ],
  "total": 11,
  "limit": 3,
  "offset": 0,
  "truncated": true,
  "next_offset": 3,
  "hint": "showing 3 of 11 — use --limit=N (0 for all) or --offset=3 for the next page"
}
```

**Text / quiet modes** — table or IDs on stdout, single-line hint on stderr:
```
$ vers status -q --limit 1
6781f925-09ba-48ab-8f2c-9adeb75eec6d
[stderr] (showing 1 of 2 — use --limit=N (0 for all) or --offset=1 for the next page)
```

## Implementation notes

- New file `internal/presenters/pagination.go` exposes:
  - `PageInfo` struct (Total/Limit/Offset/Truncated/NextOffset/Hint)
  - `ApplyPaging(total, limit, offset) (start, end, info)` — clamps and computes
  - `PrintListJSON(items, info)` — bare array vs. envelope based on `info.Truncated`
  - `PrintTruncationHint(w, info)` — no-op when not truncated; otherwise `(hint)\n`
- Unit tested in `internal/presenters/pagination_test.go` (10 sub-cases covering empty/full/truncated/unbounded/offset-past-end/negative offsets).

## Server-side pagination caveat

The Go SDK (`vers-sdk-go@v0.1.0-alpha.32`) **does not expose** `Limit`/`Offset`/`Cursor` query parameters on its list endpoints today. The `ListCommitsResponse` shape *includes* `Limit`/`Offset`/`Total` fields, suggesting the API server may accept them, but the SDK has no typed param surface for them.

Per the task instructions, I implemented client-side pagination after the full response. Marked clearly:

- One canonical TODO at the top of `internal/presenters/pagination.go` describing the constraint.
- Inline TODO comments at every call site (`cmd/status.go`, `cmd/commit.go`, `cmd/repo.go`, `cmd/tag.go`, `cmd/env.go`) noting where to plumb `--limit`/`--offset` to the SDK once supported.

When the SDK exposes the params, the change is local: replace the `ApplyPaging` block with a request that passes `limit`/`offset` through (e.g., via `option.WithQuery`), receives an already-paged response, and uses the API's `total` field instead of `len(items)` to detect truncation.

## Validation

- `go build ./...` — passes
- `go vet ./...` — clean
- `go test ./...` — all packages pass, including new `TestApplyPaging` (10 sub-cases)
- Live API checks against `/Users/tynandaly/basin/vers-cli`'s authenticated environment:
  - `vers status --limit 1 --format json` → returns 1 of 2 VMs with truncation envelope. ✓
  - `vers status --limit 1` text mode → table to stdout, hint to stderr. ✓
  - `vers status -q --limit 1` quiet mode → ID to stdout, hint to stderr. ✓
  - `vers status --limit 0` unbounded → bare array, no envelope. ✓
  - `vers commit list --limit 3 --format json` → envelope with `total: 11`, `next_offset: 3`. ✓
  - `vers commit list --limit 3 --offset 3 --format json` → next page works. ✓
  - `vers commit list --help` → new flags documented. ✓

## Out of scope (left untouched)

- `--format json` → `--json` rename (separate worktree handles F1).
- `info` → `get` rename (separate worktree handles F8).
- Existing `--format string` flags retained as-is on every list command.
- Tests in `internal/handlers/` and `internal/mcp/` were not modified — handler signatures unchanged because pagination lives in the cmd layer.

## Open questions / risks

1. **JSON shape change for `env list` when truncated.** Historical shape was `{"KEY": "VALUE", ...}` (an object). The truncated envelope uses `items: [{key, value}, ...]` because Go map iteration order is unstable, and a sorted ordered list is required for stable `next_offset` paging. When *not* truncated, the historical map shape is preserved. Agents that hit the truncated branch will see a different shape — flagged here in case downstream MCP tooling depends on the map form.

2. **`commit list` text-mode header.** `RenderCommitsList` prints `"%d commit(s)"` from `view.Total`, which is the API-reported total, not the paged count. Left as-is because the API total is more useful information; the paged count is implicit in the row count plus the stderr hint. If desired, this could be tweaked to `"showing N of TOTAL commits"`.

3. **`repo list` / `tag list` / `repo tag list` have no API-reported total.** The truncation total is `len(response)` — i.e., what we got back from the API. If the server is silently capping the response, the agent would see a wrong total. Low risk today (these endpoints appear to return everything), but worth verifying once server-side pagination lands.

## Recommended next steps

1. **Coordinate the merge order with the F1 (`--format` → `--json`) and F8 (`info` → `get`) worktrees.** This branch only touches `--limit`/`--offset` and the `info` lines in renderers/help; conflicts should be limited to flag-init blocks and Long-help text, all easy three-way merges.
2. **Open an issue in `vers-sdk-go`** requesting typed `Limit`/`Offset` parameters on every list endpoint so the client-side trim can be removed and `total`/`next_offset` come from the API.
3. **Add an MCP-tool wrapper note** so the existing MCP server surfaces `--limit`/`--offset` in tool descriptions for agent consumers (out of scope for F7 itself; deserves a dedicated follow-up).
