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
	ListOpenPRs(ctx context.Context, owner, repo string) ([]*github.PullRequest, error)
	CreatePR(ctx context.Context, owner, repo, title, head, base, body string) (*github.PullRequest, error)
}

// realGitHubClient wraps go-github client.
type realGitHubClient struct {
	client *github.Client
}

func (r *realGitHubClient) ListOpenPRs(ctx context.Context, owner, repo string) ([]*github.PullRequest, error) {
	opts := &github.PullRequestListOptions{
		State:       "open",
		ListOptions: github.ListOptions{PerPage: 100},
	}
	var all []*github.PullRequest
	for {
		prs, resp, err := r.client.PullRequests.List(ctx, owner, repo, opts)
		if err != nil {
			return nil, fmt.Errorf("list PRs: %w", err)
		}
		all = append(all, prs...)
		if resp.NextPage == 0 {
			return all, nil
		}
		opts.Page = resp.NextPage
	}
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
func NewGitHubService(ctx context.Context, token string) *GitHubService {
	ts := oauth2.StaticTokenSource(&oauth2.Token{AccessToken: token})
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)
	return &GitHubService{client: &realGitHubClient{client: client}}
}

// HasOpenSyncPR checks if there's already an open PR whose head branch was
// created by this tool (<branchPrefix>-<suffix>) in the target repository itself.
// Sync branches carry a unique suffix per run, so an exact head match never hits.
func (s *GitHubService) HasOpenSyncPR(ctx context.Context, owner, repo, branchPrefix string) (bool, error) {
	prs, err := s.client.ListOpenPRs(ctx, owner, repo)
	if err != nil {
		return false, err
	}
	for _, pr := range prs {
		head := pr.GetHead()
		if head.GetRepo().GetOwner().GetLogin() != owner {
			continue // fork からの PR は対象外
		}
		if strings.HasPrefix(head.GetRef(), branchPrefix+"-") {
			return true, nil
		}
	}
	return false, nil
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
	return "", fmt.Errorf("PR created for %s/%s but HTMLURL was nil in response", owner, repo)
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
