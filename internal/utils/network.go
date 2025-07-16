package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/hdresearch/vers-cli/internal/auth"
	"github.com/hdresearch/vers-sdk-go"
)

// GetVmAndNodeIP retrieves VM information and the node IP from headers in a single request.
// We use the lower-level client.Get() instead of client.API.Vm.Get() because Stainless
// doesn't expose response headers through the higher-level SDK methods.
func GetVmAndNodeIP(ctx context.Context, client *vers.Client, vmID string) (vers.APIVmGetResponseData, string, error) {
	// Use the lower-level client method to get both response data AND headers
	var rawResponse *http.Response
	err := client.Get(ctx, "/api/vm/"+vmID, nil, &rawResponse)
	if err != nil {
		return vers.APIVmGetResponseData{}, "", err
	}
	defer rawResponse.Body.Close()

	// Parse the response body using the SDK types
	var response vers.APIVmGetResponse
	if err := json.NewDecoder(rawResponse.Body).Decode(&response); err != nil {
		return vers.APIVmGetResponseData{}, "", fmt.Errorf("failed to decode VM response: %w", err)
	}

	// Extract the node IP from headers
	nodeIP := rawResponse.Header.Get("X-Node-IP")

	// Use the node IP from headers or fallback to env override, and then static load balancer host
	var versHost string
	if nodeIP != "" {
		versHost = nodeIP
	} else {
		versUrl, err := auth.GetVersUrl()
		if err != nil {
			return vers.APIVmGetResponseData{}, "", fmt.Errorf("failed to get host IP: %w", err)
		}

		versHost = versUrl.Hostname()
		if os.Getenv("VERS_DEBUG") == "true" {
			fmt.Printf("[DEBUG] No node IP in headers, using fallback: %s\n", versHost)
		}
	}

	return response.Data, versHost, nil
}

func HostIsLocal(hostName string) bool {
	// RFC 5735
	const LOOPBACK = "127.0.0.1"
	return hostName == "localhost" || hostName == "0.0.0.0" || hostName == LOOPBACK
}
