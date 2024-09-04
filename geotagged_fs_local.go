package show

import (
	"context"
	"fmt"
	io_fs "io/fs"
	"net/url"
	"os"
	"path/filepath"
)

const LOCAL_GEOTAGGEDFS_SCHEME string = "local"

type LocalGeotaggedFS struct {
	GeotaggedFS
	fs io_fs.FS
}

func init() {
	ctx := context.Background()
	err := RegisterGeotaggedFS(ctx, LOCAL_GEOTAGGEDFS_SCHEME, NewLocalGeotaggedFS)

	if err != nil {
		panic(err)
	}
}

func NewLocalGeotaggedFS(ctx context.Context, uri string) (GeotaggedFS, error) {

	u, err := url.Parse(uri)

	if err != nil {
		return nil, fmt.Errorf("Failed to parse URI, %w", err)
	}

	abs_path, err := filepath.Abs(u.Path)

	if err != nil {
		return nil, fmt.Errorf("Failed to derive absolute path for %s, %w", uri, err)
	}

	fs := os.DirFS(abs_path)

	local_fs := &LocalGeotaggedFS{
		fs: fs,
	}

	return local_fs, nil
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

func (f *LocalGeotaggedFS) Close() error {
	return nil
}
