package show

import (
	"context"
	"fmt"
	io_fs "io/fs"
	"net/url"
	"sort"
	"strings"

	"github.com/aaronland/go-roster"
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
	Close() error
}

var geotagged_fs_roster roster.Roster

type GeotaggedFSInitializationFunc func(ctx context.Context, uri string) (GeotaggedFS, error)

func RegisterGeotaggedFS(ctx context.Context, scheme string, init_func GeotaggedFSInitializationFunc) error {

	err := ensureGeotaggedFSRoster()

	if err != nil {
		return err
	}

	return geotagged_fs_roster.Register(ctx, scheme, init_func)
}

func ensureGeotaggedFSRoster() error {

	if geotagged_fs_roster == nil {

		r, err := roster.NewDefaultRoster()

		if err != nil {
			return err
		}

		geotagged_fs_roster = r
	}

	return nil
}

func NewGeotaggedFS(ctx context.Context, uri string) (GeotaggedFS, error) {

	u, err := url.Parse(uri)

	if err != nil {
		return nil, err
	}

	scheme := u.Scheme

	i, err := geotagged_fs_roster.Driver(ctx, scheme)

	if err != nil {
		return nil, err
	}

	if i == nil {
		return nil, fmt.Errorf("Missing initialization func for %s", scheme)
	}

	init_func := i.(GeotaggedFSInitializationFunc)
	return init_func(ctx, uri)
}

func GeotaggedFSSchemes() []string {

	ctx := context.Background()
	schemes := []string{}

	err := ensureGeotaggedFSRoster()

	if err != nil {
		return schemes
	}

	for _, dr := range geotagged_fs_roster.Drivers(ctx) {
		scheme := fmt.Sprintf("%s://", strings.ToLower(dr))
		schemes = append(schemes, scheme)
	}

	sort.Strings(schemes)
	return schemes
}
