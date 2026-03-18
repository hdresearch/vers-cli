package cmd

import (
	"context"

	"github.com/hdresearch/vers-cli/internal/handlers"
	pres "github.com/hdresearch/vers-cli/internal/presenters"
	"github.com/hdresearch/vers-cli/internal/runconfig"
	"github.com/spf13/cobra"
)

var (
	vmAlias   string
	runFormat string
	runWait   bool
)

// runCmd represents the run command
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "Start a development environment",
	Long: `Start a Vers development environment according to the configuration in vers.toml.

Use --format json for machine-readable output.
Use --wait to block until the VM is running.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := runconfig.Load()
		if err != nil {
			return err
		}
		applyFlagOverrides(cmd, cfg)
		apiCtx, cancel := context.WithTimeout(context.Background(), application.Timeouts.APILong)
		defer cancel()
		req := handlers.RunReq{
			MemSizeMib:  cfg.Machine.MemSizeMib,
			VcpuCount:   cfg.Machine.VcpuCount,
			RootfsName:  cfg.Rootfs.Name,
			KernelName:  cfg.Kernel.Name,
			FsSizeVmMib: cfg.Machine.FsSizeVmMib,
			VMAlias:     vmAlias,
			Wait:        runWait,
		}
		view, err := handlers.HandleRun(apiCtx, application, req)
		if err != nil {
			return err
		}

		format := pres.ParseFormat(false, runFormat)
		switch format {
		case pres.FormatJSON:
			pres.PrintJSON(view)
		default:
			pres.RenderRun(application, view)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(runCmd)

	runCmd.Flags().Int64("mem-size", 0, "Override memory size (MiB)")
	runCmd.Flags().Int64("vcpu-count", 0, "Override number of virtual CPUs")
	runCmd.Flags().String("rootfs", "", "Override rootfs name")
	runCmd.Flags().String("kernel", "", "Override kernel name")
	runCmd.Flags().Int64("fs-size-vm", 0, "Override VM filesystem size (MiB)")
	runCmd.Flags().StringVarP(&vmAlias, "vm-alias", "N", "", "Set an alias for the root VM")
	runCmd.Flags().StringVar(&runFormat, "format", "", "Output format (json)")
	runCmd.Flags().BoolVar(&runWait, "wait", false, "Wait until VM is running before returning")
}
