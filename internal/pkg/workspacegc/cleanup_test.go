// Copyright 2026 The Forgejo Authors.
// SPDX-License-Identifier: MIT

package workspacegc

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"
)

func TestCleanup(t *testing.T) {
	root := t.TempDir()
	old := time.Now().Add(-48 * time.Hour)
	makeScratch := func(name string, marker bool) string {
		t.Helper()
		dir := filepath.Join(root, name)
		for _, part := range []string{"act", "hostexecutor", "tmp"} {
			if err := os.MkdirAll(filepath.Join(dir, part), 0o700); err != nil {
				t.Fatal(err)
			}
		}
		if marker {
			if err := os.WriteFile(filepath.Join(dir, Marker), nil, 0o600); err != nil {
				t.Fatal(err)
			}
		}
		return dir
	}
	stale := makeScratch("0123456789abcdef", true)
	legacy := makeScratch("1111111111111111", false)
	recent := makeScratch("2222222222222222", true)
	foreign := filepath.Join(root, "3333333333333333")
	if err := os.MkdirAll(filepath.Join(foreign, "act"), 0o700); err != nil {
		t.Fatal(err)
	}
	cache := filepath.Join(root, "tool_cache")
	if err := os.MkdirAll(cache, 0o700); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{stale, legacy, foreign, cache} {
		if err := os.Chtimes(path, old, old); err != nil {
			t.Fatal(err)
		}
	}
	count, err := Cleanup(context.Background(), root, 24*time.Hour)
	if err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("removed %d, want 2", count)
	}
	for _, path := range []string{stale, legacy} {
		if _, err := os.Lstat(path); !os.IsNotExist(err) {
			t.Errorf("%q was not removed: %v", path, err)
		}
	}
	for _, path := range []string{recent, foreign, cache} {
		if _, err := os.Lstat(path); err != nil {
			t.Errorf("%q should survive: %v", path, err)
		}
	}
}

func TestCleanupPartiallyInitializedScratch(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "aaaaaaaaaaaaaaaa")
	if err := os.Mkdir(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, Marker), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-48 * time.Hour)
	if err := os.Chtimes(dir, old, old); err != nil {
		t.Fatal(err)
	}
	count, err := Cleanup(context.Background(), root, 24*time.Hour)
	if err != nil || count != 1 {
		t.Fatalf("count=%d err=%v", count, err)
	}
}

func TestCleanupRejectsSymlinks(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink permissions vary")
	}
	root := t.TempDir()
	external := t.TempDir()
	target := filepath.Join(external, "keep")
	if err := os.Mkdir(target, 0o700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "aaaaaaaaaaaaaaaa")
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	count, err := Cleanup(context.Background(), root, time.Second)
	if count != 0 || err != nil {
		t.Fatalf("count=%d err=%v", count, err)
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatal(err)
	}
	if _, err := Cleanup(context.Background(), link, time.Second); err == nil {
		t.Fatal("symlink root accepted")
	}
}

func TestCleanupDisabledAndCancellation(t *testing.T) {
	if count, err := Cleanup(context.Background(), "", 0); count != 0 || err != nil {
		t.Fatalf("disabled: %d %v", count, err)
	}
	if _, err := Cleanup(context.Background(), "", time.Hour); err == nil {
		t.Fatal("empty root accepted")
	}
	if _, err := Cleanup(context.Background(), string(filepath.Separator), time.Hour); err == nil {
		t.Fatal("filesystem root accepted")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := Cleanup(ctx, t.TempDir(), time.Hour); err == nil {
		t.Fatal("cancelled context accepted")
	}
}
