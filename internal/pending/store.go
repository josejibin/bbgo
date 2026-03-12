package pending

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Comment represents a comment queued locally before submission.
type Comment struct {
	Workspace string `json:"workspace"`
	Repo      string `json:"repo"`
	PRID      int    `json:"pr_id"`
	Body      string `json:"body"`
	File      string `json:"file,omitempty"`
	Line      int    `json:"line,omitempty"`
	Tag       string `json:"tag,omitempty"`
}

// Store manages pending comments in a JSON file.
type Store struct {
	path string
}

// NewStore creates a store at the default path (~/.bbgo/pending_comments.json).
func NewStore() (*Store, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("determining home directory: %w", err)
	}
	return &Store{
		path: filepath.Join(home, ".bbgo", "pending_comments.json"),
	}, nil
}

// NewStoreAt creates a store at a custom path (for testing).
func NewStoreAt(path string) *Store {
	return &Store{path: path}
}

// Load reads all pending comments from disk.
func (s *Store) Load() ([]Comment, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading pending comments: %w", err)
	}
	var comments []Comment
	if err := json.Unmarshal(data, &comments); err != nil {
		return nil, fmt.Errorf("parsing pending comments: %w", err)
	}
	return comments, nil
}

// Save writes all pending comments to disk.
func (s *Store) Save(comments []Comment) error {
	dir := filepath.Dir(s.path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating pending dir: %w", err)
	}
	data, err := json.MarshalIndent(comments, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling pending comments: %w", err)
	}
	if err := os.WriteFile(s.path, data, 0600); err != nil {
		return fmt.Errorf("writing pending comments: %w", err)
	}
	return nil
}

// Add appends a comment to the pending list.
func (s *Store) Add(c Comment) error {
	comments, err := s.Load()
	if err != nil {
		return err
	}
	comments = append(comments, c)
	return s.Save(comments)
}

// ForPR returns pending comments for a specific PR.
func (s *Store) ForPR(workspace, repo string, prID int) ([]Comment, error) {
	all, err := s.Load()
	if err != nil {
		return nil, err
	}
	var matched []Comment
	for _, c := range all {
		if c.Workspace == workspace && c.Repo == repo && c.PRID == prID {
			matched = append(matched, c)
		}
	}
	return matched, nil
}

// ClearPR removes all pending comments for a specific PR.
func (s *Store) ClearPR(workspace, repo string, prID int) error {
	all, err := s.Load()
	if err != nil {
		return err
	}
	var remaining []Comment
	for _, c := range all {
		if c.Workspace != workspace || c.Repo != repo || c.PRID != prID {
			remaining = append(remaining, c)
		}
	}
	if len(remaining) == 0 {
		// Remove the file if no pending comments remain
		if err := os.Remove(s.path); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("removing pending file: %w", err)
		}
		return nil
	}
	return s.Save(remaining)
}
