package test

import (
    "regexp"
    "testing"
    "time"
)

// TestClusterLifecycle creates a cluster, inspects it, and deletes it.
func TestClusterLifecycle(t *testing.T) {
    testEnv(t)
    ensureBuilt(t)

    clusterAlias, vmAlias := uniqueAliases("smoke")

    // Start a small cluster; rely on vers.toml defaults, but set aliases for easy targeting.
    out, err := runVers(t, defaultTimeout, "run", "-n", clusterAlias, "-N", vmAlias)
    if err != nil {
        t.Fatalf("vers run failed: %v\nOutput:\n%s", err, out)
    }

    // Always delete the created cluster at the end.
    registerClusterCleanup(t, clusterAlias)

    // Validate output includes the cluster ID and root VM info.
    re := regexp.MustCompile(`(?m)Cluster \(ID: ([^)]+)\) started successfully with root vm '([^']+)'\.`)
    matches := re.FindStringSubmatch(out)
    if len(matches) != 3 {
        t.Fatalf("unexpected run output; could not find cluster creation line.\nOutput:\n%s", out)
    }

    // Use status to fetch details by alias (server resolves ID or alias).
    out, err = runVers(t, 45*time.Second, "status", "-c", clusterAlias)
    if err != nil {
        t.Fatalf("vers status -c <alias> failed: %v\nOutput:\n%s", err, out)
    }
}

