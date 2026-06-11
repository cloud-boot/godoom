// Copyright (c) 2026 cloud-boot contributors
// SPDX-License-Identifier: GPL-2.0-only

package embedwad

import (
	"testing"
)

// TestDOOM1WAD_Available exercises whichever branch of doom1_wad.go was
// compiled in: when the `embedwad` tag is set, the slice should look
// like a real IWAD header ("IWAD" or "PWAD"); when the tag is unset,
// the slice should be empty. Either way the function MUST return
// without panicking.
func TestDOOM1WAD_Available(t *testing.T) {
	got := DOOM1WAD()
	if got == nil {
		// Stub branch — explicitly empty.
		return
	}
	if len(got) < 12 {
		t.Fatalf("DOOM1WAD: %d bytes (too short for an IWAD header)", len(got))
	}
	magic := string(got[:4])
	if magic != "IWAD" && magic != "PWAD" {
		t.Fatalf("DOOM1WAD: magic %q is not IWAD/PWAD — wrong file dropped?", magic)
	}
}
