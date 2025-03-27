package cmd

import (
	"fmt"

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
		machineName := args[0]
		
		// Print SSH connection message
		if sshUser == "" {
			fmt.Printf("Connecting to machine: %s\n", machineName)
		} else {
			fmt.Printf("Connecting to machine: %s as user: %s\n", machineName, sshUser)
		}

		// Call the SDK to establish an SSH connection
		if err := client.SSH(machineName, sshUser); err != nil {
			return fmt.Errorf("failed to connect to machine: %w", err)
		}

		return nil
	},
}

func init() {
	rootCmd.AddCommand(sshCmd)

	// Define flags for the ssh command
	sshCmd.Flags().StringVarP(&sshUser, "user", "u", "", "Username for SSH connection")
} 