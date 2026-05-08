package cmd

import (
	"encoding/json"
	"testing"
)

func TestAgentContextSchemaVersion(t *testing.T) {
	doc := buildAgentContext(rootCmd)
	if doc.SchemaVersion != "1" {
		t.Fatalf("schema_version = %q, want %q", doc.SchemaVersion, "1")
	}
	if doc.CLI.Name != "vers" {
		t.Errorf("cli.name = %q, want %q", doc.CLI.Name, "vers")
	}
	if doc.AvailableProfiles == nil {
		t.Errorf("available_profiles must be non-nil (empty slice)")
	}
	if doc.Feedback.LocalPath == "" {
		t.Errorf("feedback.local_path should be populated")
	}
}

func TestAgentContextWalksTopLevelCommands(t *testing.T) {
	doc := buildAgentContext(rootCmd)

	// Every visible top-level command (minus help/completion) must be present.
	for _, c := range rootCmd.Commands() {
		if shouldSkipCommand(c) {
			continue
		}
		if _, ok := doc.Commands[c.Name()]; !ok {
			t.Errorf("commands[%q] missing from agent-context output", c.Name())
		}
	}

	// help and completion must be skipped.
	if _, ok := doc.Commands["help"]; ok {
		t.Errorf("commands[help] should be skipped")
	}
	if _, ok := doc.Commands["completion"]; ok {
		t.Errorf("commands[completion] should be skipped")
	}
}

func TestAgentContextEmitsWellFormedJSON(t *testing.T) {
	doc := buildAgentContext(rootCmd)
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
	// Round-trip through generic decoder to confirm shape.
	var back map[string]any
	if err := json.Unmarshal(raw, &back); err != nil {
		t.Fatalf("json.Unmarshal round-trip failed: %v", err)
	}
	for _, key := range []string{"schema_version", "cli", "commands", "available_profiles", "feedback"} {
		if _, ok := back[key]; !ok {
			t.Errorf("emitted JSON missing top-level key %q", key)
		}
	}
}

func TestAgentContextRunIsAsync(t *testing.T) {
	doc := buildAgentContext(rootCmd)
	run, ok := doc.Commands["run"]
	if !ok {
		t.Fatalf("commands[run] missing")
	}
	if !run.Async {
		t.Errorf("run.async = false, want true (run exposes --wait)")
	}
}

func TestAgentContextEnumAnnotationHook(t *testing.T) {
	// Find any subcommand to attach a synthetic annotation to. We use a
	// fresh cobra-style fake by piggybacking on agent-context itself: it
	// has a --pretty bool flag we can decorate. The hook is purely a
	// description-time mechanism, so this exercises the path without
	// mutating production commands.
	prev := agentContextCmd.Annotations
	t.Cleanup(func() { agentContextCmd.Annotations = prev })
	agentContextCmd.Annotations = map[string]string{
		agentContextEnumAnnotationPrefix + "pretty": "true,false",
	}

	doc := buildAgentContext(rootCmd)
	ac, ok := doc.Commands["agent-context"]
	if !ok {
		t.Fatalf("commands[agent-context] missing")
	}
	flag, ok := ac.Flags["--pretty"]
	if !ok {
		t.Fatalf("flags[--pretty] missing")
	}
	if len(flag.Enum) != 2 || flag.Enum[0] != "true" || flag.Enum[1] != "false" {
		t.Errorf("flag.Enum = %v, want [true false]", flag.Enum)
	}
}
