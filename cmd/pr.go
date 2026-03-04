package cmd

import (
	"fmt"
	"strings"

	"github.com/josejibin/bbgo/internal/bitbucket"
	"github.com/josejibin/bbgo/internal/git"
	"github.com/josejibin/bbgo/internal/output"
	"github.com/urfave/cli/v2"
)

func PRCommands() *cli.Command {
	return &cli.Command{
		Name:  "pr",
		Usage: "Pull request commands",
		Subcommands: []*cli.Command{
			{
				Name:      "list",
				Usage:     "List pull requests",
				ArgsUsage: " ",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "state", Value: "open", Usage: "Filter by state: open, merged, declined, all"},
					&cli.StringFlag{Name: "author", Usage: "Filter by author username"},
					&cli.StringFlag{Name: "repo", Aliases: []string{"r"}, Usage: "workspace/repo override"},
					&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Usage: "Output format: text or json"},
				},
				Action: prList,
			},
			{
				Name:      "show",
				Usage:     "Show pull request details",
				ArgsUsage: "<PR_ID>",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "repo", Aliases: []string{"r"}, Usage: "workspace/repo override"},
					&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Usage: "Output format: text or json"},
				},
				Action: prShow,
			},
			{
				Name:      "diff",
				Usage:     "Show pull request diff",
				ArgsUsage: "<PR_ID>",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "repo", Aliases: []string{"r"}, Usage: "workspace/repo override"},
					&cli.BoolFlag{Name: "stat", Usage: "Show file summary only"},
				},
				Action: prDiff,
			},
			{
				Name:      "files",
				Usage:     "List changed files in a pull request",
				ArgsUsage: "<PR_ID>",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "repo", Aliases: []string{"r"}, Usage: "workspace/repo override"},
					&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Usage: "Output format: text or json"},
				},
				Action: prFiles,
			},
			{
				Name:      "create",
				Usage:     "Create a pull request",
				ArgsUsage: " ",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "title", Required: true, Usage: "PR title"},
					&cli.StringFlag{Name: "description", Usage: "PR description"},
					&cli.StringFlag{Name: "source", Usage: "Source branch (default: current git branch)"},
					&cli.StringFlag{Name: "dest", Usage: "Destination branch (default: repo default branch)"},
					&cli.StringFlag{Name: "reviewers", Usage: "Comma-separated list of reviewer usernames"},
					&cli.BoolFlag{Name: "close-source-branch", Usage: "Close source branch after merge"},
					&cli.BoolFlag{Name: "draft", Usage: "Create as draft PR"},
					&cli.StringFlag{Name: "repo", Aliases: []string{"r"}, Usage: "workspace/repo override"},
					&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Usage: "Output format: text or json"},
				},
				Action: prCreate,
			},
		},
	}
}

func prList(c *cli.Context) error {
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

	result, err := client.ListPRs(workspace, repo, c.String("state"), c.String("author"), 25)
	if err != nil {
		exitWithError(err)
		return nil
	}

	if getOutputFormat(c) == "json" {
		return output.PrintJSON(result.Values)
	}

	if len(result.Values) == 0 {
		fmt.Println("No pull requests found.")
		return nil
	}

	tbl := output.NewTable()
	tbl.Row("ID", "TITLE", "AUTHOR", "BRANCHES", "STATE", "CREATED")
	for _, pr := range result.Values {
		branches := fmt.Sprintf("%s → %s", pr.Source.Branch.Name, pr.Destination.Branch.Name)
		created := pr.CreatedOn.Format("2006-01-02")
		tbl.Row(
			fmt.Sprintf("#%d", pr.ID),
			truncate(pr.Title, 50),
			pr.Author.DisplayName,
			branches,
			pr.State,
			created,
		)
	}
	tbl.Flush()
	return nil
}

func prShow(c *cli.Context) error {
	id, err := requireIntArg(c, "PR_ID")
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

	pr, err := client.GetPR(workspace, repo, id)
	if err != nil {
		exitWithError(err)
		return nil
	}

	if getOutputFormat(c) == "json" {
		return output.PrintJSON(pr)
	}

	fmt.Printf("#%d: %s\n", pr.ID, pr.Title)
	fmt.Printf("State:       %s\n", pr.State)
	fmt.Printf("Author:      %s\n", pr.Author.DisplayName)
	fmt.Printf("Branches:    %s → %s\n", pr.Source.Branch.Name, pr.Destination.Branch.Name)
	fmt.Printf("Created:     %s\n", pr.CreatedOn.Format("2006-01-02 15:04"))
	fmt.Printf("Updated:     %s\n", pr.UpdatedOn.Format("2006-01-02 15:04"))
	fmt.Printf("URL:         %s\n", pr.Links.HTML.Href)

	if len(pr.Reviewers) > 0 {
		names := make([]string, len(pr.Reviewers))
		for i, r := range pr.Reviewers {
			names[i] = r.DisplayName
		}
		fmt.Printf("Reviewers:   %s\n", strings.Join(names, ", "))
	}

	if len(pr.Participants) > 0 {
		fmt.Println("Reviews:")
		for _, p := range pr.Participants {
			if p.Role == "REVIEWER" {
				fmt.Printf("  %s: %s\n", p.User.DisplayName, participantStatus(p))
			}
		}
	}

	if pr.Description != "" {
		fmt.Printf("\n%s\n", pr.Description)
	}

	return nil
}

