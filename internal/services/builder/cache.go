package builder

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
)

// cacheFile is the on-disk location for the layer cache. Sibling to .vers/.
const cacheFile = ".vers/buildcache.json"

// LayerCache maps a deterministic cache key to the commit id that resulted
// from executing that step. It is a best-effort cache: entries referring to
// commits that have been deleted server-side are transparently bypassed.
type LayerCache struct {
	Entries map[string]string `json:"entries"`
	path    string
}

// LoadCache reads the cache from .vers/buildcache.json (or returns an empty
// cache if missing). It is safe to call when the working directory is not a
// vers project — the returned cache simply never persists.
func LoadCache() *LayerCache {
	c := &LayerCache{Entries: map[string]string{}, path: cacheFile}
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return c
	}
	_ = json.Unmarshal(data, c)
	if c.Entries == nil {
		c.Entries = map[string]string{}
	}
	return c
}

// Save persists the cache to disk. Failures are silently ignored — the cache
// is an optimization, not a correctness boundary.
func (c *LayerCache) Save() {
	if c == nil || c.path == "" {
		return
	}
	// Only persist if .vers/ exists — otherwise we're outside a vers project.
	if _, err := os.Stat(".vers"); err != nil {
		return
	}
	if err := os.MkdirAll(filepath.Dir(c.path), 0755); err != nil {
		return
	}
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(c.path, data, 0644)
}

// Get returns the cached commit id for a key, or "" if not present.
func (c *LayerCache) Get(key string) string {
	if c == nil {
		return ""
	}
	return c.Entries[key]
}

// Put stores a cache entry.
func (c *LayerCache) Put(key, commitID string) {
	if c == nil {
		return
	}
	c.Entries[key] = commitID
}

// CacheKey computes a stable cache key from the parent commit id, the
// normalized instruction text, and any side-inputs (e.g. a COPY tree hash).
// Extras are sorted before hashing to avoid ordering issues.
func CacheKey(parentCommit, instruction string, extras ...string) string {
	sort.Strings(extras)
	h := sha256.New()
	fmt.Fprintf(h, "v1\n")
	fmt.Fprintf(h, "parent:%s\n", parentCommit)
	fmt.Fprintf(h, "instr:%s\n", instruction)
	for _, e := range extras {
		fmt.Fprintf(h, "extra:%s\n", e)
	}
	return hex.EncodeToString(h.Sum(nil))
}
