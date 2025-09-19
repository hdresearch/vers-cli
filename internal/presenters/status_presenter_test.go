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
    s := styles.NewStatusStyles()
    cluster := vers.APIClusterGetResponseData{
        ID:       "cid",
        Alias:    "c-alias",
        RootVmID: "root",
        Vms: []vers.VmDto{
            {ID: "root", Alias: "root-a", State: "Running"},
            {ID: "v2", Alias: "v2-a", State: "Stopped"},
        },
    }

    out := capOut(t, func() { presenters.RenderClusterStatus(&s, cluster) })
    if !strings.Contains(out, "Getting status for cluster: c-alias") {
        t.Fatalf("missing cluster header: %s", out)
    }
    if !strings.Contains(out, "VMs in this cluster:") || !strings.Contains(out, "v2-a") {
        t.Fatalf("missing VM list: %s", out)
    }
}

func TestRenderVMStatus_PrintsDetails(t *testing.T) {
    s := styles.NewStatusStyles()
    vm := vers.APIVmGetResponseData{ID: "vm1", Alias: "vm-a", State: "Running", ClusterID: "cid"}
    out := capOut(t, func() { presenters.RenderVMStatus(&s, vm) })
    if !strings.Contains(out, "Getting status for VM: vm-a") {
        t.Fatalf("missing VM header: %s", out)
    }
    if !strings.Contains(out, "Cluster: cid") {
        t.Fatalf("missing cluster id: %s", out)
    }
}

func TestRenderClusterList_PrintsList(t *testing.T) {
    s := styles.NewStatusStyles()
    clusters := []vers.APIClusterListResponseData{
        {ID: "c1", Alias: "a1", RootVmID: "r1", VmCount: 1, Vms: []vers.VmDto{{ID: "r1", Alias: "ra"}}},
        {ID: "c2", Alias: "", RootVmID: "r2", VmCount: 2},
    }
    out := capOut(t, func() { presenters.RenderClusterList(&s, clusters) })
    if !strings.Contains(out, "Cluster: a1") || !strings.Contains(out, "Cluster: c2") {
        t.Fatalf("expected cluster entries, got: %s", out)
    }
}

