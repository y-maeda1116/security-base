package main

import (
	"errors"
	"strings"
	"testing"
)

// mockRunner is a test double for CommandRunner.
type mockRunner struct {
	responses map[string]mockResponse
	calls     []string
}

type mockResponse struct {
	output string
	err    error
}

func (m *mockRunner) Run(name string, args ...string) (string, error) {
	cmd := name + " " + strings.Join(args, " ")
	m.calls = append(m.calls, cmd)
	// exact match first
	if resp, ok := m.responses[cmd]; ok {
		return resp.output, resp.err
	}
	// prefix match by command name + first arg
	for key, resp := range m.responses {
		if strings.HasPrefix(cmd, key) {
			return resp.output, resp.err
		}
	}
	return "", nil
}

func TestGitClient_Clone(t *testing.T) {
	tests := []struct {
		name    string
		runner  *mockRunner
		wantErr bool
	}{
		{
			name: "success",
			runner: &mockRunner{
				responses: map[string]mockResponse{
					"git clone": {output: "", err: nil},
				},
			},
			wantErr: false,
		},
		{
			name: "network error",
			runner: &mockRunner{
				responses: map[string]mockResponse{
					"git clone": {output: "", err: errors.New("network error")},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &GitClient{runner: tt.runner}
			err := client.Clone("https://github.com/o/r.git", "/tmp/dest")
			if (err != nil) != tt.wantErr {
				t.Errorf("Clone() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGitClient_CheckoutBranch(t *testing.T) {
	tests := []struct {
		name    string
		runner  *mockRunner
		wantErr bool
	}{
		{
			name: "success",
			runner: &mockRunner{
				responses: map[string]mockResponse{
					"git -C": {output: "", err: nil},
				},
			},
			wantErr: false,
		},
		{
			name: "branch already exists",
			runner: &mockRunner{
				responses: map[string]mockResponse{
					"git -C": {output: "", err: errors.New("already exists")},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &GitClient{runner: tt.runner}
			err := client.CheckoutBranch("/tmp/dest", "sync/branch")
			if (err != nil) != tt.wantErr {
				t.Errorf("CheckoutBranch() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGitClient_HasChanges(t *testing.T) {
	tests := []struct {
		name       string
		runner     *mockRunner
		wantResult bool
	}{
		{
			name: "no changes",
			runner: &mockRunner{
				responses: map[string]mockResponse{
					"git -C": {output: "", err: nil},
				},
			},
			wantResult: false,
		},
		{
			name: "has changes",
			runner: &mockRunner{
				responses: map[string]mockResponse{
					"git -C": {output: "some diff", err: nil},
				},
			},
			wantResult: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &GitClient{runner: tt.runner}
			result, err := client.HasChanges("/tmp/dest")
			if err != nil {
				t.Errorf("HasChanges() error = %v", err)
			}
			if result != tt.wantResult {
				t.Errorf("HasChanges() = %v, want %v", result, tt.wantResult)
			}
		})
	}
}

func TestGitClient_Commit(t *testing.T) {
	runner := &mockRunner{
		responses: map[string]mockResponse{
			"git -C": {output: "", err: nil},
		},
	}
	client := &GitClient{runner: runner}

	err := client.Commit("/tmp/dest", "test commit message")
	if err != nil {
		t.Errorf("Commit() error = %v", err)
	}

	// Should have add and commit calls
	hasAdd := false
	hasCommit := false
	for _, call := range runner.calls {
		if strings.Contains(call, "add -A") {
			hasAdd = true
		}
		if strings.Contains(call, "commit -m") {
			hasCommit = true
		}
	}
	if !hasAdd {
		t.Error("expected 'git add -A' to be called")
	}
	if !hasCommit {
		t.Error("expected 'git commit -m' to be called")
	}
}

func TestGitClient_Push(t *testing.T) {
	tests := []struct {
		name    string
		runner  *mockRunner
		wantErr bool
	}{
		{
			name: "success",
			runner: &mockRunner{
				responses: map[string]mockResponse{
					"git -C": {output: "", err: nil},
				},
			},
			wantErr: false,
		},
		{
			name: "remote rejected",
			runner: &mockRunner{
				responses: map[string]mockResponse{
					"git -C": {output: "", err: errors.New("remote rejected")},
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := &GitClient{runner: tt.runner}
			err := client.Push("/tmp/dest", "origin", "sync/branch")
			if (err != nil) != tt.wantErr {
				t.Errorf("Push() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGitClient_GetSHA(t *testing.T) {
	runner := &mockRunner{
		responses: map[string]mockResponse{
			"git -C": {output: "abc123def456\n", err: nil},
		},
	}
	client := &GitClient{runner: runner}

	sha, err := client.GetSHA("/tmp/dest")
	if err != nil {
		t.Errorf("GetSHA() error = %v", err)
	}
	if sha != "abc123def456" {
		t.Errorf("GetSHA() = %q, want %q", sha, "abc123def456")
	}
}
