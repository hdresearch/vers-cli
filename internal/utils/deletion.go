package utils

import (
	"fmt"

	"github.com/hdresearch/vers-cli/internal/output"
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
	summary := output.New()

	summary.NewLine().
		WriteStyledLine(s.Progress, "=== Operation Summary ===").
		WriteStyledLine(s.Success, fmt.Sprintf("SUCCESS: Successfully processed: %d %s", results.SuccessCount, results.ItemType))

	if results.FailCount > 0 {
		summary.WriteStyledLine(s.Error, fmt.Sprintf("Failed to process: %d %s", results.FailCount, results.ItemType))

		if len(results.Errors) > 0 {
			summary.NewLine().WriteStyledLine(s.Warning, "Error details:")
			for _, error := range results.Errors {
				summary.WriteStyledLine(s.Warning, fmt.Sprintf("  - %s", error))
			}
		}
	}

	summary.Print()
}

// HandleDeletionResult displays progress, performs deletion, and handles the result
func HandleDeletionResult(currentIndex, totalCount int, action, displayName string, deletionFunc func() ([]string, error), s *styles.KillStyles) ([]string, error) {
	// Show progress
	output.ProgressCounter(currentIndex, totalCount, action, displayName, s.Progress)

	// Perform the deletion
	deletedIDs, err := deletionFunc()
	if err != nil {
		failMsg := fmt.Sprintf("FAILED: %s", err.Error())
		output.ImmediateStyledLine(s.Error, failMsg)
		return nil, err
	}

	output.SuccessMessage("Deleted successfully", s.Success)
	return deletedIDs, nil
}
