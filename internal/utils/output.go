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
	fmt.Println("\n" + s.Progress.Render("=== "+title+" ==="))
}

// Standard status messages
func OperationCancelled(s *styles.KillStyles) {
	fmt.Println(s.NoData.Render("Operation cancelled"))
}

func NoDataFound(message string, s *styles.KillStyles) {
	fmt.Println(s.NoData.Render(message))
}
