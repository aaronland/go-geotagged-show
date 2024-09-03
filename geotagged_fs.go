package show

import (
	io_fs "io/fs"
)

// GeotaggedFS defines an interface wrapping `io/fs.FS` instances for use by the `go-geotagged-show` package.
type GeotaggedFS interface {
	// The scheme (or label) for the implementation of the interface.
	Scheme() string
	// The "root directory" to be use when invoking `io/fs.WalkDir` on the implementation.
	Root() string
	// The underlying `io/fs.FS` instance for the implementation.
	FS() io_fs.FS
	// URI performs any transformations necessary to convert a string in to a URI referencing an item in the FS for external use.
	URI(string) (string, error)
}
