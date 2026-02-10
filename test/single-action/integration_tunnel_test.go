package test

import (
	"context"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hdresearch/vers-cli/test/testutil"
)

// TestTunnelBasic tests that `vers tunnel` forwards traffic from a local port
// to a service running on the VM. It starts a simple TCP echo server on the VM,
// opens a tunnel, connects to the local end, and verifies data round-trips.
func TestTunnelBasic(t *testing.T) {
	t.Log("Starting TestTunnelBasic...")
	testutil.TestEnv(t)
	testutil.EnsureBuilt(t)

	// Create a VM
	t.Log("Running: vers run")
	out, err := testutil.RunVers(t, testutil.DefaultTimeout, "run")
	if err != nil {
		t.Fatalf("vers run failed: %v\nOutput:\n%s", err, out)
	}

	vmID, err := testutil.ParseVMID(out)
	if err != nil {
		t.Fatalf("failed to parse VM ID from output: %v\nOutput:\n%s", err, out)
	}
	t.Logf("Created VM: %s", vmID)
	testutil.RegisterVMCleanup(t, vmID, false)

	// Wait for VM to be fully ready
	t.Log("Waiting for VM networking...")
	if err := waitForVMReady(t, vmID, 60*time.Second); err != nil {
		t.Fatalf("VM did not become ready: %v", err)
	}

	// Start a TCP echo server on port 7777 inside the VM using socat.
	// socat is available on the default Vers image.
	t.Log("Starting echo server on VM port 7777...")
	_, err = testutil.RunVers(t, 30*time.Second, "execute", vmID, "--",
		"sh", "-c", "nohup socat TCP-LISTEN:7777,reuseaddr,fork EXEC:cat &")
	// execute returns after the shell backgrounds the process
	_ = err
	time.Sleep(2 * time.Second) // let the server bind

	// Verify the echo server is listening
	t.Log("Verifying echo server is listening...")
	checkOut, err := testutil.RunVers(t, 15*time.Second, "execute", vmID, "--",
		"sh", "-c", "ss -tln | grep 7777 || netstat -tln | grep 7777")
	if err != nil {
		t.Fatalf("echo server does not appear to be listening: %v\nOutput:\n%s", err, checkOut)
	}
	t.Logf("Echo server confirmed listening: %s", strings.TrimSpace(checkOut))

	// Start `vers tunnel` in the background
	// Use local port 0 so the OS picks a free port — but the CLI doesn't print
	// the assigned port yet in a machine-parseable way, so we pick a high port.
	localPort := findFreePort(t)
	tunnelSpec := fmt.Sprintf("%d:7777", localPort)

	t.Logf("Starting tunnel: vers tunnel %s %s", vmID, tunnelSpec)
	binPath, err := filepath.Abs(testutil.BinPath)
	if err != nil {
		t.Fatalf("failed to resolve binary path: %v", err)
	}

	tunnelCtx, tunnelCancel := context.WithCancel(context.Background())
	defer tunnelCancel()

	tunnelCmd := exec.CommandContext(tunnelCtx, binPath, "tunnel", vmID, tunnelSpec)
	tunnelCmd.Env = os.Environ()
	tunnelOut := &strings.Builder{}
	tunnelCmd.Stdout = tunnelOut
	tunnelCmd.Stderr = tunnelOut

	if err := tunnelCmd.Start(); err != nil {
		t.Fatalf("failed to start tunnel process: %v", err)
	}
	t.Cleanup(func() {
		tunnelCancel()
		_ = tunnelCmd.Wait()
	})

	// Give the tunnel time to establish the SSH connection and start listening
	time.Sleep(5 * time.Second)

	// Connect to the local end of the tunnel and send data
	t.Logf("Connecting to tunnel at 127.0.0.1:%d...", localPort)
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", localPort), 10*time.Second)
	if err != nil {
		t.Fatalf("failed to connect to local tunnel port %d: %v\nTunnel output:\n%s",
			localPort, err, tunnelOut.String())
	}
	defer conn.Close()

	// Send a test message
	testMsg := "hello-from-tunnel-test"
	conn.SetDeadline(time.Now().Add(10 * time.Second))

	_, err = conn.Write([]byte(testMsg))
	if err != nil {
		t.Fatalf("failed to write to tunnel: %v", err)
	}

	// Read the echoed response
	buf := make([]byte, 1024)
	n, err := conn.Read(buf)
	if err != nil {
		t.Fatalf("failed to read from tunnel: %v", err)
	}

	response := string(buf[:n])
	if response != testMsg {
		t.Fatalf("expected echo %q, got %q", testMsg, response)
	}

	t.Log("✓ Data round-tripped through tunnel successfully")
	t.Log("TestTunnelBasic completed")
}

