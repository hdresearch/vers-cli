package utils

import (
	"fmt"
	"strings"

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
	var output strings.Builder

	output.WriteString("\n") // Replace SectionHeader spacing
	output.WriteString(s.Progress.Render("=== Operation Summary ===") + "\n")

	successMsg := fmt.Sprintf("Successfully processed: %d %s", results.SuccessCount, results.ItemType)
	output.WriteString(s.Success.Render(successMsg) + "\n")

	if results.FailCount > 0 {
		failMsg := fmt.Sprintf("Failed to process: %d %s", results.FailCount, results.ItemType)
		output.WriteString(s.Error.Render(failMsg) + "\n")

		if len(results.Errors) > 0 {
			output.WriteString("\n")
			output.WriteString(s.Warning.Render("Error details:") + "\n")
			for _, error := range results.Errors {
				errorDetail := fmt.Sprintf("  - %s", error)
				output.WriteString(s.Warning.Render(errorDetail) + "\n")
			}
		}
	}

	fmt.Print(output.String())
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
