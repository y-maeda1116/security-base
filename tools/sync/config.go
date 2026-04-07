package main

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Source struct {
	Owner  string `yaml:"owner"`
	Repo   string `yaml:"repo"`
	Branch string `yaml:"branch"`
}

type FileMapping struct {
	Src string `yaml:"src"`
	Dst string `yaml:"dst"`
}

type Target struct {
	Owner        string `yaml:"owner"`
	Repo         string `yaml:"repo"`
	BranchPrefix string `yaml:"branch_prefix"`
}

type PRConfig struct {
	TitlePrefix string `yaml:"title_prefix"`
	BodyTemplate string `yaml:"body_template"`
}

type Config struct {
	Source  Source        `yaml:"source"`
	Files   []FileMapping `yaml:"files"`
	Targets []Target      `yaml:"targets"`
	PR      PRConfig      `yaml:"pr"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *Config) validate() error {
	if c.Source.Owner == "" || c.Source.Repo == "" || c.Source.Branch == "" {
		return fmt.Errorf("source owner, repo, and branch are required")
	}
	if len(c.Files) == 0 {
		return fmt.Errorf("at least one file mapping is required")
	}
	for i, f := range c.Files {
		if f.Src == "" || f.Dst == "" {
			return fmt.Errorf("files[%d]: src and dst are required", i)
		}
	}
	if len(c.Targets) == 0 {
		return fmt.Errorf("at least one target is required")
	}
	for i, t := range c.Targets {
		if t.Owner == "" || t.Repo == "" || t.BranchPrefix == "" {
			return fmt.Errorf("targets[%d]: owner, repo, and branch_prefix are required", i)
		}
	}
	return nil
}
