package cmd

import (
	"context"

	"github.com/hdresearch/vers-cli/internal/handlers"
	pres "github.com/hdresearch/vers-cli/internal/presenters"
	"github.com/hdresearch/vers-cli/internal/runconfig"
	"github.com/spf13/cobra"
)

var (
	clusterAlias string
	vmAlias      string
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run [cluster]",
	Short: "Start a development environment",
	Long:  `Start a Vers development environment according to the configuration in vers.toml.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := runconfig.Load()
		if err != nil {
			return err
		}
		applyFlagOverrides(cmd, cfg)
		apiCtx, cancel := context.WithTimeout(context.Background(), application.Timeouts.APIMedium)
		defer cancel()
		req := handlers.RunReq{
			MemSizeMib:       cfg.Machine.MemSizeMib,
			VcpuCount:        cfg.Machine.VcpuCount,
			RootfsName:       cfg.Rootfs.Name,
			KernelName:       cfg.Kernel.Name,
			FsSizeClusterMib: cfg.Machine.FsSizeClusterMib,
			FsSizeVmMib:      cfg.Machine.FsSizeVmMib,
			ClusterAlias:     clusterAlias,
			VMAlias:          vmAlias,
		}
		view, err := handlers.HandleRun(apiCtx, application, req)
		if err != nil {
			return err
		}
		pres.RenderRun(application, view)
		return nil
	},
}

// StartCluster starts a development environment according to the provided configuration
// cluster start moved into internal/handlers/run.go

func init() {
	rootCmd.AddCommand(runCmd)

	// Add flags to override toml configuration
	runCmd.Flags().Int64("mem-size", 0, "Override memory size (MiB)")
	runCmd.Flags().Int64("vcpu-count", 0, "Override number of virtual CPUs")
	runCmd.Flags().String("rootfs", "", "Override rootfs name")
	runCmd.Flags().String("kernel", "", "Override kernel name")
	runCmd.Flags().Int64("fs-size-cluster", 0, "Override cluster filesystem size (MiB)")
	runCmd.Flags().Int64("fs-size-vm", 0, "Override VM filesystem size (MiB)")
	runCmd.Flags().Int64("size-cluster", 0, "Override total cluster size (MiB)")
	runCmd.Flags().StringVarP(&clusterAlias, "cluster-alias", "n", "", "Set an alias for the cluster")
	runCmd.Flags().StringVarP(&vmAlias, "vm-alias", "N", "", "Set an alias for the root VM")
}
