// Copyright 2026 The Forgejo Authors.
// SPDX-License-Identifier: MIT

//go:build linux

package workspacegc

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// mountpointsUnder detects bind mounts as well as distinct filesystems. os.RemoveAll
// does not cross symlinks, but it *does* descend into mounts, which is unsafe here.
func mountpointsUnder(dir string) (bool, error) {
	data, err := os.ReadFile("/proc/self/mountinfo")
	if err != nil {
		return false, fmt.Errorf("read mount table: %w", err)
	}
	return mountpointsUnderFromMountinfo(dir, string(data))
}

func mountpointsUnderFromMountinfo(dir, data string) (bool, error) {
	dir = filepath.Clean(dir)
	unescape := strings.NewReplacer(
		`\040`, " ",
		`\011`, "\t",
		`\012`, "\n",
		`\134`, "\\",
	)
	for _, line := range strings.Split(data, "\n") {
		if line == "" {
			continue
		}
		before, _, ok := strings.Cut(line, " - ")
		if !ok {
			return false, fmt.Errorf("malformed mountinfo entry: %q", line)
		}
		fields := strings.Fields(before)
		if len(fields) < 5 {
			return false, fmt.Errorf("malformed mountinfo entry: %q", line)
		}
		mountPoint := filepath.Clean(unescape.Replace(fields[4]))
		if mountPoint == dir || strings.HasPrefix(mountPoint, dir+string(filepath.Separator)) {
			return true, nil
		}
	}
	return false, nil
}
