package bitbucket

import (
	"fmt"
	"regexp"
	"strings"
)

// ListComments returns comments on a PR.
func (c *Client) ListComments(workspace, repo string, prID int) ([]Comment, error) {
	path := fmt.Sprintf("/2.0/repositories/%s/%s/pullrequests/%d/comments?pagelen=100", workspace, repo, prID)
	var result PaginatedResponse[Comment]
	if err := c.GetJSON(path, &result); err != nil {
		return nil, err
	}
	return result.Values, nil
}

var validTag = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

// ValidateTag checks that a tag contains only safe characters (alphanumeric, hyphens, underscores).
func ValidateTag(tag string) error {
	if !validTag.MatchString(tag) {
		return fmt.Errorf("invalid tag %q: must contain only alphanumeric characters, hyphens, and underscores", tag)
	}
	return nil
}

// PostComment posts a comment on a PR. Supports general, inline, and tagged comments.
func (c *Client) PostComment(workspace, repo string, prID int, body string, file string, line int, tag string) (*Comment, error) {
	// Embed tag as HTML comment if provided
	if tag != "" {
		if err := ValidateTag(tag); err != nil {
			return nil, err
		}
		body = body + "\n<!-- bbgo:tag:" + tag + " -->"
	}

	payload := map[string]interface{}{
		"content": map[string]string{
			"raw": body,
		},
	}

	if file != "" {
		inline := map[string]interface{}{
			"path": file,
		}
		if line > 0 {
			inline["to"] = line
		}
		payload["inline"] = inline
	}

	path := fmt.Sprintf("/2.0/repositories/%s/%s/pullrequests/%d/comments", workspace, repo, prID)
	var comment Comment
	if err := c.PostJSON(path, payload, &comment); err != nil {
		return nil, err
	}
	return &comment, nil
}

// DeleteComment deletes a comment by ID.
func (c *Client) DeleteComment(workspace, repo string, prID, commentID int) error {
	path := fmt.Sprintf("/2.0/repositories/%s/%s/pullrequests/%d/comments/%d", workspace, repo, prID, commentID)
	resp, err := c.Delete(path)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// HasTag checks if a comment contains a bbgo tag.
func HasTag(comment Comment, tag string) bool {
	marker := "<!-- bbgo:tag:" + tag + " -->"
	return strings.Contains(comment.Content.Raw, marker)
}
