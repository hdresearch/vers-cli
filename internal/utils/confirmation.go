package utils

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/hdresearch/vers-cli/styles"
)

// AskConfirmation asks for a y/N confirmation with optional styling
func AskConfirmation(prompt ...string) bool {
	// Default prompt if none provided
	confirmPrompt := "Are you sure you want to proceed? [y/N]: "
	if len(prompt) > 0 {
		confirmPrompt = prompt[0] + " [y/N]: "
	}

	fmt.Printf(confirmPrompt)

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	input = strings.TrimSpace(input)

	return strings.EqualFold(input, "y") || strings.EqualFold(input, "yes")
}

// AskSpecialConfirmation asks for an exact text match confirmation
func AskSpecialConfirmation(requiredText string, s *styles.KillStyles) bool {
	prompt := fmt.Sprintf("Type '%s' to confirm: ", requiredText)
	fmt.Printf(s.Warning.Render(prompt))

	reader := bufio.NewReader(os.Stdin)
	input, err := reader.ReadString('\n')
	if err != nil {
		fmt.Println(s.NoData.Render("Error reading input"))
		return false
	}
	input = strings.TrimSpace(input)

	return input == requiredText
}

// ConfirmDeletion shows a deletion warning and asks for confirmation
func ConfirmDeletion(itemType, itemName string, s *styles.KillStyles) bool {
	msg := fmt.Sprintf("Warning: You are about to delete %s '%s'", itemType, itemName)
	fmt.Println(s.Warning.Render(msg))
	return AskConfirmation()
}

// ConfirmClusterDeletion shows a cluster deletion warning with VM count
func ConfirmClusterDeletion(clusterName string, vmCount int, s *styles.KillStyles) bool {
	msg := fmt.Sprintf("Warning: You are about to delete cluster '%s' containing %d VMs", clusterName, vmCount)
	fmt.Println(s.Warning.Render(msg))
	return AskConfirmation()
}

// ConfirmBatchDeletion shows a batch deletion warning
func ConfirmBatchDeletion(count int, itemType string, items []string, s *styles.KillStyles) bool {
	msg := fmt.Sprintf("Warning: You are about to delete %d %ss:", count, itemType)
	fmt.Println(s.Warning.Render(msg))
	fmt.Println()

	for i, item := range items {
		listItem := fmt.Sprintf("  %d. %s '%s'", i+1, strings.Title(itemType), item)
		fmt.Println(s.Warning.Render(listItem))
	}

	fmt.Println()
	irreversibleMsg := fmt.Sprintf("This action is IRREVERSIBLE and will delete ALL specified %ss!", itemType)
	fmt.Println(s.Warning.Render(irreversibleMsg))

	return AskConfirmation()
}
