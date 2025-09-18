package test

import (
    "regexp"
    "testing"
)

// TestCommitAndRunCommit commits a VM and starts a new cluster from that commit.
func TestCommitAndRunCommit(t *testing.T) {
    testEnv(t)
    ensureBuilt(t)

    // Original cluster aliases
    clusterAlias, vmAlias := uniqueAliases("smoke")
    // New cluster aliases for run-commit
    newClusterAlias := clusterAlias + "-from-commit"
    newVmAlias := vmAlias + "-from-commit"

    // Start a cluster
    out, err := runVers(t, defaultTimeout, "run", "-n", clusterAlias, "-N", vmAlias)
    if err != nil {
        t.Fatalf("vers run failed: %v\nOutput:\n%s", err, out)
    }
    registerClusterCleanup(t, clusterAlias)

    // Commit the VM; capture Commit ID from output
    out, err = runVers(t, defaultTimeout, "commit", vmAlias)
    if err != nil {
        if regexp.MustCompile(`(?i)Error uploading commit to S3|AWS CLI|S3 bucket`).FindString(out) != "" {
            t.Skipf("skipping commit test due to backend storage configuration: %v\nOutput:\n%s", err, out)
            return
        }
        t.Fatalf("vers commit failed: %v\nOutput:\n%s", err, out)
    }
    re := regexp.MustCompile(`(?m)^Commit ID:\s*([\w-]+)\s*$`)
    m := re.FindStringSubmatch(out)
    if len(m) != 2 {
        t.Fatalf("failed to extract commit ID from output:\n%s", out)
    }
    commitID := m[1]

    // Start a new cluster from the commit
    out, err = runVers(t, defaultTimeout, "run-commit", commitID, "-n", newClusterAlias, "-N", newVmAlias)
    if err != nil {
        t.Fatalf("vers run-commit failed: %v\nOutput:\n%s", err, out)
    }
    registerClusterCleanup(t, newClusterAlias)

    // Verify status resolves for the new cluster alias
    out, err = runVers(t, defaultTimeout, "status", "-c", newClusterAlias)
    if err != nil {
        t.Fatalf("vers status -c <new-from-commit> failed: %v\nOutput:\n%s", err, out)
    }
}
