package cmd

import (
	"context"
	"fmt"
	"time"

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
		baseCtx := context.Background()
		apiCtx, cancel := context.WithTimeout(baseCtx, 30*time.Second)
		defer cancel()

		fmt.Println("Fetching available rootfs images...")
		response, err := client.API.Rootfs.List(apiCtx)
		if err != nil {
			return fmt.Errorf("failed to list rootfs images: %w", err)
		}
		data := response.Data

		if len(data.RootfsNames) == 0 {
			fmt.Println("No rootfs images found.")
			return nil
		}

		fmt.Println("Available rootfs images:")
		for _, name := range data.RootfsNames {
			fmt.Printf("- %s\n", name)
		}

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

		// Confirm deletion if not forced
		force, _ := cmd.Flags().GetBool("force")
		if !force {
			fmt.Printf("Are you sure you want to delete rootfs '%s'? This action cannot be undone. (y/n): ", rootfsName)
			var response string
			fmt.Scanln(&response)
			if response != "y" && response != "Y" {
				fmt.Println("Deletion cancelled.")
				return nil
			}
		}

		baseCtx := context.Background()
		apiCtx, cancel := context.WithTimeout(baseCtx, 30*time.Second)
		defer cancel()

		fmt.Printf("Deleting rootfs '%s'...\n", rootfsName)
		response, err := client.API.Rootfs.Delete(apiCtx, rootfsName)
		if err != nil {
			return fmt.Errorf("failed to delete rootfs '%s': %w", rootfsName, err)
		}
		data := response.Data

		fmt.Printf("Successfully deleted rootfs '%s'.\n", data.RootfsName)
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
