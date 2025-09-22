package presenters

import (
	"fmt"
	"github.com/hdresearch/vers-cli/styles"
)

type SummaryResults struct {
	SuccessCount int
	FailCount    int
	Errors       []string
	ItemType     string
}

func ProgressCounter(current, total int, action, target string, s *styles.KillStyles) {
	if total > 1 {
		fmt.Println(s.Progress.Render(fmt.Sprintf("[%d/%d] %s '%s'...", current, total, action, target)))
	} else {
		fmt.Println(s.Progress.Render(fmt.Sprintf("%s '%s'...", action, target)))
	}
}

func SuccessMessage(message string, s *styles.KillStyles) {
	fmt.Println(s.Success.Render("SUCCESS: " + message))
}

func OperationCancelled(s *styles.KillStyles) { fmt.Println(s.NoData.Render("Operation cancelled")) }

func NoDataFound(message string, s *styles.KillStyles) { fmt.Println(s.NoData.Render(message)) }

func SectionHeader(title string, s *styles.KillStyles) {
	fmt.Println()
	fmt.Println(s.Progress.Render("=== " + title + " ==="))
}

func PrintDeletionSummary(results SummaryResults, s *styles.KillStyles) {
	SectionHeader("Operation Summary", s)
	fmt.Println(s.Success.Render(fmt.Sprintf("Successfully processed: %d %s", results.SuccessCount, results.ItemType)))
	if results.FailCount > 0 {
		fmt.Println(s.Error.Render(fmt.Sprintf("Failed to process: %d %s", results.FailCount, results.ItemType)))
		if len(results.Errors) > 0 {
			fmt.Println()
			fmt.Println(s.Warning.Render("Error details:"))
			for _, e := range results.Errors {
				fmt.Println(s.Warning.Render("  - " + e))
			}
		}
	}
}
