// Package dockerfile parses the subset of Dockerfile syntax supported by
// `vers build`. It is intentionally small: we only implement instructions
// we can faithfully execute against a Vers VM.
//
// Supported instructions:
//
//	FROM         scratch | <commit-id-or-tag>      (single-stage only in v1)
//	RUN          <shell> | ["exec","form"]
//	COPY         [--chown=U:G] <src>... <dst>      (local build context only)
//	ADD          <src>... <dst>                    (same as COPY in v1)
//	ENV          KEY=VAL [KEY=VAL ...] | KEY VAL
//	ARG          NAME[=default]
//	WORKDIR      <path>
//	USER         <name|uid>[:group]
//	LABEL        KEY=VAL [KEY=VAL ...]
//	CMD          <shell> | ["exec","form"]
//	ENTRYPOINT   <shell> | ["exec","form"]
//	EXPOSE       <port>[/proto] ...
//
// Multi-stage builds (`FROM ... AS name`, `COPY --from=...`) are parsed but
// rejected by the executor with a clear error for now.
package dockerfile

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
)

// InstructionKind is a Dockerfile instruction keyword.
type InstructionKind string

const (
	KindFrom       InstructionKind = "FROM"
	KindRun        InstructionKind = "RUN"
	KindCopy       InstructionKind = "COPY"
	KindAdd        InstructionKind = "ADD"
	KindEnv        InstructionKind = "ENV"
	KindArg        InstructionKind = "ARG"
	KindWorkdir    InstructionKind = "WORKDIR"
	KindUser       InstructionKind = "USER"
	KindLabel      InstructionKind = "LABEL"
	KindCmd        InstructionKind = "CMD"
	KindEntrypoint InstructionKind = "ENTRYPOINT"
	KindExpose     InstructionKind = "EXPOSE"
)

// Instruction is one parsed Dockerfile line (after continuations joined).
type Instruction struct {
	Kind    InstructionKind
	Raw     string // raw logical line, useful for cache keys and progress output
	LineNum int    // 1-based line number of the first physical line

	// Exactly one of the following is populated depending on Kind.
	From       *FromInstr
	Run        *RunInstr
	Copy       *CopyInstr
	Env        *KVInstr
	Label      *KVInstr
	Arg        *ArgInstr
	Workdir    *StrInstr
	User       *StrInstr
	Cmd        *ExecInstr
	Entrypoint *ExecInstr
	Expose     *ExposeInstr
}

type FromInstr struct {
	Ref  string // "scratch", a commit id, or a tag name
	As   string // stage name after AS (optional)
}

type RunInstr struct {
	// If Exec is set the user wrote exec form (JSON array).
	// Otherwise Shell holds the raw shell string to pass to `bash -c`.
	Shell string
	Exec  []string
}

type CopyInstr struct {
	Sources []string
	Dest    string
	Chown   string // optional --chown=USER[:GROUP]
	From    string // optional --from=<stage|image>; unsupported in v1
}

type KVInstr struct {
	Pairs []KV
}

type KV struct{ Key, Value string }

type ArgInstr struct {
	Name    string
	Default string
	HasDef  bool
}

type StrInstr struct{ Value string }

type ExecInstr struct {
	Shell string
	Exec  []string
}

type ExposeInstr struct {
	Ports []string
}

// ParseFile parses a Dockerfile at the given path.
func ParseFile(path string) ([]Instruction, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return Parse(f)
}

