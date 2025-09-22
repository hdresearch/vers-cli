package mcp

import (
	"errors"
	"testing"

	"github.com/hdresearch/vers-cli/internal/errorsx"
)

func TestMapMCPError(t *testing.T) {
	err := mapMCPError(&errorsx.HasChildrenError{VMID: "vm-123"})
	var e *Error
	if !errors.As(err, &e) {
		t.Fatalf("expected *Error, got %T", err)
	}
	if e.Code != E_CONFLICT {
		t.Fatalf("expected E_CONFLICT, got %s", e.Code)
	}

	err = mapMCPError(&errorsx.IsRootError{VMID: "vm-1"})
	if !errors.As(err, &e) || e.Code != E_CONFLICT {
		t.Fatalf("expected E_CONFLICT for IsRootError")
	}

	err = mapMCPError(errors.New("record not found"))
	if !errors.As(err, &e) || e.Code != E_NOT_FOUND {
		t.Fatalf("expected E_NOT_FOUND for not found")
	}
}
