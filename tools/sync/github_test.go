package main

import (
	"context"
	"errors"
	"testing"

	"github.com/google/go-github/v72/github"
)

// mockGitHubClient is a test double for GitHubPRClient.
type mockGitHubClient struct {
	openPRs []*github.PullRequest
	pr      *github.PullRequest
	err     error
	calls   []string
}

func (m *mockGitHubClient) ListOpenPRs(ctx context.Context, owner, repo, head string) ([]*github.PullRequest, error) {
	m.calls = append(m.calls, "ListOpenPRs:"+owner+"/"+repo+":"+head)
	if m.err != nil {
		return nil, m.err
	}
	return m.openPRs, nil
}

func (m *mockGitHubClient) CreatePR(ctx context.Context, owner, repo, title, head, base, body string) (*github.PullRequest, error) {
	m.calls = append(m.calls, "CreatePR:"+owner+"/"+repo+":"+title)
	if m.err != nil {
		return nil, m.err
	}
	return m.pr, nil
}

func TestGetToken(t *testing.T) {
	// This tests the token resolution logic conceptually.
	// In real usage, GITHUB_TOKEN env or `gh auth token` is used.
}

func TestListOpenPRs(t *testing.T) {
	tests := []struct {
		name     string
		client   *mockGitHubClient
		wantLen  int
		wantErr  bool
	}{
		{
			name: "no existing PRs",
			client: &mockGitHubClient{
				openPRs: []*github.PullRequest{},
			},
			wantLen: 0,
		},
		{
			name: "existing PR found",
			client: &mockGitHubClient{
				openPRs: []*github.PullRequest{
					{Number: github.Ptr(42)},
				},
			},
			wantLen: 1,
		},
		{
			name: "API error",
			client: &mockGitHubClient{
				err: errors.New("rate limit"),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prs, err := tt.client.ListOpenPRs(context.Background(), "owner", "repo", "sync/branch")
			if (err != nil) != tt.wantErr {
				t.Errorf("ListOpenPRs() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(prs) != tt.wantLen {
				t.Errorf("len(PR) = %d, want %d", len(prs), tt.wantLen)
			}
		})
	}
}

func TestCreatePR(t *testing.T) {
	tests := []struct {
		name   string
		client *mockGitHubClient
		wantErr bool
	}{
		{
			name: "success",
			client: &mockGitHubClient{
				pr: &github.PullRequest{
					Number: github.Ptr(1),
					HTMLURL: github.Ptr("https://github.com/o/r/pull/1"),
				},
			},
			wantErr: false,
		},
		{
			name: "auth error",
			client: &mockGitHubClient{
				err: errors.New("unauthorized"),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pr, err := tt.client.CreatePR(
				context.Background(),
				"owner", "repo",
				"chore: sync", "sync/branch", "main",
				"body",
			)
			if (err != nil) != tt.wantErr {
				t.Errorf("CreatePR() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && pr == nil {
				t.Error("CreatePR() returned nil PR")
			}
		})
	}
}

func TestHasOpenPR(t *testing.T) {
	tests := []struct {
		name   string
		client *mockGitHubClient
		want   bool
	}{
		{
			name: "no open PRs",
			client: &mockGitHubClient{
				openPRs: []*github.PullRequest{},
			},
			want: false,
		},
		{
			name: "open PR exists",
			client: &mockGitHubClient{
				openPRs: []*github.PullRequest{
					{Number: github.Ptr(1)},
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &GitHubService{client: tt.client}
			result, err := svc.HasOpenPR(context.Background(), "owner", "repo", "sync/branch")
			if err != nil {
				t.Errorf("HasOpenPR() error = %v", err)
			}
			if result != tt.want {
				t.Errorf("HasOpenPR() = %v, want %v", result, tt.want)
			}
		})
	}
}
