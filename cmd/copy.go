package cmd

import (
	"context"
	"time"

	"github.com/hdresearch/vers-cli/internal/handlers"
	pres "github.com/hdresearch/vers-cli/internal/presenters"
	"github.com/spf13/cobra"
)

var copyTimeout int

// copyCmd represents the copy command
var copyCmd = &cobra.Command{
	Use:   "copy [vm-id|alias] <source> <destination>",
	Short: "Copy files to/from a VM using SCP",
	Long: `Copy files between your local machine and a Vers VM using SCP.
	
Examples:
  vers copy vm-123 ./local-file.txt /remote/path/
  vers copy vm-123 /remote/path/file.txt ./local-file.txt
  vers copy ./local-file.txt /remote/path/  (uses HEAD VM)
  vers copy -r ./local-dir/ /remote/path/  (recursive directory copy)`,
	Args: cobra.RangeArgs(2, 3),
	RunE: func(cmd *cobra.Command, args []string) error {
		// Use custom timeout if specified, otherwise use default APIMedium
		timeout := application.Timeouts.APIMedium
		if copyTimeout > 0 {
			timeout = time.Duration(copyTimeout) * time.Second
		}
		apiCtx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()
		recursive, _ := cmd.Flags().GetBool("recursive")
		var target, source, destination string
		if len(args) == 2 {
			source, destination = args[0], args[1]
		} else {
			target, source, destination = args[0], args[1], args[2]
		}
		view, err := handlers.HandleCopy(apiCtx, application, handlers.CopyReq{Target: target, Source: source, Destination: destination, Recursive: recursive})
		if err != nil {
			return err
		}
		pres.RenderCopy(application, view)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(copyCmd)
	copyCmd.Flags().BoolP("recursive", "r", false, "Recursively copy directories")
	copyCmd.Flags().IntVarP(&copyTimeout, "timeout", "t", 0, "Timeout in seconds (default: 30s, use 0 for no limit)")
}
