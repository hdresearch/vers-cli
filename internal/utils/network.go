package utils

import (
	"context"
	"fmt"
	"os"

	"github.com/hdresearch/vers-cli/internal/auth"
	"github.com/hdresearch/vers-sdk-go"
)

// GetVmAndNodeIP retrieves VM information and the node IP from headers in a single request.
// We use the lower-level client.Get() instead of client.Vm.List() because we need response headers.
func GetVmAndNodeIP(ctx context.Context, client *vers.Client, vmID string) (*vers.Vm, string, error) {
	// For now, use List() to get VM data
	// TODO: Implement proper Get with headers if needed
	vms, err := client.Vm.List(ctx)
	if err != nil {
		return nil, "", err
	}

	var targetVM *vers.Vm
	for _, vm := range *vms {
		if vm.VmID == vmID {
			targetVM = &vm
			break
		}
	}

	if targetVM == nil {
		return nil, "", fmt.Errorf("VM %s not found", vmID)
	}

	// Get the host from VERS_URL
	versUrl, err := auth.GetVersUrl()
	if err != nil {
		return nil, "", fmt.Errorf("failed to get host: %w", err)
	}

	versHost := versUrl.Hostname()
	if os.Getenv("VERS_DEBUG") == "true" {
		fmt.Printf("[DEBUG] Using host: %s\n", versHost)
	}

	return targetVM, versHost, nil
}

func IsHostLocal(hostName string) bool {
	// RFC 5735
	const LOOPBACK = "127.0.0.1"
	return hostName == "localhost" || hostName == "0.0.0.0" || hostName == LOOPBACK
}
