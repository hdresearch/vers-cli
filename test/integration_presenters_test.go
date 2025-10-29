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

func TestPresenters_StatusIntegration(t *testing.T) {
	s := styles.NewStatusStyles()

	// VM status rendering - updated to use new vers.Vm type
	t.Run("VMStatus", func(t *testing.T) {
		vm := &vers.Vm{
			VmID:   "vm1",
			IP:     "192.168.1.100",
			Parent: "root",
		}
		outV := capStdout(t, func() { presenters.RenderVMStatus(&s, vm) })
		if !strings.Contains(outV, "Getting status for VM: vm1") {
			t.Fatalf("missing VM status markers.\n%s", outV)
		}
	})
}
