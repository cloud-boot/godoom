// Copyright (c) 2026 cloud-boot contributors
// SPDX-License-Identifier: GPL-2.0-only

package tamago

import (
	"testing"
)

type stubInput struct {
	q []virtioInputEvent
}

func (f *stubInput) ReadEventRaw() (uint16, uint16, uint32, bool) {
	if len(f.q) == 0 {
		return 0, 0, 0, false
	}
	ev := f.q[0]
	f.q = f.q[1:]
	return ev.Type, ev.Code, ev.Value, true
}

func TestInputAdapter_Poll_EmptyQueue(t *testing.T) {
	i := NewInputAdapter(&stubInput{})
	if _, ok := i.Poll(); ok {
		t.Fatal("empty queue should not yield event")
	}
}

func TestInputAdapter_Poll_KeyDown(t *testing.T) {
	in := &stubInput{q: []virtioInputEvent{
		{Type: 0x01, Code: 103, Value: 1}, // KEY_UP down
	}}
	i := NewInputAdapter(in)
	ev, ok := i.Poll()
	if !ok {
		t.Fatal("expected event")
	}
	if ev.HIDUsage != 0x52 || !ev.Down {
		t.Fatalf("got %+v want HID=0x52 Down=true", ev)
	}
}

func TestInputAdapter_Poll_KeyUp(t *testing.T) {
	in := &stubInput{q: []virtioInputEvent{
		{Type: 0x01, Code: 1, Value: 0}, // KEY_ESC up
	}}
	i := NewInputAdapter(in)
	ev, ok := i.Poll()
	if !ok {
		t.Fatal("expected event")
	}
	if ev.HIDUsage != 0x29 || ev.Down {
		t.Fatalf("got %+v want HID=0x29 Down=false", ev)
	}
}

func TestInputAdapter_Poll_NonKeyDropped(t *testing.T) {
	in := &stubInput{q: []virtioInputEvent{
		{Type: 0x02, Code: 0, Value: 1}, // EV_REL
	}}
	i := NewInputAdapter(in)
	if _, ok := i.Poll(); ok {
		t.Fatal("EV_REL should be dropped")
	}
}

func TestInputAdapter_Poll_UnmappedDropped(t *testing.T) {
	in := &stubInput{q: []virtioInputEvent{
		{Type: 0x01, Code: 0xFFFE, Value: 1},
	}}
	i := NewInputAdapter(in)
	if _, ok := i.Poll(); ok {
		t.Fatal("unmapped code should be dropped")
	}
}

func TestInputAdapter_NilGuards(t *testing.T) {
	var i *InputAdapter
	if _, ok := i.Poll(); ok {
		t.Fatal("nil receiver should yield no event")
	}
	i = NewInputAdapter(nil)
	if _, ok := i.Poll(); ok {
		t.Fatal("nil reader should yield no event")
	}
}

func TestEvdevCodeToHIDUsage_AllMapped(t *testing.T) {
	cases := []struct {
		code uint16
		hid  uint16
	}{
		{1, 0x29}, {15, 0x2B}, {28, 0x28}, {29, 0xE0},
		{57, 0x2C}, {97, 0xE4},
		{103, 0x52}, {105, 0x50}, {106, 0x4F}, {108, 0x51},
	}
	for _, c := range cases {
		got, ok := evdevCodeToHIDUsage(c.code)
		if !ok || got != c.hid {
			t.Fatalf("code %d: got %#x ok=%v want %#x", c.code, got, ok, c.hid)
		}
	}
}

func TestInputAdapter_Poll_Repeat_FoldedToDown(t *testing.T) {
	in := &stubInput{q: []virtioInputEvent{
		{Type: 0x01, Code: 103, Value: 2}, // KEY_UP autorepeat
	}}
	i := NewInputAdapter(in)
	ev, ok := i.Poll()
	if !ok || !ev.Down {
		t.Fatalf("repeat should fold to Down=true, got %+v ok=%v", ev, ok)
	}
}
