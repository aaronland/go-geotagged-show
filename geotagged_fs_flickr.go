package show

import (
	"context"
	"fmt"
	io_fs "io/fs"
	"net/url"

	"github.com/aaronland/go-flickr-api/client"
	flickr_fs "github.com/aaronland/go-flickr-api/fs"
)

const FLICKR_GEOTAGGEDFS_SCHEME string = "flickr"

type FlickrGeotaggedFS struct {
	GeotaggedFS
	root string
	fs   io_fs.FS
}

func init() {
	ctx := context.Background()
	err := RegisterGeotaggedFS(ctx, FLICKR_GEOTAGGEDFS_SCHEME, NewFlickrGeotaggedFS)

	if err != nil {
		panic(err)
	}
}

func NewFlickrGeotaggedFS(ctx context.Context, uri string) (GeotaggedFS, error) {

	u, err := url.Parse(uri)

	if err != nil {
		return nil, fmt.Errorf("Failed to parse URI, %w", err)
	}

	q := u.Query()
	client_uri := q.Get("client-uri")
	root_uri := q.Get("root")

	cl, err := client.NewClient(ctx, client_uri)

	if err != nil {
		return nil, fmt.Errorf("Failed to create new Flickr API client, %w", err)
	}

	fs := flickr_fs.New(ctx, cl)

	flickr_fs := &FlickrGeotaggedFS{
		root: root_uri,
		fs:   fs,
	}

	return flickr_fs, nil
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

func (f *FlickrGeotaggedFS) Close() error {
	return nil
}
