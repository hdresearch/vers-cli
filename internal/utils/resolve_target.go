package utils

import (
	"context"
	"fmt"
	"strings"

	vers "github.com/hdresearch/vers-sdk-go"
)

// TargetResult holds the result of resolving a VM target identifier.
type TargetResult struct {
	Ident    string // The resolved VM identifier (ID or alias)
	UsedHEAD bool   // Whether HEAD was used as fallback
	HeadID   string // The HEAD VM ID (only set if UsedHEAD is true)
}

// ResolveTarget resolves a VM target string. If target is empty, falls back
// to the current HEAD VM. This centralises the pattern used across handlers.
func ResolveTarget(target string) (TargetResult, error) {
	if strings.TrimSpace(target) != "" {
		return TargetResult{Ident: target}, nil
	}

	headID, err := GetCurrentHeadVM()
	if err != nil {
		return TargetResult{}, fmt.Errorf("no VM ID provided and %w", err)
	}

	return TargetResult{
		Ident:    headID,
		UsedHEAD: true,
		HeadID:   headID,
	}, nil
}

// ResolvedVM holds a verified VM identity.
type ResolvedVM struct {
	ID       string // Verified VM ID
	UsedHEAD bool   // Whether HEAD was used as fallback
}

// ResolveTargetVM resolves a target string to a verified VM ID.
// If target is empty, uses HEAD. Then verifies the VM exists via the API.
func ResolveTargetVM(ctx context.Context, client *vers.Client, target string) (ResolvedVM, error) {
	t, err := ResolveTarget(target)
	if err != nil {
		return ResolvedVM{}, err
	}

	info, err := ResolveVMIdentifier(ctx, client, t.Ident)
	if err != nil {
		return ResolvedVM{}, err
	}

	return ResolvedVM{
		ID:       info.ID,
		UsedHEAD: t.UsedHEAD,
	}, nil
}
