package show

import (
	io_fs "io/fs"
)

type LocalGeotaggedFS struct {
	GeotaggedFS
	fs io_fs.FS
}

func (f *LocalGeotaggedFS) Root() string {
	return "."
}

func (f *LocalGeotaggedFS) FS() io_fs.FS {
	return f.fs
}
