package cmd

import (
	"github.com/spf13/cobra"
)

var sshUser string

// sshCmd represents the ssh command
var sshCmd = &cobra.Command{
	Use:   "ssh <machine>",
	Short: "SSH into a running machine",
	Long:  `Connect to a running Vers machine via SSH.`,
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		// machineName := args[0]
		
		// // Print SSH connection message
		// if sshUser == "" {
		// 	fmt.Printf("Connecting to machine: %s\n", machineName)
		// } else {
		// 	fmt.Printf("Connecting to machine: %s as user: %s\n", machineName, sshUser)
		// }

		// // Initialize the context for future SDK calls
		// _ = context.Background()
		
		// // Call the SDK to establish an SSH connection
		// // This is a stub implementation - adjust based on actual SDK API
		// fmt.Println("Establishing SSH connection...")
		// // Example: response, err := client.API.Machine.SSH(ctx, machineName, sshUser)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(sshCmd)

	// Define flags for the ssh command
	sshCmd.Flags().StringVarP(&sshUser, "user", "u", "", "Username for SSH connection")
} 