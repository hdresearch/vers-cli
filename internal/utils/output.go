package utils

import (
	"fmt"

	"github.com/hdresearch/vers-cli/styles"
)

// ProgressCounter formats and prints progress messages like [1/5] Doing something...
func ProgressCounter(current, total int, action, target string, s *styles.KillStyles) {
	if total > 1 {
		msg := fmt.Sprintf("[%d/%d] %s '%s'...", current, total, action, target)
		fmt.Println(s.Progress.Render(msg))
	} else {
		msg := fmt.Sprintf("%s '%s'...", action, target)
		fmt.Println(s.Progress.Render(msg))
	}
}

// SuccessMessage prints a standardized success message
func SuccessMessage(message string, s *styles.KillStyles) {
	fmt.Println(s.Success.Render("SUCCESS: " + message))
}

// SectionHeader prints a formatted section header
func SectionHeader(title string, s *styles.KillStyles) {
	fmt.Println()
	fmt.Println(s.Progress.Render("=== " + title + " ==="))
}

func PrintSummary(results SummaryResults, s *styles.KillStyles) {
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

// Standard status messages
func OperationCancelled(s *styles.KillStyles) {
	fmt.Println(s.NoData.Render("Operation cancelled"))
}

func NoDataFound(message string, s *styles.KillStyles) {
	fmt.Println(s.NoData.Render(message))
}
