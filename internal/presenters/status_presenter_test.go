package presenters_test

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

func capOut(t *testing.T, fn func()) string {
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

func TestRenderClusterStatus_PrintsDetails(t *testing.T) {
	t.Skip("Cluster concept removed from SDK - test deprecated")
	// Note: This test uses vers.APIClusterGetResponseData which no longer exists
}

func TestRenderVMStatus_PrintsDetails(t *testing.T) {
	s := styles.NewStatusStyles()
	// Use the new vers.Vm type instead of APIVmGetResponseData
	vm := &vers.Vm{
		VmID:   "vm1",
		IP:     "192.168.1.100",
		Parent: "root",
	}
	out := capOut(t, func() { presenters.RenderVMStatus(&s, vm) })
	if !strings.Contains(out, "Getting status for VM: vm1") {
		t.Fatalf("missing VM header: %s", out)
	}
	// Note: ClusterID field no longer exists in new SDK
}

func TestRenderClusterList_PrintsList(t *testing.T) {
	t.Skip("Cluster concept removed from SDK - test deprecated")
	// Note: This test uses vers.APIClusterListResponseData which no longer exists
	// RenderClusterList function may also be removed
}
