package show

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	io_fs "io/fs"
	"log"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/aaronland/go-geotagged-show/static/www"
	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geojson"
	"github.com/rwcarlsen/goexif/exif"
	"github.com/rwcarlsen/goexif/mknote"
	"github.com/sfomuseum/go-http-protomaps"
	www_show "github.com/sfomuseum/go-www-show"
	"github.com/yalue/merged_fs"
)

const leaflet_osm_tile_url = "https://tile.openstreetmap.org/{z}/{x}/{y}.png"
const protomaps_api_tile_url string = "https://api.protomaps.com/tiles/v3/{z}/{x}/{y}.mvt?key={key}"

func Run(ctx context.Context) error {
	fs := DefaultFlagSet()
	return RunWithFlagSet(ctx, fs)
}

func RunWithFlagSet(ctx context.Context, fs *flag.FlagSet) error {

	opts, err := RunOptionsFromFlagSet(ctx, fs)

	if err != nil {
		return err
	}

	paths := fs.Args()

	geotagged_fs := make([]GeotaggedFS, 0)

	for _, uri := range paths {

		u, err := url.Parse(uri)

		if err != nil {
			return fmt.Errorf("Failed to parse path %s, %w", uri, err)
		}

		q := u.Query()

		switch u.Scheme {
		case "flickr":

			client_uri := q.Get("client-uri")
			root_uri := q.Get("root")

			if flickr_client_uri != "" && client_uri == "{flickr-client-uri}" {
				q.Del("client-uri")
				q.Set("client-uri", flickr_client_uri)
			}

			if flickr_root_uri != "" && root_uri == "{flickr-root-uri}" {
				q.Del("root")
				q.Set("root", flickr_root_uri)
			}

		case "":

			u.Scheme = "local"
		}

		u.RawQuery = q.Encode()
		uri = u.String()

		new_fs, err := NewGeotaggedFS(ctx, uri)

		if err != nil {
			return fmt.Errorf("Failed to create new geotagged FS, %w", err)
		}

		geotagged_fs = append(geotagged_fs, new_fs)
	}

	opts.GeotaggedFS = geotagged_fs

	return RunWithOptions(ctx, opts)
}

