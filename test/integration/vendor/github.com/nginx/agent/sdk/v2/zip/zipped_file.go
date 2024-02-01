/**
 * Copyright (c) F5, Inc.
 *
 * This source code is licensed under the Apache License, Version 2.0 license found in the
 * LICENSE file in the root directory of this source tree.
 */

// Package zip provide convenience utilities to work with config files.
package zip

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"sync"

	"github.com/nginx/agent/sdk/v2/checksum"
	"github.com/nginx/agent/sdk/v2/files"
	"github.com/nginx/agent/sdk/v2/proto"
)

var ErrFlushed = errors.New("zipped file: already flushed")

const (
	DefaultFileMode = 0o644
)

// Writer is a helper for building the proto ZippedFile for sending with multiple file contents.
// Uses in memory bytes.Buffer, should be extended if larger file content is expected.
type Writer struct {
	sync.Mutex
	flushed bool
	prefix  string
	wrote   int
	buf     *bytes.Buffer
	gzip    *gzip.Writer
	writer  *tar.Writer
}

type Reader struct {
	prefix string
	gzip   *gzip.Reader
	reader *tar.Reader
}

// NewWriter returns a writer for to create the ZippedFile proto.
func NewWriter(prefix string) (*Writer, error) {
	if prefix == "" {
		return nil, fmt.Errorf("zip prefix path can not be empty")
	}
	b := bytes.Buffer{}
	gz := gzip.NewWriter(&b)
	return &Writer{
		prefix: prefix,
		buf:    &b,
		gzip:   gz,
		writer: tar.NewWriter(gz),
	}, nil
}

// Payloads returns the content, prefix, and checksum for the files written to the writer.
func (z *Writer) Payloads() ([]byte, string, string, error) {
	z.Lock()
	defer z.Unlock()
	if z.flushed {
		return nil, "", "", ErrFlushed
	}
	// close the writer, so it flushes to the buffer, this also means we can/should only
	// do this once.
	if err := z.writer.Close(); err != nil {
		return nil, "", "", err
	}
	if err := z.gzip.Close(); err != nil {
		return nil, "", "", err
	}
	z.flushed = true
	content := z.content()
	return content, z.prefix, checksum.HexChecksum(content), nil
}

func (z *Writer) Proto() (*proto.ZippedFile, error) {
	var err error
	p := &proto.ZippedFile{}
	p.Contents, p.RootDirectory, p.Checksum, err = z.Payloads()
	if err != nil {
		return nil, err
	}

	return p, nil
}

func (z *Writer) Add(name string, mode os.FileMode, r io.Reader) error {
	z.Lock()
	defer z.Unlock()
	if z.flushed {
		return ErrFlushed
	}
	b := bytes.NewBuffer([]byte{})
	n, err := io.Copy(b, r)
	if err != nil {
		return fmt.Errorf("zipped file: copy error %s", err)
	}
	z.wrote++
	h := &tar.Header{
		Name: name,
		Mode: int64(mode),
		Size: n,
	}
	err = z.writer.WriteHeader(h)
	if err != nil {
		return fmt.Errorf("zipped file: write header error %s", err)
	}
	_, err = z.writer.Write(b.Bytes())
	if err != nil {
		return fmt.Errorf("zipped file: write error %s", err)
	}
	return err
}

func (z *Writer) AddFile(fullPath string) error {
	r, err := os.Open(fullPath)
	if err != nil {
		return err
	}
	defer func() {
		_ = r.Close()
	}()
	return z.Add(fullPath, DefaultFileMode, r)
}

func (z *Writer) FileLen() int {
	return z.wrote
}

func (z *Writer) content() []byte {
	return z.buf.Bytes()
}

// NewReader returns a Reader to help with extracting the files from the provided ZippedFile proto.
func NewReader(p *proto.ZippedFile) (*Reader, error) {
	return NewReaderFromPayloads(p.Contents, p.RootDirectory, p.Checksum)
}

// NewReaderFromPayloads returns a Reader to help with extracting the provided zipped content
func NewReaderFromPayloads(content []byte, prefix, cs string) (*Reader, error) {
	validateChecksum := checksum.HexChecksum(content)

	if validateChecksum != cs {
		return nil, fmt.Errorf("checksum validation failed %x", validateChecksum)
	}
	gz, err := gzip.NewReader(bytes.NewReader(content))
	if err != nil {
		return nil, err
	}

	return &Reader{
		gzip:   gz,
		reader: tar.NewReader(gz),
		prefix: prefix,
	}, nil
}

type FileReaderCallback = func(err error, path string, mode os.FileMode, r io.Reader) bool

// RangeFileReaders calls f sequentially for each file in the zip archive. If f returns false, range stops the iteration.
func (r *Reader) RangeFileReaders(callback FileReaderCallback) {
	for {
		header, err := r.reader.Next()

		if err == io.EOF {
			break
		}

		if header == nil {
			continue
		}
		callback(err, header.Name, os.FileMode(header.Mode), r.reader)
	}
}

func (r *Reader) Prefix() string {
	return r.prefix
}

func (r *Reader) Close() error {
	return r.gzip.Close()
}

func UnPack(zipFile *proto.ZippedFile) ([]*proto.File, error) {
	zipContentsReader, err := NewReader(zipFile)
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = zipContentsReader.Close()
	}()

	rawFiles := make([]*proto.File, 0)
	zipContentsReader.RangeFileReaders(func(err error, path string, mode os.FileMode, rc io.Reader) bool {
		if err != nil {
			log.Print(err)
		}

		b := bytes.NewBuffer([]byte{})
		_, err = io.Copy(b, rc)
		if err != nil {
			return false
		}

		rawFiles = append(rawFiles, &proto.File{
			Name:        path,
			Permissions: files.GetPermissions(mode),
			Contents:    b.Bytes(),
		})
		return true
	})
	return rawFiles, err
}
