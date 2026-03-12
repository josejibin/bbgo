package cmd

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/josejibin/bbgo/internal/bitbucket"
	"github.com/josejibin/bbgo/internal/output"
	"github.com/josejibin/bbgo/internal/pending"
	"github.com/urfave/cli/v2"
)

var commentFlags = []cli.Flag{
	&cli.StringFlag{Name: "body", Usage: "Comment body (use - for stdin)"},
	&cli.StringFlag{Name: "file", Usage: "File path for inline comment"},
	&cli.IntFlag{Name: "line", Usage: "Line number for inline comment"},
	&cli.StringFlag{Name: "tag", Usage: "Tag for later cleanup (embedded as HTML comment)"},
	&cli.StringFlag{Name: "repo", Aliases: []string{"r"}, Usage: "workspace/repo override"},
	&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Usage: "Output format: text or json"},
}

// printPendingNotice prints a reminder if there are pending comments for the given PR.
// Returns true if a notice was printed.
func printPendingNotice(workspace, repo string, prID int) bool {
	store, err := pending.NewStore()
	if err != nil {
		return false
	}
	comments, err := store.ForPR(workspace, repo, prID)
	if err != nil || len(comments) == 0 {
		return false
	}
	_, _ = fmt.Fprintf(os.Stderr, "Note: %d pending comment(s) for this PR. Run 'bbgo comment submit %d' to post, or 'bbgo comment discard %d' to discard.\n", len(comments), prID, prID)
	return true
}

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
				Name:      "add",
				Usage:     "Queue a comment locally (use 'submit' to post all at once)",
				ArgsUsage: "<PR_ID> [BODY]",
				Flags: append(commentFlags,
					&cli.BoolFlag{Name: "now", Usage: "Post immediately instead of queuing"},
				),
				Action: commentAdd,
			},
			{
				Name:      "submit",
				Usage:     "Post all pending comments for a PR",
				ArgsUsage: "<PR_ID>",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "repo", Aliases: []string{"r"}, Usage: "workspace/repo override"},
					&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Usage: "Output format: text or json"},
				},
				Action: commentSubmit,
			},
			{
				Name:      "pending",
				Usage:     "List pending (not yet posted) comments for a PR",
				ArgsUsage: "<PR_ID>",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "repo", Aliases: []string{"r"}, Usage: "workspace/repo override"},
					&cli.StringFlag{Name: "output", Aliases: []string{"o"}, Usage: "Output format: text or json"},
				},
				Action: commentPending,
			},
			{
				Name:      "discard",
				Usage:     "Discard all pending comments for a PR",
				ArgsUsage: "<PR_ID>",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "repo", Aliases: []string{"r"}, Usage: "workspace/repo override"},
				},
				Action: commentDiscard,
			},
			{
				Name:      "post",
				Usage:     "Post a single comment immediately (sends notification)",
				ArgsUsage: "<PR_ID> [BODY]",
				Flags:     commentFlags,
				Action:    commentPost,
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

// parseCommentBody extracts the comment body from flags/args/stdin.
func parseCommentBody(c *cli.Context) (string, error) {
	body := getString(c, "body")
	if body == "" {
		if arg := c.Args().Get(1); arg != "" && !strings.HasPrefix(arg, "-") {
			body = arg
		}
	}
	if body == "-" {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return "", fmt.Errorf("reading stdin: %w", err)
		}
		body = strings.TrimRight(string(data), "\n")
	}
	if body == "" {
		return "", fmt.Errorf("body is required: use --body flag, positional arg, or --body - for stdin")
	}
	return body, nil
}

// validateInlineFlags checks that --file and --line are used correctly.
func validateInlineFlags(c *cli.Context) (string, int, error) {
	file := getString(c, "file")
	line, lineSet, err := getOptionalInt(c, "line")
	if err != nil {
		return "", 0, err
	}
	if lineSet && line <= 0 {
		return "", 0, fmt.Errorf("--line must be greater than zero")
	}
	if lineSet && file == "" {
		return "", 0, fmt.Errorf("--file is required when --line is set")
	}
	return file, line, nil
}

