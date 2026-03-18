package errorsx_test

import (
	"errors"
	"testing"

	"github.com/hdresearch/vers-cli/internal/errorsx"
)

func TestExitCodeFromError(t *testing.T) {
	tests := []struct {
		err  error
		want int
	}{
		{nil, errorsx.ExitOK},
		{errors.New("401 Unauthorized"), errorsx.ExitAuth},
		{errors.New("forbidden"), errorsx.ExitAuth},
		{errors.New("VM not found"), errorsx.ExitNotFound},
		{errors.New("404"), errorsx.ExitNotFound},
		{errors.New("409 conflict"), errorsx.ExitConflict},
		{errors.New("400 bad request"), errorsx.ExitBadRequest},
		{errors.New("context deadline exceeded"), errorsx.ExitTimeout},
		{errors.New("timed out waiting"), errorsx.ExitTimeout},
		{errors.New("operation cancelled by user"), errorsx.ExitCancelled},
		{errors.New("something random"), errorsx.ExitGeneral},
	}

	for _, tt := range tests {
		got := errorsx.ExitCodeFromError(tt.err)
		if got != tt.want {
			name := "<nil>"
			if tt.err != nil {
				name = tt.err.Error()
			}
			t.Errorf("ExitCodeFromError(%q) = %d, want %d", name, got, tt.want)
		}
	}
}
