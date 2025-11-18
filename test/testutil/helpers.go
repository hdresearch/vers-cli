package testutil

import (
	"context"
	"fmt"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

const (
	DefaultTimeout = 60 * time.Second
)

var (
	BinPath     string
	envFileRoot string
)

func init() {
	// Dynamically determine paths based on working directory
	// Tests can run from test/ or test/single-action/
	if _, err := os.Stat("../cmd/vers"); err == nil {
		// Running from test/ directory
		BinPath = "../bin/vers"
		envFileRoot = "../.env"
	} else {
		// Running from test/single-action/ or similar
		BinPath = "../../bin/vers"
		envFileRoot = "../../.env"
	}
}

// TestEnv ensures required env vars are present; loads root .env if found.
func TestEnv(t TLike) {
	// Load root .env and local .env for convenience if present
	_ = godotenv.Load(envFileRoot)
	_ = godotenv.Load(".env")

	missing := []string{}
	for _, k := range []string{"VERS_URL", "VERS_API_KEY"} {
		if strings.TrimSpace(os.Getenv(k)) == "" {
			missing = append(missing, k)
		}
	}
	if len(missing) > 0 {
		t.Skipf("missing required env vars for integration tests: %s", strings.Join(missing, ", "))
	}

	// Normalize VERS_URL to include a scheme if missing (CLI requires it)
	if url := strings.TrimSpace(os.Getenv("VERS_URL")); url != "" {
		if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
			os.Setenv("VERS_URL", "http://"+url)
		}
	}
}

// EnsureBuilt builds the CLI binary if it doesn't exist.
func EnsureBuilt(t TLike) {
	// Always rebuild to pick up latest changes during dev/test
	if err := os.MkdirAll(filepath.Dir(BinPath), 0o755); err != nil {
		t.Fatalf("failed to create bin dir: %v", err)
	}
	// Determine the correct path to cmd/vers
	cmdPath := "../cmd/vers"
	if _, err := os.Stat(cmdPath); err != nil {
		cmdPath = "../../cmd/vers"
	}
	cmd := exec.Command("go", "build", "-o", BinPath, cmdPath)
	cmd.Env = os.Environ()
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("failed to build CLI: %v\n%s", err, string(out))
	}
}

// RunVers executes the CLI with a timeout and returns combined stdout/stderr and error.
func RunVers(t TLike, timeout time.Duration, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, BinPath, args...)
	// Inherit env so VERS_URL and VERS_API_KEY are visible
	cmd.Env = os.Environ()
	out, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return string(out), fmt.Errorf("command timed out: vers %s", strings.Join(args, " "))
	}
	return string(out), err
}

// RegisterVMCleanup ensures a VM is deleted at test end.
func RegisterVMCleanup(t TLike, identifier string, recursive bool) {
	t.Cleanup(func() {
		args := []string{"kill", "-y"}
		if recursive {
			args = append(args, "-r")
		}
		args = append(args, identifier)
		_, _ = RunVers(t, DefaultTimeout, args...)
	})
}

// UniqueAlias returns a unique alias string scoped to a test run.
func UniqueAlias(prefix string) string {
	// Keep it readable and collision-resistant without external deps.
	ts := time.Now().UTC().Format("20060102-150405")
	randPart := rand.Intn(1_000_000)
	return fmt.Sprintf("%s-it-%s-%06d", prefix, ts, randPart)
}

// ParseVMID extracts the VM ID from `vers run` output.
// The output format is: "VM '<vmID>' started successfully."
func ParseVMID(output string) (string, error) {
	// Look for the pattern: VM '<id>' started successfully
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "started successfully") {
			// Extract text between "VM '" and "' started"
			start := strings.Index(line, "VM '")
			if start == -1 {
				continue
			}
			start += 4 // Move past "VM '"
			end := strings.Index(line[start:], "' started")
			if end == -1 {
				continue
			}
			vmID := line[start : start+end]
			if vmID != "" {
				return vmID, nil
			}
		}
	}
	return "", fmt.Errorf("could not parse VM ID from output")
}

// TLike is the subset of *testing.T methods we use; helps reuse in helpers.
type TLike interface {
	Cleanup(func())
	Fatalf(string, ...any)
	Skipf(string, ...any)
}
