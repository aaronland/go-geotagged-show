package show

import (
	io_fs "io/fs"
)

type FlickrGeotaggedFS struct {
	GeotaggedFS
	root string
	fs   io_fs.FS
}

func (f *FlickrGeotaggedFS) Root() string {
	return f.root
}

func (f *FlickrGeotaggedFS) FS() io_fs.FS {
	return f.fs
}
