package output

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Writer provides efficient batched output with styling support
type Writer struct {
	builder strings.Builder
}

// New creates a new Writer instance
func New() *Writer {
	return &Writer{}
}

// --- Core methods (keep only what's actually used) ---

func (w *Writer) Write(text string) *Writer {
	w.builder.WriteString(text)
	return w
}

func (w *Writer) WriteLine(text string) *Writer {
	w.builder.WriteString(text + "\n")
	return w
}

func (w *Writer) WriteLinef(format string, args ...interface{}) *Writer {
	w.builder.WriteString(fmt.Sprintf(format, args...) + "\n")
	return w
}

func (w *Writer) WriteStyled(style lipgloss.Style, text string) *Writer {
	w.builder.WriteString(style.Render(text))
	return w
}

func (w *Writer) WriteStyledLine(style lipgloss.Style, text string) *Writer {
	w.builder.WriteString(style.Render(text) + "\n")
	return w
}

func (w *Writer) WriteStyledLinef(style lipgloss.Style, format string, args ...interface{}) *Writer {
	w.builder.WriteString(style.Render(fmt.Sprintf(format, args...)) + "\n")
	return w
}

func (w *Writer) NewLine() *Writer {
	w.builder.WriteString("\n")
	return w
}

func (w *Writer) Print() {
	fmt.Print(w.builder.String())
	w.builder.Reset()
}

func (w *Writer) String() string {
	return w.builder.String()
}

func (w *Writer) IsEmpty() bool {
	return w.builder.Len() == 0
}

// --- Standalone functions (these are what you actually use) ---

// ProgressCounter outputs progress messages like "[1/5] Doing something..."
func ProgressCounter(current, total int, action, target string, style lipgloss.Style) {
	w := New()
	if total > 1 {
		w.WriteStyledLinef(style, "[%d/%d] %s '%s'...", current, total, action, target)
	} else {
		w.WriteStyledLinef(style, "%s '%s'...", action, target)
	}
	w.Print()
}

// SuccessMessage outputs a standardized success message
func SuccessMessage(message string, style lipgloss.Style) {
	New().WriteStyledLine(style, "SUCCESS: "+message).Print()
}

// OperationCancelled outputs a standard cancellation message
func OperationCancelled(style lipgloss.Style) {
	New().WriteStyledLine(style, "Operation cancelled").Print()
}

// NoDataFound outputs a standard "no data found" message
func NoDataFound(message string, style lipgloss.Style) {
	New().WriteStyledLine(style, message).Print()
}

// ImmediateLine outputs a line immediately without buffering
func ImmediateLine(text string) {
	fmt.Println(text)
}

// ImmediateStyledLine outputs styled text with newline immediately
func ImmediateStyledLine(style lipgloss.Style, text string) {
	fmt.Println(style.Render(text))
}

// PrintDeletionSummary outputs a deletion summary
func PrintDeletionSummary(results DeletionSummaryResults, s DeletionStyleSet) {
	summary := New()
	summary.NewLine().WriteStyledLine(s.Progress, "=== Operation Summary ===")
	summary.WriteStyledLine(s.Success, fmt.Sprintf("SUCCESS: Successfully processed: %d %s", results.SuccessCount, results.ItemType))

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

// DeletionSummaryResults represents summary data for deletion operations
type DeletionSummaryResults struct {
	SuccessCount int
	FailCount    int
	Errors       []string
	ItemType     string
}

// DeletionStyleSet represents the styles needed for deletion summary output
type DeletionStyleSet struct {
	Progress lipgloss.Style
	Success  lipgloss.Style
	Error    lipgloss.Style
	Warning  lipgloss.Style
}
