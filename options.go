package show

import (
	"context"
	"flag"
	"fmt"

	"github.com/sfomuseum/go-flags/flagset"
	www_show "github.com/sfomuseum/go-www-show"
)

type RunOptions struct {
	MapProvider     string
	MapTileURI      string
	ProtomapsTheme  string
	Port            int
	Style           *LeafletStyle
	PointStyle      *LeafletStyle
	LabelProperties []string
	GeotaggedFS     []GeotaggedFS
	Browser         www_show.Browser
	Verbose         bool
}

func RunOptionsFromFlagSet(ctx context.Context, fs *flag.FlagSet) (*RunOptions, error) {

	flagset.Parse(fs)

	err := flagset.SetFlagsFromEnvVars(fs, "SHOW")

	if err != nil {
		return nil, fmt.Errorf("Failed to assing flags from environment variables, %w", err)
	}
	
	opts := &RunOptions{
		MapProvider:     map_provider,
		MapTileURI:      map_tile_uri,
		ProtomapsTheme:  protomaps_theme,
		Port:            port,
		LabelProperties: label_properties,
		Verbose:         verbose,
	}

	br, err := www_show.NewBrowser(ctx, "web://")

	if err != nil {
		return nil, fmt.Errorf("Failed to create new browser, %w", err)
	}

	opts.Browser = br

	if style != "" {

		s, err := UnmarshalStyle(style)

		if err != nil {
			return nil, fmt.Errorf("Failed to unmarshal style, %w", err)
		}

		opts.Style = s
	}

	if point_style != "" {

		s, err := UnmarshalStyle(point_style)

		if err != nil {
			return nil, fmt.Errorf("Failed to unmarshal point style, %w", err)
		}

		opts.PointStyle = s
	}

	return opts, nil
}
