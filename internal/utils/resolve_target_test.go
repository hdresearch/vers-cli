package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveTarget_WithExplicitTarget(t *testing.T) {
	result, err := ResolveTarget("my-vm-id")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Ident != "my-vm-id" {
		t.Errorf("expected ident 'my-vm-id', got %q", result.Ident)
	}
	if result.UsedHEAD {
		t.Error("expected UsedHEAD=false when target is explicit")
	}
}

func TestResolveTarget_FallsBackToHEAD(t *testing.T) {
	// Create a temp .vers/HEAD
	origDir, _ := os.Getwd()
	tmp := t.TempDir()
	os.Chdir(tmp)
	defer os.Chdir(origDir)

	os.MkdirAll(filepath.Join(VersDir), 0755)
	os.WriteFile(filepath.Join(VersDir, HeadFile), []byte("head-vm-123"), 0644)

	result, err := ResolveTarget("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Ident != "head-vm-123" {
		t.Errorf("expected ident 'head-vm-123', got %q", result.Ident)
	}
	if !result.UsedHEAD {
		t.Error("expected UsedHEAD=true")
	}
	if result.HeadID != "head-vm-123" {
		t.Errorf("expected HeadID 'head-vm-123', got %q", result.HeadID)
	}
}

func TestResolveTarget_NoHEAD_ReturnsError(t *testing.T) {
	origDir, _ := os.Getwd()
	tmp := t.TempDir()
	os.Chdir(tmp)
	defer os.Chdir(origDir)

	_, err := ResolveTarget("")
	if err == nil {
		t.Fatal("expected error when no HEAD exists and target is empty")
	}
}
