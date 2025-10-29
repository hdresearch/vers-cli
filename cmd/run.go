package cmd

import (
	"context"

	"github.com/hdresearch/vers-cli/internal/handlers"
	pres "github.com/hdresearch/vers-cli/internal/presenters"
	"github.com/hdresearch/vers-cli/internal/runconfig"
	"github.com/spf13/cobra"
)

var vmAlias string

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Start a development environment",
	Long:  `Start a Vers development environment according to the configuration in vers.toml.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := runconfig.Load()
		if err != nil {
			return err
		}
		applyFlagOverrides(cmd, cfg)
		apiCtx, cancel := context.WithTimeout(context.Background(), application.Timeouts.APIMedium)
		defer cancel()
		req := handlers.RunReq{
			MemSizeMib:  cfg.Machine.MemSizeMib,
			VcpuCount:   cfg.Machine.VcpuCount,
			RootfsName:  cfg.Rootfs.Name,
			KernelName:  cfg.Kernel.Name,
			FsSizeVmMib: cfg.Machine.FsSizeVmMib,
			VMAlias:     vmAlias,
		}
		view, err := handlers.HandleRun(apiCtx, application, req)
		if err != nil {
			return err
		}
		pres.RenderRun(application, view)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(runCmd)

	// Add flags to override toml configuration
	runCmd.Flags().Int64("mem-size", 0, "Override memory size (MiB)")
	runCmd.Flags().Int64("vcpu-count", 0, "Override number of virtual CPUs")
	runCmd.Flags().String("rootfs", "", "Override rootfs name")
	runCmd.Flags().String("kernel", "", "Override kernel name")
	runCmd.Flags().Int64("fs-size-vm", 0, "Override VM filesystem size (MiB)")
	runCmd.Flags().StringVarP(&vmAlias, "vm-alias", "N", "", "Set an alias for the root VM")
}
