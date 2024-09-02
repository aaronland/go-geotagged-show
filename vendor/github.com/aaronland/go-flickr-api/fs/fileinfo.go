package fs

import (
	io_fs "io/fs"
	"time"
)

type apiFileInfo struct {
	name    string
	size    int64
	modTime time.Time
	mode    io_fs.FileMode
	is_spr  bool
}

// base name of the file
func (fi *apiFileInfo) Name() string {
	return fi.name
}

// length in bytes for regular files; system-dependent for others
func (fi *apiFileInfo) Size() int64 {
	return fi.size
}

// file mode bits
func (fi *apiFileInfo) Mode() io_fs.FileMode {
	return fi.mode
}

// modification time
func (fi *apiFileInfo) ModTime() time.Time {
	return fi.modTime
}

// abbreviation for Mode().IsDir()
func (fi *apiFileInfo) IsDir() bool {
	return fi.is_spr
}

// underlying data source (can return nil)
func (fi *apiFileInfo) Sys() interface{} {
	return nil
}