// Parse parses Dockerfile bytes from r.
func Parse(r io.Reader) ([]Instruction, error) {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)

	// First pass: read physical lines, join continuations, drop comments.
	// Parser directives (# syntax=...) are ignored in v1.
	type logical struct {
		text    string
		lineNum int
	}
	var logicals []logical

	var (
		builder strings.Builder
		startLn int
		lineNum int
	)
	flush := func() {
		text := strings.TrimSpace(builder.String())
		builder.Reset()
		if text == "" {
			return
		}
		logicals = append(logicals, logical{text: text, lineNum: startLn})
	}

	for scanner.Scan() {
		lineNum++
		raw := scanner.Text()
		// Strip BOM on first line
		if lineNum == 1 {
			raw = strings.TrimPrefix(raw, "\uFEFF")
		}
		trimmed := strings.TrimSpace(raw)
		// Comment line (only if not a continuation)
		if strings.HasPrefix(trimmed, "#") && builder.Len() == 0 {
			continue
		}
		// Start of a new logical line
		if builder.Len() == 0 {
			startLn = lineNum
		}
		// Handle line continuation: trailing backslash (with optional trailing whitespace)
		noTrail := strings.TrimRight(raw, " \t")
		if strings.HasSuffix(noTrail, "\\") {
			// strip the backslash, append a space
			builder.WriteString(strings.TrimSuffix(noTrail, "\\"))
			builder.WriteByte(' ')
			continue
		}
		builder.WriteString(raw)
		flush()
	}
	flush()
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("read dockerfile: %w", err)
	}

	var out []Instruction
	for _, l := range logicals {
		instr, err := parseLogical(l.text, l.lineNum)
		if err != nil {
			return nil, fmt.Errorf("line %d: %w", l.lineNum, err)
		}
		out = append(out, instr)
	}
	return out, nil
}

func parseLogical(text string, lineNum int) (Instruction, error) {
	// Split into KEYWORD + rest
	sp := strings.IndexAny(text, " \t")
	if sp < 0 {
		return Instruction{}, fmt.Errorf("missing arguments: %q", text)
	}
	keyword := strings.ToUpper(text[:sp])
	rest := strings.TrimSpace(text[sp+1:])

	instr := Instruction{
		Kind:    InstructionKind(keyword),
		Raw:     text,
		LineNum: lineNum,
	}

	switch instr.Kind {
	case KindFrom:
		f, err := parseFrom(rest)
		if err != nil {
			return instr, err
		}
		instr.From = f
	case KindRun:
		instr.Run = parseRun(rest)
	case KindCopy, KindAdd:
		c, err := parseCopy(rest)
		if err != nil {
			return instr, err
		}
		instr.Copy = c
	case KindEnv:
		kv, err := parseKV(rest, true)
		if err != nil {
			return instr, err
		}
		instr.Env = kv
	case KindLabel:
		kv, err := parseKV(rest, false)
		if err != nil {
			return instr, err
		}
		instr.Label = kv
	case KindArg:
		instr.Arg = parseArg(rest)
	case KindWorkdir:
		instr.Workdir = &StrInstr{Value: rest}
	case KindUser:
		instr.User = &StrInstr{Value: rest}
	case KindCmd:
		instr.Cmd = parseExec(rest)
	case KindEntrypoint:
		instr.Entrypoint = parseExec(rest)
	case KindExpose:
		instr.Expose = &ExposeInstr{Ports: fieldsNonEmpty(rest)}
	default:
		return instr, fmt.Errorf("unsupported instruction %q", keyword)
	}
	return instr, nil
}

func parseFrom(s string) (*FromInstr, error) {
	fields := fieldsNonEmpty(s)
	if len(fields) == 0 {
		return nil, fmt.Errorf("FROM requires an argument")
	}
	f := &FromInstr{Ref: fields[0]}
	if len(fields) >= 3 && strings.EqualFold(fields[1], "AS") {
		f.As = fields[2]
	} else if len(fields) != 1 {
		return nil, fmt.Errorf("FROM syntax: expected 'FROM <ref> [AS name]'")
	}
	return f, nil
}

func parseRun(s string) *RunInstr {
	if exec, ok := tryParseExecArray(s); ok {
		return &RunInstr{Exec: exec}
	}
	return &RunInstr{Shell: s}
}

func parseExec(s string) *ExecInstr {
	if exec, ok := tryParseExecArray(s); ok {
		return &ExecInstr{Exec: exec}
	}
	return &ExecInstr{Shell: s}
}

func tryParseExecArray(s string) ([]string, bool) {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "[") {
		return nil, false
	}
	var out []string
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		return nil, false
	}
	return out, true
}

func parseCopy(s string) (*CopyInstr, error) {
	fields, err := splitCopyArgs(s)
	if err != nil {
		return nil, err
	}
	c := &CopyInstr{}
	// Collect leading --flag=value entries
	for len(fields) > 0 && strings.HasPrefix(fields[0], "--") {
		flag := fields[0]
		fields = fields[1:]
		eq := strings.IndexByte(flag, '=')
		if eq < 0 {
			return nil, fmt.Errorf("COPY flag %q requires =value", flag)
		}
		name, val := flag[:eq], flag[eq+1:]
		switch name {
		case "--chown":
			c.Chown = val
		case "--from":
			c.From = val
		default:
			return nil, fmt.Errorf("COPY: unsupported flag %q", name)
		}
	}
	if len(fields) < 2 {
		return nil, fmt.Errorf("COPY requires at least <src> <dst>")
	}
	c.Dest = fields[len(fields)-1]
	c.Sources = fields[:len(fields)-1]
	return c, nil
}

