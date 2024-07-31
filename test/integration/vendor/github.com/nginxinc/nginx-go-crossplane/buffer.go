/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

package crossplane

import (
	"io"
	"strings"
)

// Creator abstracts file creation (to write configs to something other than files).
type Creator interface {
	Create(string) (io.WriteCloser, error)
	Reset()
}

// FileString is a string representation of a file.
type FileString struct {
	Name string
	w    strings.Builder
}

// Write makes this an io.Writer.
func (fs *FileString) Write(b []byte) (int, error) {
	return fs.w.Write(b)
}

// Close makes this an io.Closer.
func (fs *FileString) Close() error {
	fs.w.WriteByte('\n')
	return nil
}

// String makes this a Stringer.
func (fs *FileString) String() string {
	return fs.w.String()
}

// StringsCreator is an option for rendering config files to strings(s).
type StringsCreator struct {
	Files []*FileString
}

// Create makes this a Creator.
func (sc *StringsCreator) Create(file string) (io.WriteCloser, error) {
	wc := &FileString{Name: file}
	sc.Files = append(sc.Files, wc)
	return wc, nil
}

// Reset returns the Creator to its initial state.
func (sc *StringsCreator) Reset() {
	sc.Files = []*FileString{}
}
