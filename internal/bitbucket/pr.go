package bitbucket

import (
	"fmt"
	"io"
	"net/url"
	"strings"
)

// ListPRs returns pull requests for the given workspace/repo.
func (c *Client) ListPRs(workspace, repo, state, author, source, dest string, pagelen int) (*PaginatedResponse[PullRequest], error) {
	params := url.Values{}
	if state != "" && state != "open" {
		if state == "all" {
			// Omit state filter to get all
		} else {
			params.Set("state", stateToAPI(state))
		}
	} else {
		params.Set("state", "OPEN")
	}
	if pagelen > 0 {
		params.Set("pagelen", fmt.Sprintf("%d", pagelen))
	}
	var filters []string
	if author != "" {
		filters = append(filters, fmt.Sprintf(`author.nickname="%s"`, author))
	}
	if source != "" {
		filters = append(filters, fmt.Sprintf(`source.branch.name="%s"`, source))
	}
	if dest != "" {
		filters = append(filters, fmt.Sprintf(`destination.branch.name="%s"`, dest))
	}
	if len(filters) > 0 {
		params.Set("q", strings.Join(filters, " AND "))
	}

	path := fmt.Sprintf("/2.0/repositories/%s/%s/pullrequests", workspace, repo)
	if len(params) > 0 {
		path += "?" + params.Encode()
	}

	var result PaginatedResponse[PullRequest]
	if err := c.GetJSON(path, &result); err != nil {
		return nil, err
	}
	return &result, nil
}

// GetPR returns a single pull request by ID.
func (c *Client) GetPR(workspace, repo string, id int) (*PullRequest, error) {
	path := fmt.Sprintf("/2.0/repositories/%s/%s/pullrequests/%d", workspace, repo, id)
	var pr PullRequest
	if err := c.GetJSON(path, &pr); err != nil {
		return nil, err
	}
	return &pr, nil
}

// GetDiff returns the raw unified diff for a PR.
func (c *Client) GetDiff(workspace, repo string, id int) (string, error) {
	path := fmt.Sprintf("/2.0/repositories/%s/%s/pullrequests/%d/diff", workspace, repo, id)
	resp, err := c.Get(path)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading diff: %w", err)
	}
	return string(data), nil
}

// GetDiffStat returns the list of changed files for a PR.
func (c *Client) GetDiffStat(workspace, repo string, id int) ([]DiffStat, error) {
	path := fmt.Sprintf("/2.0/repositories/%s/%s/pullrequests/%d/diffstat", workspace, repo, id)
	var result PaginatedResponse[DiffStat]
	if err := c.GetJSON(path, &result); err != nil {
		return nil, err
	}
	return result.Values, nil
}

// CreatePR creates a new pull request.
func (c *Client) CreatePR(workspace, repo string, req CreatePRRequest) (*PullRequest, error) {
	path := fmt.Sprintf("/2.0/repositories/%s/%s/pullrequests", workspace, repo)
	var pr PullRequest
	if err := c.PostJSON(path, req, &pr); err != nil {
		return nil, err
	}
	return &pr, nil
}

// GetRepoInfo returns repository information including the default branch.
func (c *Client) GetRepoInfo(workspace, repo string) (*RepoInfo, error) {
	path := fmt.Sprintf("/2.0/repositories/%s/%s", workspace, repo)
	var info RepoInfo
	if err := c.GetJSON(path, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

// CreatePRRequest is the payload for creating a pull request.
type CreatePRRequest struct {
	Title             string     `json:"title"`
	Description       string     `json:"description,omitempty"`
	Source            BranchRef  `json:"source"`
	Destination       BranchRef  `json:"destination"`
	CloseSourceBranch bool       `json:"close_source_branch"`
	Reviewers         []UserRef  `json:"reviewers,omitempty"`
	Draft             bool       `json:"draft,omitempty"`
}

type UserRef struct {
	Username string `json:"username"`
}

func stateToAPI(state string) string {
	switch state {
	case "open":
		return "OPEN"
	case "merged":
		return "MERGED"
	case "declined":
		return "DECLINED"
	case "superseded":
		return "SUPERSEDED"
	default:
		return state
	}
}
