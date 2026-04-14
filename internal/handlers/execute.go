package handlers

import (
	"bufio"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"

	"github.com/hdresearch/vers-cli/internal/app"
	"github.com/hdresearch/vers-cli/internal/presenters"
	vmSvc "github.com/hdresearch/vers-cli/internal/services/vm"
	sshutil "github.com/hdresearch/vers-cli/internal/ssh"
	"github.com/hdresearch/vers-cli/internal/utils"
	"golang.org/x/crypto/ssh"
)

type ExecuteReq struct {
	Target     string
	Command    []string
	WorkingDir string
	Env        map[string]string
	TimeoutSec uint64
	UseSSH     bool
	Stdin      string
}

// streamResponse represents a single NDJSON line from the exec stream.
// The orchestrator flattens the agent protocol into:
//
//	{"type":"chunk","stream":"stdout","data_b64":"...","cursor":N,"exec_id":"..."}
//	{"type":"exit","exit_code":0,"cursor":N,"exec_id":"..."}
//	{"type":"error","code":"...","message":"..."}
type streamResponse struct {
	Type     string `json:"type"`
	Stream   string `json:"stream,omitempty"`
	DataB64  string `json:"data_b64,omitempty"`
	ExitCode *int   `json:"exit_code,omitempty"`
	Cursor   uint64 `json:"cursor,omitempty"`
	Code     string `json:"code,omitempty"`
	Message  string `json:"message,omitempty"`
}

func HandleExecute(ctx context.Context, a *app.App, r ExecuteReq) (presenters.ExecuteView, error) {
	v := presenters.ExecuteView{}

	t, err := utils.ResolveTarget(r.Target)
	if err != nil {
		return v, err
	}
	v.UsedHEAD = t.UsedHEAD
	v.HeadID = t.HeadID

	if r.UseSSH {
		return handleExecuteSSH(ctx, a, r, t, v)
	}

	return handleExecuteAPI(ctx, a, r, t, v)
}

// handleExecuteAPI runs the command via the orchestrator exec/stream API.
func handleExecuteAPI(ctx context.Context, a *app.App, r ExecuteReq, t utils.TargetResult, v presenters.ExecuteView) (presenters.ExecuteView, error) {
	// Wrap the command in bash -c so shell features work
	command := []string{"bash", "-c", utils.ShellJoin(r.Command)}

	body, err := vmSvc.ExecStream(ctx, t.Ident, vmSvc.ExecRequest{
		Command:    command,
		Env:        r.Env,
		WorkingDir: r.WorkingDir,
		Stdin:      r.Stdin,
		TimeoutSec: r.TimeoutSec,
	})
	if err != nil {
		return v, fmt.Errorf("exec: %w", err)
	}
	defer body.Close()

	exitCode, err := streamExecOutput(body, a.IO.Out, a.IO.Err)
	if err != nil {
		return v, fmt.Errorf("exec stream: %w", err)
	}

	v.ExitCode = exitCode
	return v, nil
}

// handleExecuteSSH runs the command via direct SSH (legacy fallback).
func handleExecuteSSH(ctx context.Context, a *app.App, r ExecuteReq, t utils.TargetResult, v presenters.ExecuteView) (presenters.ExecuteView, error) {
	info, err := vmSvc.GetConnectInfo(ctx, a.Client, t.Ident)
	if err != nil {
		return v, fmt.Errorf("failed to get VM information: %w", err)
	}

	cmdStr := utils.ShellJoin(r.Command)
	client := sshutil.NewClient(info.Host, info.KeyPath, info.VMDomain)

	if r.Stdin != "" {
		return handleExecuteSSHWithStdin(ctx, client, cmdStr, r.Stdin, a, v)
	}

	err = client.Execute(ctx, cmdStr, a.IO.Out, a.IO.Err)
	if err != nil {
		if exitErr, ok := err.(*ssh.ExitError); ok {
			v.ExitCode = exitErr.ExitStatus()
			return v, nil
		}
		return v, fmt.Errorf("failed to execute command: %w", err)
	}
	return v, nil
}

// handleExecuteSSHWithStdin runs a command via SSH, piping stdin data to the remote process.
func handleExecuteSSHWithStdin(ctx context.Context, client *sshutil.Client, cmd, stdinData string, a *app.App, v presenters.ExecuteView) (presenters.ExecuteView, error) {
	sess, err := client.StartSession(ctx)
	if err != nil {
		return v, fmt.Errorf("failed to start SSH session: %w", err)
	}
	defer sess.Close()

	// Copy stdout/stderr in background
	go io.Copy(a.IO.Out, sess.Stdout())
	go io.Copy(a.IO.Err, sess.Stderr())

	if err := sess.Start(cmd); err != nil {
		return v, fmt.Errorf("failed to start command: %w", err)
	}

	// Write stdin data and close to signal EOF
	if _, err := io.WriteString(sess.Stdin(), stdinData); err != nil {
		return v, fmt.Errorf("failed to write stdin: %w", err)
	}
	sess.Stdin().Close()

	err = sess.Wait()
	if err != nil {
		if exitErr, ok := err.(*ssh.ExitError); ok {
			v.ExitCode = exitErr.ExitStatus()
			return v, nil
		}
		return v, fmt.Errorf("command failed: %w", err)
	}
	return v, nil
}

// streamExecOutput reads NDJSON from the exec stream, writes stdout/stderr
// to the provided writers, and returns the exit code.
func streamExecOutput(body io.Reader, stdout, stderr io.Writer) (int, error) {
	scanner := bufio.NewScanner(body)
	// Allow large lines (agent can send up to 10MB of output)
	scanner.Buffer(make([]byte, 0, 64*1024), 16*1024*1024)

	exitCode := 0

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var resp streamResponse
		if err := json.Unmarshal(line, &resp); err != nil {
			// Skip unparseable lines
			continue
		}

		switch resp.Type {
		case "chunk":
			data, err := base64.StdEncoding.DecodeString(resp.DataB64)
			if err != nil {
				continue
			}
			switch resp.Stream {
			case "stdout":
				stdout.Write(data)
			case "stderr":
				stderr.Write(data)
			}

		case "exit":
			if resp.ExitCode != nil {
				exitCode = *resp.ExitCode
			}
			return exitCode, nil

		case "error":
			return 1, fmt.Errorf("exec error [%s]: %s", resp.Code, resp.Message)
		}
	}

	if err := scanner.Err(); err != nil {
		return 1, fmt.Errorf("stream read error: %w", err)
	}

	return exitCode, nil
}
