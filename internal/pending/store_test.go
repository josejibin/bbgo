package pending

import (
	"path/filepath"
	"testing"
)

func TestStoreAddAndLoad(t *testing.T) {
	dir := t.TempDir()
	s := NewStoreAt(filepath.Join(dir, "pending.json"))

	c := Comment{
		Workspace: "ws",
		Repo:      "repo",
		PRID:      1,
		Body:      "test comment",
		Tag:       "review",
	}
	if err := s.Add(c); err != nil {
		t.Fatalf("Add: %v", err)
	}

	comments, err := s.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(comments) != 1 {
		t.Fatalf("expected 1 comment, got %d", len(comments))
	}
	if comments[0].Body != "test comment" {
		t.Errorf("expected body 'test comment', got %q", comments[0].Body)
	}
}

func TestStoreForPR(t *testing.T) {
	dir := t.TempDir()
	s := NewStoreAt(filepath.Join(dir, "pending.json"))

	_ = s.Add(Comment{Workspace: "ws", Repo: "repo", PRID: 1, Body: "pr1"})
	_ = s.Add(Comment{Workspace: "ws", Repo: "repo", PRID: 2, Body: "pr2"})
	_ = s.Add(Comment{Workspace: "ws", Repo: "repo", PRID: 1, Body: "pr1-again"})

	matched, err := s.ForPR("ws", "repo", 1)
	if err != nil {
		t.Fatalf("ForPR: %v", err)
	}
	if len(matched) != 2 {
		t.Fatalf("expected 2 comments for PR 1, got %d", len(matched))
	}
}

func TestStoreClearPR(t *testing.T) {
	dir := t.TempDir()
	s := NewStoreAt(filepath.Join(dir, "pending.json"))

	_ = s.Add(Comment{Workspace: "ws", Repo: "repo", PRID: 1, Body: "pr1"})
	_ = s.Add(Comment{Workspace: "ws", Repo: "repo", PRID: 2, Body: "pr2"})

	if err := s.ClearPR("ws", "repo", 1); err != nil {
		t.Fatalf("ClearPR: %v", err)
	}

	all, err := s.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(all) != 1 {
		t.Fatalf("expected 1 remaining comment, got %d", len(all))
	}
	if all[0].PRID != 2 {
		t.Errorf("expected remaining comment for PR 2, got PR %d", all[0].PRID)
	}
}

func TestStoreLoadEmpty(t *testing.T) {
	dir := t.TempDir()
	s := NewStoreAt(filepath.Join(dir, "nonexistent.json"))

	comments, err := s.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(comments) != 0 {
		t.Fatalf("expected 0 comments, got %d", len(comments))
	}
}

func TestStoreClearPRRemovesFile(t *testing.T) {
	dir := t.TempDir()
	s := NewStoreAt(filepath.Join(dir, "pending.json"))

	_ = s.Add(Comment{Workspace: "ws", Repo: "repo", PRID: 1, Body: "only"})

	if err := s.ClearPR("ws", "repo", 1); err != nil {
		t.Fatalf("ClearPR: %v", err)
	}

	comments, err := s.Load()
	if err != nil {
		t.Fatalf("Load after clear: %v", err)
	}
	if len(comments) != 0 {
		t.Fatalf("expected 0 comments after clearing all, got %d", len(comments))
	}
}

func TestStoreInlineComment(t *testing.T) {
	dir := t.TempDir()
	s := NewStoreAt(filepath.Join(dir, "pending.json"))

	c := Comment{
		Workspace: "ws",
		Repo:      "repo",
		PRID:      1,
		Body:      "inline note",
		File:      "main.go",
		Line:      42,
	}
	if err := s.Add(c); err != nil {
		t.Fatalf("Add: %v", err)
	}

	comments, err := s.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if comments[0].File != "main.go" || comments[0].Line != 42 {
		t.Errorf("inline fields not preserved: file=%q line=%d", comments[0].File, comments[0].Line)
	}
}
