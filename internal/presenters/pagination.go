package presenters

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
)

// PageInfo describes pagination metadata for list-style command outputs.
//
// When emitted as JSON, fields with zero values are kept (Total/Limit/Offset)
// so consumers can rely on a stable shape; NextOffset is omitted when there is
// no next page.
//
// TODO: Once the underlying API exposes server-side pagination via typed
// query parameters in the Go SDK, plumb Limit/Offset/Cursor through to the
// network call instead of trimming the full response client-side. Today the
// SDK list endpoints do not accept pagination params, so we client-side
// paginate after the response.
type PageInfo struct {
	Total      int    `json:"total"`
	Limit      int    `json:"limit"`
	Offset     int    `json:"offset"`
	Truncated  bool   `json:"truncated"`
	NextOffset *int   `json:"next_offset,omitempty"`
	Hint       string `json:"hint,omitempty"`
}

// ApplyPaging clamps the requested offset/limit against a list of length total
// and returns the slice indices [start, end) along with PageInfo describing
// the result.
//
// limit == 0 (or negative) means "unbounded": return everything from offset
// onwards.
func ApplyPaging(total, limit, offset int) (start, end int, info PageInfo) {
	if offset < 0 {
		offset = 0
	}
	if offset > total {
		offset = total
	}
	start = offset
	if limit <= 0 {
		end = total
	} else {
		end = offset + limit
		if end > total {
			end = total
		}
	}

	info = PageInfo{
		Total:     total,
		Limit:     limit,
		Offset:    offset,
		Truncated: end < total,
	}
	if info.Truncated {
		next := end
		info.NextOffset = &next
		shown := end - start
		info.Hint = fmt.Sprintf(
			"showing %d of %d — use --limit=N (0 for all) or --offset=%d for the next page",
			shown, total, end,
		)
	}
	return start, end, info
}

// PaginatedJSON is the wire shape used when a list response is truncated.
// When not truncated, callers should emit the bare items array (preserving
// pre-pagination output shape for backwards compatibility).
type PaginatedJSON struct {
	Items      interface{} `json:"items"`
	Total      int         `json:"total"`
	Limit      int         `json:"limit"`
	Offset     int         `json:"offset"`
	Truncated  bool        `json:"truncated"`
	NextOffset *int        `json:"next_offset,omitempty"`
	Hint       string      `json:"hint,omitempty"`
}

// PrintListJSON emits items as JSON. When info.Truncated is true, items are
// wrapped in a PaginatedJSON envelope with hint and next_offset. Otherwise
// the bare items value is emitted (matching the pre-pagination shape).
func PrintListJSON(items interface{}, info PageInfo) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	if info.Truncated {
		return enc.Encode(PaginatedJSON{
			Items:      items,
			Total:      info.Total,
			Limit:      info.Limit,
			Offset:     info.Offset,
			Truncated:  true,
			NextOffset: info.NextOffset,
			Hint:       info.Hint,
		})
	}
	return enc.Encode(items)
}

// PrintTruncationHint writes a one-line truncation hint to stderr (so it does
// not pollute stdout data streams). It is a no-op if the page was not
// truncated.
func PrintTruncationHint(w io.Writer, info PageInfo) {
	if !info.Truncated {
		return
	}
	if w == nil {
		w = os.Stderr
	}
	fmt.Fprintf(w, "(%s)\n", info.Hint)
}
