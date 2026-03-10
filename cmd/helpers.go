package cmd

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/josejibin/bbgo/internal/bitbucket"
	"github.com/josejibin/bbgo/internal/config"
	"github.com/josejibin/bbgo/internal/git"
	"github.com/josejibin/bbgo/internal/secrets"
	"github.com/urfave/cli/v2"
)

// resolveRepo determines workspace and repo from: flag → config → git detect.
func resolveRepo(c *cli.Context) (workspace, repo string, err error) {
	repoFlag := getString(c, "repo", "r")

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
		if loadErr := secrets.LastLoadError(); loadErr != nil {
			return nil, fmt.Errorf("token not available: %v\nRun `bbgo config set --token` to re-store the token", loadErr)
		}
		return nil, fmt.Errorf("no token configured — run `bbgo config set --token` or set BBGO_TOKEN env var")
	}

	return bitbucket.NewClient(token, getBool(c, "verbose")), nil
}

// getString returns a string flag value, falling back to manual arg parsing.
// Workaround for urfave/cli v2 not parsing flags after positional args in nested subcommands.
func getString(c *cli.Context, name string, aliases ...string) string {
	if v := c.String(name); v != "" {
		return v
	}
	return stringFlagFromArgs(c, append([]string{name}, aliases...)...)
}

// getBool returns a bool flag value, falling back to manual arg parsing.
func getBool(c *cli.Context, name string, aliases ...string) bool {
	if c.Bool(name) {
		return true
	}
	return boolFlagFromArgs(c, append([]string{name}, aliases...)...)
}

// getOptionalInt returns an int flag value, whether it was set, and any parse error.
func getOptionalInt(c *cli.Context, name string, aliases ...string) (int, bool, error) {
	if c.IsSet(name) {
		return c.Int(name), true, nil
	}

	if s := stringFlagFromArgs(c, append([]string{name}, aliases...)...); s != "" {
		n, err := strconv.Atoi(strings.TrimSpace(s))
		if err != nil {
			return 0, true, fmt.Errorf("invalid --%s value %q (must be a number)", name, s)
		}
		return n, true, nil
	}

	return 0, false, nil
}

// stringFlagFromArgs extracts a string flag value from remaining args.
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

// ExitCodeForError maps error types to CLI exit codes.
func ExitCodeForError(err error) int {
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
