package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/josejibin/bbgo/internal/bitbucket"
	"github.com/josejibin/bbgo/internal/config"
	"github.com/josejibin/bbgo/internal/git"
	"github.com/josejibin/bbgo/internal/secrets"
	"github.com/urfave/cli/v2"
)

// resolveRepo determines workspace and repo from: flag → config → git detect.
func resolveRepo(c *cli.Context) (workspace, repo string, err error) {
	repoFlag := c.String("repo")

	// 1. From --repo flag
	if repoFlag != "" {
		parts := strings.SplitN(repoFlag, "/", 2)
		if len(parts) != 2 {
			return "", "", fmt.Errorf("--repo must be in format workspace/repo")
		}
		return parts[0], parts[1], nil
	}

	// 2. From config
	cfgPath := c.String("config")
	cfg, err := config.Load(cfgPath)
	if err == nil && cfg.DefaultRepo != "" {
		parts := strings.SplitN(cfg.DefaultRepo, "/", 2)
		if len(parts) == 2 {
			return parts[0], parts[1], nil
		}
	}

	// If workspace is set in config, try to combine with git-detected repo
	if err == nil && cfg.Workspace != "" {
		ws, r, gitErr := git.DetectRepo()
		if gitErr == nil {
			_ = ws // use workspace from config
			return cfg.Workspace, r, nil
		}
	}

	// 3. From git remote
	ws, r, gitErr := git.DetectRepo()
	if gitErr == nil {
		return ws, r, nil
	}

	return "", "", fmt.Errorf("cannot determine repo — use --repo flag, set default_repo in config, or run from a git repo with a Bitbucket remote")
}

// newClient creates a Bitbucket API client from stored credentials.
func newClient(c *cli.Context) (*bitbucket.Client, error) {
	token := secrets.Token()
	if token == "" {
		return nil, fmt.Errorf("no token configured — run `bbgo config set --token`")
	}

	return bitbucket.NewClient(token, c.Bool("verbose")), nil
}

// exitWithError prints the error and exits with the appropriate code.
func exitWithError(err error) {
	code := exitCodeForError(err)
	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	os.Exit(code)
}

func exitCodeForError(err error) int {
	switch err.(type) {
	case *bitbucket.AuthError:
		return 2
	case *bitbucket.NotFoundError:
		return 3
	default:
		if strings.Contains(err.Error(), "not a git repo") || strings.Contains(err.Error(), "no remote origin") {
			return 5
		}
		return 1
	}
}
