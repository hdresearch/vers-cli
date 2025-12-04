package test

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/hdresearch/vers-cli/internal/auth"
	"github.com/hdresearch/vers-cli/test/testutil"
	vers "github.com/hdresearch/vers-sdk-go"
)

// TestConnect_SSHOverTLS tests the connect command using SSH-over-TLS.
func TestConnect_SSHOverTLS(t *testing.T) {
	t.Log("Starting TestConnect_SSHOverTLS...")
	testutil.TestEnv(t)
	testutil.EnsureBuilt(t)

	// Create a VM
	t.Log("Running: vers run")
	out, err := testutil.RunVers(t, testutil.DefaultTimeout, "run")
	if err != nil {
		t.Fatalf("vers run failed: %v\nOutput:\n%s", err, out)
	}

	// Parse VM ID
	vmID, err := testutil.ParseVMID(out)
	if err != nil {
		t.Fatalf("failed to parse VM ID from output: %v\nOutput:\n%s", err, out)
	}
	t.Logf("Created VM: %s", vmID)
	testutil.RegisterVMCleanup(t, vmID, false)

	// Wait for VM to be fully ready (networking configured)
	t.Log("Waiting for VM networking to be configured...")
	if err := waitForVMReady(t, vmID, 60*time.Second); err != nil {
		t.Fatalf("VM did not become ready: %v", err)
	}

	// Test connect command by running a simple command
	// Note: we can't use `vers connect` directly in a test because it spawns an interactive shell
	// Instead, we'll test the SSH connection using the same method the connect command uses
	t.Log("Testing SSH-over-TLS connection...")

	// Get client
	clientOptions, err := auth.GetClientOptions()
	if err != nil {
		t.Fatalf("failed to get client options: %v", err)
	}
	client := vers.NewClient(clientOptions...)

	// Get SSH key
	ctx := context.Background()
	keyPath, err := auth.GetOrCreateSSHKey(vmID, client, ctx)
	if err != nil {
		t.Fatalf("failed to get SSH key: %v", err)
	}
	t.Logf("SSH key path: %s", keyPath)

	// Test SSH connection using SSH-over-TLS with retry logic
	// Based on our debugging, first connections may fail if networking isn't fully ready
	vmHostname := fmt.Sprintf("%s.vm.vers.sh", vmID)
	proxyCommand := fmt.Sprintf("openssl s_client -connect %s:443 -servername %s -quiet", vmHostname, vmHostname)

	sshArgs := []string{
		fmt.Sprintf("root@%s", vmHostname),
		"-o", fmt.Sprintf("ProxyCommand=%s", proxyCommand),
		"-o", "StrictHostKeyChecking=no",
		"-o", "UserKnownHostsFile=/dev/null",
		"-o", "IdentitiesOnly=yes",
		"-o", "PreferredAuthentications=publickey",
		"-o", "LogLevel=ERROR",
		"-o", "ConnectTimeout=30",
		"-o", "ServerAliveInterval=10",
		"-i", keyPath,
		"whoami",
	}

	// Retry SSH connection up to 3 times (accounting for networking race condition)
	var sshOut []byte
	var sshErr error
	maxRetries := 3

	for attempt := 1; attempt <= maxRetries; attempt++ {
		t.Logf("SSH connection attempt %d/%d...", attempt, maxRetries)
		sshCmd := exec.Command("ssh", sshArgs...)
		sshOut, sshErr = sshCmd.CombinedOutput()

		if sshErr == nil {
			// Success!
			break
		}

		t.Logf("Attempt %d failed: %v", attempt, sshErr)
		if attempt < maxRetries {
			t.Logf("Waiting 5 seconds before retry...")
			time.Sleep(5 * time.Second)
		}
	}

	if sshErr != nil {
		t.Fatalf("SSH command failed after %d attempts: %v\nLast output:\n%s", maxRetries, sshErr, string(sshOut))
	}

	// Verify output - filter out OpenSSL certificate messages
	output := strings.TrimSpace(string(sshOut))
	t.Logf("SSH command output: %s", output)

	// The output may contain OpenSSL certificate verification messages
	// Extract the last non-empty line which should be the actual command output
	lines := strings.Split(output, "\n")
	var lastLine string
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line != "" && !strings.Contains(line, "verify") && !strings.Contains(line, "depth=") {
			lastLine = line
			break
		}
	}

	if lastLine != "root" {
		t.Errorf("expected 'root' from whoami command, got: %s", lastLine)
	}

	t.Log("✓ SSH-over-TLS connection successful")
	t.Log("TestConnect_SSHOverTLS completed")
}

// waitForVMReady waits for the VM to appear and gives time for networking to be configured.
// Since there's a known race condition where VMs become API-queryable before networking
// is fully configured, we wait for the VM to appear then give extra time for WireGuard
// and DNAT rules to be set up.
func waitForVMReady(t *testing.T, vmID string, timeout time.Duration) error {
	t.Helper()

	clientOptions, err := auth.GetClientOptions()
	if err != nil {
		return fmt.Errorf("failed to get client options: %w", err)
	}
	client := vers.NewClient(clientOptions...)

	deadline := time.Now().Add(timeout)
	attempt := 0

	// First, wait for the VM to appear in the list
	for time.Now().Before(deadline) {
		attempt++
		t.Logf("Checking if VM exists (attempt %d)...", attempt)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		vms, err := client.Vm.List(ctx)
		cancel()

		if err != nil {
			t.Logf("Error listing VMs: %v (will retry)", err)
			time.Sleep(2 * time.Second)
			continue
		}

		// Find our VM
		found := false
		for _, v := range *vms {
			if v.VmID == vmID {
				found = true
				break
			}
		}

		if found {
			t.Logf("VM found in list, waiting for networking to be configured...")
			// Give Chelsea time to finish WireGuard setup and DNAT rules
			// Based on our earlier debugging, first connections may fail if networking isn't ready
			time.Sleep(10 * time.Second)
			return nil
		}

		t.Logf("VM not found in list (will retry)")
		time.Sleep(2 * time.Second)
	}

	return fmt.Errorf("VM did not appear within %v", timeout)
}

// TestConnect_InvalidVM tests that connect fails gracefully with a non-existent VM.
func TestConnect_InvalidVM(t *testing.T) {
	t.Log("Starting TestConnect_InvalidVM...")
	testutil.TestEnv(t)
	testutil.EnsureBuilt(t)

	// Try to connect to a non-existent VM
	invalidVMID := "00000000-0000-0000-0000-000000000000"
	t.Logf("Attempting to connect to invalid VM: %s", invalidVMID)

	// We expect this to fail
	_, err := testutil.RunVers(t, testutil.DefaultTimeout, "connect", invalidVMID)

	if err == nil {
		t.Fatal("expected error when connecting to non-existent VM, got nil")
	}

	t.Logf("Got expected error: %v", err)
	t.Log("✓ Connect correctly fails for non-existent VM")
	t.Log("TestConnect_InvalidVM completed")
}
