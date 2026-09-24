package main

import (
	"os"
	"path/filepath"
	"testing"
)

// FuzzLoadConfig は任意のYAML入力に対して LoadConfig が panic・ハングせず、
// エラーまたは検証済みの *Config を返すことを保証する。
// go test ./... では seed corpus のみ実行され、継続的な探索は
// go test -fuzz=FuzzLoadConfig -fuzztime=30s で行う。
func FuzzLoadConfig(f *testing.F) {
	f.Add([]byte("source:\n  owner: o\n  repo: r\n  branch: b\nfiles:\n  - src: a\n    dst: b\ntargets:\n  - owner: o\n    repo: r\n    branch_prefix: p\n"))
	f.Add([]byte(""))
	f.Add([]byte("not: [valid: yaml"))
	f.Add([]byte("files:\n  - src: a\ntargets: []\n"))
	f.Add([]byte("source: 123\nfiles: \"string\"\n"))
	f.Add([]byte("source:\n  owner: o\n  repo: r\n  branch: b\nfiles:\n  - src: a\n    dst: b\ntargets:\n  - owner: o\n    repo: r\n    branch_prefix: p\npr:\n  title_prefix: \"chore: sync\"\n  body_template: \"{{.SHA}}\"\n"))

	f.Fuzz(func(t *testing.T, data []byte) {
		dir := t.TempDir()
		path := filepath.Join(dir, "config.yaml")
		if err := os.WriteFile(path, data, 0600); err != nil {
			t.Fatalf("write config: %v", err)
		}

		cfg, err := LoadConfig(path)
		if err != nil {
			// 不正な入力に対するエラーは正常系
			return
		}
		if cfg == nil {
			t.Fatal("LoadConfig() = (nil, nil)")
		}
		// エラーなしで返った場合は検証を通過しているはず
		if cfg.Source.Owner == "" || cfg.Source.Repo == "" || cfg.Source.Branch == "" {
			t.Fatalf("unvalidated config returned: %+v", cfg.Source)
		}
		if len(cfg.Files) == 0 || len(cfg.Targets) == 0 {
			t.Fatalf("unvalidated config returned: files=%d targets=%d", len(cfg.Files), len(cfg.Targets))
		}
	})
}
