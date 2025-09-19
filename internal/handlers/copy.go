package handlers

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/hdresearch/vers-cli/internal/app"
	"github.com/hdresearch/vers-cli/internal/presenters"
	runrt "github.com/hdresearch/vers-cli/internal/runtime"
	vmSvc "github.com/hdresearch/vers-cli/internal/services/vm"
	sshutil "github.com/hdresearch/vers-cli/internal/ssh"
	"github.com/hdresearch/vers-cli/internal/utils"
)

type CopyReq struct {
	Target      string
	Source      string
	Destination string
	Recursive   bool
}

func HandleCopy(ctx context.Context, a *app.App, r CopyReq) (presenters.CopyView, error) {
	v := presenters.CopyView{}
	ident := r.Target
	if ident == "" {
		head, err := utils.GetCurrentHeadVM()
		if err != nil {
			return v, fmt.Errorf("no VM ID provided and %w", err)
		}
		ident = head
		v.UsedHEAD = true
		v.HeadID = head
	}

	info, err := vmSvc.GetConnectInfo(ctx, a.Client, ident)
	if err != nil {
		return v, fmt.Errorf("failed to get VM information: %w", err)
	}

	vmInfo := utils.CreateVMInfoFromGetResponse(info.VM)
	v.VMName = vmInfo.DisplayName
	if info.VM.State != "Running" {
		return v, fmt.Errorf("VM is not running (current state: %s)", info.VM.State)
	}
	if info.VM.NetworkInfo.SSHPort == 0 {
		return v, fmt.Errorf("VM does not have SSH port information available")
	}

	versHost := info.Host
	sshHost := versHost
	sshPort := fmt.Sprintf("%d", info.VM.NetworkInfo.SSHPort)
	if utils.IsHostLocal(versHost) {
		sshHost = info.VM.IPAddress
		sshPort = "22"
	}

	scpTarget := fmt.Sprintf("root@%s", sshHost)
	var scpSource, scpDest string

	// Determine transfer direction
	if strings.HasPrefix(r.Source, "/") && !strings.HasPrefix(r.Destination, "/") {
		// Remote -> Local
		scpSource = fmt.Sprintf("%s:%s", scpTarget, r.Source)
		scpDest = r.Destination
		v.Action = "Downloading"
	} else if !strings.HasPrefix(r.Source, "/") && strings.HasPrefix(r.Destination, "/") {
		// Local -> Remote
		scpSource = r.Source
		scpDest = fmt.Sprintf("%s:%s", scpTarget, r.Destination)
		v.Action = "Uploading"
	} else {
		if _, err := os.Stat(r.Source); err == nil {
			scpSource = r.Source
			scpDest = fmt.Sprintf("%s:%s", scpTarget, r.Destination)
			v.Action = "Uploading"
		} else {
			scpSource = fmt.Sprintf("%s:%s", scpTarget, r.Source)
			scpDest = r.Destination
			v.Action = "Downloading"
		}
	}

	// Expand ~ and clean paths for local files for nicer output
	v.Src = scpSource
	v.Dest = scpDest
	if !strings.Contains(scpSource, ":") {
		v.Src = expandTilde(scpSource)
	}
	if !strings.Contains(scpDest, ":") {
		v.Dest = expandTilde(scpDest)
	}

	args := sshutil.SCPArgs(sshPort, info.KeyPath, r.Recursive)
	args = append(args, scpSource, scpDest)
	err = a.Runner.Run(ctx, "scp", args, runrt.Stdio{Out: a.IO.Out, Err: a.IO.Err})
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return v, fmt.Errorf("scp command exited with code %d", ee.ExitCode())
		}
		return v, fmt.Errorf("failed to run SCP command: %w", err)
	}
	return v, nil
}

func expandTilde(p string) string {
	if strings.HasPrefix(p, "~") {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, strings.TrimPrefix(p, "~"))
		}
	}
	return p
}
