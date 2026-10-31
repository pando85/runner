// Copyright 2026 The Forgejo Authors.
// SPDX-License-Identifier: MIT

package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestWorkspaceGCDurations(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yml")
	if err := os.WriteFile(path, []byte("runner:\n  workdir_cleanup_age: 0s\n  idle_cleanup_interval: 30m\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := New(FromFile(path))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Runner.WorkdirCleanupAge != 0 {
		t.Fatalf("workdir_cleanup_age=%s, want disabled", cfg.Runner.WorkdirCleanupAge)
	}
	if cfg.Runner.IdleCleanupInterval != 30*time.Minute {
		t.Fatalf("idle_cleanup_interval=%s, want 30m", cfg.Runner.IdleCleanupInterval)
	}
}
