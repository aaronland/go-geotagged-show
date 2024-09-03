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
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/aaronland/go-flickr-api/client"
	flickr_fs "github.com/aaronland/go-flickr-api/fs"
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

		var fs io_fs.FS

		switch u.Scheme {
		case "flickr":

			q := u.Query()
			client_uri := q.Get("client-uri")
			root := q.Get("root")

			if flickr_client_uri != "" && strings.Contains(client_uri, "{flickr-client-uri}") {
				client_uri = strings.Replace(client_uri, "{flickr-client-uri}", flickr_client_uri, 1)
			}

			cl, err := client.NewClient(ctx, client_uri)

			if err != nil {
				return fmt.Errorf("Failed to create new Flickr API client, %w", err)
			}

			fs = flickr_fs.New(ctx, cl)

			flickr_fs := &FlickrGeotaggedFS{
				root: root,
				fs:   fs,
			}

			geotagged_fs = append(geotagged_fs, flickr_fs)

		default:
			abs_path, err := filepath.Abs(uri)

			if err != nil {
				return fmt.Errorf("Failed to derive absolute path for %s, %w", uri, err)
			}

			fs = os.DirFS(abs_path)

			local_fs := &LocalGeotaggedFS{
				fs: fs,
			}

			geotagged_fs = append(geotagged_fs, local_fs)

		}

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

	for _, geotagged_fs := range opts.GeotaggedFS {

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

				if flickr_fs.MatchesPhotoURL(path) {

					v, err := flickr_fs.DerivePhotoURL(path)

					if err != nil {
						logger.Debug("Failed to derive Flickr photo URL, skipping", "error", err)
						return
					}

					logger = logger.With("new path", v)
					logger.Debug("Reassign path derived from Flickr URL")
					path = v
				}

				f.Properties["image:path"] = path
				fc.Append(f)

				logger.Info("Add feature for photo", "image:path", path, "latitude", lat, "longitude", lon)
				return
			}(path)

			return nil
		}

		err := io_fs.WalkDir(geotagged_fs.FS(), geotagged_fs.Root(), walk_func)

		if err != nil {
			return fmt.Errorf("Failed to walk geotagged FS, %w", err)
		}

	}

	wg.Wait()

	mux := http.NewServeMux()

	www_fs := http.FS(www.FS)
	mux.Handle("/", http.FileServer(www_fs))

	var combined_fs io_fs.FS
	count_geotagged := len(opts.GeotaggedFS)

	switch count_geotagged {
	case 0:
		return fmt.Errorf("No geotagged photo sources to crawl")
	case 1:
		combined_fs = opts.GeotaggedFS[0].FS()
	default:

		combined_fs = opts.GeotaggedFS[0].FS()

		for i := 1; i < count_geotagged; i++ {
			combined_fs = merged_fs.NewMergedFS(combined_fs, opts.GeotaggedFS[i].FS())
		}
	}

	photos_fs := http.FS(combined_fs)
	photos_prefix := "/photos/"

	mux.Handle(photos_prefix, http.StripPrefix(photos_prefix, http.FileServer(photos_fs)))

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
