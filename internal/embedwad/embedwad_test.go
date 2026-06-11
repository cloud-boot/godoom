// Copyright (c) 2026 cloud-boot contributors
// SPDX-License-Identifier: GPL-2.0-only

package embedwad

import (
	"io"
	"testing"
	"testing/fstest"
)

func TestNew_BasicReadback(t *testing.T) {
	payload := []byte("IWAD\x00\x01\x02\x03")
	f := New("doom1.wad", payload)

	fp, err := f.Open("doom1.wad")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer fp.Close()

	got, err := io.ReadAll(fp)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if string(got) != string(payload) {
		t.Fatalf("payload mismatch: got %q want %q", got, payload)
	}
}

func TestNew_MissingFile(t *testing.T) {
	f := New("doom1.wad", []byte("x"))
	if _, err := f.Open("doom2.wad"); err == nil {
		t.Fatalf("expected error opening unknown file")
	}
}

func TestNew_StdFSContract(t *testing.T) {
	f := New("doom1.wad", []byte("payload-bytes"))
	if err := fstest.TestFS(f, "doom1.wad"); err != nil {
		t.Fatalf("fstest.TestFS: %v", err)
	}
}

func TestNew_StatSize(t *testing.T) {
	payload := make([]byte, 4096)
	f := New("doom1.wad", payload)
	fp, err := f.Open("doom1.wad")
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	defer fp.Close()
	st, err := fp.Stat()
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}
	if st.Size() != int64(len(payload)) {
		t.Fatalf("Size: got %d want %d", st.Size(), len(payload))
	}
	if st.IsDir() {
		t.Fatalf("IsDir should be false")
	}
	if st.Name() != "doom1.wad" {
		t.Fatalf("Name: got %q want %q", st.Name(), "doom1.wad")
	}
}
