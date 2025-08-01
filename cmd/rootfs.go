package cmd

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/hdresearch/vers-cli/internal/utils"
	"github.com/hdresearch/vers-cli/styles"
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

		// Build complete output
		var output strings.Builder

		if len(data.RootfsNames) == 0 {
			output.WriteString("No rootfs images found.\n")
		} else {
			output.WriteString("Available rootfs images:\n")
			for _, name := range data.RootfsNames {
				output.WriteString(fmt.Sprintf("- %s\n", name))
			}
		}

		// Print complete list
		fmt.Print(output.String())
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
		s := styles.NewKillStyles() // Reuse kill styles for consistency

		// Confirm deletion if not forced
		force, _ := cmd.Flags().GetBool("force")
		if !force {
			fmt.Print(s.Warning.Render(fmt.Sprintf("Are you sure you want to delete rootfs '%s'? This action cannot be undone.", rootfsName) + " [y/N]: "))
			reader := bufio.NewReader(os.Stdin)
			input, err := reader.ReadString('\n')
			if err != nil || (!strings.EqualFold(strings.TrimSpace(input), "y") && !strings.EqualFold(strings.TrimSpace(input), "yes")) {
				utils.OperationCancelled(&s)
				return nil
			}
		}

		baseCtx := context.Background()
		apiCtx, cancel := context.WithTimeout(baseCtx, 30*time.Second)
		defer cancel()

		utils.ProgressCounter(1, 1, "Deleting rootfs", rootfsName, &s)
		response, err := client.API.Rootfs.Delete(apiCtx, rootfsName)
		if err != nil {
			return fmt.Errorf("failed to delete rootfs '%s': %w", rootfsName, err)
		}
		data := response.Data

		utils.SuccessMessage(fmt.Sprintf("Successfully deleted rootfs '%s'", data.RootfsName), &s)
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
