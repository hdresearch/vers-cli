package test

import (
    "strings"
    "testing"
)

// TestRun_InvalidFsSizes ensures the CLI preflight validation blocks invalid sizes.
func TestRun_InvalidFsSizes(t *testing.T) {
    testEnv(t)
    ensureBuilt(t)

    // VM > cluster should be rejected before API call.
    out, err := runVers(t, defaultTimeout, "run", "--fs-size-cluster", "256", "--fs-size-vm", "1024")
    if err == nil {
        t.Fatalf("expected failure for invalid fs sizes, got success. Output:\n%s", out)
    }
    if !strings.Contains(out, "invalid configuration: VM filesystem size") {
        t.Fatalf("expected friendly validation error, got:\n%s", out)
    }
}
