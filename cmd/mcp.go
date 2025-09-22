package cmd

import (
	"context"
	"fmt"
	"time"

	imcp "github.com/hdresearch/vers-cli/internal/mcp"
	"github.com/spf13/cobra"
)

var (
	mcpTransport           string
	mcpAddr                string
	mcpAllowInsecureSetKey bool
)

// mcpCmd is the top-level command for MCP-related actions.
var mcpCmd = &cobra.Command{
	Use:   "mcp",
	Short: "Run the MCP server to expose vers tools",
	Long:  "Starts an MCP server so agent clients can call vers operations as tools.",
}

// mcpServeCmd starts the MCP server.
var mcpServeCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the MCP server",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Ensure application is initialized by root PersistentPreRunE
		if application == nil {
			return fmt.Errorf("application not initialized")
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		opts := imcp.Options{
			Transport:           mcpTransport,
			Addr:                mcpAddr,
			AllowInsecureSetKey: mcpAllowInsecureSetKey,
			Verbose:             verbose,
		}

		// Give a brief startup message; real logging stays internal.
		fmt.Fprintf(application.IO.Out, "Starting MCP server (transport=%s, addr=%s)\n", opts.Transport, opts.Addr)

        // Provide a grace period for clean shutdown on CTRL-C.
        // In future we can wire signal.NotifyContext here.
        ctx, cancel2 := context.WithTimeout(ctx, application.Timeouts.APILong+time.Minute)
        defer cancel2()
        return imcp.StartServer(ctx, application, opts)
    },
}

func init() {
	rootCmd.AddCommand(mcpCmd)
	mcpCmd.AddCommand(mcpServeCmd)

	mcpServeCmd.Flags().StringVar(&mcpTransport, "transport", imcp.TransportStdio, "Transport: stdio or http")
	mcpServeCmd.Flags().StringVar(&mcpAddr, "addr", ":3920", "Listen address for HTTP transport")
	mcpServeCmd.Flags().BoolVar(&mcpAllowInsecureSetKey, "allow-insecure-set-key", false, "Allow setting API key via tool (local dev only)")
}
