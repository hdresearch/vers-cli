package ssh

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/pkg/sftp"
)

// Upload copies a local file or directory to the remote host.
func (c *Client) Upload(ctx context.Context, localPath, remotePath string, recursive bool) error {
	client, err := c.Connect(ctx)
	if err != nil {
		return err
	}
	defer client.Close()

	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		return fmt.Errorf("SFTP client: %w", err)
	}
	defer sftpClient.Close()

	// Check if local path is a directory
	info, err := os.Stat(localPath)
	if err != nil {
		return fmt.Errorf("stat local path: %w", err)
	}

	if info.IsDir() {
		if !recursive {
			return fmt.Errorf("source is a directory, use recursive mode")
		}
		return uploadDir(sftpClient, localPath, remotePath)
	}

	return uploadFile(sftpClient, localPath, remotePath)
}

// uploadFile copies a single file to the remote host.
func uploadFile(sftpClient *sftp.Client, localPath, remotePath string) error {
	localFile, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("open local file: %w", err)
	}
	defer localFile.Close()

	// Get local file info for permissions
	info, err := localFile.Stat()
	if err != nil {
		return fmt.Errorf("stat local file: %w", err)
	}

	// Create remote file
	remoteFile, err := sftpClient.Create(remotePath)
	if err != nil {
		return fmt.Errorf("create remote file: %w", err)
	}
	defer remoteFile.Close()

	// Copy contents
	if _, err := io.Copy(remoteFile, localFile); err != nil {
		return fmt.Errorf("copy file: %w", err)
	}

	// Set permissions
	if err := sftpClient.Chmod(remotePath, info.Mode()); err != nil {
		// Non-fatal: some systems may not support all permission bits
		_ = err
	}

	return nil
}

// uploadDir recursively copies a directory to the remote host.
func uploadDir(sftpClient *sftp.Client, localDir, remoteDir string) error {
	return filepath.Walk(localDir, func(localPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Calculate relative path
		relPath, err := filepath.Rel(localDir, localPath)
		if err != nil {
			return fmt.Errorf("relative path: %w", err)
		}

		// Build remote path (use forward slashes for remote)
		remotePath := remoteDir + "/" + filepath.ToSlash(relPath)

		if info.IsDir() {
			// Create remote directory
			if err := sftpClient.MkdirAll(remotePath); err != nil {
				return fmt.Errorf("mkdir %s: %w", remotePath, err)
			}
			return nil
		}

		// Upload file
		return uploadFile(sftpClient, localPath, remotePath)
	})
}

// Download copies a remote file or directory to the local host.
func (c *Client) Download(ctx context.Context, remotePath, localPath string, recursive bool) error {
	client, err := c.Connect(ctx)
	if err != nil {
		return err
	}
	defer client.Close()

	sftpClient, err := sftp.NewClient(client)
	if err != nil {
		return fmt.Errorf("SFTP client: %w", err)
	}
	defer sftpClient.Close()

	// Check if remote path is a directory
	info, err := sftpClient.Stat(remotePath)
	if err != nil {
		return fmt.Errorf("stat remote path: %w", err)
	}

	if info.IsDir() {
		if !recursive {
			return fmt.Errorf("source is a directory, use recursive mode")
		}
		return downloadDir(sftpClient, remotePath, localPath)
	}

	return downloadFile(sftpClient, remotePath, localPath)
}

// downloadFile copies a single file from the remote host.
func downloadFile(sftpClient *sftp.Client, remotePath, localPath string) error {
	remoteFile, err := sftpClient.Open(remotePath)
	if err != nil {
		return fmt.Errorf("open remote file: %w", err)
	}
	defer remoteFile.Close()

	// Get remote file info for permissions
	info, err := remoteFile.Stat()
	if err != nil {
		return fmt.Errorf("stat remote file: %w", err)
	}

	// Create local file
	localFile, err := os.Create(localPath)
	if err != nil {
		return fmt.Errorf("create local file: %w", err)
	}
	defer localFile.Close()

	// Copy contents
	if _, err := io.Copy(localFile, remoteFile); err != nil {
		return fmt.Errorf("copy file: %w", err)
	}

	// Set permissions
	if err := os.Chmod(localPath, info.Mode()); err != nil {
		// Non-fatal
		_ = err
	}

	return nil
}

// downloadDir recursively copies a directory from the remote host.
func downloadDir(sftpClient *sftp.Client, remoteDir, localDir string) error {
	// Create local base directory
	if err := os.MkdirAll(localDir, 0755); err != nil {
		return fmt.Errorf("mkdir %s: %w", localDir, err)
	}

	// Walk remote directory
	walker := sftpClient.Walk(remoteDir)
	for walker.Step() {
		if err := walker.Err(); err != nil {
			return err
		}

		remotePath := walker.Path()
		info := walker.Stat()

		// Calculate relative path
		relPath, err := filepath.Rel(remoteDir, remotePath)
		if err != nil {
			return fmt.Errorf("relative path: %w", err)
		}

		// Build local path
		localPath := filepath.Join(localDir, relPath)

		if info.IsDir() {
			// Create local directory
			if err := os.MkdirAll(localPath, info.Mode()); err != nil {
				return fmt.Errorf("mkdir %s: %w", localPath, err)
			}
			continue
		}

		// Download file
		if err := downloadFile(sftpClient, remotePath, localPath); err != nil {
			return err
		}
	}

	return nil
}
