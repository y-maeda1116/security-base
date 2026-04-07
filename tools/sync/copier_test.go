package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestCopyFile(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T, srcDir, dstDir string)
		wantErr bool
		check   func(t *testing.T, dstDir string)
	}{
		{
			name: "normal copy",
			setup: func(t *testing.T, srcDir, dstDir string) {
				os.WriteFile(filepath.Join(srcDir, "a.yml"), []byte("content"), 0644)
			},
			wantErr: false,
			check: func(t *testing.T, dstDir string) {
				data, err := os.ReadFile(filepath.Join(dstDir, "a.yml"))
				if err != nil {
					t.Fatalf("read dst: %v", err)
				}
				if string(data) != "content" {
					t.Errorf("content = %q, want %q", string(data), "content")
				}
			},
		},
		{
			name: "src not found",
			setup: func(t *testing.T, srcDir, dstDir string) {
				// intentionally do not create source file
			},
			wantErr: true,
		},
		{
			name: "dst directory auto-created",
			setup: func(t *testing.T, srcDir, dstDir string) {
				os.WriteFile(filepath.Join(srcDir, "a.yml"), []byte("data"), 0644)
			},
			wantErr: false,
			check: func(t *testing.T, dstDir string) {
				sub := filepath.Join(dstDir, "sub", "dir")
				// copy to nested path
				srcFile := filepath.Join(filepath.Dir(dstDir), "src", "a.yml")
				dstFile := filepath.Join(sub, "a.yml")
				if err := CopyFile(srcFile, dstFile); err != nil {
					t.Fatalf("CopyFile to nested dir: %v", err)
				}
				data, err := os.ReadFile(dstFile)
				if err != nil {
					t.Fatalf("read: %v", err)
				}
				if string(data) != "data" {
					t.Errorf("content = %q, want %q", string(data), "data")
				}
			},
		},
		{
			name: "overwrite existing",
			setup: func(t *testing.T, srcDir, dstDir string) {
				os.WriteFile(filepath.Join(srcDir, "a.yml"), []byte("new"), 0644)
				os.WriteFile(filepath.Join(dstDir, "a.yml"), []byte("old"), 0644)
			},
			wantErr: false,
			check: func(t *testing.T, dstDir string) {
				data, err := os.ReadFile(filepath.Join(dstDir, "a.yml"))
				if err != nil {
					t.Fatalf("read: %v", err)
				}
				if string(data) != "new" {
					t.Errorf("content = %q, want %q", string(data), "new")
				}
			},
		},
		{
			name: "empty file",
			setup: func(t *testing.T, srcDir, dstDir string) {
				os.WriteFile(filepath.Join(srcDir, "empty.yml"), []byte(""), 0644)
			},
			wantErr: false,
			check: func(t *testing.T, dstDir string) {
				data, err := os.ReadFile(filepath.Join(dstDir, "empty.yml"))
				if err != nil {
					t.Fatalf("read: %v", err)
				}
				if len(data) != 0 {
					t.Errorf("len(data) = %d, want 0", len(data))
				}
			},
		},
		{
			name: "permission preserved",
			setup: func(t *testing.T, srcDir, dstDir string) {
				if runtime.GOOS == "windows" {
					t.Skip("permissions not meaningful on Windows")
				}
				os.WriteFile(filepath.Join(srcDir, "perm.yml"), []byte("p"), 0755)
			},
			wantErr: false,
			check: func(t *testing.T, dstDir string) {
				if runtime.GOOS == "windows" {
					return
				}
				info, err := os.Stat(filepath.Join(dstDir, "perm.yml"))
				if err != nil {
					t.Fatalf("stat: %v", err)
				}
				if info.Mode().Perm() != 0755 {
					t.Errorf("perm = %o, want %o", info.Mode().Perm(), 0755)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			base := t.TempDir()
			srcDir := filepath.Join(base, "src")
			dstDir := filepath.Join(base, "dst")
			os.MkdirAll(srcDir, 0755)
			os.MkdirAll(dstDir, 0755)

			tt.setup(t, srcDir, dstDir)

			// Find source files
			entries, _ := os.ReadDir(srcDir)
			for _, e := range entries {
				srcFile := filepath.Join(srcDir, e.Name())
				dstFile := filepath.Join(dstDir, e.Name())

				err := CopyFile(srcFile, dstFile)
				if (err != nil) != tt.wantErr {
					t.Errorf("CopyFile() error = %v, wantErr %v", err, tt.wantErr)
				}
			}

			if !tt.wantErr && tt.check != nil {
				// For "dst directory auto-created", the check handles its own copy
				if tt.name == "dst directory auto-created" {
					tt.check(t, dstDir)
				} else {
					tt.check(t, dstDir)
				}
			}
		})
	}
}

func TestCopyFiles(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	os.WriteFile(filepath.Join(srcDir, "a.yml"), []byte("aaa"), 0644)
	os.WriteFile(filepath.Join(srcDir, "b.yml"), []byte("bbb"), 0644)

	mappings := []FileMapping{
		{Src: "a.yml", Dst: "a.yml"},
		{Src: "b.yml", Dst: "sub/b.yml"},
	}

	copied, err := CopyFiles(srcDir, dstDir, mappings)
	if err != nil {
		t.Fatalf("CopyFiles() error = %v", err)
	}
	if len(copied) != 2 {
		t.Errorf("len(copied) = %d, want 2", len(copied))
	}

	data, _ := os.ReadFile(filepath.Join(dstDir, "a.yml"))
	if string(data) != "aaa" {
		t.Errorf("a.yml = %q, want %q", string(data), "aaa")
	}

	data, _ = os.ReadFile(filepath.Join(dstDir, "sub", "b.yml"))
	if string(data) != "bbb" {
		t.Errorf("sub/b.yml = %q, want %q", string(data), "bbb")
	}
}

func TestCopyFilesSrcMissing(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := t.TempDir()

	mappings := []FileMapping{
		{Src: "nonexistent.yml", Dst: "nonexistent.yml"},
	}

	_, err := CopyFiles(srcDir, dstDir, mappings)
	if err == nil {
		t.Error("CopyFiles() should return error for missing src")
	}
}
