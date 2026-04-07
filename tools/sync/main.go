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

	gitClient := NewGitClient()
	ghService := NewGitHubService(token)

	syncer := NewSyncer(gitClient, ghService, cfg, "")

	results := syncer.SyncAll(context.Background())

	fmt.Println("\n=== Sync Results ===")
	for _, r := range results {
		if r.Error != nil {
			fmt.Printf("[ERROR] %s/%s: %v\n", r.Target.Owner, r.Target.Repo, r.Error)
		} else if r.Skipped {
			fmt.Printf("[SKIP]  %s/%s: no changes or PR already exists\n", r.Target.Owner, r.Target.Repo)
		} else {
			fmt.Printf("[OK]    %s/%s: %s\n", r.Target.Owner, r.Target.Repo, r.PRURL)
		}
	}

	// Exit 1 if any target failed
	for _, r := range results {
		if r.Error != nil {
			os.Exit(1)
		}
	}
}
