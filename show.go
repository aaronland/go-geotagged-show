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

	"github.com/paulmach/orb"
	"github.com/paulmach/orb/geojson"
	"github.com/rwcarlsen/goexif/exif"
	"github.com/rwcarlsen/goexif/mknote"
	"github.com/sfomuseum/go-geojson-show/static/www"
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

	geotagged_photos := make([]io_fs.FS, len(paths))

	for idx, path := range paths {

		abs_path, err := filepath.Abs(path)

		if err != nil {
			return fmt.Errorf("Failed to derive absolute path for %s, %w", path, err)
		}

		slog.Info("Add filesystem", "path", abs_path)
		geotagged_photos[idx] = os.DirFS(abs_path)
	}

	opts.GeotaggedPhotos = geotagged_photos

	return RunWithOptions(ctx, opts)
}

func RunWithOptions(ctx context.Context, opts *RunOptions) error {

	if opts.Verbose {
		slog.SetLogLoggerLevel(slog.LevelDebug)
		slog.Debug("Verbose logging enabled")
	}

	exif.RegisterParsers(mknote.All...)

	var geotagged_fs io_fs.FS
	count_geotagged := len(opts.GeotaggedPhotos)

	switch count_geotagged {
	case 0:
		return fmt.Errorf("No geotagged photo sources to crawl")
	case 1:
		geotagged_fs = opts.GeotaggedPhotos[0]
	default:

		geotagged_fs = opts.GeotaggedPhotos[0]

		for i := 1; i < count_geotagged; i++ {
			geotagged_fs = merged_fs.NewMergedFS(geotagged_fs, opts.GeotaggedPhotos[i])
		}
	}

	fc := geojson.NewFeatureCollection()
	wg := new(sync.WaitGroup)

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

			r, err := geotagged_fs.Open(path)

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

			f.Properties["image:path"] = path
			fc.Append(f)

			logger.Info("Add feature for photo", "image:path", path, "latitude", lat, "longitude", lon)
			return
		}(path)

		return nil
	}

	err := io_fs.WalkDir(geotagged_fs, ".", walk_func)

	if err != nil {
		return fmt.Errorf("Failed to walk geotagged FS, %w", err)
	}

	wg.Wait()

	mux := http.NewServeMux()

	www_fs := http.FS(www.FS)
	mux.Handle("/", http.FileServer(www_fs))

	photos_fs := http.FS(geotagged_fs)
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
