package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var force bool

// killCmd represents the kill command
var killCmd = &cobra.Command{
	Use:   "kill <branch|cluster>",
	Short: "Forcefully terminate a branch or cluster",
	Long:  `Forcefully terminate a branch, cluster, or VM in the Vers environment.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		targetName := args[0]
		
		if force {
			fmt.Printf("Forcefully terminating: %s\n", targetName)
		} else {
			fmt.Printf("Terminating: %s\n", targetName)
		}

		// Call the SDK to kill the branch
		if err := client.KillBranch(targetName, force); err != nil {
			return fmt.Errorf("kill operation failed: %w", err)
		}

		fmt.Printf("Successfully terminated: %s\n", targetName)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(killCmd)

	// Define flags for the kill command
	killCmd.Flags().BoolVarP(&force, "force", "f", false, "Force termination without confirmation")
} 