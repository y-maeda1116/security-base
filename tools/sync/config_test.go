package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantErr bool
		check   func(t *testing.T, cfg *Config)
	}{
		{
			name: "valid config",
			content: `source:
  owner: y-maeda1116
  repo: security-base
  branch: main
files:
  - src: .github/workflows/ci.yml
    dst: .github/workflows/ci.yml
targets:
  - owner: y-maeda1116
    repo: python-template-base
    branch_prefix: sync/security-base
pr:
  title_prefix: "chore: sync security-base"
`,
			wantErr: false,
			check: func(t *testing.T, cfg *Config) {
				if cfg.Source.Owner != "y-maeda1116" {
					t.Errorf("Source.Owner = %q, want %q", cfg.Source.Owner, "y-maeda1116")
				}
				if cfg.Source.Repo != "security-base" {
					t.Errorf("Source.Repo = %q, want %q", cfg.Source.Repo, "security-base")
				}
				if len(cfg.Files) != 1 {
					t.Fatalf("len(Files) = %d, want 1", len(cfg.Files))
				}
				if cfg.Files[0].Src != ".github/workflows/ci.yml" {
					t.Errorf("Files[0].Src = %q, want %q", cfg.Files[0].Src, ".github/workflows/ci.yml")
				}
				if len(cfg.Targets) != 1 {
					t.Fatalf("len(Targets) = %d, want 1", len(cfg.Targets))
				}
				if cfg.Targets[0].Repo != "python-template-base" {
					t.Errorf("Targets[0].Repo = %q", cfg.Targets[0].Repo)
				}
				if cfg.PR.TitlePrefix != "chore: sync security-base" {
					t.Errorf("PR.TitlePrefix = %q", cfg.PR.TitlePrefix)
				}
			},
		},
		{
			name:    "empty config",
			content: ``,
			wantErr: true,
		},
		{
			name: "missing source",
			content: `files:
  - src: a.yml
    dst: a.yml
targets:
  - owner: o
    repo: r
    branch_prefix: sync/sb
`,
			wantErr: true,
		},
		{
			name: "missing targets",
			content: `source:
  owner: y-maeda1116
  repo: security-base
  branch: main
files:
  - src: a.yml
    dst: a.yml
`,
			wantErr: true,
		},
		{
			name: "missing files",
			content: `source:
  owner: y-maeda1116
  repo: security-base
  branch: main
targets:
  - owner: o
    repo: r
    branch_prefix: sync/sb
`,
			wantErr: true,
		},
		{
			name: "multiple files and targets",
			content: `source:
  owner: y-maeda1116
  repo: security-base
  branch: main
files:
  - src: a.yml
    dst: a.yml
  - src: b.yml
    dst: b.yml
targets:
  - owner: y-maeda1116
    repo: repo-a
    branch_prefix: sync/sb
  - owner: y-maeda1116
    repo: repo-b
    branch_prefix: sync/sb
pr:
  title_prefix: "sync"
`,
			wantErr: false,
			check: func(t *testing.T, cfg *Config) {
				if len(cfg.Files) != 2 {
					t.Errorf("len(Files) = %d, want 2", len(cfg.Files))
				}
				if len(cfg.Targets) != 2 {
					t.Errorf("len(Targets) = %d, want 2", len(cfg.Targets))
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "config.yaml")
			if err := os.WriteFile(path, []byte(tt.content), 0644); err != nil {
				t.Fatalf("write config: %v", err)
			}

			cfg, err := LoadConfig(path)
			if (err != nil) != tt.wantErr {
				t.Errorf("LoadConfig() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err == nil && tt.check != nil {
				tt.check(t, cfg)
			}
		})
	}
}

func TestLoadConfigFileNotFound(t *testing.T) {
	_, err := LoadConfig("/nonexistent/config.yaml")
	if err == nil {
		t.Error("LoadConfig() should return error for missing file")
	}
}

func TestConfigDefaults(t *testing.T) {
	dir := t.TempDir()
	content := `source:
  owner: y-maeda1116
  repo: security-base
  branch: main
files:
  - src: a.yml
    dst: a.yml
targets:
  - owner: o
    repo: r
    branch_prefix: sync/sb
`
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	// PR section has default title_prefix
	if cfg.PR.TitlePrefix != "" {
		t.Errorf("PR.TitlePrefix = %q, want empty (not set)", cfg.PR.TitlePrefix)
	}
}
