package output

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Writer provides a fluent interface for building and outputting text
// It replaces the repetitive strings.Builder pattern throughout the codebase
type Writer struct {
	builder strings.Builder
}

// New creates a new Writer instance
func New() *Writer {
	return &Writer{}
}

// Write adds text to the buffer
func (w *Writer) Write(text string) *Writer {
	w.builder.WriteString(text)
	return w
}

// Writef adds formatted text to the buffer
func (w *Writer) Writef(format string, args ...interface{}) *Writer {
	w.builder.WriteString(fmt.Sprintf(format, args...))
	return w
}

// WriteLine adds text with a newline to the buffer
func (w *Writer) WriteLine(text string) *Writer {
	w.builder.WriteString(text + "\n")
	return w
}

// WriteLinef adds formatted text with a newline to the buffer
func (w *Writer) WriteLinef(format string, args ...interface{}) *Writer {
	w.builder.WriteString(fmt.Sprintf(format, args...) + "\n")
	return w
}

// WriteStyled adds styled text to the buffer
func (w *Writer) WriteStyled(style lipgloss.Style, text string) *Writer {
	w.builder.WriteString(style.Render(text))
	return w
}

// WriteStyledLine adds styled text with a newline to the buffer
func (w *Writer) WriteStyledLine(style lipgloss.Style, text string) *Writer {
	w.builder.WriteString(style.Render(text) + "\n")
	return w
}

// WriteStyledLinef adds formatted styled text with a newline to the buffer
func (w *Writer) WriteStyledLinef(style lipgloss.Style, format string, args ...interface{}) *Writer {
	w.builder.WriteString(style.Render(fmt.Sprintf(format, args...)) + "\n")
	return w
}

// NewLine adds a newline to the buffer
func (w *Writer) NewLine() *Writer {
	w.builder.WriteString("\n")
	return w
}

// Print outputs the buffer contents and resets the buffer for reuse
func (w *Writer) Print() {
	fmt.Print(w.builder.String())
	w.builder.Reset()
}

// PrintTo outputs the buffer contents without resetting (useful for testing)
func (w *Writer) String() string {
	return w.builder.String()
}

// Reset clears the buffer
func (w *Writer) Reset() *Writer {
	w.builder.Reset()
	return w
}

// Len returns the length of the current buffer
func (w *Writer) Len() int {
	return w.builder.Len()
}

// IsEmpty returns true if the buffer is empty
func (w *Writer) IsEmpty() bool {
	return w.builder.Len() == 0
}

// --- Convenience methods for common patterns ---

// WriteIf conditionally writes text based on a condition
func (w *Writer) WriteIf(condition bool, text string) *Writer {
	if condition {
		w.builder.WriteString(text)
	}
	return w
}

// WriteLineIf conditionally writes a line based on a condition
func (w *Writer) WriteLineIf(condition bool, text string) *Writer {
	if condition {
		w.builder.WriteString(text + "\n")
	}
	return w
}

// WriteStyledIf conditionally writes styled text based on a condition
func (w *Writer) WriteStyledIf(condition bool, style lipgloss.Style, text string) *Writer {
	if condition {
		w.builder.WriteString(style.Render(text))
	}
	return w
}

// --- Domain-specific convenience methods ---
// Based on actual patterns found in the codebase

// WriteVMInfo writes VM information in a consistent format (like in branch.go)
func (w *Writer) WriteVMInfo(label, value string, labelStyle, valueStyle lipgloss.Style) *Writer {
	w.WriteStyled(labelStyle, label)
	w.Write(": ")
	w.WriteStyledLine(valueStyle, value)
	return w
}

// WriteListItemStyled writes a styled list item (common in branch.go, status.go)
func (w *Writer) WriteListItemStyled(itemStyle lipgloss.Style, content string) *Writer {
	w.WriteStyledLine(itemStyle, content)
	return w
}

// WriteSuccessMessage writes a success message with consistent formatting
func (w *Writer) WriteSuccessMessage(style lipgloss.Style, message string) *Writer {
	w.WriteStyledLine(style, "✓ "+message)
	return w
}

// WriteWarningMessage writes a warning message with consistent formatting
func (w *Writer) WriteWarningMessage(style lipgloss.Style, message string) *Writer {
	w.WriteStyledLine(style, "⚠️  "+message)
	return w
}

// WriteProgressMessage writes a progress message
func (w *Writer) WriteProgressMessage(style lipgloss.Style, message string) *Writer {
	w.WriteStyledLine(style, message)
	return w
}

// WriteTip writes a tip message (common pattern)
func (w *Writer) WriteTip(style lipgloss.Style, message string) *Writer {
	w.WriteStyledLine(style, message)
	return w
}

