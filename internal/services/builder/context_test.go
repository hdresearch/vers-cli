package builder

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestDockerIgnore(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, ".dockerignore"), "node_modules\n*.log\n!important.log\n")
	bc, err := LoadContext(dir)
	if err != nil {
		t.Fatal(err)
	}
	cases := map[string]bool{
		"src/main.go":             false,
		"node_modules":            true,
		"node_modules/foo/bar.js": true,
		"a/b/node_modules/x":      true,
		"debug.log":               true,
		"important.log":           false,
		"src/ok.log":              true,
	}
	for path, want := range cases {
		if got := bc.IsIgnored(path); got != want {
			t.Errorf("IsIgnored(%q): got %v want %v", path, got, want)
		}
	}
}

func TestResolveSourceAndHash(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "a.txt"), "hello")
	writeFile(t, filepath.Join(dir, "sub/b.txt"), "world")
	writeFile(t, filepath.Join(dir, "sub/c.log"), "ignored")
	writeFile(t, filepath.Join(dir, ".dockerignore"), "*.log\n")

	bc, err := LoadContext(dir)
	if err != nil {
		t.Fatal(err)
	}
	entries, err := bc.ResolveSource("sub")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].RelPath != "b.txt" {
		t.Errorf("got entries: %+v", entries)
	}

	h1, err := bc.HashSources([]string{"a.txt", "sub"})
	if err != nil {
		t.Fatal(err)
	}
	// Mutate a.txt → hash must change
	writeFile(t, filepath.Join(dir, "a.txt"), "goodbye")
	h2, err := bc.HashSources([]string{"a.txt", "sub"})
	if err != nil {
		t.Fatal(err)
	}
	if h1 == h2 {
		t.Errorf("expected hash to change after file mutation")
	}
}

func TestResolveSourceRejectsEscape(t *testing.T) {
	dir := t.TempDir()
	bc, err := LoadContext(dir)
	if err != nil {
		t.Fatal(err)
	}
	cases := []string{"../etc/passwd", "/absolute", "a/../../b"}
	for _, c := range cases {
		if _, err := bc.ResolveSource(c); err == nil {
			t.Errorf("expected error for %q", c)
		}
	}
}

func TestCacheKeyStability(t *testing.T) {
	a := CacheKey("parent", "RUN echo hi", "env=FOO=bar", "wd=/app")
	b := CacheKey("parent", "RUN echo hi", "wd=/app", "env=FOO=bar") // different order
	if a != b {
		t.Errorf("cache key not order-stable: %s vs %s", a, b)
	}
	c := CacheKey("parent2", "RUN echo hi", "env=FOO=bar", "wd=/app")
	if a == c {
		t.Errorf("cache key collides across parents")
	}
}
