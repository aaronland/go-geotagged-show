package show

import (
	"context"
	"fmt"
	io_fs "io/fs"
	"log/slog"

	_ "gocloud.dev/blob/fileblob"

	"github.com/aaronland/gocloud-blob/bucket"
	"gocloud.dev/blob"
)

type BlobGeotaggedFS struct {
	GeotaggedFS
	bucket *blob.Bucket
}

func init() {
	ctx := context.Background()

	slog.Info("blob")

	for _, scheme := range blob.DefaultURLMux().BucketSchemes() {

		slog.Info("register", "scheme", scheme)
		err := RegisterGeotaggedFS(ctx, scheme, NewBlobGeotaggedFS)

		if err != nil {
			panic(err)
		}
	}
}

func NewBlobGeotaggedFS(ctx context.Context, uri string) (GeotaggedFS, error) {

	b, err := bucket.OpenBucket(ctx, uri)

	if err != nil {
		return nil, fmt.Errorf("Failed to open bucket, %w", err)
	}

	b.SetIOFSCallback(func() (context.Context, *blob.ReaderOptions) {
		ctx := context.Background()
		return ctx, nil
	})

	fs := &BlobGeotaggedFS{
		bucket: b,
	}

	return fs, nil
}

func (f *BlobGeotaggedFS) Scheme() string {
	return "blob"
}

func (f *BlobGeotaggedFS) Root() string {
	return "."
}

func (f *BlobGeotaggedFS) FS() io_fs.FS {
	return f.bucket
}

func (f *BlobGeotaggedFS) URI(path string) (string, error) {
	return path, nil
}

func (f *BlobGeotaggedFS) Close() error {
	return f.bucket.Close()
}
