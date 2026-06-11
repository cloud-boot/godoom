// Copyright (c) 2026 cloud-boot contributors
// SPDX-License-Identifier: GPL-2.0-only

// Package embedwad is a scaffold for serving the DOOM WAD to the engine
// in a TamaGo bare-metal target, where there is no filesystem.
//
// The gore engine consumes its WAD through gore.SetVirtualFileSystem(fs.FS),
// so embedwad just needs to expose an fs.FS over an in-memory byte blob.
// Two construction paths are supported:
//
//  1. Compile-time: an integrator runs `go:embed doom1.wad` in their main
//     package and hands the bytes to [New]. This is what the cloud-boot
//     "livedoom" boot artifact will do for the first demo.
//  2. Run-time: the bytes are streamed from a cloud-boot OCI artifact
//     mount point and handed to [New] after boot completes. This path is
//     a follow-up sprint and is documented in PORT.md.
package embedwad

import (
	"bytes"
	"errors"
	"io"
	"io/fs"
	"time"
)

// New returns an fs.FS that contains a single file named `name` (typically
// "doom1.wad") whose contents are the provided byte slice. The slice is
// retained, not copied; callers MUST NOT mutate it after the call returns.
func New(name string, wad []byte) fs.FS {
	return &wadFS{name: name, data: wad, modTime: time.Now()}
}

type wadFS struct {
	name    string
	data    []byte
	modTime time.Time
}

func (f *wadFS) Open(name string) (fs.File, error) {
	if name == "." {
		return &wadDir{f: f}, nil
	}
	if name != f.name {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}
	return &wadFile{f: f, r: bytes.NewReader(f.data)}, nil
}

type wadFile struct {
	f *wadFS
	r *bytes.Reader
}

func (w *wadFile) Stat() (fs.FileInfo, error)         { return wadInfo{f: w.f}, nil }
func (w *wadFile) Read(p []byte) (int, error)         { return w.r.Read(p) }
func (w *wadFile) Close() error                       { return nil }
func (w *wadFile) Seek(off int64, w2 int) (int64, error) { return w.r.Seek(off, w2) }

// ReadAt satisfies io.ReaderAt — required by gore's w_Read path which
// asserts the WAD file is an io.ReaderAt for random-access lump
// reading (doom.go:40794). bytes.Reader has ReadAt; just delegate.
func (w *wadFile) ReadAt(p []byte, off int64) (int, error) { return w.r.ReadAt(p, off) }

type wadInfo struct{ f *wadFS }

func (i wadInfo) Name() string       { return i.f.name }
func (i wadInfo) Size() int64        { return int64(len(i.f.data)) }
func (i wadInfo) Mode() fs.FileMode  { return 0o444 }
func (i wadInfo) ModTime() time.Time { return i.f.modTime }
func (i wadInfo) IsDir() bool        { return false }
func (i wadInfo) Sys() any           { return nil }

type wadDir struct {
	f      *wadFS
	served bool
}

func (d *wadDir) Stat() (fs.FileInfo, error)             { return dirInfo{}, nil }
func (d *wadDir) Read([]byte) (int, error)               { return 0, errors.New("is a directory") }
func (d *wadDir) Close() error                           { return nil }
func (d *wadDir) ReadDir(n int) ([]fs.DirEntry, error) {
	if d.served {
		if n <= 0 {
			return nil, nil
		}
		return nil, io.EOF
	}
	d.served = true
	return []fs.DirEntry{dirEntry{f: d.f}}, nil
}

type dirInfo struct{}

func (dirInfo) Name() string       { return "." }
func (dirInfo) Size() int64        { return 0 }
func (dirInfo) Mode() fs.FileMode  { return fs.ModeDir | 0o555 }
func (dirInfo) ModTime() time.Time { return time.Time{} }
func (dirInfo) IsDir() bool        { return true }
func (dirInfo) Sys() any           { return nil }

type dirEntry struct{ f *wadFS }

func (e dirEntry) Name() string               { return e.f.name }
func (e dirEntry) IsDir() bool                { return false }
func (e dirEntry) Type() fs.FileMode          { return 0 }
func (e dirEntry) Info() (fs.FileInfo, error) { return wadInfo{f: e.f}, nil }
