package cmd

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/josejibin/bbgo/internal/bitbucket"
	"github.com/josejibin/bbgo/internal/output"
	"github.com/urfave/cli/v2"
)

func CommentCommands() *cli.Command {
	return &cli.Command{
		Name:  "comment",
		Usage: "PR comment commands",
		Subcommands: []*cli.Command{
			{
				Name:      "list",
				Usage:     "List comments on a PR",
				ArgsUsage: "<PR_ID>",
				Flags: []cli.Flag{
					&cli.BoolFlag{Name: "inline-only", Usage: "Show only inline comments"},
					&cli.StringFlag{Name: "repo", Aliases: []string{"r"}, Usage: "workspace/repo override"},
					&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Usage: "Output format: text or json"},
				},
				Action: commentList,
			},
			{
				Name:      "post",
				Usage:     "Post a comment on a PR",
				ArgsUsage: "<PR_ID> [BODY]",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "body", Usage: "Comment body (use - for stdin)"},
					&cli.StringFlag{Name: "file", Usage: "File path for inline comment"},
					&cli.IntFlag{Name: "line", Usage: "Line number for inline comment"},
					&cli.StringFlag{Name: "tag", Usage: "Tag for later cleanup (embedded as HTML comment)"},
					&cli.StringFlag{Name: "repo", Aliases: []string{"r"}, Usage: "workspace/repo override"},
					&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Usage: "Output format: text or json"},
				},
				Action: commentPost,
			},
			{
				Name:      "delete",
				Usage:     "Delete a comment or all comments with a tag",
				ArgsUsage: "<PR_ID> [COMMENT_ID]",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "tag", Usage: "Delete all comments with this tag"},
					&cli.BoolFlag{Name: "dry-run", Usage: "Show what would be deleted"},
					&cli.StringFlag{Name: "repo", Aliases: []string{"r"}, Usage: "workspace/repo override"},
				},
				Action: commentDelete,
			},
		},
	}
}

func commentList(c *cli.Context) error {
	prID, err := requireIntArg(c, "PR_ID")
	if err != nil {
		return err
	}

	client, err := newClient(c)
	if err != nil {
		return err
	}

	workspace, repo, err := resolveRepo(c)
	if err != nil {
		return err
	}

	comments, err := client.ListComments(workspace, repo, prID)
	if err != nil {
		return err
	}

	// Filter deleted comments
	var filtered []bitbucket.Comment
	for _, cm := range comments {
		if cm.Deleted {
			continue
		}
		if getBool(c, "inline-only") && cm.Inline == nil {
			continue
		}
		filtered = append(filtered, cm)
	}

	if getOutputFormat(c) == "json" {
		return output.PrintJSON(filtered)
	}

	if len(filtered) == 0 {
		fmt.Println("No comments found.")
		return nil
	}

	for _, cm := range filtered {
		location := "general"
		if cm.Inline != nil {
			if cm.Inline.To != nil {
				location = fmt.Sprintf("%s:%d", cm.Inline.Path, *cm.Inline.To)
			} else {
				location = cm.Inline.Path
			}
		}
		fmt.Printf("[#%d] %s (%s) — %s\n", cm.ID, cm.User.DisplayName, location, cm.CreatedOn.Format("2006-01-02 15:04"))
		fmt.Printf("  %s\n\n", truncate(cm.Content.Raw, 200))
	}

	return nil
}

func commentPost(c *cli.Context) error {
	prID, err := requireIntArg(c, "PR_ID")
	if err != nil {
		return err
	}

	client, err := newClient(c)
	if err != nil {
		return err
	}

	workspace, repo, err := resolveRepo(c)
	if err != nil {
		return err
	}

	body := getString(c, "body")
	if body == "" {
		// Accept body as second positional arg, but only if it's not a flag
		if arg := c.Args().Get(1); arg != "" && !strings.HasPrefix(arg, "-") {
			body = arg
		}
	}
	if body == "-" {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("reading stdin: %w", err)
		}
		body = strings.TrimRight(string(data), "\n")
	}
	if body == "" {
		return fmt.Errorf("body is required: use --body flag, positional arg, or --body - for stdin")
	}

	file := getString(c, "file")
	line, lineSet, err := getOptionalInt(c, "line")
	if err != nil {
		return err
	}
	tag := getString(c, "tag")

	if lineSet && line <= 0 {
		return fmt.Errorf("--line must be greater than zero")
	}
	if lineSet && file == "" {
		return fmt.Errorf("--file is required when --line is set")
	}

	comment, err := client.PostComment(
		workspace, repo, prID,
		body,
		file,
		line,
		tag,
	)
	if err != nil {
		return err
	}

	if getOutputFormat(c) == "json" {
		return output.PrintJSON(map[string]any{
			"id": comment.ID,
		})
	}

	fmt.Printf("Comment #%d posted.\n", comment.ID)
	return nil
}

func commentDelete(c *cli.Context) error {
	prID, err := requireIntArg(c, "PR_ID")
	if err != nil {
		return err
	}

	client, err := newClient(c)
	if err != nil {
		return err
	}

	workspace, repo, err := resolveRepo(c)
	if err != nil {
		return err
	}

	tag := getString(c, "tag")
	dryRun := getBool(c, "dry-run")

	// Delete by tag
	if tag != "" {
		comments, err := client.ListComments(workspace, repo, prID)
		if err != nil {
			return err
		}

		var toDelete []bitbucket.Comment
		for _, cm := range comments {
			if !cm.Deleted && bitbucket.HasTag(cm, tag) {
				toDelete = append(toDelete, cm)
			}
		}

		if len(toDelete) == 0 {
			fmt.Printf("No comments found with tag %q.\n", tag)
			return nil
		}

		for _, cm := range toDelete {
			if dryRun {
				fmt.Printf("[dry-run] Would delete comment #%d: %s\n", cm.ID, truncate(cm.Content.Raw, 80))
			} else {
				if err := client.DeleteComment(workspace, repo, prID, cm.ID); err != nil {
					fmt.Fprintf(c.App.ErrWriter, "Failed to delete comment #%d: %v\n", cm.ID, err)
					continue
				}
				fmt.Printf("Deleted comment #%d\n", cm.ID)
			}
		}
		return nil
	}

	// Delete by comment ID (second arg)
	commentIDStr := c.Args().Get(1)
	if commentIDStr == "" {
		return fmt.Errorf("provide a COMMENT_ID or --tag")
	}
	commentID, err := strconv.Atoi(strings.TrimSpace(commentIDStr))
	if err != nil {
		return fmt.Errorf("invalid COMMENT_ID: %q", commentIDStr)
	}

	if dryRun {
		fmt.Printf("[dry-run] Would delete comment #%d\n", commentID)
		return nil
	}

	if err := client.DeleteComment(workspace, repo, prID, commentID); err != nil {
		return err
	}
	fmt.Printf("Deleted comment #%d\n", commentID)
	return nil
}
