package presenters_test

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	presenters "github.com/hdresearch/vers-cli/internal/presenters"
	vers "github.com/hdresearch/vers-sdk-go"
)

func captureOutput(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w
	fn()
	_ = w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	_ = r.Close()
	return buf.String()
}

func TestRenderTree_PrintsClusterAndHead(t *testing.T) {
	cluster := vers.APIClusterGetResponseData{
		ID:       "cluster-123",
		Alias:    "my-cluster",
		VmCount:  2,
		RootVmID: "vm-root",
		Vms: []vers.VmDto{
			{ID: "vm-root", Alias: "root", State: "Running", Children: []string{"vm-child"}},
			{ID: "vm-child", Alias: "child-a", State: "Running"},
		},
	}

	out := captureOutput(t, func() {
		_ = presenters.RenderTree(cluster, "vm-child")
	})

	// Key markers
	if !strings.Contains(out, "Cluster: my-cluster (Total VMs: 2)") {
		t.Fatalf("expected cluster header, got:\n%s", out)
	}
	if !strings.Contains(out, "child-a") || !strings.Contains(out, "<- HEAD") {
		t.Fatalf("expected HEAD marker on child-a, got:\n%s", out)
	}
	if !strings.Contains(out, "Legend:") {
		t.Fatalf("expected legend in output, got:\n%s", out)
	}
}
