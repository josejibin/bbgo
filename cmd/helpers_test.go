package cmd

import (
	"fmt"
	"testing"

	"github.com/josejibin/bbgo/internal/bitbucket"
)

func TestExitCodeForError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want int
	}{
		{"auth error", &bitbucket.AuthError{Msg: "auth"}, 2},
		{"not found error", &bitbucket.NotFoundError{Msg: "nf"}, 3},
		{"git not a repo", fmt.Errorf("not a git repo"), 5},
		{"git no remote", fmt.Errorf("no remote origin found"), 5},
		{"generic error", fmt.Errorf("something else"), 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ExitCodeForError(tt.err)
			if got != tt.want {
				t.Errorf("ExitCodeForError(%v) = %d, want %d", tt.err, got, tt.want)
			}
		})
	}
}

func TestTruncate(t *testing.T) {
	tests := []struct {
		input string
		max   int
		want  string
	}{
		{"hello", 10, "hello"},
		{"hello", 5, "hello"},
		{"hello world", 8, "hello..."},
		{"abc", 3, "abc"},
		{"abcd", 3, "..."},
	}
	for _, tt := range tests {
		t.Run(fmt.Sprintf("%s_%d", tt.input, tt.max), func(t *testing.T) {
			got := truncate(tt.input, tt.max)
			if got != tt.want {
				t.Errorf("truncate(%q, %d) = %q, want %q", tt.input, tt.max, got, tt.want)
			}
		})
	}
}

func TestDiffStatPath(t *testing.T) {
	tests := []struct {
		name string
		stat bitbucket.DiffStat
		want string
	}{
		{
			"new file",
			bitbucket.DiffStat{New: &bitbucket.DiffFile{Path: "new.go"}},
			"new.go",
		},
		{
			"old file only",
			bitbucket.DiffStat{Old: &bitbucket.DiffFile{Path: "old.go"}},
			"old.go",
		},
		{
			"both prefers new",
			bitbucket.DiffStat{
				New: &bitbucket.DiffFile{Path: "new.go"},
				Old: &bitbucket.DiffFile{Path: "old.go"},
			},
			"new.go",
		},
		{
			"neither",
			bitbucket.DiffStat{},
			"(unknown)",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := diffStatPath(tt.stat)
			if got != tt.want {
				t.Errorf("diffStatPath() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestRequireIntArg(t *testing.T) {
	tests := []struct {
		input string
		want  int
		err   bool
	}{
		{"42", 42, false},
		{"0", 0, false},
		{"abc", 0, true},
		{"", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			id, err := requireIntArg(testCLIContext([]string{tt.input}), "PR_ID")
			if tt.err {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if id != tt.want {
				t.Errorf("got %d, want %d", id, tt.want)
			}
		})
	}
}

func TestRequireIntArgRejectsTrailingText(t *testing.T) {
	_, err := requireIntArg(testCLIContext([]string{"42abc"}), "PR_ID")
	if err == nil {
		t.Fatal("expected error for malformed numeric argument")
	}
}

func TestParticipantStatus(t *testing.T) {
	tests := []struct {
		name string
		p    bitbucket.Participant
		want string
	}{
		{"approved", bitbucket.Participant{Approved: true}, "approved"},
		{"changes requested", bitbucket.Participant{State: "changes_requested"}, "changes_requested"},
		{"pending", bitbucket.Participant{}, "pending"},
		{"approved takes precedence", bitbucket.Participant{Approved: true, State: "changes_requested"}, "approved"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := participantStatus(tt.p)
			if got != tt.want {
				t.Errorf("participantStatus() = %q, want %q", got, tt.want)
			}
		})
	}
}
