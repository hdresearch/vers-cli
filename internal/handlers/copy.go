package handlers

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hdresearch/vers-cli/internal/app"
	"github.com/hdresearch/vers-cli/internal/presenters"
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

	vmInfo := utils.CreateVMInfoFromVM(*info.VM)
	v.VMName = vmInfo.DisplayName

	// Use VM ID as host for SSH-over-TLS
	sshHost := info.Host

	var localPath, remotePath string
	var isUpload bool

	// Determine transfer direction
	if strings.HasPrefix(r.Source, "/") && !strings.HasPrefix(r.Destination, "/") {
		// Remote -> Local (source starts with /, dest doesn't)
		remotePath = r.Source
		localPath = r.Destination
		isUpload = false
		v.Action = "Downloading"
	} else if !strings.HasPrefix(r.Source, "/") && strings.HasPrefix(r.Destination, "/") {
		// Local -> Remote (source doesn't start with /, dest does)
		localPath = r.Source
		remotePath = r.Destination
		isUpload = true
		v.Action = "Uploading"
	} else {
		// Ambiguous - check if source exists locally
		if _, err := os.Stat(r.Source); err == nil {
			localPath = r.Source
			remotePath = r.Destination
			isUpload = true
			v.Action = "Uploading"
		} else {
			remotePath = r.Source
			localPath = r.Destination
			isUpload = false
			v.Action = "Downloading"
		}
	}

	// Expand ~ and clean paths for nicer output
	v.Src = r.Source
	v.Dest = r.Destination
	if isUpload {
		v.Src = expandTilde(localPath)
	} else {
		v.Dest = expandTilde(localPath)
	}

	// Use native SFTP client
	client := sshutil.NewClient(sshHost, info.KeyPath)
	if isUpload {
		err = client.Upload(ctx, expandTilde(localPath), remotePath, r.Recursive)
	} else {
		err = client.Download(ctx, remotePath, expandTilde(localPath), r.Recursive)
	}
	if err != nil {
		return v, fmt.Errorf("file transfer failed: %w", err)
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
