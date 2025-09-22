package runtime

import (
	"context"
	"os/exec"
)

// Stdio provides standard IO wiring for processes.
type Stdio struct {
	In  any // io.Reader, but kept as any to avoid import cycles in call sites
	Out any // io.Writer
	Err any // io.Writer
}

// Runner runs external commands.
type Runner interface {
	Run(ctx context.Context, name string, args []string, stdio Stdio) error
}

// ExecRunner implements Runner using os/exec.
type ExecRunner struct{}

func NewExecRunner() *ExecRunner { return &ExecRunner{} }

func (r *ExecRunner) Run(ctx context.Context, name string, args []string, stdio Stdio) error {
	cmd := exec.CommandContext(ctx, name, args...)
	if stdio.In != nil {
		if rd, ok := stdio.In.(interface{ Read([]byte) (int, error) }); ok {
			cmd.Stdin = rd
		}
	}
	if stdio.Out != nil {
		if wr, ok := stdio.Out.(interface{ Write([]byte) (int, error) }); ok {
			cmd.Stdout = wr
		}
	}
	if stdio.Err != nil {
		if wr, ok := stdio.Err.(interface{ Write([]byte) (int, error) }); ok {
			cmd.Stderr = wr
		}
	}
	return cmd.Run()
}
