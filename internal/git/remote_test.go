package git

import "testing"

func TestParseRemoteURL(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		workspace string
		repo      string
		wantErr   bool
	}{
		{
			name:      "SSH",
			url:       "git@bitbucket.org:myteam/myrepo.git",
			workspace: "myteam",
			repo:      "myrepo",
		},
		{
			name:      "SSH without .git",
			url:       "git@bitbucket.org:myteam/myrepo",
			workspace: "myteam",
			repo:      "myrepo",
		},
		{
			name:      "HTTPS",
			url:       "https://bitbucket.org/myteam/myrepo.git",
			workspace: "myteam",
			repo:      "myrepo",
		},
		{
			name:      "HTTPS without .git",
			url:       "https://bitbucket.org/myteam/myrepo",
			workspace: "myteam",
			repo:      "myrepo",
		},
		{
			name:      "HTTPS with user",
			url:       "https://user@bitbucket.org/myteam/myrepo.git",
			workspace: "myteam",
			repo:      "myrepo",
		},
		{
			name:    "GitHub URL",
			url:     "git@github.com:user/repo.git",
			wantErr: true,
		},
		{
			name:    "invalid URL",
			url:     "not-a-url",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ws, repo, err := ParseRemoteURL(tt.url)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got ws=%q repo=%q", ws, repo)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if ws != tt.workspace {
				t.Errorf("workspace: got %q, want %q", ws, tt.workspace)
			}
			if repo != tt.repo {
				t.Errorf("repo: got %q, want %q", repo, tt.repo)
			}
		})
	}
}