// TestTunnelInvalidVM tests that tunnel fails gracefully with a non-existent VM.
func TestTunnelInvalidVM(t *testing.T) {
	t.Log("Starting TestTunnelInvalidVM...")
	testutil.TestEnv(t)
	testutil.EnsureBuilt(t)

	invalidVMID := "00000000-0000-0000-0000-000000000000"
	out, err := testutil.RunVers(t, 30*time.Second, "tunnel", invalidVMID, "8080:80")

	if err == nil {
		t.Fatal("expected error when tunneling to non-existent VM, got nil")
	}

	t.Logf("Got expected error: %v\nOutput:\n%s", err, out)
	t.Log("✓ Tunnel correctly fails for non-existent VM")
	t.Log("TestTunnelInvalidVM completed")
}

// TestTunnelInvalidSpec tests that tunnel rejects malformed port specs.
func TestTunnelInvalidSpec(t *testing.T) {
	t.Log("Starting TestTunnelInvalidSpec...")
	testutil.TestEnv(t)
	testutil.EnsureBuilt(t)

	specs := []string{
		"not-a-port",
		"abc:80",
		"8080:abc",
		"70000:80",
		"8080:0",
		"1:2:3:4",
	}

	for _, spec := range specs {
		t.Run(spec, func(t *testing.T) {
			// Use a dummy VM ID — spec validation should fail before VM resolution
			out, err := testutil.RunVers(t, 10*time.Second, "tunnel", "dummy-vm", spec)
			if err == nil {
				t.Fatalf("expected error for spec %q, got none. Output:\n%s", spec, out)
			}
			t.Logf("Got expected error for %q: %s", spec, strings.TrimSpace(out))
		})
	}

	t.Log("✓ Tunnel correctly rejects invalid specs")
	t.Log("TestTunnelInvalidSpec completed")
}

// TestTunnelUsesHEAD tests that tunnel falls back to HEAD when no VM is specified.
func TestTunnelUsesHEAD(t *testing.T) {
	t.Log("Starting TestTunnelUsesHEAD...")
	testutil.TestEnv(t)
	testutil.EnsureBuilt(t)

	// Create a VM (sets HEAD)
	tempDir, err := os.MkdirTemp("", "vers-tunnel-head-test-")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	t.Log("Running: vers run (in temp dir)")
	out, err := testutil.RunVersInDir(t, tempDir, testutil.DefaultTimeout, "run")
	if err != nil {
		t.Fatalf("vers run failed: %v\nOutput:\n%s", err, out)
	}

	vmID, err := testutil.ParseVMID(out)
	if err != nil {
		t.Fatalf("failed to parse VM ID from output: %v\nOutput:\n%s", err, out)
	}
	t.Logf("Created VM: %s", vmID)
	testutil.RegisterVMCleanup(t, vmID, false)

	// Wait for VM
	if err := waitForVMReady(t, vmID, 60*time.Second); err != nil {
		t.Fatalf("VM did not become ready: %v", err)
	}

	// Start tunnel from the same temp dir WITHOUT specifying VM ID — should use HEAD
	localPort := findFreePort(t)
	tunnelSpec := fmt.Sprintf("%d:22", localPort) // port 22 is always listening (sshd)

	binPath, err := filepath.Abs(testutil.BinPath)
	if err != nil {
		t.Fatalf("failed to resolve binary path: %v", err)
	}

	tunnelCtx, tunnelCancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer tunnelCancel()

	tunnelCmd := exec.CommandContext(tunnelCtx, binPath, "tunnel", tunnelSpec)
	tunnelCmd.Dir = tempDir
	tunnelCmd.Env = os.Environ()
	tunnelOut := &strings.Builder{}
	tunnelCmd.Stdout = tunnelOut
	tunnelCmd.Stderr = tunnelOut

	if err := tunnelCmd.Start(); err != nil {
		t.Fatalf("failed to start tunnel: %v", err)
	}
	t.Cleanup(func() {
		tunnelCancel()
		_ = tunnelCmd.Wait()
	})

	// Give tunnel time to start
	time.Sleep(5 * time.Second)

	output := tunnelOut.String()
	t.Logf("Tunnel output:\n%s", output)

	// Verify it mentions HEAD
	if !strings.Contains(output, "HEAD") {
		t.Logf("Warning: tunnel output doesn't mention HEAD. Output:\n%s", output)
	}

	// Verify the local port is accepting connections (meaning the tunnel is up)
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", localPort), 5*time.Second)
	if err != nil {
		t.Fatalf("tunnel not listening on local port %d (HEAD resolution may have failed): %v\nTunnel output:\n%s",
			localPort, err, output)
	}
	conn.Close()

	t.Log("✓ Tunnel correctly uses HEAD VM")
	t.Log("TestTunnelUsesHEAD completed")
}

// findFreePort asks the OS for a free TCP port.
func findFreePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to find free port: %v", err)
	}
	port := l.Addr().(*net.TCPAddr).Port
	l.Close()
	return port
}
