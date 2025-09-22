package cmd

import (
	"context"
	"fmt"

	"github.com/hdresearch/vers-cli/internal/handlers"
	"github.com/spf13/cobra"
)

// rootfsCmd represents the rootfs command
var rootfsCmd = &cobra.Command{
	Use:   "rootfs",
	Short: "Manage rootfs images",
	Long:  `Commands to list and delete rootfs images on the Vers platform.`,
}

// rootfsListCmd represents the rootfs list command
var rootfsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available rootfs images",
	Long:  `List all available rootfs images on the Vers platform.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		apiCtx, cancel := context.WithTimeout(context.Background(), application.Timeouts.APIMedium)
		defer cancel()
		fmt.Println("Fetching available rootfs images...")
		view, err := handlers.HandleRootfsList(apiCtx, application, handlers.RootfsListReq{})
		if err != nil {
			return fmt.Errorf("failed to list rootfs images: %w", err)
		}
		handlers.RenderRootfsList(application, view)
		return nil
	},
}

// rootfsDeleteCmd represents the rootfs delete command
var rootfsDeleteCmd = &cobra.Command{
	Use:   "delete [rootfs-name]",
	Short: "Delete a rootfs image",
	Long:  `Delete a specific rootfs image from the Vers platform.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		rootfsName := args[0]
		force, _ := cmd.Flags().GetBool("force")
		apiCtx, cancel := context.WithTimeout(context.Background(), application.Timeouts.APIMedium)
		defer cancel()
		view, err := handlers.HandleRootfsDelete(apiCtx, application, handlers.RootfsDeleteReq{Name: rootfsName, Force: force})
		if err != nil {
			return err
		}
		handlers.RenderRootfsDelete(application, view)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(rootfsCmd)
	rootfsCmd.AddCommand(rootfsListCmd)
	rootfsCmd.AddCommand(rootfsDeleteCmd)

	// Add flags for delete command
	rootfsDeleteCmd.Flags().BoolP("force", "f", false, "Force deletion without confirmation")
}
