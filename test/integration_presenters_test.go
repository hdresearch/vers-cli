package test

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	presenters "github.com/hdresearch/vers-cli/internal/presenters"
	"github.com/hdresearch/vers-cli/styles"
	vers "github.com/hdresearch/vers-sdk-go"
)

func capStdout(t *testing.T, fn func()) string {
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

func TestPresenters_TreeIntegration(t *testing.T) {
	// No backend required: present a synthetic cluster and assert output shape
	cluster := vers.APIClusterGetResponseData{
		ID:       "cluster-x",
		Alias:    "it-cluster",
		VmCount:  3,
		RootVmID: "vm-root",
		Vms: []vers.VmDto{
			{ID: "vm-root", Alias: "root", State: "Running", Children: []string{"vm-a", "vm-b"}},
			{ID: "vm-a", Alias: "alpha", State: "Running"},
			{ID: "vm-b", Alias: "beta", State: "Stopped"},
		},
	}
	out := capStdout(t, func() { _ = presenters.RenderTree(cluster, "vm-a") })
	if !strings.Contains(out, "Cluster: it-cluster (Total VMs: 3)") {
		t.Fatalf("expected cluster header, got:\n%s", out)
	}
	if !strings.Contains(out, "alpha") || !strings.Contains(out, "<- HEAD") {
		t.Fatalf("expected HEAD marker on alpha, got:\n%s", out)
	}
	if !strings.Contains(out, "Legend:") || !strings.Contains(out, "[R]") || !strings.Contains(out, "[S]") {
		t.Fatalf("expected legend and state markers, got:\n%s", out)
	}
}

func TestPresenters_StatusIntegration(t *testing.T) {
	s := styles.NewStatusStyles()

	// Cluster status rendering
	cluster := vers.APIClusterGetResponseData{
		ID:       "cid",
		Alias:    "c1",
		RootVmID: "r1",
		Vms: []vers.VmDto{
			{ID: "r1", Alias: "root", State: "Running"},
			{ID: "v2", Alias: "node-2", State: "Paused"},
		},
	}
	outC := capStdout(t, func() { presenters.RenderClusterStatus(&s, cluster) })
	if !strings.Contains(outC, "Getting status for cluster: c1") || !strings.Contains(outC, "VMs in this cluster:") {
		t.Fatalf("missing cluster status markers.\n%s", outC)
	}

	// VM status rendering
	vm := vers.APIVmGetResponseData{ID: "vm1", Alias: "v1", State: "Running", ClusterID: "cid"}
	outV := capStdout(t, func() { presenters.RenderVMStatus(&s, vm) })
	if !strings.Contains(outV, "Getting status for VM: v1") || !strings.Contains(outV, "Cluster: cid") {
		t.Fatalf("missing VM status markers.\n%s", outV)
	}
}
