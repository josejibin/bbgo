package git

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

var (
	// SSH: git@bitbucket.org:workspace/repo.git
	sshPattern = regexp.MustCompile(`git@bitbucket\.org:([^/]+)/([^/.]+)(?:\.git)?$`)
	// HTTPS: https://bitbucket.org/workspace/repo.git
	httpsPattern = regexp.MustCompile(`https?://(?:[^@]+@)?bitbucket\.org/([^/]+)/([^/.]+)(?:\.git)?$`)
)

// DetectRepo returns workspace and repo slug from the git remote "origin".
func DetectRepo() (workspace, repo string, err error) {
	out, err := exec.Command("git", "remote", "get-url", "origin").Output()
	if err != nil {
		return "", "", fmt.Errorf("not a git repo or no remote origin")
	}
	url := strings.TrimSpace(string(out))
	return ParseRemoteURL(url)
}

// ParseRemoteURL extracts workspace and repo from a Bitbucket remote URL.
func ParseRemoteURL(url string) (workspace, repo string, err error) {
	if m := sshPattern.FindStringSubmatch(url); m != nil {
		return m[1], m[2], nil
	}
	if m := httpsPattern.FindStringSubmatch(url); m != nil {
		return m[1], m[2], nil
	}
	return "", "", fmt.Errorf("cannot parse Bitbucket remote URL: %s", url)
}

// CurrentBranch returns the current git branch name.
func CurrentBranch() (string, error) {
	out, err := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD").Output()
	if err != nil {
		return "", fmt.Errorf("cannot determine current branch: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

// IsBranchPushed checks whether the given branch exists on the remote.
func IsBranchPushed(branch string) bool {
	err := exec.Command("git", "rev-parse", "--verify", "refs/remotes/origin/"+branch).Run()
	return err == nil
}
