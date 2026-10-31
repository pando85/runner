// Copyright 2026 The Forgejo Authors.
// SPDX-License-Identifier: MIT

//go:build !linux

package workspacegc

// Other platforms do not expose Linux mountinfo. Filesystem workspace
// directories must not be shared with independently mounted paths.
func mountpointsUnder(string) (bool, error) { return false, nil }
