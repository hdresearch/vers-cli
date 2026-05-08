package cmd

import (
	"fmt"
	"text/tabwriter"
	"time"

	"github.com/hdresearch/vers-cli/internal/jobs"
	pres "github.com/hdresearch/vers-cli/internal/presenters"
	"github.com/spf13/cobra"
)

var jobsCmd = &cobra.Command{
	Use:   "jobs",
	Short: "Manage the durable job ledger",
	Long: `View and manage the local job ledger.

Every '--wait' invocation of an async-submitting command (run, branch, deploy,
resume, run-commit) appends entries to a JSONL ledger at ~/.vers/jobs.jsonl
(override with VERS_JOBS_DIR). This command reads, summarises, and prunes that
ledger.

Note: Phase 1 ships journaling only — entries are written for audit and
introspection. Resumption of in-flight jobs is not yet implemented.`,
	Run: func(cmd *cobra.Command, args []string) {
		_ = cmd.Help()
	},
}

// --- jobs list ---

var (
	jobsListJSON   bool
	jobsListLimit  int
	jobsListOffset int
	jobsListStatus string
)

var jobsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List recent jobs",
	Long: `List the latest state of every job in the ledger, most recent first.

Use --status to filter to one of: submitted, complete, failed.
Use --limit and --offset to page (default --limit 50, 0 = unbounded).`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if jobsListStatus != "" {
			switch jobsListStatus {
			case jobs.StatusSubmitted, jobs.StatusComplete, jobs.StatusFailed:
			default:
				return fmt.Errorf("--status must be one of: submitted, complete, failed (got: %q)", jobsListStatus)
			}
		}
		entries, err := jobs.Latest(jobsListStatus)
		if err != nil {
			return err
		}

		start, end, info := pres.ApplyPaging(len(entries), jobsListLimit, jobsListOffset)
		paged := entries[start:end]

		if jobsListJSON {
			items := make([]jobs.Entry, len(paged))
			copy(items, paged)
			return pres.PrintListJSON(items, info)
		}

		if len(paged) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No jobs recorded.")
			pres.PrintTruncationHint(cmd.ErrOrStderr(), info)
			return nil
		}
		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "JOB_ID\tSTATUS\tKIND\tSTARTED\tDURATION")
		for _, e := range paged {
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\n",
				e.ID,
				e.Status,
				e.Kind,
				e.StartedAt.Local().Format(time.RFC3339),
				formatDuration(e),
			)
		}
		w.Flush()
		pres.PrintTruncationHint(cmd.ErrOrStderr(), info)
		return nil
	},
}

func formatDuration(e jobs.Entry) string {
	if e.DurationMs != nil {
		d := time.Duration(*e.DurationMs) * time.Millisecond
		return d.Truncate(time.Millisecond).String()
	}
	if e.Status == jobs.StatusSubmitted {
		return "—"
	}
	return ""
}

// --- jobs get ---

var jobsGetJSON bool

var jobsGetCmd = &cobra.Command{
	Use:   "get <job-id>",
	Short: "Show one job entry",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		entry, ok, err := jobs.Get(args[0])
		if err != nil {
			return err
		}
		if !ok {
			cmd.SilenceUsage = true
			// errorsx maps "not found" -> ExitNotFound (3).
			return fmt.Errorf("job not found: %s", args[0])
		}
		if jobsGetJSON {
			return pres.PrintJSON(entry)
		}
		w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 0, 2, ' ', 0)
		fmt.Fprintf(w, "ID:\t%s\n", entry.ID)
		fmt.Fprintf(w, "Kind:\t%s\n", entry.Kind)
		fmt.Fprintf(w, "Command:\t%s\n", entry.Command)
		if len(entry.Args) > 0 {
			fmt.Fprintf(w, "Args:\t%v\n", entry.Args)
		}
		fmt.Fprintf(w, "Status:\t%s\n", entry.Status)
		fmt.Fprintf(w, "Started:\t%s\n", entry.StartedAt.Local().Format(time.RFC3339))
		if entry.CompletedAt != nil {
			fmt.Fprintf(w, "Completed:\t%s\n", entry.CompletedAt.Local().Format(time.RFC3339))
		}
		if entry.DurationMs != nil {
			fmt.Fprintf(w, "Duration:\t%s\n", time.Duration(*entry.DurationMs)*time.Millisecond)
		}
		if entry.ResultID != "" {
			fmt.Fprintf(w, "Result:\t%s\n", entry.ResultID)
		}
		if entry.Error != "" {
			fmt.Fprintf(w, "Error:\t%s\n", entry.Error)
		}
		return w.Flush()
	},
}

// --- jobs prune ---

var (
	jobsPruneOlderThan string
	jobsPruneAll       bool
	jobsPruneDryRun    bool
)

var jobsPruneCmd = &cobra.Command{
	Use:   "prune",
	Short: "Remove old job entries from the ledger",
	Long: `Remove ledger entries older than --older-than (default 7d), or all entries with --all.

Use --dry-run to preview without modifying the ledger.

Examples:
  vers jobs prune --older-than 7d
  vers jobs prune --older-than 24h --dry-run
  vers jobs prune --all`,
	RunE: func(cmd *cobra.Command, args []string) error {
		opts := jobs.PruneOptions{
			All:    jobsPruneAll,
			DryRun: jobsPruneDryRun,
		}
		if !jobsPruneAll {
			d, err := jobs.ParseDuration(jobsPruneOlderThan)
			if err != nil {
				return err
			}
			opts.OlderThan = d
		}
		res, err := jobs.Prune(opts)
		if err != nil {
			return err
		}
		prefix := "pruned"
		if jobsPruneDryRun {
			prefix = "would prune"
		}
		fmt.Fprintf(cmd.OutOrStdout(), "%s %d entries (%d kept)\n", prefix, res.Pruned, res.Kept)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(jobsCmd)
	jobsCmd.AddCommand(jobsListCmd, jobsGetCmd, jobsPruneCmd)

	jobsListCmd.Flags().BoolVar(&jobsListJSON, "json", false, "Output as JSON")
	jobsListCmd.Flags().IntVar(&jobsListLimit, "limit", 50, "Maximum number of jobs to return (0 = unbounded)")
	jobsListCmd.Flags().IntVar(&jobsListOffset, "offset", 0, "Number of jobs to skip (for paging)")
	jobsListCmd.Flags().StringVar(&jobsListStatus, "status", "", "Filter by status: submitted, complete, failed")

	jobsGetCmd.Flags().BoolVar(&jobsGetJSON, "json", false, "Output as JSON")

	jobsPruneCmd.Flags().StringVar(&jobsPruneOlderThan, "older-than", "7d", "Prune entries older than this duration (e.g. 7d, 24h, 30m)")
	jobsPruneCmd.Flags().BoolVar(&jobsPruneAll, "all", false, "Prune all entries regardless of age")
	jobsPruneCmd.Flags().BoolVar(&jobsPruneDryRun, "dry-run", false, "Preview what would be pruned without writing")
}
