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
	if repoFlag == "" {
		repoFlag = stringFlagFromArgs(c, "repo", "r")
	}

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

// stringFlagFromArgs extracts a string flag value from remaining args.
// Workaround for urfave/cli v2 not parsing flags after positional args in nested subcommands.
func stringFlagFromArgs(c *cli.Context, names ...string) string {
	args := c.Args().Slice()
	for i, arg := range args {
		for _, name := range names {
			prefix := "-"
			if len(name) > 1 {
				prefix = "--"
			}
			flag := prefix + name
			if arg == flag && i+1 < len(args) {
				return args[i+1]
			}
			if strings.HasPrefix(arg, flag+"=") {
				return arg[len(flag)+1:]
			}
		}
	}
	return ""
}

// boolFlagFromArgs checks if a boolean flag is present in remaining args.
func boolFlagFromArgs(c *cli.Context, names ...string) bool {
	args := c.Args().Slice()
	for _, arg := range args {
		for _, name := range names {
			prefix := "-"
			if len(name) > 1 {
				prefix = "--"
			}
			if arg == prefix+name {
				return true
			}
		}
	}
	return false
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
