package mcp

import "testing"

func TestValidateExecute(t *testing.T) {
	if err := validateExecute(ExecuteInput{}); err == nil {
		t.Fatalf("expected error when command is empty")
	}
	if err := validateExecute(ExecuteInput{Command: []string{"echo", "hi"}, TimeoutSeconds: -1}); err == nil {
		t.Fatalf("expected error for negative timeout")
	}
	if err := validateExecute(ExecuteInput{Command: []string{"echo", "ok"}}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateRun(t *testing.T) {
	if err := validateRun(RunInput{FsSizeVmMib: -1}); err == nil {
		t.Fatalf("expected error for negative filesystem size")
	}
	if err := validateRun(RunInput{FsSizeVmMib: 512}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateKill(t *testing.T) {
	if err := validateKill(KillInput{}); err == nil {
		t.Fatalf("expected confirmation-required error for MCP kill without skipConfirmation")
	}
	if err := validateKill(KillInput{SkipConfirmation: true}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
