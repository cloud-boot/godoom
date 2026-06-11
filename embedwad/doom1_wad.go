// Copyright (c) 2026 cloud-boot contributors
// SPDX-License-Identifier: GPL-2.0-only
//
// Build-tag-gated WAD embed. The `embedwad` build tag opts the binary
// in to the compile-time //go:embed of `doom1.wad` from this directory.
// Without the tag, [DOOM1WAD] is empty and the variable still resolves
// at link time, keeping the package usable in environments that ship
// the WAD via runtime (OCI stream, virtio-fs, ...) instead of embed.
//
// Why a tag: the WAD blob is large (≈4 MiB for the shareware DOOM1.WAD;
// ≈28 MiB for the Freedoom phase-1 WAD) — too big to commit. The file
// itself is `.gitignore`d (see this directory's .gitignore). Operators
// drop the WAD next to this source before running `task doomboot:efi:amd64`,
// then the build sees it through //go:embed.
//
// Recommended WADs (in license-clarity order):
//
//  1. freedoom1.wad — BSD-3-equivalent, drop-in replacement for the
//     shareware DOOM1.WAD. https://freedoom.github.io/
//  2. doom1.wad (shareware) — freely-distributable per id Software's
//     1993 release terms, but technically not BSD-3 / Apache-2.0
//     compatible. Avoid for downstream redistribution.
//
// The variable name stays DOOM1WAD regardless of which WAD is dropped
// in — the engine identifies the IWAD by its lump table, not by file
// name.

//go:build embedwad

package embedwad

import (
	_ "embed"
)

// doom1WAD is the raw bytes of the embedded WAD. The blob is only read
// at engine start (gore.SetVirtualFileSystem reads the entire IWAD into
// memory anyway), so a single immutable slice on the rodata segment is
// fine — no double-allocation overhead.
//
//go:embed doom1.wad
var doom1WAD []byte

// DOOM1WAD returns the embedded WAD bytes. The caller MUST NOT mutate
// the returned slice; the underlying storage lives in rodata.
func DOOM1WAD() []byte {
	return doom1WAD
}
