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
	outputResults := output.DeletionSummaryResults{
		SuccessCount: results.SuccessCount,
		FailCount:    results.FailCount,
		Errors:       results.Errors,
		ItemType:     results.ItemType,
	}

	styleSet := output.DeletionStyleSet{
		Progress: s.Progress,
		Success:  s.Success,
		Error:    s.Error,
		Warning:  s.Warning,
	}

	output.PrintDeletionSummary(outputResults, styleSet)
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
		output.ImmediateStyledLine(s.Error, failMsg)
		return nil, err
	}

	SuccessMessage("Deleted successfully", s)
	return deletedIDs, nil
}
