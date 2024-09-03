package show

import (
	io_fs "io/fs"
)

const LOCAL_GEOTAGGEDFS_SCHEME string = "local"

type LocalGeotaggedFS struct {
	GeotaggedFS
	fs io_fs.FS
}

func (f *LocalGeotaggedFS) Scheme() string {
	return LOCAL_GEOTAGGEDFS_SCHEME
}

func (f *LocalGeotaggedFS) Root() string {
	return "."
}

func (f *LocalGeotaggedFS) FS() io_fs.FS {
	return f.fs
}

func (f *LocalGeotaggedFS) URI(path string) (string, error) {
	return path, nil
}
