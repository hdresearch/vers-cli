package mcp

import (
	"errors"
	"testing"
)

func TestMapMCPError(t *testing.T) {
	err := mapMCPError(errors.New("record not found"))
	var e *Error
	if !errors.As(err, &e) || e.Code != E_NOT_FOUND {
		t.Fatalf("expected E_NOT_FOUND for not found")
	}

	// Non-matching errors pass through
	orig := errors.New("something else")
	err = mapMCPError(orig)
	if err != orig {
		t.Fatalf("expected passthrough, got %v", err)
	}

	// Nil stays nil
	if mapMCPError(nil) != nil {
		t.Fatal("expected nil")
	}
}
