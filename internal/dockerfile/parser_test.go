package dockerfile

import (
	"strings"
	"testing"
)

func TestParseBasic(t *testing.T) {
	src := `# A comment
FROM scratch
ENV FOO=bar BAZ="hello world"
WORKDIR /app
COPY . /app
RUN echo hello && \
    echo world
CMD ["node", "server.js"]
EXPOSE 8080
`
	instrs, err := Parse(strings.NewReader(src))
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if len(instrs) != 7 {
		t.Fatalf("got %d instrs, want 7: %+v", len(instrs), instrs)
	}
	if instrs[0].Kind != KindFrom || instrs[0].From.Ref != "scratch" {
		t.Errorf("FROM: %+v", instrs[0])
	}
	if instrs[1].Kind != KindEnv || len(instrs[1].Env.Pairs) != 2 {
		t.Errorf("ENV: %+v", instrs[1].Env)
	}
	if instrs[1].Env.Pairs[1].Value != "hello world" {
		t.Errorf("ENV quoted: %q", instrs[1].Env.Pairs[1].Value)
	}
	if instrs[4].Kind != KindRun || !strings.Contains(instrs[4].Run.Shell, "echo hello") || !strings.Contains(instrs[4].Run.Shell, "echo world") {
		t.Errorf("RUN continuation: %q", instrs[4].Run.Shell)
	}
	if instrs[5].Kind != KindCmd || len(instrs[5].Cmd.Exec) != 2 || instrs[5].Cmd.Exec[0] != "node" {
		t.Errorf("CMD exec form: %+v", instrs[5].Cmd)
	}
	if instrs[6].Kind != KindExpose || instrs[6].Expose.Ports[0] != "8080" {
		t.Errorf("EXPOSE: %+v", instrs[6].Expose)
	}
}

func TestParseFromAs(t *testing.T) {
	instrs, err := Parse(strings.NewReader("FROM abc123 AS builder\n"))
	if err != nil {
		t.Fatal(err)
	}
	if instrs[0].From.As != "builder" {
		t.Errorf("want As=builder, got %+v", instrs[0].From)
	}
}

func TestParseCopyFlags(t *testing.T) {
	instrs, err := Parse(strings.NewReader("COPY --chown=root:root a b /dst/\n"))
	if err != nil {
		t.Fatal(err)
	}
	c := instrs[0].Copy
	if c.Chown != "root:root" {
		t.Errorf("chown: %q", c.Chown)
	}
	if len(c.Sources) != 2 || c.Sources[0] != "a" || c.Sources[1] != "b" {
		t.Errorf("sources: %+v", c.Sources)
	}
	if c.Dest != "/dst/" {
		t.Errorf("dest: %q", c.Dest)
	}
}

func TestParseEnvLegacy(t *testing.T) {
	instrs, err := Parse(strings.NewReader("ENV MY_VAR this is the value\n"))
	if err != nil {
		t.Fatal(err)
	}
	p := instrs[0].Env.Pairs
	if len(p) != 1 || p[0].Key != "MY_VAR" || p[0].Value != "this is the value" {
		t.Errorf("legacy ENV: %+v", p)
	}
}

func TestParseArg(t *testing.T) {
	cases := []struct {
		in      string
		name    string
		def     string
		hasDef  bool
	}{
		{"ARG FOO\n", "FOO", "", false},
		{"ARG FOO=bar\n", "FOO", "bar", true},
		{"ARG FOO=\n", "FOO", "", true},
	}
	for _, c := range cases {
		instrs, err := Parse(strings.NewReader(c.in))
		if err != nil {
			t.Fatalf("%s: %v", c.in, err)
		}
		a := instrs[0].Arg
		if a.Name != c.name || a.Default != c.def || a.HasDef != c.hasDef {
			t.Errorf("%q: got %+v", c.in, a)
		}
	}
}

func TestParseCommentsAndBlanks(t *testing.T) {
	src := "# top\n\nFROM scratch\n# mid\n\nRUN true\n"
	instrs, err := Parse(strings.NewReader(src))
	if err != nil {
		t.Fatal(err)
	}
	if len(instrs) != 2 {
		t.Errorf("expected 2 instrs, got %d", len(instrs))
	}
}

func TestParseRejectsUnknown(t *testing.T) {
	_, err := Parse(strings.NewReader("HEALTHCHECK CMD foo\n"))
	if err == nil {
		t.Error("expected error for unsupported instruction")
	}
}
