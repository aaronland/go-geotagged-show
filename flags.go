package show

import (
	"flag"
	"fmt"
	"os"

	"github.com/sfomuseum/go-flags/flagset"
	"github.com/sfomuseum/go-flags/multi"
)

var port int

var map_provider string
var map_tile_uri string
var protomaps_theme string

var style string
var point_style string

var label_properties multi.MultiString
var verbose bool

var flickr_client_uri string
var flickr_root_uri string

func DefaultFlagSet() *flag.FlagSet {

	fs := flagset.NewFlagSet("show")

	fs.StringVar(&map_provider, "map-provider", "leaflet", "Valid options are: leaflet, protomaps")
	fs.StringVar(&map_tile_uri, "map-tile-uri", leaflet_osm_tile_url, "A valid Leaflet tile layer URI. See documentation for special-case (interpolated tile) URIs.")
	fs.StringVar(&protomaps_theme, "protomaps-theme", "white", "A valid Protomaps theme label.")

	fs.StringVar(&style, "style", "", "A custom Leaflet style definition for geometries. This may either be a JSON-encoded string or a path on disk.")
	fs.StringVar(&point_style, "point-style", "", "A custom Leaflet style definition for point geometries. This may either be a JSON-encoded string or a path on disk.")
	fs.IntVar(&port, "port", 0, "The port number to listen for requests on (on localhost). If 0 then a random port number will be chosen.")

	// TBD
	// fs.Var(&label_properties, "label", "Zero or more (GeoJSON Feature) properties to use to construct a label for a feature's popup menu when it is clicked on.")

	fs.StringVar(&flickr_client_uri, "flickr-client-uri", "", "This is a helper flag. If defined, any of the URIs with the \"flickr://\" scheme passed to the (show) tool containing the string \"{flickr-client-uri}\" will have those strings wil be replaced with this value. Expected to be a valid aaronland/go-flickr-api/client.Client URI")

	fs.StringVar(&flickr_root_uri, "flickr-root-uri", "", "This is a helper flag. If defined, any of the URIs with the \"flickr://\" scheme passed to the (show) tool containing the string \"{flickr-root-uri}\" will have those strings wil be replaced with this value. Expected to be a string-encoded set of query parameters that can be passed to the aaronland/go-flickr-api/fs.ReadDir method.")

	fs.BoolVar(&verbose, "verbose", false, "Enable verbose (debug) logging.")

	fs.Usage = func() {
		fmt.Fprintf(os.Stderr, "Command-line tool for showing one or more collections of geotagged photos on a map from an on-demand web server.\n")
		fmt.Fprintf(os.Stderr, "Usage:\n\t %s uri(N) uri(N)\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Valid options are:\n")
		fs.PrintDefaults()
	}

	return fs
}
