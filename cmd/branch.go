package cmd

import (
	"context"
	"fmt"
	"time"

	vers "github.com/hdresearch/vers-sdk-go"
	"github.com/spf13/cobra"
)

var fromBranch string

// branchCmd represents the branch command
var branchCmd = &cobra.Command{
	Use:   "branch <name>",
	Short: "Branch a machine",
	Long:  `Branch the state of a given machine.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		vmName := args[0]
		
		fmt.Printf("Creating branch of vm '%s' \n", vmName)


		baseCtx := context.Background()
		client = vers.NewClient()
		apiCtx, cancel := context.WithTimeout(baseCtx, 30*time.Second)
		defer cancel() 

		branchParams := vers.APIVmNewBranchParams {
			Body: map[string]interface{}{},
		}

		fmt.Println("Creating branch...")
		branchInfo, err := client.API.Vm.NewBranch(apiCtx, vmName, branchParams)

		if err != nil {
			return fmt.Errorf("failed to create branch of vm '%s': %w", vmName, err)
		}
		fmt.Printf("Branch created successfully with ID: %s\n", branchInfo.ID)
		fmt.Printf("Branch IP address: %s\n", branchInfo.IPAddress)
		fmt.Printf("Branch state: %s\n", branchInfo.State)
		
		return nil
	},
}

func init() {
	rootCmd.AddCommand(branchCmd)

	// Define flags for the branch command
	branchCmd.Flags().StringVarP(&fromBranch, "from", "f", "", "Source branch or commit (default: current state)")
} 