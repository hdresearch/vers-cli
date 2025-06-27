package utils

import (
	"fmt"

	"github.com/hdresearch/vers-cli/styles"
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

// HandleDeletionResult displays progress, performs deletion, and handles the result
// This is the common pattern used by both VM and cluster processors
func HandleDeletionResult(currentIndex, totalCount int, action, displayName string, deletionFunc func() ([]string, error), s *styles.KillStyles) ([]string, error) {
	// Show progress
	ProgressCounter(currentIndex, totalCount, action, displayName, s)

	// Perform the deletion
	deletedIDs, err := deletionFunc()
	if err != nil {
		failMsg := fmt.Sprintf("FAILED: %s", err.Error())
		fmt.Println(s.Error.Render(failMsg))
		return nil, err
	}

	SuccessMessage("Deleted successfully", s)
	return deletedIDs, nil
}
