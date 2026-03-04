package cmd

import (
	"fmt"

	"github.com/josejibin/bbgo/internal/bitbucket"
	"github.com/josejibin/bbgo/internal/output"
	"github.com/urfave/cli/v2"
)

func participantStatus(p bitbucket.Participant) string {
	if p.Approved {
		return "approved"
	}
	if p.State == "changes_requested" {
		return "changes_requested"
	}
	return "pending"
}

func ReviewCommands() *cli.Command {
	return &cli.Command{
		Name:  "review",
		Usage: "PR review commands",
		Subcommands: []*cli.Command{
			{
				Name:      "list",
				Usage:     "List reviewers and their status",
				ArgsUsage: "<PR_ID>",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "repo", Aliases: []string{"r"}, Usage: "workspace/repo override"},
					&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Usage: "Output format: text or json"},
				},
				Action: reviewList,
			},
			{
				Name:      "approve",
				Usage:     "Approve a pull request",
				ArgsUsage: "<PR_ID>",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "repo", Aliases: []string{"r"}, Usage: "workspace/repo override"},
				},
				Action: reviewApprove,
			},
			{
				Name:      "request-changes",
				Usage:     "Request changes on a pull request",
				ArgsUsage: "<PR_ID>",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "repo", Aliases: []string{"r"}, Usage: "workspace/repo override"},
				},
				Action: reviewRequestChanges,
			},
			{
				Name:      "unapprove",
				Usage:     "Remove approval from a pull request",
				ArgsUsage: "<PR_ID>",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "repo", Aliases: []string{"r"}, Usage: "workspace/repo override"},
				},
				Action: reviewUnapprove,
			},
		},
	}
}

func reviewList(c *cli.Context) error {
	prID, err := requireIntArg(c, "PR_ID")
	if err != nil {
		return err
	}

	client, err := newClient(c)
	if err != nil {
		exitWithError(err)
		return nil
	}

	workspace, repo, err := resolveRepo(c)
	if err != nil {
		exitWithError(err)
		return nil
	}

	participants, err := client.ListReviewers(workspace, repo, prID)
	if err != nil {
		exitWithError(err)
		return nil
	}

	if getOutputFormat(c) == "json" {
		type reviewEntry struct {
			User   string `json:"user"`
			Role   string `json:"role"`
			Status string `json:"status"`
		}
		var entries []reviewEntry
		for _, p := range participants {
			entries = append(entries, reviewEntry{
				User:   p.User.DisplayName,
				Role:   p.Role,
				Status: participantStatus(p),
			})
		}
		return output.PrintJSON(entries)
	}

	if len(participants) == 0 {
		fmt.Println("No reviewers found.")
		return nil
	}

	tbl := output.NewTable()
	tbl.Row("REVIEWER", "ROLE", "STATUS")
	for _, p := range participants {
		tbl.Row(p.User.DisplayName, p.Role, participantStatus(p))
	}
	tbl.Flush()
	return nil
}

func reviewApprove(c *cli.Context) error {
	prID, err := requireIntArg(c, "PR_ID")
	if err != nil {
		return err
	}

	client, err := newClient(c)
	if err != nil {
		exitWithError(err)
		return nil
	}

	workspace, repo, err := resolveRepo(c)
	if err != nil {
		exitWithError(err)
		return nil
	}

	if err := client.ApprovePR(workspace, repo, prID); err != nil {
		exitWithError(err)
		return nil
	}

	fmt.Printf("PR #%d approved.\n", prID)
	return nil
}

func reviewRequestChanges(c *cli.Context) error {
	prID, err := requireIntArg(c, "PR_ID")
	if err != nil {
		return err
	}

	client, err := newClient(c)
	if err != nil {
		exitWithError(err)
		return nil
	}

	workspace, repo, err := resolveRepo(c)
	if err != nil {
		exitWithError(err)
		return nil
	}

	if err := client.RequestChanges(workspace, repo, prID); err != nil {
		exitWithError(err)
		return nil
	}

	fmt.Printf("Changes requested on PR #%d.\n", prID)
	return nil
}

func reviewUnapprove(c *cli.Context) error {
	prID, err := requireIntArg(c, "PR_ID")
	if err != nil {
		return err
	}

	client, err := newClient(c)
	if err != nil {
		exitWithError(err)
		return nil
	}

	workspace, repo, err := resolveRepo(c)
	if err != nil {
		exitWithError(err)
		return nil
	}

	if err := client.UnapprovePR(workspace, repo, prID); err != nil {
		exitWithError(err)
		return nil
	}

	fmt.Printf("Approval removed from PR #%d.\n", prID)
	return nil
}
