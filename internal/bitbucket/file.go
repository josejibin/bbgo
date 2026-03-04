package bitbucket

import (
	"fmt"
	"io"
)

// GetFileContent retrieves file content from a repository at a given commit or branch.
func (c *Client) GetFileContent(workspace, repo, ref, filePath string) (string, error) {
	path := fmt.Sprintf("/2.0/repositories/%s/%s/src/%s/%s", workspace, repo, ref, filePath)
	resp, err := c.Get(path)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading file content: %w", err)
	}
	return string(data), nil
}
