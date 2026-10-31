// Copyright 2026 The Forgejo Authors.
// SPDX-License-Identifier: MIT

package poll

import (
	"context"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"code.forgejo.org/forgejo/runner/v13/internal/pkg/config"
	"code.forgejo.org/forgejo/runner/v13/internal/pkg/workspacegc"
)

func TestWorkspaceGCOnlyWhenIdle(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "0123456789abcdef")
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, workspacegc.Marker), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	stale := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(dir, stale, stale); err != nil {
		t.Fatal(err)
	}
	p := &poller{
		pollingCtx: context.Background(),
		cfg: &config.Config{
			Runner: config.Runner{WorkdirCleanupAge: 24 * time.Hour},
			Host:   config.Host{WorkdirParent: root},
		},
	}
	var inProgress atomic.Int64
	inProgress.Store(1)
	p.cleanupIdleWorkspaces(&inProgress)
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("active task scratch directory was removed: %v", err)
	}
	inProgress.Store(0)
	p.cleanupIdleWorkspaces(&inProgress)
	if _, err := os.Stat(dir); !os.IsNotExist(err) {
		t.Fatalf("idle cleanup did not remove stale scratch directory: %v", err)
	}
}
