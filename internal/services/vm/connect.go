package vm

import (
	"context"
	"net/url"

	"github.com/hdresearch/vers-cli/internal/auth"
	"github.com/hdresearch/vers-cli/internal/utils"
	vers "github.com/hdresearch/vers-sdk-go"
)

// Info contains details needed to establish an SSH connection to a VM.
type Info struct {
	VM      *vers.Vm
	Host    string // preferred host (node IP or fallback base hostname)
	KeyPath string // local path to SSH private key
	BaseURL *url.URL
}

// GetConnectInfo resolves the VM and returns connection information, including Node IP and SSH key.
func GetConnectInfo(ctx context.Context, client *vers.Client, identifier string) (Info, error) {
	var out Info

	vm, nodeIP, err := utils.GetVmAndNodeIP(ctx, client, identifier)
	if err != nil {
		return out, err
	}
	out.VM = vm

	versURL, err := auth.GetVersUrl()
	if err != nil {
		return out, err
	}
	out.BaseURL = versURL

	host := nodeIP
	if host == "" {
		host = versURL.Hostname()
	}
	out.Host = host

	keyPath, err := auth.GetOrCreateSSHKey(vm.VmID, client, ctx)
	if err != nil {
		return out, err
	}
	out.KeyPath = keyPath
	return out, nil
}
