package cmd

import "testing"

// TestLooksLikeRepoRef verifies the heuristic used to suggest --ref when
// the user passes a repo:tag-shaped argument. False positives are cheap
// (we just show an extra suggestion); false negatives mean the user hits
// a cryptic 422 from the API, so the positive cases get careful coverage.
func TestLooksLikeRepoRef(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  bool
	}{
		// Positive: canonical repo:tag shapes
		{"simple", "my-app:latest", true},
		{"minimal", "a:b", true},
		{"semver-tag", "my_app:v1.0.0", true},
		{"dotted-parts", "x.y:1.2", true},
		{"numeric-parts", "123:456", true},
		{"mixed-legal-chars", "pi-agent:2026-04-19-v4", true},

		// Negative: no colon
		{"empty", "", false},
		{"no-colon", "abc-123", false},
		{"uuid", "aa84beb8-a080-4be5-bb7f-86374a68b380", false},

		// Negative: malformed colon placement
		{"colon-at-start", ":latest", false},
		{"colon-at-end", "my-app:", false},
		{"lone-colon", ":", false},

		// Negative: contains characters not valid in repo/tag names
		{"space", "my app:latest", false},
		{"slash", "owner/my-app:latest", false},
		{"at-sign", "my-app@v1:latest", false},
		{"hash", "my-app#1:latest", false},
		{"non-ascii", "my-app:päivä", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := looksLikeRepoRef(tt.input)
			if got != tt.want {
				t.Errorf("looksLikeRepoRef(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// TestLooksLikeRepoRef_MultipleColons documents the current behavior for
// strings with more than one colon. The first colon splits repo from tag,
// and subsequent characters (including another colon) must be legal name
// chars. A bare `a:b:c` has a colon in the tag part, which fails the
// legal-char check and returns false.
func TestLooksLikeRepoRef_MultipleColons(t *testing.T) {
	if looksLikeRepoRef("a:b:c") {
		t.Error("expected a:b:c to NOT look like a repo:tag (colon in tag part is not a legal name char)")
	}
}
