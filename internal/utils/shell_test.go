package utils

import "testing"

func TestShellQuote(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"", "''"},
		{"hello", "hello"},
		{"ls", "ls"},
		{"-la", "-la"},
		{"/usr/bin/foo", "/usr/bin/foo"},
		{"KEY=VALUE", "KEY=VALUE"},
		{"hello world", "'hello world'"},
		{"it's", `'it'\''s'`},
		{"echo hello world", "'echo hello world'"},
		{"$(rm -rf /)", `'$(rm -rf /)'`},
		{`"quoted"`, `'"quoted"'`},
		{"a\nb", "'a\nb'"},
		{"foo;bar", "'foo;bar'"},
		{"a&b", "'a&b'"},
		{"file name.txt", "'file name.txt'"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			got := ShellQuote(tt.in)
			if got != tt.want {
				t.Errorf("ShellQuote(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestShellJoin(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{
			name: "simple command",
			args: []string{"echo", "hello"},
			want: "echo hello",
		},
		{
			name: "sh -c with compound command",
			args: []string{"sh", "-c", "echo hello world"},
			want: "sh -c 'echo hello world'",
		},
		{
			name: "arguments with spaces",
			args: []string{"grep", "-r", "hello world", "/tmp"},
			want: "grep -r 'hello world' /tmp",
		},
		{
			name: "single argument",
			args: []string{"ls"},
			want: "ls",
		},
		{
			name: "arguments with special chars",
			args: []string{"sh", "-c", "echo $HOME && ls"},
			want: "sh -c 'echo $HOME && ls'",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ShellJoin(tt.args)
			if got != tt.want {
				t.Errorf("ShellJoin(%q) = %q, want %q", tt.args, got, tt.want)
			}
		})
	}
}
