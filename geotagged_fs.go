package show

import (
	io_fs "io/fs"
)

type GeotaggedFS interface {
	Root() string
	FS() io_fs.FS
}
