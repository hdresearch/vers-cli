package utils

import (
	"context"
	"fmt"
	"strings"

	"github.com/hdresearch/vers-cli/styles"
	vers "github.com/hdresearch/vers-sdk-go"
)

// SummaryResults for deletion operations
type SummaryResults struct {
	SuccessCount int
	FailCount    int
	Errors       []string
	ItemType     string
}

// PrintDeletionSummary prints results for multiple target deletions
func PrintDeletionSummary(results SummaryResults, s *styles.KillStyles) {
	SectionHeader("Operation Summary", s)

	successMsg := fmt.Sprintf("Successfully processed: %d %s", results.SuccessCount, results.ItemType)
	fmt.Println(s.Success.Render(successMsg))

	if results.FailCount > 0 {
		failMsg := fmt.Sprintf("Failed to process: %d %s", results.FailCount, results.ItemType)
		fmt.Println(s.Error.Render(failMsg))

		if len(results.Errors) > 0 {
			fmt.Println()
			fmt.Println(s.Warning.Render("Error details:"))
			for _, error := range results.Errors {
				errorDetail := fmt.Sprintf("  - %s", error)
				fmt.Println(s.Warning.Render(errorDetail))
			}
		}
	}
}

// ValidateResourcesExist validates that all resources exist via API calls
// Generic function that works for both VMs and clusters
func ValidateResourcesExist(ctx context.Context, client *vers.Client, resourceIDs []string, resourceType string, isCluster bool) error {
	var invalidResources []string

	for _, resourceID := range resourceIDs {
		var err error
		if isCluster {
			_, err = client.API.Cluster.Get(ctx, resourceID)
		} else {
			_, err = client.API.Vm.Get(ctx, resourceID)
		}

		if err != nil {
			invalidResources = append(invalidResources, resourceID)
		}
	}

	if len(invalidResources) > 0 {
		if len(invalidResources) == 1 {
			return fmt.Errorf("%s '%s' not found", resourceType, invalidResources[0])
		}
		return fmt.Errorf("%ss not found: %s", resourceType, strings.Join(invalidResources, ", "))
	}

	return nil
}