func prDiff(c *cli.Context) error {
	id, err := requireIntArg(c, "PR_ID")
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

	if c.Bool("stat") || boolFlagFromArgs(c, "stat") {
		stats, err := client.GetDiffStat(workspace, repo, id)
		if err != nil {
			exitWithError(err)
			return nil
		}
		tbl := output.NewTable()
		tbl.Row("STATUS", "FILE", "+ADDED", "-REMOVED")
		for _, s := range stats {
			file := diffStatPath(s)
			tbl.Row(s.Status, file, fmt.Sprintf("+%d", s.LinesAdded), fmt.Sprintf("-%d", s.LinesRemoved))
		}
		tbl.Flush()
		return nil
	}

	diff, err := client.GetDiff(workspace, repo, id)
	if err != nil {
		exitWithError(err)
		return nil
	}
	fmt.Print(diff)
	return nil
}

func prFiles(c *cli.Context) error {
	id, err := requireIntArg(c, "PR_ID")
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

	stats, err := client.GetDiffStat(workspace, repo, id)
	if err != nil {
		exitWithError(err)
		return nil
	}

	if getOutputFormat(c) == "json" {
		type fileEntry struct {
			Path   string `json:"path"`
			Status string `json:"status"`
		}
		entries := make([]fileEntry, len(stats))
		for i, s := range stats {
			entries[i] = fileEntry{Path: diffStatPath(s), Status: s.Status}
		}
		return output.PrintJSON(entries)
	}

	tbl := output.NewTable()
	tbl.Row("STATUS", "FILE")
	for _, s := range stats {
		tbl.Row(s.Status, diffStatPath(s))
	}
	tbl.Flush()
	return nil
}

func prCreate(c *cli.Context) error {
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

	source := c.String("source")
	if source == "" {
		branch, err := git.CurrentBranch()
		if err != nil {
			return fmt.Errorf("cannot detect source branch: %w", err)
		}
		source = branch
	}

	// Warn if branch not pushed
	if !git.IsBranchPushed(source) {
		fmt.Fprintf(c.App.ErrWriter, "Warning: branch %q may not be pushed to remote\n", source)
	}

	dest := c.String("dest")
	if dest == "" {
		info, err := client.GetRepoInfo(workspace, repo)
		if err != nil {
			return fmt.Errorf("cannot get repo default branch: %w", err)
		}
		if info.MainBranch != nil {
			dest = info.MainBranch.Name
		} else {
			dest = "main"
		}
	}

	req := bitbucket.CreatePRRequest{
		Title:             c.String("title"),
		Description:       c.String("description"),
		Source:            bitbucket.BranchRef{Branch: bitbucket.Branch{Name: source}},
		Destination:       bitbucket.BranchRef{Branch: bitbucket.Branch{Name: dest}},
		CloseSourceBranch: c.Bool("close-source-branch"),
		Draft:             c.Bool("draft"),
	}

	if reviewers := c.String("reviewers"); reviewers != "" {
		for _, r := range strings.Split(reviewers, ",") {
			r = strings.TrimSpace(r)
			if r != "" {
				req.Reviewers = append(req.Reviewers, bitbucket.UserRef{Username: r})
			}
		}
	}

	pr, err := client.CreatePR(workspace, repo, req)
	if err != nil {
		exitWithError(err)
		return nil
	}

	if getOutputFormat(c) == "json" {
		return output.PrintJSON(map[string]interface{}{
			"id":  pr.ID,
			"url": pr.Links.HTML.Href,
		})
	}

	fmt.Printf("PR #%d created: %s\n", pr.ID, pr.Links.HTML.Href)
	return nil
}

// --- helpers ---

func diffStatPath(s bitbucket.DiffStat) string {
	if s.New != nil {
		return s.New.Path
	}
	if s.Old != nil {
		return s.Old.Path
	}
	return "(unknown)"
}

func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

func getOutputFormat(c *cli.Context) string {
	if f := c.String("output"); f != "" {
		return f
	}
	// Check remaining args for unparsed --output flag (urfave/cli v2 limitation)
	if f := stringFlagFromArgs(c, "output", "o"); f != "" {
		return f
	}
	// Walk up the context lineage to find the global --output flag
	for _, parent := range c.Lineage() {
		if f := parent.String("output"); f != "" && f != "text" {
			return f
		}
	}
	return "text"
}

func requireIntArg(c *cli.Context, name string) (int, error) {
	arg := c.Args().First()
	if arg == "" {
		return 0, fmt.Errorf("missing required argument: %s", name)
	}
	var id int
	_, err := fmt.Sscanf(arg, "%d", &id)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %q (must be a number)", name, arg)
	}
	return id, nil
}
