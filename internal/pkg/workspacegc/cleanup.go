// Copyright 2026 The Forgejo Authors.
// SPDX-License-Identifier: MIT

package workspacegc

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Marker identifies newly created host execution scratch directories.
const Marker = ".forgejo-runner-scratch"

// Cleanup removes stale host scratch directories immediately below root.
// Callers must run it while the runner is idle and must not share root with
// another independently running daemon. All other cache contents are retained.
func Cleanup(ctx context.Context, root string, maxAge time.Duration) (int, error) {
	if maxAge <= 0 {
		return 0, nil
	}
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if root == "" {
		return 0, errors.New("workspace GC root is empty")
	}
	root, err := filepath.Abs(root)
	if err != nil {
		return 0, fmt.Errorf("resolve workspace GC root: %w", err)
	}
	if root == filepath.Dir(root) {
		return 0, fmt.Errorf("refusing to clean filesystem root %q", root)
	}
	rootInfo, err := os.Lstat(root)
	if errors.Is(err, os.ErrNotExist) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("inspect workspace GC root: %w", err)
	}
	if !rootInfo.IsDir() {
		return 0, fmt.Errorf("workspace GC root %q is not a directory", root)
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		return 0, fmt.Errorf("read workspace GC root: %w", err)
	}
	cutoff := time.Now().Add(-maxAge)
	removed := 0
	var errs []error
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return removed, errors.Join(append(errs, err)...)
		}
		if !entry.IsDir() || !isScratchName(entry.Name()) {
			continue
		}
		dir := filepath.Join(root, entry.Name())
		info, err := os.Lstat(dir)
		if err != nil {
			errs = append(errs, fmt.Errorf("stat %q: %w", dir, err))
			continue
		}
		if !info.IsDir() || info.ModTime().After(cutoff) {
			continue
		}
		owned, err := isScratchDirectory(dir)
		if err != nil {
			errs = append(errs, fmt.Errorf("inspect %q: %w", dir, err))
			continue
		}
		if !owned {
			continue
		}
		if err := os.RemoveAll(dir); err != nil {
			errs = append(errs, fmt.Errorf("remove stale scratch directory %q: %w", dir, err))
			continue
		}
		removed++
	}
	return removed, errors.Join(errs...)
}

func isScratchName(name string) bool {
	if len(name) != 16 {
		return false
	}
	for _, c := range name {
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}

func isScratchDirectory(dir string) (bool, error) {
	marker, err := os.Lstat(filepath.Join(dir, Marker))
	if err == nil {
		return marker.Mode().IsRegular(), nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return false, err
	}
	// Older runners did not write a marker. Require their full scratch layout
	// rather than assuming every hex-named directory is safe to remove.
	for _, name := range [...]string{"act", "hostexecutor", "tmp"} {
		info, err := os.Lstat(filepath.Join(dir, name))
		if errors.Is(err, os.ErrNotExist) {
			return false, nil
		}
		if err != nil {
			return false, err
		}
		if !info.IsDir() {
			return false, nil
		}
	}
	return true, nil
}
