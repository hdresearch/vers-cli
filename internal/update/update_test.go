package update

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hdresearch/vers-cli/internal/config"
)

func TestIsNewerSemver(t *testing.T) {
	cases := []struct {
		current, latest string
		want            bool
	}{
		{"v0.9.0", "v0.10.0", true},
		{"0.9.0", "0.10.0", true},
		{"v0.10.0", "v0.9.0", false},
		{"v0.10.0", "v0.10.0", false},
		{"v1.0.0", "v1.0.1", true},
		{"v1.0.1", "v1.0.0", false},
		{"v1.2.3", "v2.0.0", true},
		{"v0.9", "v0.10", true}, // shorter current
		{"v0.9.0", "v0.9.0-rc1", false},
		{"", "v0.1.0", false}, // empty current is treated as dev
		// Lexical fallback for unparseable versions: the function returns
		// (current != latest) which preserves the historical behavior the
		// old implementation depended on.
		{"main", "abc", true},
	}
	for _, c := range cases {
		got := isNewerSemver(c.current, c.latest)
		if got != c.want {
			t.Errorf("isNewerSemver(%q, %q) = %v, want %v", c.current, c.latest, got, c.want)
		}
	}
}

func TestIsDevVersion(t *testing.T) {
	dev := []string{
		"dev", "unknown", "dev-abc1234", "dev-abc1234-dirty",
		"v0.10.0-dirty", "",
		// Go module pseudo-versions, as produced by `go install` /
		// `go run` against an untagged commit.
		"v0.0.0-20260506213344-eb1e5d25fa7c",
		"v0.0.0-20260101000000-000000000000",
	}
	notDev := []string{"v0.10.0", "0.10.0", "v1.2.3", "v0.10.0-rc1"}
	for _, v := range dev {
		if !IsDevVersion(v) {
			t.Errorf("IsDevVersion(%q) = false, want true", v)
		}
	}
	for _, v := range notDev {
		if IsDevVersion(v) {
			t.Errorf("IsDevVersion(%q) = true, want false", v)
		}
	}
}

func TestMaybeNotifyUpdate_EnvVarSilencesCheck(t *testing.T) {
	withTempHome(t)

	// Server that fails the test if hit.
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		http.Error(w, "should not be called", 500)
	}))
	defer srv.Close()
	withRewrittenHTTP(t, strings.TrimPrefix(srv.URL, "http://"))

	for _, varName := range []string{"VERS_NO_UPDATE_CHECK", "NO_UPDATE_NOTIFIER"} {
		t.Run(varName, func(t *testing.T) {
			t.Setenv(varName, "1")
			out := captureStderr(t, func() {
				MaybeNotifyUpdate(context.Background(), "v0.9.0", "https://github.com/hdresearch/vers-cli", 500*time.Millisecond, false)
			})
			if hits != 0 {
				t.Errorf("expected zero network calls when %s=1, got %d", varName, hits)
			}
			if out != "" {
				t.Errorf("expected silent output when %s=1, got %q", varName, out)
			}
		})
	}
}

func TestMaybeNotifyUpdate_EnvVarFalsyValuesDoNotSilence(t *testing.T) {
	// "0", "false", "no", "off" should NOT silence the nag — only positive
	// truthy values do. This mirrors common CLI conventions.
	for _, val := range []string{"0", "false", "FALSE", "no", "off", ""} {
		t.Run("VERS_NO_UPDATE_CHECK="+val, func(t *testing.T) {
			withTempHome(t)
			t.Setenv("VERS_NO_UPDATE_CHECK", val)
			t.Setenv("NO_UPDATE_NOTIFIER", "")

			cfg := &config.CLIConfig{
				UpdateCheck: config.UpdateCheckConfig{
					NextCheck:     time.Now().Add(-1 * time.Hour),
					CheckInterval: 3600,
				},
			}
			_ = config.SaveCLIConfig(cfg)

			hits := 0
			srv := fakeReleasesServer(t, "v0.10.0", &hits)
			defer srv.Close()
			withRewrittenHTTP(t, strings.TrimPrefix(srv.URL, "http://"))

			out := captureStderr(t, func() {
				MaybeNotifyUpdate(context.Background(), "v0.9.0", "https://github.com/hdresearch/vers-cli", 1*time.Second, false)
			})
			if !strings.Contains(out, "v0.9.0 -> v0.10.0") {
				t.Errorf("expected banner with VERS_NO_UPDATE_CHECK=%q, got %q", val, out)
			}
		})
	}
}

// withTempHome redirects $HOME to a temporary directory for the duration of
// the test so config.LoadCLIConfig / SaveCLIConfig don't touch the user's
// real ~/.vers/config.json. Returns the path to the temp config file.
func withTempHome(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	// On some platforms os.UserHomeDir consults USERPROFILE / etc.; force HOME.
	t.Setenv("USERPROFILE", dir)
	return filepath.Join(dir, ".vers", "config.json")
}

