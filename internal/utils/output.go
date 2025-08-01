package utils

import (
	"github.com/hdresearch/vers-cli/internal/output"
	"github.com/hdresearch/vers-cli/styles"
)

// ProgressCounter formats and prints progress messages like [1/5] Doing something...
func ProgressCounter(current, total int, action, target string, s *styles.KillStyles) {
	output.ProgressCounter(current, total, action, target, s.Progress)
}

// SuccessMessage prints a standardized success message
func SuccessMessage(message string, s *styles.KillStyles) {
	output.SuccessMessage(message, s.Success)
}

// SectionHeader prints a formatted section header
func SectionHeader(title string, s *styles.KillStyles) {
	output.SectionHeader(title, s.Progress)
}

// OperationCancelled prints a standard cancellation message
func OperationCancelled(s *styles.KillStyles) {
	output.OperationCancelled(s.NoData)
}

// NoDataFound prints a standard "no data found" message
func NoDataFound(message string, s *styles.KillStyles) {
	output.NoDataFound(message, s.NoData)
}
