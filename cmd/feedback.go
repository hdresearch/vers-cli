package cmd

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/hdresearch/vers-cli/internal/feedback"
	pres "github.com/hdresearch/vers-cli/internal/presenters"
	"github.com/spf13/cobra"
)

// Flags for `vers feedback <message>`.
var (
	feedbackJSON bool
)

// Flags for `vers feedback list`.
var (
	feedbackListJSON   bool
	feedbackListLimit  int
	feedbackListOffset int
)

// feedbackResult is the JSON envelope emitted by `vers feedback <msg> --json`.
type feedbackResult struct {
	ID             string `json:"id"`
	Timestamp      string `json:"timestamp"`
	VersVersion    string `json:"vers_version"`
	Message        string `json:"message"`
	SentUpstream   bool   `json:"sent_upstream"`
	UpstreamStatus int    `json:"upstream_status,omitempty"`
	UpstreamError  string `json:"upstream_error,omitempty"`
	JournalPath    string `json:"journal_path"`
}

var feedbackCmd = &cobra.Command{
	Use:   "feedback <message>",
	Short: "Report friction or feedback about the vers CLI",
	Long: `Record a feedback entry locally (and optionally POST upstream).

Each entry is appended to a JSONL journal at ~/.vers/feedback.jsonl. When the
VERS_FEEDBACK_ENDPOINT environment variable is set, the entry is also POSTed
to that URL as application/json with a 5s timeout. Upstream delivery is
best-effort: a failed POST does not fail the command — the local journal
entry is still written.

Use 'vers feedback list' to inspect recent entries.

Use --json for machine-readable output.

Examples:
  vers feedback "the --tier flag rejects 'enterprise' but docs list it as valid"
  VERS_FEEDBACK_ENDPOINT=https://example.com/cli-feedback vers feedback "race in --wait"
  vers feedback list --limit 5 --json`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		message := strings.TrimSpace(strings.Join(args, " "))
		if message == "" {
			return fmt.Errorf("feedback message must not be empty. Usage: vers feedback <message>")
		}

		format, err := pres.ParseFormat(false, feedbackJSON, "")
		if err != nil {
			return err
		}

		path, err := feedback.DefaultPath()
		if err != nil {
			return err
		}

		id, err := feedback.NewID()
		if err != nil {
			return err
		}

		entry := feedback.Entry{
			ID:           id,
			Timestamp:    feedback.Now(),
			VersVersion:  Version,
			Message:      message,
			SentUpstream: false,
		}

		// Try upstream POST first (if configured) so we can record the result
		// in the journal. A failure here is non-fatal.
		endpoint := strings.TrimSpace(os.Getenv(feedback.EndpointEnvVar))
		var upstreamStatus int
		var upstreamErr error
		if endpoint != "" {
			upstreamStatus, upstreamErr = feedback.PostUpstream(context.Background(), endpoint, entry)
			if upstreamErr == nil {
				entry.SentUpstream = true
			}
		}

		// Always write the local journal entry, even if the upstream POST failed.
		if err := feedback.Append(path, entry); err != nil {
			return fmt.Errorf("write feedback journal: %w", err)
		}

		result := feedbackResult{
			ID:             entry.ID,
			Timestamp:      entry.Timestamp,
			VersVersion:    entry.VersVersion,
			Message:        entry.Message,
			SentUpstream:   entry.SentUpstream,
			UpstreamStatus: upstreamStatus,
			JournalPath:    path,
		}
		if upstreamErr != nil {
			result.UpstreamError = upstreamErr.Error()
		}

		// Emit the upstream-failed warning to stderr in both text and JSON modes
		// so agents see it without polluting machine-readable stdout.
		if endpoint != "" && upstreamErr != nil {
			fmt.Fprintf(cmd.ErrOrStderr(),
				"feedback recorded locally; upstream POST failed: %s\n", upstreamErr.Error())
		}

		switch format {
		case pres.FormatJSON:
			return pres.PrintJSON(result)
		default:
			switch {
			case endpoint == "":
				fmt.Fprintln(cmd.OutOrStdout(), "feedback recorded locally (1 entry)")
			case upstreamErr == nil:
				fmt.Fprintf(cmd.OutOrStdout(),
					"feedback recorded locally and sent upstream (status: %d)\n", upstreamStatus)
			default:
				// Warning already went to stderr; confirm the local write on stdout.
				fmt.Fprintln(cmd.OutOrStdout(), "feedback recorded locally (1 entry)")
			}
		}
		return nil
	},
}

var feedbackListCmd = &cobra.Command{
	Use:   "list",
	Short: "List recent feedback entries from the local journal",
	Long: `Read recent entries from the local feedback journal (~/.vers/feedback.jsonl).

Pagination:
  --limit N    Cap results at N (default 10). Use 0 for unbounded.
  --offset N   Skip the first N results (use with --limit to page).

Entries are returned newest-first. When the result is truncated, a hint with
--offset for the next page is printed to stderr (text mode) or included in
the JSON envelope.

Use --json for machine-readable output.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		format, err := pres.ParseFormat(false, feedbackListJSON, "")
		if err != nil {
			return err
		}

		path, err := feedback.DefaultPath()
		if err != nil {
			return err
		}

		entries, err := feedback.ReadAll(path)
		if err != nil {
			return err
		}

		// Newest-first ordering.
		reversed := make([]feedback.Entry, len(entries))
		for i, e := range entries {
			reversed[len(entries)-1-i] = e
		}

		start, end, info := pres.ApplyPaging(len(reversed), feedbackListLimit, feedbackListOffset)
		paged := reversed[start:end]

		switch format {
		case pres.FormatJSON:
			// Always emit a non-nil items slice so JSON consumers see [] not null.
			if paged == nil {
				paged = []feedback.Entry{}
			}
			return pres.PrintListJSON(paged, info)
		default:
			if len(paged) == 0 {
				fmt.Fprintln(cmd.OutOrStdout(), "No feedback entries.")
				return nil
			}
			for _, e := range paged {
				upstream := "local"
				if e.SentUpstream {
					upstream = "local+upstream"
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%s  %s  [%s]  %s\n",
					e.Timestamp, e.ID, upstream, e.Message)
			}
			pres.PrintTruncationHint(cmd.ErrOrStderr(), info)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(feedbackCmd)
	feedbackCmd.Flags().BoolVar(&feedbackJSON, "json", false, "Output as JSON")

	feedbackCmd.AddCommand(feedbackListCmd)
	feedbackListCmd.Flags().BoolVar(&feedbackListJSON, "json", false, "Output as JSON")
	feedbackListCmd.Flags().IntVar(&feedbackListLimit, "limit", 10, "Maximum number of entries to return (0 = unbounded)")
	feedbackListCmd.Flags().IntVar(&feedbackListOffset, "offset", 0, "Number of entries to skip (for paging)")
}
