package prompts

import (
	"bufio"
	"fmt"
	"io"
	"strings"
)

// Prompter provides confirmation prompts.
type Prompter interface {
	YesNo(prompt string) (bool, error)
	ConfirmExact(prompt, required string) (bool, error)
}

// TermPrompter implements Prompter for terminal IO.
type TermPrompter struct {
	in  *bufio.Reader
	out io.Writer
}

func NewTermPrompter(in io.Reader, out io.Writer) *TermPrompter {
	return &TermPrompter{in: bufio.NewReader(in), out: out}
}

func (p *TermPrompter) YesNo(prompt string) (bool, error) {
	if _, err := fmt.Fprint(p.out, prompt+" [y/N]: "); err != nil {
		return false, err
	}
	line, err := p.in.ReadString('\n')
	if err != nil {
		return false, err
	}
	s := strings.TrimSpace(line)
	return strings.EqualFold(s, "y") || strings.EqualFold(s, "yes"), nil
}

func (p *TermPrompter) ConfirmExact(prompt, required string) (bool, error) {
	if _, err := fmt.Fprintf(p.out, "%s (type '%s'): ", prompt, required); err != nil {
		return false, err
	}
	line, err := p.in.ReadString('\n')
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(line) == required, nil
}
