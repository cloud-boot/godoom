// Copyright (c) 2026 cloud-boot contributors
// SPDX-License-Identifier: GPL-2.0-only
//
// This file is part of cloud-boot/godoom, a fork of github.com/AndreRenaud/gore
// (Pure-Go minimal Doom implementation, GPL-2.0). The TamaGo frontend itself is
// NEW code authored for cloud-boot and is released under the same license to
// preserve the engine's GPL boundary; cloud-boot's other components remain
// BSD-3-Clause.

package tamago

import (
	"image"

	godoom "github.com/cloud-boot/godoom"
)

// NOTE: the upstream package literal is `gore`, because the engine source
// (doom.go) is mechanically transpiled from doomgeneric and we deliberately
// avoid touching it. Internally we alias it as `godoom` to match the module
// path; the `gore.` qualifier therefore appears as `godoom.` throughout
// this file.

// GPU is the minimal subset of a virtio-gpu driver that the DOOM frontend
// needs to flip a frame. The real driver in github.com/go-virtio/gpu will
// satisfy this interface.
type GPU interface {
	// Flip presents the provided RGBA frame on the primary scanout.
	// Implementations may scale or letterbox internally. The caller retains
	// ownership of the underlying pixel slice and may mutate it after the
	// call returns.
	Flip(frame *image.RGBA) error
}

// Sound is the minimal subset of a virtio-sound driver that the DOOM
// frontend needs to play SFX. MIDI music routing is deliberately out of
// scope for the first cloud-boot demo and will be added later.
type Sound interface {
	// Cache uploads a PCM lump under a stable name. The data slice is in
	// the original DOOM dmx format (8-bit unsigned PCM, 11025 Hz mono with
	// an 8-byte header).
	Cache(name string, data []byte) error

	// Play schedules the named lump for playback on the given channel.
	// vol and sep are DOOM's 0..127 mixer values; the backend is free to
	// linearise them as it sees fit.
	Play(name string, channel, vol, sep int) error
}

// Input is the minimal subset of a virtio-input driver that the DOOM
// frontend needs to read the keyboard. The frontend is responsible for
// translating HID usage IDs to DOOM scancodes.
type Input interface {
	// Poll drains zero or more queued key events. It MUST be non-blocking;
	// it returns false when the queue is empty.
	Poll() (ev KeyEvent, ok bool)
}

// KeyEvent is a single keyboard transition surfaced by the [Input] device.
type KeyEvent struct {
	// HIDUsage is the raw HID Keyboard usage ID (e.g. 0x52 for Up Arrow).
	HIDUsage uint16
	// Down is true for a press, false for a release.
	Down bool
}

// Frontend implements [gore.DoomFrontend] on top of cloud-boot virtio
// devices. It is created via [New] and then handed to [gore.Run].
type Frontend struct {
	gpu   GPU
	snd   Sound
	in    Input
	title string
}

// New returns a Frontend wired to the given devices. Any of gpu, snd, or in
// may be nil; the matching method then becomes a no-op. This is what lets
// downstream callers compile against this package today, while the actual
// go-virtio drivers are still being implemented in sibling sprints.
func New(gpu GPU, snd Sound, in Input) *Frontend {
	return &Frontend{gpu: gpu, snd: snd, in: in}
}

// DrawFrame ships the rendered DOOM frame to the virtio-gpu scanout.
func (f *Frontend) DrawFrame(img *image.RGBA) {
	if f.gpu == nil {
		return
	}
	_ = f.gpu.Flip(img)
}

// SetTitle records the current WAD title. With no console wired up yet,
// it is otherwise dropped.
func (f *Frontend) SetTitle(title string) {
	f.title = title
}

// Title returns the most recently announced WAD title. Useful for the host
// operator to introspect via virtio-console in a follow-up sprint.
func (f *Frontend) Title() string {
	return f.title
}

// CacheSound forwards a sound lump to the virtio-sound device.
func (f *Frontend) CacheSound(name string, data []byte) {
	if f.snd == nil {
		return
	}
	_ = f.snd.Cache(name, data)
}

// PlaySound triggers playback of a previously cached SFX lump.
func (f *Frontend) PlaySound(name string, channel, vol, sep int) {
	if f.snd == nil {
		return
	}
	_ = f.snd.Play(name, channel, vol, sep)
}

// GetEvent drains one event from the virtio-input device and translates it
// to a DOOM event. It returns false when no event is currently queued.
func (f *Frontend) GetEvent(ev *godoom.DoomEvent) bool {
	if f.in == nil {
		return false
	}
	ke, ok := f.in.Poll()
	if !ok {
		return false
	}
	doomKey, mapped := hidUsageToDoomKey(ke.HIDUsage)
	if !mapped {
		return false
	}
	if ke.Down {
		ev.Type = godoom.Ev_keydown
	} else {
		ev.Type = godoom.Ev_keyup
	}
	ev.Key = doomKey
	return true
}

// hidUsageToDoomKey translates a USB HID Keyboard usage ID into the DOOM
// scancode expected by the engine. The mapping is intentionally minimal
// until a real virtio-input driver lands and we can iterate on coverage.
func hidUsageToDoomKey(usage uint16) (uint8, bool) {
	switch usage {
	case 0x52: // Up Arrow
		return godoom.KEY_UPARROW1, true
	case 0x51: // Down Arrow
		return godoom.KEY_DOWNARROW1, true
	case 0x50: // Left Arrow
		return godoom.KEY_LEFTARROW1, true
	case 0x4f: // Right Arrow
		return godoom.KEY_RIGHTARROW1, true
	case 0x28: // Return / Enter
		return godoom.KEY_ENTER, true
	case 0x29: // Escape
		return godoom.KEY_ESCAPE, true
	case 0x2c: // Space -> Use
		return godoom.KEY_USE1, true
	case 0x2b: // Tab -> Automap
		return godoom.KEY_TAB, true
	case 0xe0, 0xe4: // Left / Right Ctrl -> Fire
		return godoom.KEY_FIRE1, true
	case 0x50 | 0x100: // (reserved) strafe-left placeholder
		return godoom.KEY_STRAFE_L1, true
	case 0x4f | 0x100: // (reserved) strafe-right placeholder
		return godoom.KEY_STRAFE_R1, true
	default:
		return 0, false
	}
}
