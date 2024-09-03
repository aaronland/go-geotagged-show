package show

import (
	"fmt"
	io_fs "io/fs"

	flickr_fs "github.com/aaronland/go-flickr-api/fs"
)

const FLICKR_GEOTAGGEDFS_SCHEME string = "flickr"

type FlickrGeotaggedFS struct {
	GeotaggedFS
	root string
	fs   io_fs.FS
}

func (f *FlickrGeotaggedFS) Scheme() string {
	return FLICKR_GEOTAGGEDFS_SCHEME
}

func (f *FlickrGeotaggedFS) Root() string {
	return f.root
}

func (f *FlickrGeotaggedFS) FS() io_fs.FS {
	return f.fs
}

func (f *FlickrGeotaggedFS) URI(path string) (string, error) {

	if flickr_fs.MatchesPhotoURL(path) {

		v, err := flickr_fs.DerivePhotoURL(path)

		if err != nil {
			return "", fmt.Errorf("Failed to derive photo url from %s, %w", path, err)
		}

		path = v
	}

	return path, nil
}
