package history

import (
	"context"
	"fmt"

	vers "github.com/hdresearch/vers-sdk-go"
)

type CommitEntry struct {
	ID               string   `json:"ID"`
	Message          string   `json:"Message"`
	Timestamp        int64    `json:"Timestamp"`
	Tags             []string `json:"Tags"`
	Author           string   `json:"Author"`
	VMID             string   `json:"VMID"`
	Alias            string   `json:"Alias"`
	HostArchitecture string   `json:"HostArchitecture"`
}

type commitResponse struct {
	Commits []CommitEntry `json:"commits"`
}

// GetCommits fetches commit entries for a VM via the low-level client.Get to access custom endpoint shape.
func GetCommits(ctx context.Context, client *vers.Client, vmID string) ([]CommitEntry, error) {
	var resp commitResponse
	if err := client.Get(ctx, fmt.Sprintf("/api/vm/%s/commits", vmID), nil, &resp); err != nil {
		return nil, err
	}
	return resp.Commits, nil
}
