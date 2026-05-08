package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// agentContextSchemaVersion is bumped whenever the agent-context JSON shape
// changes in a backwards-incompatible way.
const agentContextSchemaVersion = "1"

// agentContextEnumAnnotationPrefix is the cobra Annotations key prefix used to
// declare a known enum set for a flag. Example:
//
//	cmd.Annotations["vers:enum:visibility"] = "public,private,unlisted"
const agentContextEnumAnnotationPrefix = "vers:enum:"

var agentContextPretty bool

var agentContextCmd = &cobra.Command{
	Use:   "agent-context",
	Short: "Emit a versioned JSON description of the CLI for agent consumption",
	Long: `Emit a versioned, machine-readable JSON description of every command,
subcommand, and flag exposed by the CLI.

Agents should consume this output instead of parsing --help text. The top-level
"schema_version" field lets consumers detect breaking shape changes.`,
	SilenceUsage:  true,
	SilenceErrors: true,
	Run: func(cmd *cobra.Command, args []string) {
		doc := buildAgentContext(cmd.Root())

		var (
			out []byte
			err error
		)
		if agentContextPretty {
			out, err = json.MarshalIndent(doc, "", "  ")
		} else {
			out, err = json.Marshal(doc)
		}
		if err != nil {
			// Per spec this command must never fail; fall back to a minimal
			// stub document and exit 0.
			fmt.Fprintln(os.Stdout, `{"schema_version":"`+agentContextSchemaVersion+`","commands":{}}`)
			return
		}
		fmt.Fprintln(os.Stdout, string(out))
	},
}

func init() {
	agentContextCmd.Flags().BoolVar(&agentContextPretty, "pretty", false, "Emit indented JSON instead of compact JSON")
	rootCmd.AddCommand(agentContextCmd)
}

// agentContextDoc is the top-level JSON shape emitted by `vers agent-context`.
type agentContextDoc struct {
	SchemaVersion     string                          `json:"schema_version"`
	CLI               agentContextCLI                 `json:"cli"`
	Commands          map[string]*agentContextCommand `json:"commands"`
	AvailableProfiles []string                        `json:"available_profiles"`
	Feedback          agentContextFeedback            `json:"feedback"`
}

type agentContextCLI struct {
	Name        string `json:"name"`
	Version     string `json:"version"`
	Description string `json:"description"`
}

type agentContextCommand struct {
	Use         string                          `json:"use"`
	Short       string                          `json:"short"`
	Long        string                          `json:"long,omitempty"`
	Aliases     []string                        `json:"aliases,omitempty"`
	Args        agentContextArgs                `json:"args"`
	Async       bool                            `json:"async"`
	Flags       map[string]*agentContextFlag    `json:"flags"`
	Subcommands map[string]*agentContextCommand `json:"subcommands,omitempty"`
}

type agentContextArgs struct {
	Min int `json:"min"`
	Max int `json:"max"`
}

type agentContextFlag struct {
	Shorthand string   `json:"shorthand,omitempty"`
	Type      string   `json:"type"`
	Default   string   `json:"default"`
	Usage     string   `json:"usage"`
	Required  bool     `json:"required"`
	Enum      []string `json:"enum,omitempty"`
}

type agentContextFeedback struct {
	LocalPath          string `json:"local_path"`
	EndpointConfigured bool   `json:"endpoint_configured"`
}

func buildAgentContext(root *cobra.Command) *agentContextDoc {
	doc := &agentContextDoc{
		SchemaVersion: agentContextSchemaVersion,
		CLI: agentContextCLI{
			Name:        root.Name(),
			Version:     Version,
			Description: strings.TrimSpace(root.Long),
		},
		Commands:          map[string]*agentContextCommand{},
		AvailableProfiles: []string{},
		Feedback: agentContextFeedback{
			LocalPath:          "~/.vers/feedback.jsonl",
			EndpointConfigured: os.Getenv("VERS_FEEDBACK_ENDPOINT") != "",
		},
	}

	for _, c := range root.Commands() {
		if shouldSkipCommand(c) {
			continue
		}
		doc.Commands[c.Name()] = describeCommand(c)
	}
	return doc
}

func shouldSkipCommand(c *cobra.Command) bool {
	if c.Hidden {
		return true
	}
	switch c.Name() {
	case "help", "completion":
		return true
	}
	return false
}

func describeCommand(c *cobra.Command) *agentContextCommand {
	out := &agentContextCommand{
		Use:   c.Use,
		Short: c.Short,
		Long:  strings.TrimSpace(c.Long),
		Args:  agentContextArgs{Min: 0, Max: -1},
		Flags: map[string]*agentContextFlag{},
	}
	if len(c.Aliases) > 0 {
		out.Aliases = append(out.Aliases, c.Aliases...)
	}

	// Collect flags (local + inherited from parents), excluding hidden &
	// deprecated. Inherited flags give agents a complete picture without
	// having to re-walk the parent chain themselves.
	c.Flags().VisitAll(func(f *pflag.Flag) {
		if entry := describeFlag(c, f); entry != nil {
			out.Flags["--"+f.Name] = entry
		}
	})

	// async = visible --wait flag exists.
	if w := c.Flags().Lookup("wait"); w != nil && !w.Hidden && len(w.Deprecated) == 0 {
		out.Async = true
	}

	for _, sub := range c.Commands() {
		if shouldSkipCommand(sub) {
			continue
		}
		if out.Subcommands == nil {
			out.Subcommands = map[string]*agentContextCommand{}
		}
		out.Subcommands[sub.Name()] = describeCommand(sub)
	}

	return out
}

func describeFlag(parent *cobra.Command, f *pflag.Flag) *agentContextFlag {
	if f.Hidden || len(f.Deprecated) > 0 {
		return nil
	}
	entry := &agentContextFlag{
		Shorthand: f.Shorthand,
		Type:      f.Value.Type(),
		Default:   f.DefValue,
		Usage:     f.Usage,
	}
	if _, ok := f.Annotations[cobra.BashCompOneRequiredFlag]; ok {
		entry.Required = true
	}
	if parent != nil && parent.Annotations != nil {
		if raw, ok := parent.Annotations[agentContextEnumAnnotationPrefix+f.Name]; ok && raw != "" {
			parts := strings.Split(raw, ",")
			vals := make([]string, 0, len(parts))
			for _, p := range parts {
				if v := strings.TrimSpace(p); v != "" {
					vals = append(vals, v)
				}
			}
			if len(vals) > 0 {
				entry.Enum = vals
			}
		}
	}
	return entry
}
