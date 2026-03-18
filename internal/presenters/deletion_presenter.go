package presenters

import "fmt"

type SummaryResults struct {
	SuccessCount int
	FailCount    int
	Errors       []string
	ItemType     string
}

func ProgressCounter(current, total int, action, target string) {
	if total > 1 {
		fmt.Printf("[%d/%d] %s '%s'...\n", current, total, action, target)
	} else {
		fmt.Printf("%s '%s'...\n", action, target)
	}
}

func SuccessMessage(message string) {
	fmt.Printf("✓ %s\n", message)
}

func OperationCancelled() { fmt.Println("Operation cancelled") }

func NoDataFound(message string) { fmt.Println(message) }

func SectionHeader(title string) {
	fmt.Println()
	fmt.Printf("=== %s ===\n", title)
}

func PrintDeletionSummary(results SummaryResults) {
	SectionHeader("Operation Summary")
	fmt.Printf("✓ Successfully processed: %d %s\n", results.SuccessCount, results.ItemType)
	if results.FailCount > 0 {
		fmt.Printf("✗ Failed to process: %d %s\n", results.FailCount, results.ItemType)
		if len(results.Errors) > 0 {
			fmt.Println()
			fmt.Println("Error details:")
			for _, e := range results.Errors {
				fmt.Printf("  - %s\n", e)
			}
		}
	}
}