// splitCopyArgs splits a COPY argument string. If the string is a JSON array
// (exec form for COPY: ["a", "b"]) we honor that, otherwise we split on
// whitespace.
func splitCopyArgs(s string) ([]string, error) {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "[") {
		var out []string
		if err := json.Unmarshal([]byte(s), &out); err != nil {
			return nil, fmt.Errorf("COPY: bad JSON array: %w", err)
		}
		return out, nil
	}
	return fieldsNonEmpty(s), nil
}

// parseKV parses ENV/LABEL. ENV allows legacy single-pair form (`ENV K V...`)
// when no '=' is present in the first token.
func parseKV(s string, allowLegacy bool) (*KVInstr, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, fmt.Errorf("expected key=value")
	}
	// Legacy form: "ENV K rest of line" → one pair, value = rest of line joined
	if allowLegacy {
		first := firstField(s)
		if !strings.Contains(first, "=") {
			rest := strings.TrimSpace(s[len(first):])
			if rest == "" {
				return nil, fmt.Errorf("ENV legacy form requires a value")
			}
			return &KVInstr{Pairs: []KV{{Key: first, Value: rest}}}, nil
		}
	}
	pairs, err := tokenizeKVPairs(s)
	if err != nil {
		return nil, err
	}
	return &KVInstr{Pairs: pairs}, nil
}

func parseArg(s string) *ArgInstr {
	s = strings.TrimSpace(s)
	eq := strings.IndexByte(s, '=')
	if eq < 0 {
		return &ArgInstr{Name: s}
	}
	return &ArgInstr{Name: s[:eq], Default: s[eq+1:], HasDef: true}
}

// tokenizeKVPairs handles space-separated KEY=VAL pairs with optional quoted values:
//
//	FOO=bar BAZ="hello world" QUX='a b'
func tokenizeKVPairs(s string) ([]KV, error) {
	var pairs []KV
	i := 0
	for i < len(s) {
		// Skip whitespace
		for i < len(s) && (s[i] == ' ' || s[i] == '\t') {
			i++
		}
		if i >= len(s) {
			break
		}
		// Key: read until '='
		keyStart := i
		for i < len(s) && s[i] != '=' && s[i] != ' ' && s[i] != '\t' {
			i++
		}
		if i >= len(s) || s[i] != '=' {
			return nil, fmt.Errorf("expected '=' after key %q", s[keyStart:i])
		}
		key := s[keyStart:i]
		i++ // skip '='
		// Value: quoted or bare
		val, n, err := readValue(s[i:])
		if err != nil {
			return nil, err
		}
		i += n
		pairs = append(pairs, KV{Key: key, Value: val})
	}
	if len(pairs) == 0 {
		return nil, fmt.Errorf("expected key=value pairs")
	}
	return pairs, nil
}

func readValue(s string) (string, int, error) {
	if len(s) == 0 {
		return "", 0, nil
	}
	if s[0] == '"' || s[0] == '\'' {
		quote := s[0]
		var b strings.Builder
		i := 1
		for i < len(s) {
			c := s[i]
			if c == '\\' && i+1 < len(s) && quote == '"' {
				// backslash escape only meaningful inside double quotes
				b.WriteByte(s[i+1])
				i += 2
				continue
			}
			if c == quote {
				return b.String(), i + 1, nil
			}
			b.WriteByte(c)
			i++
		}
		return "", 0, fmt.Errorf("unterminated %c-quoted value", quote)
	}
	i := 0
	for i < len(s) && s[i] != ' ' && s[i] != '\t' {
		i++
	}
	return s[:i], i, nil
}

func firstField(s string) string {
	for i := 0; i < len(s); i++ {
		if s[i] == ' ' || s[i] == '\t' {
			return s[:i]
		}
	}
	return s
}

func fieldsNonEmpty(s string) []string {
	return strings.Fields(s)
}