// fakeReleasesServer returns an httptest server that serves a single
// /repos/.../releases/latest payload with the given tag.
func fakeReleasesServer(t *testing.T, tag string, hits *int) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if hits != nil {
			*hits++
		}
		if !strings.HasSuffix(r.URL.Path, "/releases/latest") {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(GitHubRelease{TagName: tag, Name: tag})
	}))
}

// repoStringForServer builds a "repository" value that GetLatestReleaseContext
// will rewrite into the fake server's URL. We exploit the fact that the code
// strips the "https://github.com/" prefix and concatenates "https://api.github.com".
// To redirect to httptest, we instead inject a custom http.RoundTripper.
type rewriteTransport struct {
	target string
	base   http.RoundTripper
}

func (rt *rewriteTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	// Rewrite api.github.com → httptest server
	if strings.Contains(r.URL.Host, "api.github.com") {
		u := *r.URL
		// httptest server URL is like http://127.0.0.1:PORT
		// Parse it once.
		// We stash the target in rt.target as "host:port".
		u.Scheme = "http"
		u.Host = rt.target
		r2 := r.Clone(r.Context())
		r2.URL = &u
		r2.Host = rt.target
		return rt.base.RoundTrip(r2)
	}
	return rt.base.RoundTrip(r)
}

func withRewrittenHTTP(t *testing.T, host string) {
	t.Helper()
	orig := http.DefaultClient.Transport
	http.DefaultClient.Transport = &rewriteTransport{target: host, base: http.DefaultTransport}
	t.Cleanup(func() { http.DefaultClient.Transport = orig })
}

func TestMaybeNotifyUpdate_PrintsFromCacheWithoutNetwork(t *testing.T) {
	configPath := withTempHome(t)

	// Pre-seed config with a known-newer LatestVersion and a NextCheck in
	// the future so the slow path is skipped entirely.
	cfg := &config.CLIConfig{
		UpdateCheck: config.UpdateCheckConfig{
			LastCheck:     time.Now(),
			NextCheck:     time.Now().Add(1 * time.Hour),
			CheckInterval: 3600,
			LatestVersion: "v0.10.0",
		},
	}
	if err := config.SaveCLIConfig(cfg); err != nil {
		t.Fatalf("seed config: %v", err)
	}

	// Use a server that, if hit, would fail the test.
	hits := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		http.Error(w, "should not be called", 500)
	}))
	defer srv.Close()
	withRewrittenHTTP(t, strings.TrimPrefix(srv.URL, "http://"))

	// Capture stderr.
	out := captureStderr(t, func() {
		MaybeNotifyUpdate(context.Background(), "v0.9.0", "https://github.com/hdresearch/vers-cli", 500*time.Millisecond, false)
	})

	if hits != 0 {
		t.Errorf("expected zero network calls on fast path, got %d", hits)
	}
	if !strings.Contains(out, "v0.9.0 -> v0.10.0") {
		t.Errorf("expected update banner in output, got %q", out)
	}
	// Sanity: config was not rewritten.
	got, _ := config.LoadCLIConfig()
	if got.UpdateCheck.LatestVersion != "v0.10.0" {
		t.Errorf("config LatestVersion changed unexpectedly: %q", got.UpdateCheck.LatestVersion)
	}
	_ = configPath
}

func TestMaybeNotifyUpdate_RefreshesWhenStale(t *testing.T) {
	withTempHome(t)

	// Seed config with NextCheck in the past so the slow path runs.
	cfg := &config.CLIConfig{
		UpdateCheck: config.UpdateCheckConfig{
			LastCheck:     time.Now().Add(-2 * time.Hour),
			NextCheck:     time.Now().Add(-1 * time.Hour),
			CheckInterval: 3600,
			// Empty LatestVersion: nothing to print on the fast path.
		},
	}
	if err := config.SaveCLIConfig(cfg); err != nil {
		t.Fatalf("seed config: %v", err)
	}

	hits := 0
	srv := fakeReleasesServer(t, "v0.10.0", &hits)
	defer srv.Close()
	withRewrittenHTTP(t, strings.TrimPrefix(srv.URL, "http://"))

	out := captureStderr(t, func() {
		MaybeNotifyUpdate(context.Background(), "v0.9.0", "https://github.com/hdresearch/vers-cli", 2*time.Second, false)
	})

	if hits != 1 {
		t.Errorf("expected exactly 1 network call, got %d", hits)
	}
	if !strings.Contains(out, "v0.9.0 -> v0.10.0") {
		t.Errorf("expected update banner, got %q", out)
	}

	// Cache should now be populated and NextCheck pushed into the future.
	got, _ := config.LoadCLIConfig()
	if got.UpdateCheck.LatestVersion != "v0.10.0" {
		t.Errorf("expected LatestVersion=v0.10.0, got %q", got.UpdateCheck.LatestVersion)
	}
	if !got.UpdateCheck.NextCheck.After(time.Now().Add(30 * time.Minute)) {
		t.Errorf("expected NextCheck pushed forward, got %v", got.UpdateCheck.NextCheck)
	}
}

