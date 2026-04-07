package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/google/go-github/v72/github"
	"golang.org/x/oauth2"
)

// GitHubPRClient abstracts GitHub PR operations for testability.
type GitHubPRClient interface {
	ListOpenPRs(ctx context.Context, owner, repo, head string) ([]*github.PullRequest, error)
	CreatePR(ctx context.Context, owner, repo, title, head, base, body string) (*github.PullRequest, error)
}

// realGitHubClient wraps go-github client.
type realGitHubClient struct {
	client *github.Client
}

func (r *realGitHubClient) ListOpenPRs(ctx context.Context, owner, repo, head string) ([]*github.PullRequest, error) {
	opts := &github.PullRequestListOptions{
		State: "open",
		Head:  head,
	}
	prs, _, err := r.client.PullRequests.List(ctx, owner, repo, opts)
	if err != nil {
		return nil, fmt.Errorf("list PRs: %w", err)
	}
	return prs, nil
}

func (r *realGitHubClient) CreatePR(ctx context.Context, owner, repo, title, head, base, body string) (*github.PullRequest, error) {
	newPR := &github.NewPullRequest{
		Title: &title,
		Head:  &head,
		Base:  &base,
		Body:  &body,
	}
	pr, _, err := r.client.PullRequests.Create(ctx, owner, repo, newPR)
	if err != nil {
		return nil, fmt.Errorf("create PR: %w", err)
	}
	return pr, nil
}

// GitHubService provides high-level GitHub operations.
type GitHubService struct {
	client GitHubPRClient
}

// NewGitHubService creates a GitHubService with a real GitHub client.
func NewGitHubService(token string) *GitHubService {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(context.Background(), ts)
	client := github.NewClient(tc)
	return &GitHubService{client: &realGitHubClient{client: client}}
}

// HasOpenPR checks if there's already an open PR from the given head branch.
func (s *GitHubService) HasOpenPR(ctx context.Context, owner, repo, head string) (bool, error) {
	prs, err := s.client.ListOpenPRs(ctx, owner, repo, head)
	if err != nil {
		return false, err
	}
	return len(prs) > 0, nil
}

// CreatePullRequest creates a new pull request. Returns the PR URL.
func (s *GitHubService) CreatePullRequest(ctx context.Context, owner, repo, title, head, base, body string) (string, error) {
	pr, err := s.client.CreatePR(ctx, owner, repo, title, head, base, body)
	if err != nil {
		return "", err
	}
	if pr.HTMLURL != nil {
		return *pr.HTMLURL, nil
	}
	return "", nil
}

// GetToken resolves a GitHub token from environment or gh CLI.
func GetToken() (string, error) {
	if token := os.Getenv("GITHUB_TOKEN"); token != "" {
		return token, nil
	}
	out, err := exec.Command("gh", "auth", "token").Output()
	if err != nil {
		return "", fmt.Errorf("GITHUB_TOKEN not set and `gh auth token` failed: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}
