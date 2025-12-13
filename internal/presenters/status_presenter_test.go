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

func TestRenderVMStatus_PrintsDetails(t *testing.T) {
	s := styles.NewStatusStyles()
	vm := &vers.Vm{
		VmID: "vm1",
	}
	out := capOut(t, func() { presenters.RenderVMStatus(&s, vm) })
	if !strings.Contains(out, "Getting status for VM: vm1") {
		t.Fatalf("missing VM header: %s", out)
	}
	if !strings.Contains(out, "VM: vm1") {
		t.Fatalf("missing VM details: %s", out)
	}
}
