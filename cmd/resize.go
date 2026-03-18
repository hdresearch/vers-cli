package cmd

import (
	"context"
	"fmt"

	"github.com/hdresearch/vers-cli/internal/handlers"
	"github.com/spf13/cobra"
)

var resizeDiskSize int64

var resizeCmd = &cobra.Command{
	Use:   "resize [vm-id|alias]",
	Short: "Resize a VM's disk",
	Long: `Resize a VM's disk to a new size. The new size must be strictly greater than the
current size. Size is specified in MiB using the --size flag. If no VM is specified, uses the current HEAD.`,
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
		fmt.Printf("✓ Disk resized to %d MiB for VM %s\n", resizeDiskSize, vmID)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(resizeCmd)
	resizeCmd.Flags().Int64Var(&resizeDiskSize, "size", 0, "New disk size in MiB (required, must be greater than current size)")
	resizeCmd.MarkFlagRequired("size")
}
