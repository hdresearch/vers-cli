package handlers

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hdresearch/vers-cli/internal/app"
)

// HandleBuild: error paths that don't require a live orchestrator.
//
// These exercise the pre-flight work (locating the Dockerfile, parsing it,
// loading the build context) which happens before any SDK calls. A zero-value
// *app.App is fine for these cases because we never reach the executor.

func TestHandleBuild_MissingDockerfile(t *testing.T) {
	dir := t.TempDir()
	_, err := HandleBuild(context.Background(), &app.App{}, BuildReq{ContextDir: dir})
	if err == nil {
		t.Fatal("expected error for missing Dockerfile")
	}
	if !strings.Contains(err.Error(), "parse") {
		t.Errorf("want parse error, got: %v", err)
	}
}

func TestHandleBuild_ParseError(t *testing.T) {
	dir := t.TempDir()
	// Syntax the parser actively rejects (unsupported keyword).
	if err := os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte("HEALTHCHECK CMD foo\n"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := HandleBuild(context.Background(), &app.App{}, BuildReq{ContextDir: dir})
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestHandleBuild_ContextNotADirectory(t *testing.T) {
	f, err := os.CreateTemp(t.TempDir(), "notadir")
	if err != nil {
		t.Fatal(err)
	}
	f.Close()
	// Use a real Dockerfile path, but point context at the file itself.
	df := filepath.Join(t.TempDir(), "Dockerfile")
	_ = os.WriteFile(df, []byte("FROM scratch\n"), 0644)
	_, err = HandleBuild(context.Background(), &app.App{}, BuildReq{
		ContextDir: f.Name(),
		Dockerfile: df,
	})
	if err == nil {
		t.Fatal("expected 'not a directory' error")
	}
}

func TestHandleBuild_FromScratchRequiresSizing(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte("FROM scratch\nRUN echo hi\n"), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := HandleBuild(context.Background(), &app.App{}, BuildReq{ContextDir: dir})
	if err == nil {
		t.Fatal("expected sizing error")
	}
	if !strings.Contains(err.Error(), "mem-size") {
		t.Errorf("want sizing error, got: %v", err)
	}
}

func TestHandleBuild_RejectsMultiStage(t *testing.T) {
	dir := t.TempDir()
	df := "FROM scratch\nFROM scratch AS next\n"
	if err := os.WriteFile(filepath.Join(dir, "Dockerfile"), []byte(df), 0644); err != nil {
		t.Fatal(err)
	}
	_, err := HandleBuild(context.Background(), &app.App{}, BuildReq{
		ContextDir:  dir,
		MemSizeMib:  512,
		VcpuCount:   1,
		FsSizeVmMib: 1024,
	})
	if err == nil {
		t.Fatal("expected multi-stage rejection")
	}
	if !strings.Contains(err.Error(), "multi-stage") {
		t.Errorf("want multi-stage error, got: %v", err)
	}
}
