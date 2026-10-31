// Copyright 2026 The Forgejo Authors.
// SPDX-License-Identifier: MIT

//go:build linux

package workspacegc

import "testing"

func TestMountpointsUnderFromMountinfo(t *testing.T) {
	const mounts = "24 1 0:1 / / rw,relatime - ext4 /dev/sda rw\n" +
		"31 24 0:2 / /cache/aaaaaaaaaaaaaaaa/mounted rw,relatime - tmpfs tmpfs rw\n" +
		"32 24 0:3 / /cache/something-else rw,relatime - tmpfs tmpfs rw\n" +
		"33 24 0:4 / /cache/bbbbbbbbbbbbbbbb/mount\\040name rw,relatime - tmpfs tmpfs rw\n"
	for _, tt := range []struct {
		name string
		dir string
		want bool
	}{
		{"nested", "/cache/aaaaaaaaaaaaaaaa", true},
		{"mount itself", "/cache/aaaaaaaaaaaaaaaa/mounted", true},
		{"other directory", "/cache/cccccccccccccccc", false},
		{"prefix sibling", "/cache/aaaaaaaaaaaaaaa", false},
		{"escaped mount name", "/cache/bbbbbbbbbbbbbbbb", true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			got, err := mountpointsUnderFromMountinfo(tt.dir, mounts)
			if err != nil || got != tt.want {
				t.Fatalf("got %v, error %v, want %v", got, err, tt.want)
			}
		})
	}
	if _, err := mountpointsUnderFromMountinfo("/cache/aaaaaaaaaaaaaaaa", "invalid\n"); err == nil {
		t.Fatal("expected malformed mountinfo to be rejected")
	}
}
