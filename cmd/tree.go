package cmd

import (
	"context"
	"fmt"

	"github.com/hdresearch/vers-cli/internal/handlers"
	pres "github.com/hdresearch/vers-cli/internal/presenters"
	vers "github.com/hdresearch/vers-sdk-go"
	"github.com/spf13/cobra"
)

var treeCmd = &cobra.Command{
	Use:   "tree [cluster-id|cluster-alias]",
	Short: "Print the tree of the cluster",
	Long:  `Print a visual tree representation of the cluster and its VMs. If no cluster ID or alias is provided, uses the cluster from current HEAD.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		apiCtx, cancel := context.WithTimeout(context.Background(), application.Timeouts.APIMedium)
		defer cancel()
		var clusterArg string
		if len(args) > 0 {
			clusterArg = args[0]
		}
		clusterAny, head, err := handlers.HandleTree(apiCtx, application, handlers.TreeReq{ClusterIdentifier: clusterArg})
		if err != nil {
			return fmt.Errorf("failed to resolve cluster: %w", err)
		}
		var finding string
		if clusterArg == "" && head != "" {
			finding = fmt.Sprintf("Finding cluster for current HEAD VM: %s", head)
		}
		cluster, ok := clusterAny.(vers.APIClusterGetResponseData)
		if !ok {
			return fmt.Errorf("unexpected cluster payload type")
		}
		return pres.RenderTreeController(cluster, head, finding)
	},
}

func init() {
	rootCmd.AddCommand(treeCmd)
}
