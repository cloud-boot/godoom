// Copyright (c) 2026 cloud-boot contributors
// SPDX-License-Identifier: GPL-2.0-only
//
// This file is part of cloud-boot/godoom, a fork of github.com/AndreRenaud/gore
// (Pure-Go minimal Doom implementation, GPL-2.0). The TamaGo frontend adapter
// itself is NEW code authored for cloud-boot and is released under the same
// license to preserve the engine's GPL boundary; cloud-boot's other components
// remain BSD-3-Clause.

package tamago

import (
	"image"
)

// virtioFramebuffer is the minimal subset of github.com/go-virtio/gpu's
// *Framebuffer the adapter needs. Defining it as a local interface keeps
// this file decoupled from a compile-time import of go-virtio/gpu — the
// concrete *gpu.Framebuffer satisfies it via duck typing, so cloud-boot's
// probe wires the real driver into [NewGPUAdapter] without godoom growing
// a virtio/gpu module dependency.
type virtioFramebuffer interface {
	// Flush pushes the current Pix bytes to the host scanout
	// (TRANSFER_TO_HOST_2D + RESOURCE_FLUSH in virtio-gpu terms).
	Flush() error
}

// GPUAdapter is the GPU implementation that wraps a virtio-gpu
// Framebuffer. The Framebuffer is acquired by the caller (typically the
// cloud-boot/tamago-uefi probe via go-virtio/gpu.OpenVirtioGPU +
// SetupFramebuffer) and the adapter owns nothing beyond it; lifetime
// management stays with the caller, who must keep the underlying
// VirtioGPU device alive for the duration of the DOOM run.
//
// The adapter performs the RGBA → BGRA byte swap DOOM's frame format
// requires for virtio-gpu's VIRTIO_GPU_FORMAT_B8G8R8A8_UNORM resource,
// and the destination-rectangle copy when the framebuffer is wider /
// taller than the 320×200 DOOM canvas (cloud-boot/tamago-uefi typically
// asks the device for a native 1024×768 scanout; the adapter centers
// the DOOM image inside it without a scale to keep the hot path cheap).
type GPUAdapter struct {
	fb     virtioFramebuffer
	pix    []byte // BGRA backing store, aliased onto fb.Pix
	width  int    // framebuffer width in pixels
	height int    // framebuffer height in pixels
	// xOff / yOff are the top-left of the DOOM canvas inside the
	// framebuffer. Zero when the framebuffer matches the DOOM canvas
	// exactly; positive when the framebuffer is bigger (we letterbox).
	xOff int
	yOff int
}

// NewGPUAdapter wraps the supplied virtio-gpu Framebuffer. `pix` MUST be
// the same byte slice as `fb.Pix` so the adapter can blit DOOM frames
// directly into the device-backed memory without a second copy; width and
// height are the framebuffer's dimensions in pixels.
//
// The caller (the cloud-boot probe) typically does:
//
//	g, _ := gpu.OpenVirtioGPU(transport)
//	displays, _ := g.DisplayInfo()
//	fb, _ := g.SetupFramebuffer(displays[0].ScanoutID, displays[0].Width, displays[0].Height)
//	adapter := tamago.NewGPUAdapter(fb, fb.Pix, int(fb.Width), int(fb.Height))
func NewGPUAdapter(fb virtioFramebuffer, pix []byte, width, height int) *GPUAdapter {
	return &GPUAdapter{
		fb:     fb,
		pix:    pix,
		width:  width,
		height: height,
	}
}

// Flip blits the DOOM RGBA frame into the BGRA framebuffer backing store
// and asks the device to push it to the scanout.
//
// Centers the DOOM image in the framebuffer on the first call (cached in
// xOff / yOff), then for every subsequent call performs only the
// per-pixel RGBA → BGRA copy in the live region. The unchanged border
// remains as the zero-initialised pages the PageAllocator handed out —
// the bare-metal DOOM demo doesn't need a per-frame border repaint.
func (g *GPUAdapter) Flip(frame *image.RGBA) error {
	if g == nil || g.fb == nil || frame == nil || len(g.pix) == 0 {
		return nil
	}
	b := frame.Bounds()
	fw := b.Dx()
	fh := b.Dy()
	if fw <= 0 || fh <= 0 || g.width <= 0 || g.height <= 0 {
		return nil
	}
	// Recompute centering if dimensions changed; first frame is the
	// canonical sizing.
	if fw <= g.width {
		g.xOff = (g.width - fw) / 2
	} else {
		g.xOff = 0
		fw = g.width
	}
	if fh <= g.height {
		g.yOff = (g.height - fh) / 2
	} else {
		g.yOff = 0
		fh = g.height
	}
	srcStride := frame.Stride
	dstStride := g.width * 4
	for y := 0; y < fh; y++ {
		srcRow := frame.Pix[y*srcStride : y*srcStride+fw*4]
		dstRow := g.pix[(y+g.yOff)*dstStride+g.xOff*4 : (y+g.yOff)*dstStride+(g.xOff+fw)*4]
		for x := 0; x < fw; x++ {
			r := srcRow[x*4+0]
			gn := srcRow[x*4+1]
			bl := srcRow[x*4+2]
			a := srcRow[x*4+3]
			// virtio-gpu's VIRTIO_GPU_FORMAT_B8G8R8A8_UNORM lays bytes
			// out as { B, G, R, A } in little-endian memory order.
			dstRow[x*4+0] = bl
			dstRow[x*4+1] = gn
			dstRow[x*4+2] = r
			dstRow[x*4+3] = a
		}
	}
	return g.fb.Flush()
}

// Compile-time interface conformance assertion — *GPUAdapter must
// satisfy the GPU contract the Frontend expects.
var _ GPU = (*GPUAdapter)(nil)
