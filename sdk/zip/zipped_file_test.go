package zip

import (
	"bytes"
	"io"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nginx/agent/sdk/v2/proto"
)

type fileDef struct {
	name    string
	mode    os.FileMode
	content []byte
}

type writeTest struct {
	prefix           string
	constErr         bool
	fileErr          bool
	protoErr         bool
	preFileCallback  func(f *Writer)
	preProtoCallback func(f *Writer)
	files            []fileDef
}

func TestWriter(t *testing.T) {
	for _, tt := range []writeTest{
		{prefix: "", constErr: true},
		{prefix: "/root"},
		{
			// should not be able to double flush
			prefix: "/root", preProtoCallback: func(f *Writer) {
				_, _ = f.Proto()
			}, protoErr: true,
		},
		{
			prefix: "/root",
			files: []fileDef{
				{"foo", 0600, []byte("foo")},
			},
		},
		{ // flushed, so should not able to add file
			prefix: "/root",
			preFileCallback: func(f *Writer) {
				_, _ = f.Proto()
			},
			fileErr: true,
			files: []fileDef{
				{"foo", 0600, []byte("foo")},
			},
		},
	} {
		f, err := NewWriter(tt.prefix)
		if tt.constErr {
			assert.Error(t, err)
			continue
		} else {
			assert.NoError(t, err)
		}
		assert.NotNil(t, f)
		if tt.preFileCallback != nil {
			tt.preFileCallback(f)
		}
		if tt.files != nil {
			for _, ff := range tt.files {
				b := bytes.NewReader(ff.content)
				err = f.Add(ff.name, ff.mode, b)
				if tt.fileErr {
					assert.Error(t, err)
					break
				} else {
					assert.NoError(t, err)
				}
			}
		}
		if tt.fileErr {
			continue
		}
		if tt.preProtoCallback != nil {
			tt.preProtoCallback(f)
		}
		p, err := f.Proto()
		if tt.protoErr {
			assert.Error(t, err)
			continue
		} else {
			assert.NoError(t, err)
		}
		assert.NotNil(t, p)

		assert.Equal(t, p.RootDirectory, tt.prefix)
		assert.NotNil(t, p.Contents)
		assert.NotEmpty(t, p.Checksum)
		if len(tt.files) > 0 {
			files := make(map[string][]byte, len(tt.files))
			for _, ff := range tt.files {
				files[ff.name] = ff.content[:]
			}
			var r *Reader
			r, err = NewReader(p)
			require.NoError(t, err)
			r.RangeFileReaders(func(err error, path string, mode os.FileMode, r io.Reader) bool {
				var b []byte
				b, err = io.ReadAll(r)
				require.NoError(t, err)
				require.Equal(t, files[path], b)
				delete(files, path)
				return true
			})
			require.Empty(t, files)
		}
	}
}

type readTest struct {
	zipFile *proto.ZippedFile
	prefix  string
	files   []fileDef
}

func createReadTest(t *testing.T, prefix string, files []fileDef) readTest {
	w, err := NewWriter(prefix)
	assert.NoError(t, err)

	for _, f := range files {
		r := bytes.NewReader(f.content)
		err = w.Add(f.name, f.mode, r)
		assert.NoError(t, err)
	}

	zipFile, err := w.Proto()
	assert.NoError(t, err)
	assert.NotEmpty(t, zipFile.Contents)
	return readTest{
		zipFile: zipFile,
		prefix:  prefix,
		files:   files,
	}
}

func TestReader(t *testing.T) {
	for _, tt := range []readTest{
		createReadTest(t, "/root", []fileDef{}),
		createReadTest(t, "/root",
			[]fileDef{
				{
					name:    "foo",
					mode:    0600,
					content: []byte("bar"),
				},
			},
		),
		createReadTest(t, "/tmp/nginx/conf",
			[]fileDef{
				{
					name: "nginx.conf",
					mode: 0600,
					content: []byte(`
					worker_processes  2;
					user              www-data;
					
					events {
						use           epoll;
						worker_connections  128;
					}
					access_log    /tmp/testdata/logs/access.log  combined;
					}`),
				},
			},
		),
	} {
		r, err := NewReader(tt.zipFile)
		defer func() {
			assert.NoError(t, r.Close())
		}()
		assert.NoError(t, err)
		assert.Equal(t, tt.prefix, r.Prefix())

		f := make([]fileDef, 0)
		r.RangeFileReaders(func(err error, path string, mode os.FileMode, rc io.Reader) bool {
			assert.NoError(t, err)

			b := bytes.NewBuffer([]byte{})
			_, err = io.Copy(b, rc)

			assert.NoError(t, err)
			f = append(f, fileDef{
				name:    path,
				mode:    mode,
				content: b.Bytes(),
			})
			return true
		})
		assert.Equal(t, f, tt.files)
	}
}

func TestUnPack(t *testing.T) {
	for _, tt := range []readTest{
		createReadTest(t, "/tmp/nginx/conf",
			[]fileDef{
				{
					name: "nginx.conf",
					mode: 0600,
					content: []byte(`
					worker_processes  2;
					user              www-data;
					
					events {
						use           epoll;
						worker_connections  128;
					}
					access_log    /tmp/testdata/logs/access.log  combined;
					}`),
				},
			},
		),
	} {
		confFiles, err := UnPack(tt.zipFile)
		assert.NoError(t, err)
		assert.NotNil(t, confFiles)
		assert.NotEmpty(t, confFiles)
	}
}
