package build

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// CreateWorkspaceTar creates a tar archive of the CWD, excluding .vers, vers.toml, and the tar file itself.
// It returns the tar bytes and a cleanup function for temporary file removal.
func CreateWorkspaceTar() ([]byte, func(), error) {
	tempFile, err := os.CreateTemp("", "vers-rootfs-*.tar")
	if err != nil {
		return nil, func() {}, fmt.Errorf("failed to create temporary file: %w", err)
	}
	cleanup := func() { _ = os.Remove(tempFile.Name()) }

	tw := tar.NewWriter(tempFile)
	// We close when reading
	// Walk the working directory
	workDir, err := os.Getwd()
	if err != nil {
		cleanup()
		return nil, func() {}, fmt.Errorf("failed to get working directory: %w", err)
	}
	err = filepath.Walk(workDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(workDir, path)
		if err != nil {
			return fmt.Errorf("failed to get relative path: %w", err)
		}
		if rel == ".vers" || strings.HasPrefix(rel, ".vers"+string(os.PathSeparator)) || rel == "vers.toml" || rel == tempFile.Name() {
			if info.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if rel == "." {
			return nil
		}
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return fmt.Errorf("failed to create tar header: %w", err)
		}
		header.Name = rel
		if err := tw.WriteHeader(header); err != nil {
			return fmt.Errorf("failed to write tar header: %w", err)
		}
		if info.Mode().IsRegular() {
			f, err := os.Open(path)
			if err != nil {
				return fmt.Errorf("failed to open file %s: %w", path, err)
			}
			defer f.Close()
			if _, err := io.Copy(tw, f); err != nil {
				return fmt.Errorf("failed to copy file contents: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		_ = tw.Close()
		cleanup()
		return nil, func() {}, err
	}
	_ = tw.Close()
	if _, err := tempFile.Seek(0, 0); err != nil {
		cleanup()
		return nil, func() {}, fmt.Errorf("failed to reset file pointer: %w", err)
	}
	bytes, err := os.ReadFile(tempFile.Name())
	if err != nil {
		cleanup()
		return nil, func() {}, fmt.Errorf("failed to read tar file: %w", err)
	}
	return bytes, cleanup, nil
}