func RunWithOptions(ctx context.Context, opts *RunOptions) error {

	if opts.Verbose {
		slog.SetLogLoggerLevel(slog.LevelDebug)
		slog.Debug("Verbose logging enabled")
	}

	exif.RegisterParsers(mknote.All...)

	fc := geojson.NewFeatureCollection()
	wg := new(sync.WaitGroup)
	mu := new(sync.RWMutex)

	// Walk each GeotaggedFS separately and derive suitable images for showing on
	// a map. Originally this was done by walking a single "merge" FS but that started
	// causing all kinds of headaches. It is easier just to be stupid and direct.

	for _, geotagged_fs := range opts.GeotaggedFS {

		fs_scheme := geotagged_fs.Scheme()
		fs_root := geotagged_fs.Root()

		logger := slog.Default()
		logger = logger.With("scheme", fs_scheme, "root", fs_root)

		logger.Debug("Walk filesystem")

		walk_func := func(path string, d io_fs.DirEntry, err error) error {

			if err != nil {
				return err
			}

			if d.IsDir() {
				return nil
			}

			wg.Add(1)

			go func(path string) {

				defer wg.Done()

				logger := slog.Default()
				logger = logger.With("scheme", fs_scheme)
				logger = logger.With("path", path)

				r, err := geotagged_fs.FS().Open(path)

				if err != nil {
					logger.Debug("Failed to open image for reading, skipping", "error", err)
					return
				}

				defer r.Close()

				x, err := exif.Decode(r)

				if err != nil {
					logger.Debug("Failed to decode EXIF data, skipping", "error", err)
					return
				}

				lat, lon, err := x.LatLong()

				if err != nil {
					logger.Debug("Failed to derive lat,lon from EXIF data, skipping", "error", err)
					return
				}

				pt := orb.Point([2]float64{lon, lat})
				f := geojson.NewFeature(pt)

				path, err = geotagged_fs.URI(path)

				if err != nil {
					logger.Error("Failed to derive path for scheme", "error", err)
					return
				}

				// This bit is important. It is used in conjunction with a FS "lookup" table
				// defined below to determine which FS to use for serving any given image based
				// on the image prefix (scheme)

				image_path, err := url.JoinPath(geotagged_fs.Scheme(), path)

				if err != nil {
					logger.Error("Failed to derive image path from scheme", "error", err)
					return
				}

				f.Properties["image:path"] = image_path

				// The "Append" method does not do this so we do
				// https://github.com/paulmach/orb/blob/v0.11.1/geojson/feature_collection.go#L39

				mu.Lock()
				defer mu.Unlock()

				fc.Append(f)

				logger.Info("Add feature for photo", "image:path", image_path, "latitude", lat, "longitude", lon)
				return
			}(path)

			return nil
		}

		err := io_fs.WalkDir(geotagged_fs.FS(), fs_root, walk_func)

		if err != nil {
			return fmt.Errorf("Failed to walk geotagged FS, %w", err)
		}

	}

	wg.Wait()

	mux := http.NewServeMux()

	www_fs := http.FS(www.FS)
	mux.Handle("/", http.FileServer(www_fs))

	// Loop through all the geotagged FS instances and group them by scheme
	// creating a new "merge" FS instance for each set stored in a lookup
	// table. That lookup table is used to decide which FS to use to serve
	// individual image requests. Remember: the scheme (prefix) for image
	// requests is set above in the image:path GeoJSON property

	fs_sorted := make(map[string][]io_fs.FS)
	fs_lookup := make(map[string]io_fs.FS)

	for i := 0; i < len(opts.GeotaggedFS); i++ {

		geotagged_fs := opts.GeotaggedFS[i]
		fs_scheme := geotagged_fs.Scheme()

		fs_col, exists := fs_sorted[fs_scheme]

		if !exists {
			fs_col = make([]io_fs.FS, 0)
		}

		fs_col = append(fs_col, geotagged_fs.FS())
		fs_sorted[fs_scheme] = fs_col
	}

	for scheme, fs_col := range fs_sorted {
		combined_fs := merged_fs.MergeMultiple(fs_col...)
		fs_lookup[scheme] = combined_fs
	}

	photos_prefix := "/photos/"

	photos_handler := photoHandler(fs_lookup)
	mux.Handle(photos_prefix, http.StripPrefix(photos_prefix, photos_handler))

	data_handler := dataHandler(fc)
	mux.Handle("/features.geojson", data_handler)

	//

	map_cfg := &mapConfig{
		Provider:        opts.MapProvider,
		TileURL:         opts.MapTileURI,
		Style:           opts.Style,
		PointStyle:      opts.PointStyle,
		LabelProperties: opts.LabelProperties,
	}

	if map_provider == "protomaps" {

		u, err := url.Parse(opts.MapTileURI)

		if err != nil {
			log.Fatalf("Failed to parse Protomaps tile URL, %w", err)
		}

		switch u.Scheme {
		case "file":

			mux_url, mux_handler, err := protomaps.FileHandlerFromPath(u.Path, "")

			if err != nil {
				log.Fatalf("Failed to determine absolute path for '%s', %v", opts.MapTileURI, err)
			}

			mux.Handle(mux_url, mux_handler)
			map_cfg.TileURL = mux_url

		case "api":
			key := u.Host
			map_cfg.TileURL = strings.Replace(protomaps_api_tile_url, "{key}", key, 1)
		}

		map_cfg.Protomaps = &protomapsConfig{
			Theme: opts.ProtomapsTheme,
		}
	}

	map_cfg_handler := mapConfigHandler(map_cfg)

	mux.Handle("/map.json", map_cfg_handler)

	www_show_opts := &www_show.RunOptions{
		Port:    opts.Port,
		Browser: opts.Browser,
		Mux:     mux,
	}

	return www_show.RunWithOptions(ctx, www_show_opts)
}

func dataHandler(fc *geojson.FeatureCollection) http.Handler {

	fn := func(rsp http.ResponseWriter, req *http.Request) {

		enc_json, err := fc.MarshalJSON()

		if err != nil {
			http.Error(rsp, "Internal server error", http.StatusInternalServerError)
			return
		}

		rsp.Header().Set("Content-type", "application/json")
		rsp.Write(enc_json)
		return
	}

	return http.HandlerFunc(fn)
}

func mapConfigHandler(cfg *mapConfig) http.Handler {

	fn := func(rsp http.ResponseWriter, req *http.Request) {

		rsp.Header().Set("Content-type", "application/json")

		enc := json.NewEncoder(rsp)
		err := enc.Encode(cfg)

		if err != nil {
			slog.Error("Failed to encode map config", "error", err)
			http.Error(rsp, "Internal server error", http.StatusInternalServerError)
		}

		return
	}

	return http.HandlerFunc(fn)
}

func photoHandler(fs_lookup map[string]io_fs.FS) http.Handler {

	fn := func(rsp http.ResponseWriter, req *http.Request) {

		logger := slog.Default()
		logger = logger.With("url", req.URL.Path)

		logger.Debug("Handle photos request")

		path := req.URL.Path
		path = strings.TrimLeft(path, "/")

		parts := strings.Split(path, "/")
		scheme := parts[0]

		logger = logger.With("scheme", scheme)

		geotagged_fs, exists := fs_lookup[scheme]

		if !exists {
			logger.Error("Failed to locate FS for scheme")
			http.Error(rsp, "Not found", http.StatusNotFound)
			return
		}

		scheme_prefix := fmt.Sprintf("%s/", scheme)

		photos_fs := http.FS(geotagged_fs)
		h := http.StripPrefix(scheme_prefix, http.FileServer(photos_fs))

		logger.Info("Serve, stripping prefix", "prefix", scheme_prefix)
		h.ServeHTTP(rsp, req)
		return
	}

	return http.HandlerFunc(fn)
}
