package builder

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// BuildContext represents the on-disk source tree that COPY reads from.
type BuildContext struct {
	Root    string   // absolute path to the context root
	Ignores []string // parsed .dockerignore patterns
}

// LoadContext resolves root to an absolute path and reads .dockerignore if
// present.
func LoadContext(root string) (*BuildContext, error) {
	abs, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}
	st, err := os.Stat(abs)
	if err != nil {
		return nil, fmt.Errorf("build context %q: %w", root, err)
	}
	if !st.IsDir() {
		return nil, fmt.Errorf("build context %q is not a directory", root)
	}
	ign, err := loadDockerIgnore(filepath.Join(abs, ".dockerignore"))
	if err != nil {
		return nil, err
	}
	return &BuildContext{Root: abs, Ignores: ign}, nil
}

func loadDockerIgnore(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer f.Close()
	var out []string
	scan := bufio.NewScanner(f)
	for scan.Scan() {
		line := strings.TrimSpace(scan.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		out = append(out, line)
	}
	return out, scan.Err()
}

// IsIgnored reports whether a path (relative to the context root, using
// forward slashes) is ignored. It implements a subset of .dockerignore
// semantics: literal paths, `*` glob segments, and leading `!` negation.
// Directory prefix matching is honored so `node_modules` ignores the whole
// tree.
func (c *BuildContext) IsIgnored(rel string) bool {
	rel = filepath.ToSlash(rel)
	ignored := false
	for _, pat := range c.Ignores {
		neg := false
		if strings.HasPrefix(pat, "!") {
			neg = true
			pat = pat[1:]
		}
		if matchIgnore(pat, rel) {
			ignored = !neg
		}
	}
	return ignored
}

func matchIgnore(pattern, path string) bool {
	pattern = strings.TrimPrefix(pattern, "./")
	pattern = strings.TrimPrefix(pattern, "/")
	// Exact or directory-prefix match for literal patterns without globs
	if !strings.ContainsAny(pattern, "*?[") {
		if pattern == path {
			return true
		}
		if strings.HasPrefix(path, pattern+"/") {
			return true
		}
		// Match any segment equal to the pattern (e.g. node_modules at any depth)
		parts := strings.Split(path, "/")
		for _, p := range parts {
			if p == pattern {
				return true
			}
		}
		return false
	}
	// Glob fallback — filepath.Match on the full path.
	ok, _ := filepath.Match(pattern, path)
	if ok {
		return true
	}
	// Try last segment
	last := path
	if i := strings.LastIndex(path, "/"); i >= 0 {
		last = path[i+1:]
	}
	ok, _ = filepath.Match(pattern, last)
	return ok
}

// FileEntry is a resolved file or directory to be copied.
type FileEntry struct {
	AbsPath string // on-disk absolute path
	RelPath string // relative to source spec root (for preserving tree)
	IsDir   bool
	Mode    os.FileMode
	Size    int64
}

// ResolveSource expands one COPY/ADD source spec into file entries.
// The spec is relative to the context root; `..` escapes are rejected.
func (c *BuildContext) ResolveSource(spec string) ([]FileEntry, error) {
	spec = filepath.ToSlash(strings.TrimPrefix(spec, "./"))
	if spec == "" || strings.HasPrefix(spec, "/") || strings.Contains(spec, "..") {
		return nil, fmt.Errorf("copy source %q must be a relative path inside the build context", spec)
	}
	abs := filepath.Join(c.Root, filepath.FromSlash(spec))
	// Ensure still inside root after join
	rel, err := filepath.Rel(c.Root, abs)
	if err != nil || strings.HasPrefix(rel, "..") {
		return nil, fmt.Errorf("copy source %q escapes the build context", spec)
	}

	st, err := os.Stat(abs)
	if err != nil {
		return nil, fmt.Errorf("copy source %q: %w", spec, err)
	}

	var out []FileEntry
	if !st.IsDir() {
		if c.IsIgnored(filepath.ToSlash(rel)) {
			return nil, nil
		}
		out = append(out, FileEntry{AbsPath: abs, RelPath: filepath.Base(abs), IsDir: false, Mode: st.Mode(), Size: st.Size()})
		return out, nil
	}

	err = filepath.Walk(abs, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		r, err := filepath.Rel(abs, p)
		if err != nil {
			return err
		}
		if r == "." {
			return nil
		}
		fullRel, err := filepath.Rel(c.Root, p)
		if err != nil {
			return err
		}
		if c.IsIgnored(filepath.ToSlash(fullRel)) {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		out = append(out, FileEntry{
			AbsPath: p,
			RelPath: r,
			IsDir:   info.IsDir(),
			Mode:    info.Mode(),
			Size:    info.Size(),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(out, func(i, j int) bool { return out[i].RelPath < out[j].RelPath })
	return out, nil
}

// HashSources returns a stable sha256 of the file tree referenced by all
// sources. We hash (relPath, mode, size, contentHash) of every regular file
// so that any change busts the cache.
func (c *BuildContext) HashSources(sources []string) (string, error) {
	h := sha256.New()
	for _, src := range sources {
		entries, err := c.ResolveSource(src)
		if err != nil {
			return "", err
		}
		fmt.Fprintf(h, "SRC %s\n", src)
		for _, e := range entries {
			if e.IsDir {
				fmt.Fprintf(h, "D %s %o\n", filepath.ToSlash(e.RelPath), e.Mode.Perm())
				continue
			}
			sum, err := hashFile(e.AbsPath)
			if err != nil {
				return "", err
			}
			fmt.Fprintf(h, "F %s %o %d %s\n", filepath.ToSlash(e.RelPath), e.Mode.Perm(), e.Size, sum)
		}
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func hashFile(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
