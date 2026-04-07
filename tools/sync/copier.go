package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// CopyFile copies a single file from src to dst, preserving permissions.
// It creates destination directories as needed.
func CopyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open source: %w", err)
	}
	defer srcFile.Close()

	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("create dst dir: %w", err)
	}

	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("open dest: %w", err)
	}
	defer dstFile.Close()

	if _, err := io.Copy(dstFile, srcFile); err != nil {
		return fmt.Errorf("copy data: %w", err)
	}

	// Preserve source permissions
	srcInfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("stat source: %w", err)
	}
	if err := os.Chmod(dst, srcInfo.Mode().Perm()); err != nil {
		return fmt.Errorf("chmod dest: %w", err)
	}

	return nil
}

// CopyFiles copies all mapped files from srcDir to dstDir.
// Returns list of successfully copied relative destination paths.
func CopyFiles(srcDir, dstDir string, mappings []FileMapping) ([]string, error) {
	var copied []string

	for _, m := range mappings {
		srcPath := filepath.Join(srcDir, m.Src)
		dstPath := filepath.Join(dstDir, m.Dst)

		if err := CopyFile(srcPath, dstPath); err != nil {
			return nil, fmt.Errorf("copy %s → %s: %w", m.Src, m.Dst, err)
		}
		copied = append(copied, m.Dst)
	}

	return copied, nil
}
