// Copyright (c) 2026 cloud-boot contributors
// SPDX-License-Identifier: GPL-2.0-only

package tamago

import (
	"errors"
	"testing"
)

type stubSound struct {
	writes  [][]byte
	stream  uint32
	wantErr error
}

func (f *stubSound) Write(streamID uint32, frames []byte) (int, error) {
	if f.wantErr != nil {
		return 0, f.wantErr
	}
	cp := make([]byte, len(frames))
	copy(cp, frames)
	f.writes = append(f.writes, cp)
	f.stream = streamID
	return len(frames), nil
}

func dmxLump(samples ...byte) []byte {
	n := uint32(len(samples))
	hdr := []byte{
		3, 0, // format = 3
		0x11, 0x2B, // rate = 11025
		byte(n), byte(n >> 8), byte(n >> 16), byte(n >> 24),
	}
	return append(hdr, samples...)
}

func TestSoundAdapter_Cache_Play_RoundTrip(t *testing.T) {
	snd := &stubSound{}
	s := NewSoundAdapter(snd, 7)
	if err := s.Cache("DSPISTOL", dmxLump(0, 128, 255)); err != nil {
		t.Fatalf("Cache: %v", err)
	}
	if err := s.Play("DSPISTOL", 0, 64, 64); err != nil {
		t.Fatalf("Play: %v", err)
	}
	if len(snd.writes) != 1 {
		t.Fatalf("writes: got %d want 1", len(snd.writes))
	}
	if snd.stream != 7 {
		t.Fatalf("stream: got %d want 7", snd.stream)
	}
	// Each u8 sample is expanded to 2 s16 bytes.
	if got := len(snd.writes[0]); got != 6 {
		t.Fatalf("written len: got %d want 6", got)
	}
	// 0 → s16 = (0-128)<<8 = -32768 = 0x8000
	// 128 → s16 = 0
	// 255 → s16 = (255-128)<<8 = 32512 = 0x7F00
	want := []byte{0x00, 0x80, 0x00, 0x00, 0x00, 0x7F}
	for i, b := range want {
		if snd.writes[0][i] != b {
			t.Fatalf("byte %d: got 0x%02X want 0x%02X", i, snd.writes[0][i], b)
		}
	}
}

func TestSoundAdapter_Play_UnknownIsSilent(t *testing.T) {
	snd := &stubSound{}
	s := NewSoundAdapter(snd, 0)
	if err := s.Play("DSBOGUS", 0, 0, 0); err != nil {
		t.Fatalf("unknown Play: %v", err)
	}
	if len(snd.writes) != 0 {
		t.Fatalf("unknown Play wrote: %d", len(snd.writes))
	}
}

func TestSoundAdapter_Play_WriteError(t *testing.T) {
	snd := &stubSound{wantErr: errors.New("boom")}
	s := NewSoundAdapter(snd, 0)
	_ = s.Cache("X", dmxLump(0))
	if err := s.Play("X", 0, 0, 0); err == nil {
		t.Fatal("expected error from underlying Write")
	}
}

func TestSoundAdapter_Cache_ShortLumpEmpty(t *testing.T) {
	s := NewSoundAdapter(&stubSound{}, 0)
	// Less than 8-byte header: dropped silently.
	if err := s.Cache("SHORT", []byte{1, 2, 3}); err != nil {
		t.Fatalf("Cache short: %v", err)
	}
	// numSamples larger than actual remaining bytes: clamped.
	bad := []byte{3, 0, 0x11, 0x2B, 0xFF, 0xFF, 0xFF, 0xFF, 1, 2}
	if err := s.Cache("BAD", bad); err != nil {
		t.Fatalf("Cache bad: %v", err)
	}
}

func TestSoundAdapter_NilGuards(t *testing.T) {
	var s *SoundAdapter
	if err := s.Cache("X", dmxLump(0)); err != nil {
		t.Fatalf("nil Cache: %v", err)
	}
	if err := s.Play("X", 0, 0, 0); err != nil {
		t.Fatalf("nil Play: %v", err)
	}
	s = NewSoundAdapter(nil, 0)
	_ = s.Cache("X", dmxLump(0, 0, 0))
	if err := s.Play("X", 0, 0, 0); err != nil {
		t.Fatalf("nil snd Play: %v", err)
	}
}

func TestConvertDMXToS16LE_Edges(t *testing.T) {
	if got := convertDMXToS16LE(nil); got != nil {
		t.Fatalf("nil: got %v", got)
	}
	if got := convertDMXToS16LE([]byte{1, 2}); got != nil {
		t.Fatalf("short: got %v", got)
	}
	// numSamples = 0
	hdr := []byte{3, 0, 0x11, 0x2B, 0, 0, 0, 0}
	if got := convertDMXToS16LE(hdr); got != nil {
		t.Fatalf("zero samples: got %v", got)
	}
}
