package presenters

import (
	"testing"
)

func TestApplyPaging(t *testing.T) {
	cases := []struct {
		name               string
		total, limit, off  int
		wantStart, wantEnd int
		wantTrunc          bool
		wantNextOffset     int // -1 = nil
	}{
		{"empty", 0, 50, 0, 0, 0, false, -1},
		{"under limit", 10, 50, 0, 0, 10, false, -1},
		{"exactly limit", 50, 50, 0, 0, 50, false, -1},
		{"truncated first page", 142, 50, 0, 0, 50, true, 50},
		{"truncated middle page", 142, 50, 50, 50, 100, true, 100},
		{"truncated last page", 142, 50, 100, 100, 142, false, -1},
		{"unbounded", 142, 0, 0, 0, 142, false, -1},
		{"unbounded with offset", 142, 0, 50, 50, 142, false, -1},
		{"offset past end", 10, 50, 100, 10, 10, false, -1},
		{"negative offset clamped", 10, 5, -3, 0, 5, true, 5},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			start, end, info := ApplyPaging(c.total, c.limit, c.off)
			if start != c.wantStart || end != c.wantEnd {
				t.Fatalf("indices = (%d,%d), want (%d,%d)", start, end, c.wantStart, c.wantEnd)
			}
			if info.Truncated != c.wantTrunc {
				t.Fatalf("truncated = %v, want %v", info.Truncated, c.wantTrunc)
			}
			if c.wantNextOffset == -1 {
				if info.NextOffset != nil {
					t.Fatalf("NextOffset = %d, want nil", *info.NextOffset)
				}
			} else {
				if info.NextOffset == nil {
					t.Fatalf("NextOffset = nil, want %d", c.wantNextOffset)
				}
				if *info.NextOffset != c.wantNextOffset {
					t.Fatalf("NextOffset = %d, want %d", *info.NextOffset, c.wantNextOffset)
				}
			}
			if info.Truncated && info.Hint == "" {
				t.Fatalf("expected non-empty hint when truncated")
			}
		})
	}
}
