package bitbucket

import (
	"fmt"
)

// ListReviewers returns the participants/reviewers for a PR.
func (c *Client) ListReviewers(workspace, repo string, prID int) ([]Participant, error) {
	pr, err := c.GetPR(workspace, repo, prID)
	if err != nil {
		return nil, err
	}
	return pr.Participants, nil
}

// ApprovePR approves a pull request.
func (c *Client) ApprovePR(workspace, repo string, prID int) error {
	path := fmt.Sprintf("/2.0/repositories/%s/%s/pullrequests/%d/approve", workspace, repo, prID)
	resp, err := c.Post(path, nil)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// UnapprovePR removes approval from a pull request.
func (c *Client) UnapprovePR(workspace, repo string, prID int) error {
	path := fmt.Sprintf("/2.0/repositories/%s/%s/pullrequests/%d/approve", workspace, repo, prID)
	resp, err := c.Delete(path)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}

// RequestChanges requests changes on a pull request.
func (c *Client) RequestChanges(workspace, repo string, prID int) error {
	path := fmt.Sprintf("/2.0/repositories/%s/%s/pullrequests/%d/request-changes", workspace, repo, prID)
	resp, err := c.Post(path, nil)
	if err != nil {
		return err
	}
	resp.Body.Close()
	return nil
}