func commentAdd(c *cli.Context) error {
	prID, err := requireIntArg(c, "PR_ID")
	if err != nil {
		return err
	}

	workspace, repo, err := resolveRepo(c)
	if err != nil {
		return err
	}

	body, err := parseCommentBody(c)
	if err != nil {
		return err
	}

	file, line, err := validateInlineFlags(c)
	if err != nil {
		return err
	}

	tag := getString(c, "tag")
	if tag != "" {
		if err := bitbucket.ValidateTag(tag); err != nil {
			return err
		}
	}

	// --now: skip the queue and post immediately
	if getBool(c, "now") {
		client, err := newClient(c)
		if err != nil {
			return err
		}
		comment, err := client.PostComment(workspace, repo, prID, body, file, line, tag)
		if err != nil {
			return err
		}
		if getOutputFormat(c) == "json" {
			return output.PrintJSON(map[string]any{"id": comment.ID})
		}
		fmt.Printf("Comment #%d posted.\n", comment.ID)
		return nil
	}

	store, err := pending.NewStore()
	if err != nil {
		return err
	}

	pc := pending.Comment{
		Workspace: workspace,
		Repo:      repo,
		PRID:      prID,
		Body:      body,
		File:      file,
		Line:      line,
		Tag:       tag,
	}
	if err := store.Add(pc); err != nil {
		return err
	}

	allPending, err := store.ForPR(workspace, repo, prID)
	if err != nil {
		return err
	}

	if getOutputFormat(c) == "json" {
		return output.PrintJSON(map[string]any{
			"status":        "queued",
			"pending_count": len(allPending),
		})
	}

	fmt.Printf("Comment queued. %d pending comment(s) for PR #%d. Run 'bbgo comment submit %d' to post all.\n", len(allPending), prID, prID)
	return nil
}

func commentSubmit(c *cli.Context) error {
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

	store, err := pending.NewStore()
	if err != nil {
		return err
	}

	comments, err := store.ForPR(workspace, repo, prID)
	if err != nil {
		return err
	}

	if len(comments) == 0 {
		fmt.Printf("No pending comments for PR #%d.\n", prID)
		return nil
	}

	var posted []int
	var failed int
	for _, pc := range comments {
		comment, err := client.PostComment(
			workspace, repo, prID,
			pc.Body,
			pc.File,
			pc.Line,
			pc.Tag,
		)
		if err != nil {
			_, _ = fmt.Fprintf(c.App.ErrWriter, "Failed to post comment: %v\n", err)
			failed++
			continue
		}
		posted = append(posted, comment.ID)
	}

	// Clear submitted comments
	if err := store.ClearPR(workspace, repo, prID); err != nil {
		_, _ = fmt.Fprintf(c.App.ErrWriter, "Warning: failed to clear pending comments: %v\n", err)
	}

	if getOutputFormat(c) == "json" {
		return output.PrintJSON(map[string]any{
			"posted": posted,
			"failed": failed,
		})
	}

	fmt.Printf("Posted %d comment(s) on PR #%d.", len(posted), prID)
	if failed > 0 {
		fmt.Printf(" %d failed.", failed)
	}
	fmt.Println()
	return nil
}

func commentPending(c *cli.Context) error {
	prID, err := requireIntArg(c, "PR_ID")
	if err != nil {
		return err
	}

	workspace, repo, err := resolveRepo(c)
	if err != nil {
		return err
	}

	store, err := pending.NewStore()
	if err != nil {
		return err
	}

	comments, err := store.ForPR(workspace, repo, prID)
	if err != nil {
		return err
	}

	if getOutputFormat(c) == "json" {
		return output.PrintJSON(comments)
	}

	if len(comments) == 0 {
		fmt.Printf("No pending comments for PR #%d.\n", prID)
		return nil
	}

	fmt.Printf("%d pending comment(s) for PR #%d:\n\n", len(comments), prID)
	for i, pc := range comments {
		location := "general"
		if pc.File != "" {
			if pc.Line > 0 {
				location = fmt.Sprintf("%s:%d", pc.File, pc.Line)
			} else {
				location = pc.File
			}
		}
		fmt.Printf("[%d] (%s)", i+1, location)
		if pc.Tag != "" {
			fmt.Printf(" [tag:%s]", pc.Tag)
		}
		fmt.Printf("\n  %s\n\n", truncate(pc.Body, 200))
	}
	return nil
}

func commentDiscard(c *cli.Context) error {
	prID, err := requireIntArg(c, "PR_ID")
	if err != nil {
		return err
	}

	workspace, repo, err := resolveRepo(c)
	if err != nil {
		return err
	}

	store, err := pending.NewStore()
	if err != nil {
		return err
	}

	comments, err := store.ForPR(workspace, repo, prID)
	if err != nil {
		return err
	}

	if len(comments) == 0 {
		fmt.Printf("No pending comments for PR #%d.\n", prID)
		return nil
	}

	if err := store.ClearPR(workspace, repo, prID); err != nil {
		return err
	}

	fmt.Printf("Discarded %d pending comment(s) for PR #%d.\n", len(comments), prID)
	return nil
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

	printPendingNotice(workspace, repo, prID)

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

	body, err := parseCommentBody(c)
	if err != nil {
		return err
	}

	file, line, err := validateInlineFlags(c)
	if err != nil {
		return err
	}
	tag := getString(c, "tag")

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
					_, _ = fmt.Fprintf(c.App.ErrWriter, "Failed to delete comment #%d: %v\n", cm.ID, err)
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
