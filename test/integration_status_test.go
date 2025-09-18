package test

import (
    "strings"
    "testing"
    "time"
)

// TestStatus_Smoke verifies we can reach the backend and list status.
func TestStatus_Smoke(t *testing.T) {
    testEnv(t)
    ensureBuilt(t)

    out, err := runVers(t, 30*time.Second, "status")
    if err != nil {
        t.Fatalf("vers status failed: %v\nOutput:\n%s", err, out)
    }

    // Basic sanity: output includes common markers indicating successful execution.
    ok := strings.Contains(out, "Cluster details:") ||
        strings.Contains(out, "Tip:") ||
        strings.Contains(out, "No clusters found.") ||
        strings.Contains(out, "Fetching list of clusters")
    if !ok {
        t.Fatalf("unexpected status output; got:\n%s", out)
    }
}
