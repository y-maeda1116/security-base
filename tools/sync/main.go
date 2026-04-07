package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
)

func main() {
	configPath := flag.String("config", "config.yaml", "path to config.yaml")
	flag.Parse()

	cfg, err := LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	token, err := GetToken()
	if err != nil {
		log.Fatalf("get token: %v", err)
	}

	ctx := context.Background()
	gitClient := NewGitClient()
	ghService := NewGitHubService(ctx, token)

	syncer := NewSyncer(gitClient, ghService, cfg, "")

	results := syncer.SyncAll(ctx)

	fmt.Println("\n=== Sync Results ===")
	hasError := false
	for _, r := range results {
		if r.Error != nil {
			fmt.Printf("[ERROR] %s/%s: %v\n", r.Target.Owner, r.Target.Repo, r.Error)
			hasError = true
		} else if r.Skipped {
			fmt.Printf("[SKIP]  %s/%s: no changes or PR already exists\n", r.Target.Owner, r.Target.Repo)
		} else {
			fmt.Printf("[OK]    %s/%s: %s\n", r.Target.Owner, r.Target.Repo, r.PRURL)
		}
	}

	if hasError {
		os.Exit(1)
	}
}
