package cmd

import (
	"fmt"

	"github.com/urfave/cli/v2"
)

func FileCommands() *cli.Command {
	return &cli.Command{
		Name:  "file",
		Usage: "File content commands",
		Subcommands: []*cli.Command{
			{
				Name:      "get",
				Usage:     "Get file content from repository",
				ArgsUsage: "<path>",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "commit", Usage: "Commit hash"},
					&cli.StringFlag{Name: "branch", Usage: "Branch name"},
					&cli.StringFlag{Name: "repo", Aliases: []string{"r"}, Usage: "workspace/repo override"},
				},
				Action: fileGet,
			},
		},
	}
}

func fileGet(c *cli.Context) error {
	filePath := c.Args().First()
	if filePath == "" {
		return fmt.Errorf("missing required argument: path")
	}

	client, err := newClient(c)
	if err != nil {
		return err
	}

	workspace, repo, err := resolveRepo(c)
	if err != nil {
		return err
	}

	// Determine ref: commit > branch > default branch
	ref := getString(c, "commit")
	if ref == "" {
		ref = getString(c, "branch")
	}
	if ref == "" {
		// Get repo default branch
		info, err := client.GetRepoInfo(workspace, repo)
		if err != nil {
			return fmt.Errorf("cannot determine default branch: %w", err)
		}
		if info.MainBranch != nil {
			ref = info.MainBranch.Name
		} else {
			ref = "main"
		}
	}

	content, err := client.GetFileContent(workspace, repo, ref, filePath)
	if err != nil {
		return err
	}

	fmt.Print(content)
	return nil
}