// WriteProgressCounter writes progress messages like "[1/5] Doing something..."
// Migrated from utils.ProgressCounter
func (w *Writer) WriteProgressCounter(current, total int, action, target string, style lipgloss.Style) *Writer {
	var msg string
	if total > 1 {
		msg = fmt.Sprintf("[%d/%d] %s '%s'...", current, total, action, target)
	} else {
		msg = fmt.Sprintf("%s '%s'...", action, target)
	}
	w.WriteStyledLine(style, msg)
	return w
}

// WriteStandardSuccess writes a standardized "SUCCESS: message" format
// Migrated from utils.SuccessMessage
func (w *Writer) WriteStandardSuccess(message string, style lipgloss.Style) *Writer {
	w.WriteStyledLine(style, "SUCCESS: "+message)
	return w
}

// WriteSectionHeader writes a formatted section header
// Migrated from utils.SectionHeader
func (w *Writer) WriteSectionHeader(title string, style lipgloss.Style) *Writer {
	w.NewLine()
	w.WriteStyledLine(style, "=== "+title+" ===")
	return w
}

// WriteOperationCancelled writes a standard cancellation message
func (w *Writer) WriteOperationCancelled(style lipgloss.Style) *Writer {
	w.WriteStyledLine(style, "Operation cancelled")
	return w
}

// WriteNoDataFound writes a standard "no data found" message
func (w *Writer) WriteNoDataFound(message string, style lipgloss.Style) *Writer {
	w.WriteStyledLine(style, message)
	return w
}

// --- Output phase methods ---
// Based on the patterns I see in your code

// SetupPhase creates a new writer for setup output
func SetupPhase() *Writer {
	return New()
}

// DetailsPhase creates a new writer for details output
func DetailsPhase() *Writer {
	return New()
}

// FinalPhase creates a new writer for final output
func FinalPhase() *Writer {
	return New()
}

// --- Utility functions ---

// Immediate outputs text immediately without buffering
func Immediate(text string) {
	fmt.Print(text)
}

// ImmediateLine outputs a line immediately without buffering
func ImmediateLine(text string) {
	fmt.Println(text)
}

// ImmediateStyled outputs styled text immediately
func ImmediateStyled(style lipgloss.Style, text string) {
	fmt.Print(style.Render(text))
}

// ImmediateStyledLine outputs styled text with newline immediately
func ImmediateStyledLine(style lipgloss.Style, text string) {
	fmt.Println(style.Render(text))
}

// --- Convenient standalone functions (migrate from utils) ---

// ProgressCounter outputs progress messages immediately like "[1/5] Doing something..."
// This replaces utils.ProgressCounter for cases where immediate output is needed
func ProgressCounter(current, total int, action, target string, style lipgloss.Style) {
	New().WriteProgressCounter(current, total, action, target, style).Print()
}

// SuccessMessage outputs a standardized success message immediately
// This replaces utils.SuccessMessage for cases where immediate output is needed
func SuccessMessage(message string, style lipgloss.Style) {
	New().WriteStandardSuccess(message, style).Print()
}

// SectionHeader outputs a formatted section header immediately
// This replaces utils.SectionHeader for cases where immediate output is needed
func SectionHeader(title string, style lipgloss.Style) {
	New().WriteSectionHeader(title, style).Print()
}

// OperationCancelled outputs a standard cancellation message immediately
func OperationCancelled(style lipgloss.Style) {
	New().WriteOperationCancelled(style).Print()
}

// NoDataFound outputs a standard "no data found" message immediately
func NoDataFound(message string, style lipgloss.Style) {
	New().WriteNoDataFound(message, style).Print()
}

// PrintDeletionSummary outputs a deletion summary immediately
// This replaces the utils.PrintDeletionSummary function
func PrintDeletionSummary(results DeletionSummaryResults, s DeletionStyleSet) {
	summary := New()

	summary.WriteSectionHeader("Operation Summary", s.Progress).
		WriteStandardSuccess(fmt.Sprintf("Successfully processed: %d %s", results.SuccessCount, results.ItemType), s.Success)

	if results.FailCount > 0 {
		summary.WriteStyledLine(s.Error, fmt.Sprintf("Failed to process: %d %s", results.FailCount, results.ItemType))

		if len(results.Errors) > 0 {
			summary.NewLine().
				WriteStyledLine(s.Warning, "Error details:")
			for _, error := range results.Errors {
				summary.WriteStyledLine(s.Warning, fmt.Sprintf("  - %s", error))
			}
		}
	}

	summary.Print()
}

// DeletionSummaryResults represents summary data for deletion operations
type DeletionSummaryResults struct {
	SuccessCount int
	FailCount    int
	Errors       []string
	ItemType     string
}

// DeletionStyleSet represents the styles needed for deletion summary output
// This should be satisfied by styles.KillStyles and similar style structs
type DeletionStyleSet struct {
	Progress lipgloss.Style
	Success  lipgloss.Style
	Error    lipgloss.Style
	Warning  lipgloss.Style
}