func TestMaybeNotifyUpdate_NetworkFailureDoesNotSuppressForFullInterval(t *testing.T) {
	withTempHome(t)

	cfg := &config.CLIConfig{
		UpdateCheck: config.UpdateCheckConfig{
			NextCheck:     time.Now().Add(-1 * time.Hour),
			CheckInterval: 3600,
		},
	}
	if err := config.SaveCLIConfig(cfg); err != nil {
		t.Fatalf("seed config: %v", err)
	}

	// Server that always 500s.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "boom", 500)
	}))
	defer srv.Close()
	withRewrittenHTTP(t, strings.TrimPrefix(srv.URL, "http://"))

	_ = captureStderr(t, func() {
		MaybeNotifyUpdate(context.Background(), "v0.9.0", "https://github.com/hdresearch/vers-cli", 500*time.Millisecond, false)
	})

	got, _ := config.LoadCLIConfig()
	// NextCheck should be ~5min from now (the short backoff), not ~1h.
	if got.UpdateCheck.NextCheck.After(time.Now().Add(30 * time.Minute)) {
		t.Errorf("expected short backoff after failure, NextCheck=%v", got.UpdateCheck.NextCheck)
	}
	if got.UpdateCheck.NextCheck.Before(time.Now().Add(1 * time.Minute)) {
		t.Errorf("expected NextCheck at least ~5min in future, got %v", got.UpdateCheck.NextCheck)
	}
}

func TestMaybeNotifyUpdate_DevVersionSkipsEverything(t *testing.T) {
	withTempHome(t)
	hits := 0
	srv := fakeReleasesServer(t, "v9.9.9", &hits)
	defer srv.Close()
	withRewrittenHTTP(t, strings.TrimPrefix(srv.URL, "http://"))

	out := captureStderr(t, func() {
		MaybeNotifyUpdate(context.Background(), "dev-abc1234", "https://github.com/hdresearch/vers-cli", 500*time.Millisecond, false)
	})
	if hits != 0 {
		t.Errorf("dev version should not hit network, got %d hits", hits)
	}
	if out != "" {
		t.Errorf("dev version should not print, got %q", out)
	}
}

func TestMaybeNotifyUpdate_NoBannerWhenAlreadyOnLatest(t *testing.T) {
	withTempHome(t)
	cfg := &config.CLIConfig{
		UpdateCheck: config.UpdateCheckConfig{
			NextCheck:     time.Now().Add(-1 * time.Hour),
			CheckInterval: 3600,
		},
	}
	_ = config.SaveCLIConfig(cfg)

	hits := 0
	srv := fakeReleasesServer(t, "v0.10.0", &hits)
	defer srv.Close()
	withRewrittenHTTP(t, strings.TrimPrefix(srv.URL, "http://"))

	out := captureStderr(t, func() {
		MaybeNotifyUpdate(context.Background(), "v0.10.0", "https://github.com/hdresearch/vers-cli", 1*time.Second, false)
	})
	if out != "" {
		t.Errorf("no banner expected when on latest, got %q", out)
	}
	if hits != 1 {
		t.Errorf("expected 1 hit, got %d", hits)
	}
}

// captureStderr runs fn and returns whatever it wrote to os.Stderr.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	orig := os.Stderr
	os.Stderr = w
	done := make(chan string, 1)
	go func() {
		buf := make([]byte, 0, 1024)
		tmp := make([]byte, 512)
		for {
			n, err := r.Read(tmp)
			if n > 0 {
				buf = append(buf, tmp[:n]...)
			}
			if err != nil {
				done <- string(buf)
				return
			}
		}
	}()
	fn()
	w.Close()
	os.Stderr = orig
	return <-done
}

// Sanity: assert the banner format hasn't drifted, since downstream tooling
// (and the slack message we link users to) reference its wording.
func TestPrintUpdateBannerFormat(t *testing.T) {
	out := captureStderr(t, func() {
		printUpdateBanner("v0.9.0", "v0.10.0")
	})
	if !strings.Contains(out, "vers update available") {
		t.Errorf("banner missing identifying phrase: %q", out)
	}
	if !strings.Contains(out, "vers upgrade") {
		t.Errorf("banner should tell users to run 'vers upgrade': %q", out)
	}
	// Avoid using fmt-style placeholders by accident.
	if strings.Contains(out, "%s") || strings.Contains(out, "%v") {
		t.Errorf("banner contains unrendered placeholder: %q", out)
	}
	_ = fmt.Sprintf // keep fmt import used even if we drop a check above
}
