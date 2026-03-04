package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSave(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")

	cfg := &Config{
		Workspace:   "myteam",
		DefaultRepo: "myteam/myrepo",
	}

	if err := Save(path, cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Verify file permissions
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0600 {
		t.Errorf("expected perm 0600, got %o", perm)
	}

	loaded, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.Workspace != cfg.Workspace {
		t.Errorf("workspace: got %q, want %q", loaded.Workspace, cfg.Workspace)
	}
	if loaded.DefaultRepo != cfg.DefaultRepo {
		t.Errorf("default_repo: got %q, want %q", loaded.DefaultRepo, cfg.DefaultRepo)
	}
}

func TestLoadMissing(t *testing.T) {
	cfg, err := Load("/nonexistent/path/config.yaml")
	if err != nil {
		t.Fatalf("Load missing file should not error: %v", err)
	}
	if cfg.Workspace != "" {
		t.Errorf("expected empty workspace, got %q", cfg.Workspace)
	}
}
