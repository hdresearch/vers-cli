package output

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
	fmt.Println(s.Success.Render("  ✓ " + message))
}

// ErrorMessage prints a standardized error message
func ErrorMessage(message string, s *styles.KillStyles) {
	fmt.Println(s.Error.Render("  ❌ " + message))
}

// WarningMessage prints a standardized warning message
func WarningMessage(message string, s *styles.KillStyles) {
	fmt.Println(s.Warning.Render("  ⚠️  " + message))
}

// SectionHeader prints a formatted section header
func SectionHeader(title string, s *styles.KillStyles) {
	fmt.Println()
	fmt.Println(s.Progress.Render("=== " + title + " ==="))
}

// SummaryResults prints deletion/operation summary results
type SummaryResults struct {
	SuccessCount int
	FailCount    int
	Errors       []string
	ItemType     string // "clusters", "VMs", etc.
}

func PrintSummary(results SummaryResults, s *styles.KillStyles) {
	SectionHeader("Operation Summary", s)

	successMsg := fmt.Sprintf("✓ Successfully processed: %d %s", results.SuccessCount, results.ItemType)
	fmt.Println(s.Success.Render(successMsg))

	if results.FailCount > 0 {
		failMsg := fmt.Sprintf("❌ Failed to process: %d %s", results.FailCount, results.ItemType)
		fmt.Println(s.Error.Render(failMsg))

		if len(results.Errors) > 0 {
			fmt.Println()
			fmt.Println(s.Warning.Render("Error details:"))
			for _, error := range results.Errors {
				errorDetail := fmt.Sprintf("  • %s", error)
				fmt.Println(s.Warning.Render(errorDetail))
			}
		}
	}
}

// PrintList prints a numbered list of items with consistent formatting
func PrintList(items []string, itemType string, s *styles.KillStyles) {
	for i, item := range items {
		listItem := fmt.Sprintf("  %d. %s '%s'", i+1, itemType, item)
		fmt.Println(s.Warning.Render(listItem))
	}
}

// PrintClusterList prints a numbered list of clusters with VM counts
func PrintClusterList(clusters []ClusterInfo, s *styles.KillStyles) {
	for i, cluster := range clusters {
		listItem := fmt.Sprintf("  %d. Cluster '%s' (%d VMs)", i+1, cluster.DisplayName, cluster.VmCount)
		fmt.Println(s.Warning.Render(listItem))
	}
}

type ClusterInfo struct {
	DisplayName string
	VmCount     int
}

// Standard status messages
func OperationCancelled(s *styles.KillStyles) {
	fmt.Println(s.NoData.Render("Operation cancelled"))
}

func NoDataFound(message string, s *styles.KillStyles) {
	fmt.Println(s.NoData.Render(message))
}

func ProcessingMessage(count int, itemType string, s *styles.KillStyles) {
	msg := fmt.Sprintf("Processing %d %s...", count, itemType)
	fmt.Println(s.Progress.Render(msg))
}

// Commonly used final messages
func AllSuccessMessage(itemType string, s *styles.KillStyles) {
	fmt.Println()
	msg := fmt.Sprintf("🎉 All %s processed successfully!", itemType)
	fmt.Println(s.Success.Render(msg))
}

func HeadClearedMessage(reason string, s *styles.KillStyles) {
	fmt.Println()
	msg := fmt.Sprintf("HEAD cleared (%s)", reason)
	fmt.Println(s.NoData.Render(msg))
}
