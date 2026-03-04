package bitbucket

import (
	"fmt"
	"io"
	"net/url"
	"strings"
)

// GetFileContent retrieves file content from a repository at a given commit or branch.
func (c *Client) GetFileContent(workspace, repo, ref, filePath string) (string, error) {
	// URL-encode ref and each segment of the file path to handle spaces/special chars.
	encodedRef := url.PathEscape(ref)
	segments := strings.Split(filePath, "/")
	for i, seg := range segments {
		segments[i] = url.PathEscape(seg)
	}
	encodedPath := strings.Join(segments, "/")
	path := fmt.Sprintf("/2.0/repositories/%s/%s/src/%s/%s", workspace, repo, encodedRef, encodedPath)
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
