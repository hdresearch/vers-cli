package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var treeCmd = &cobra.Command{
	Use:   "tree [cluster-id|cluster-alias]",
	Short: "Print the tree of the cluster (DEPRECATED)",
	Long:  `Tree command is deprecated - cluster concept has been removed from the API. Use 'vers status' to view VMs.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Println("Tree command is not available - cluster concept has been removed from the API")
		fmt.Println("Use 'vers status' to view all VMs")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(treeCmd)
}
