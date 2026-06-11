// Copyright (c) 2026 cloud-boot contributors
// SPDX-License-Identifier: GPL-2.0-only
//
// Default build (no `embedwad` tag): DOOM1WAD returns an empty slice so
// downstream callers can compile + import the package without first
// dropping a WAD next to doom1_wad.go. The cloud-boot probe will refuse
// to start the engine when the slice is empty, surfacing a clear
// "DOOMBOOT FAIL: no WAD embedded — rebuild with -tags embedwad after
// dropping doom1.wad into internal/embedwad/" diagnostic.

//go:build !embedwad

package embedwad

// DOOM1WAD returns an empty slice. The `embedwad` build variant in
// doom1_wad.go replaces this with the //go:embed'd bytes.
func DOOM1WAD() []byte {
	return nil
}
