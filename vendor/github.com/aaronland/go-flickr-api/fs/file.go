package fs

import (
	"io"
	io_fs "io/fs"
	"os"
	"time"
)

type apiFile struct {
	name           string
	perm           os.FileMode
	content        io.ReadCloser
	content_length int64
	modTime        time.Time
	closed         bool
	is_spr         bool
}

func (f *apiFile) Stat() (io_fs.FileInfo, error) {

	if f.closed {
		return nil, io_fs.ErrClosed
	}

	fi := apiFileInfo{
		name:    f.name,
		size:    f.content_length,
		modTime: f.modTime,
		mode:    f.perm,
		is_spr:  f.is_spr,
	}

	return &fi, nil
}

func (f *apiFile) Read(b []byte) (int, error) {

	if f.is_spr {
		return 0, nil
	}

	if f.closed {
		return 0, io_fs.ErrClosed
	}

	return f.content.Read(b)
}

func (f *apiFile) ReadDir(n int) ([]io_fs.DirEntry, error) {
	return make([]io_fs.DirEntry, 0), nil
}

func (f *apiFile) Close() error {

	if f.closed {
		return io_fs.ErrClosed
	}

	if f.is_spr {
		f.closed = true
		return nil
	}

	err := f.content.Close()

	if err != nil {
		return err
	}

	f.closed = true
	return nil
}
