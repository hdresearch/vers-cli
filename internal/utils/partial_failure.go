package utils

import (
	"fmt"
	"strings"

	"github.com/hdresearch/vers-cli/styles"
	"github.com/hdresearch/vers-sdk-go"
)

// HandleVmDeleteErrors processes VM deletion results and prints error messages
// Returns true if there were partial failures, false if completely successful
func HandleVmDeleteErrors(result *vers.APIVmDeleteResponse, s *styles.KillStyles) bool {
	hasErrors := len(result.Data.Errors) > 0

	if !hasErrors {
		return false
	}

	fmt.Println(s.Warning.Render("One or more VMs failed to delete:"))
	for _, error := range result.Data.Errors {
		fmt.Printf(s.Warning.Render("  • %s: %s\n"), error.ID, error.Error)
	}

	return true
}

// GetClusterDeleteErrorSummary returns a summary string of cluster deletion errors
// for use in bulk operations. Returns empty string if no errors.
func GetClusterDeleteErrorSummary(result *vers.APIClusterDeleteResponse) string {
	hasErrors := len(result.Data.Vms.Errors) > 0 || result.Data.FsError != ""

	if !hasErrors {
		return ""
	}

	errorDetails := []string{}

	if result.Data.FsError != "" {
		errorDetails = append(errorDetails, result.Data.FsError)
	}

	for _, vmError := range result.Data.Vms.Errors {
		errorDetails = append(errorDetails, fmt.Sprintf("%s: %s", vmError.ID, vmError.Error))
	}

	return strings.Join(errorDetails, "; ")
}
