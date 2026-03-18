package cmd

import (
	"context"
	"fmt"

	"github.com/hdresearch/vers-cli/internal/handlers"
	pres "github.com/hdresearch/vers-cli/internal/presenters"
	"github.com/spf13/cobra"
)

var (
	resizeDiskSize int64
	resizeFormat   string
)

var resizeCmd = &cobra.Command{
	Use:   "resize [vm-id|alias]",
	Short: "Resize a VM's disk",
	Long: `Resize a VM's disk to a new size. The new size must be strictly greater than the
current size. Size is specified in MiB using the --size flag. If no VM is specified, uses the current HEAD.

Use --format json for machine-readable output.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		target := ""
		if len(args) > 0 {
			target = args[0]
		}

		apiCtx, cancel := context.WithTimeout(context.Background(), application.Timeouts.APIMedium)
		defer cancel()

		vmID, err := handlers.HandleResize(apiCtx, application, handlers.ResizeReq{
			Target:    target,
			FsSizeMib: resizeDiskSize,
		})
		if err != nil {
			return err
		}

		format := pres.ParseFormat(false, resizeFormat)
		switch format {
		case pres.FormatJSON:
			pres.PrintJSON(map[string]interface{}{"vm_id": vmID, "fs_size_mib": resizeDiskSize})
		default:
			fmt.Printf("✓ Disk resized to %d MiB for VM %s\n", resizeDiskSize, vmID)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(resizeCmd)
	resizeCmd.Flags().Int64Var(&resizeDiskSize, "size", 0, "New disk size in MiB (required, must be greater than current size)")
	resizeCmd.MarkFlagRequired("size")
	resizeCmd.Flags().StringVar(&resizeFormat, "format", "", "Output format (json)")
}
