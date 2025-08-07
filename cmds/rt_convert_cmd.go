package cmds

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image/png"
	"os"
	"strings"

	sm "github.com/flopp/go-staticmaps"
	"github.com/golang/geo/s2"
	"github.com/interline-io/transitland-lib/request"
	"github.com/interline-io/transitland-lib/rt"
	"github.com/interline-io/transitland-lib/rt/pb"
	"github.com/interline-io/transitland-lib/tlcli"
	"github.com/spf13/pflag"
	"google.golang.org/protobuf/encoding/protojson"
)

// RTConvertCommand
type RTConvertCommand struct {
	InputFile    string
	OutputFile   string
	Format       string
	TileProvider string
	MapWidth     int
	MapHeight    int
	MarkerColor  string
}

func (cmd *RTConvertCommand) HelpDesc() (string, string) {
	return "Convert GTFS Realtime to readable formats.", "Eases inspecting live feeds. Enables processing with JSON-based tools like jq. For feeds with vehicle positions, you can also convert to GeoJSON (FeatureCollection), GeoJSONL (one feature per line), or PNG (rendered map with an optional OpenStreetMap basemap). See https://www.interline.io/blog/geojsonl-extracts/ for more information about GeoJSONL. See https://operations.osmfoundation.org/policies/tiles/ and https://github.com/CartoDB/basemap-styles for more information about base maps."
}

func (cmd *RTConvertCommand) HelpExample() string {
	return `% {{.ParentCommand}} {{.Command}} "trips.pb"
% {{.ParentCommand}} {{.Command}} --format geojson "vehicle_positions.pb"
% {{.ParentCommand}} {{.Command}} --format geojsonl "vehicle_positions.pb"
% {{.ParentCommand}} {{.Command}} --format png --out vehicles.png "vehicle_positions.pb"
% {{.ParentCommand}} {{.Command}} --format png --tiles carto-light --out vehicles.png "vehicle_positions.pb"
% {{.ParentCommand}} {{.Command}} --format png --tiles none --out vehicles.png "vehicle_positions.pb"
% {{.ParentCommand}} {{.Command}} --format png --width 800 --height 600 --color red --out vehicles.png "vehicle_positions.pb"
% {{.ParentCommand}} {{.Command}} --format png --color "#FF0000" --out vehicles.png "vehicle_positions.pb"
% {{.ParentCommand}} {{.Command}} --format json --out output.json "alerts.pb"
% {{.ParentCommand}} {{.Command}} --format geojson --out mbta_vehicles.geojson "https://cdn.mbta.com/realtime/VehiclePositions.pb"
% {{.ParentCommand}} {{.Command}} --format geojson "https://developer.trimet.org/ws/gtfs/VehiclePositions"
% {{.ParentCommand}} {{.Command}} --format json "https://developer.trimet.org/ws/V1/TripUpdate"`
}

func (cmd *RTConvertCommand) HelpArgs() string {
	return "[flags] <input pb>"
}

func (cmd *RTConvertCommand) AddFlags(fl *pflag.FlagSet) {
	fl.StringVarP(&cmd.OutputFile, "out", "o", "", "Write output to file; defaults to stdout")
	fl.StringVarP(&cmd.Format, "format", "f", "json", "Output format: json, geojson, geojsonl, png (only vehicle positions will be included in non-JSON formats)")
	fl.StringVarP(&cmd.TileProvider, "tiles", "t", "none", "Tile provider for PNG maps: carto-dark, carto-light, osm, none (transparent background)")
	fl.IntVarP(&cmd.MapWidth, "width", "w", 1200, "Map width in pixels")
	fl.IntVarP(&cmd.MapHeight, "height", "e", 900, "Map height in pixels")
	fl.StringVarP(&cmd.MarkerColor, "color", "c", "orange", "Marker color for vehicle positions")
}

func (cmd *RTConvertCommand) Parse(args []string) error {
	fl := tlcli.NewNArgs(args)
	if fl.NArg() < 1 {
		return errors.New("requires input pb")
	}
	cmd.InputFile = fl.Arg(0)
	return nil
}

func (cmd *RTConvertCommand) Run(ctx context.Context) error {
	// Fetch
	msg, err := rt.ReadURL(ctx, cmd.InputFile, request.WithAllowLocal)
	if err != nil {
		return err
	}

	var outputData []byte

	// Handle different output formats
	switch strings.ToLower(cmd.Format) {
	case "json":
		// Create json
		mOpts := protojson.MarshalOptions{UseProtoNames: true, Indent: "  "}
		outputData, err = mOpts.Marshal(msg)
		if err != nil {
			return err
		}
	case "geojson", "geojsonl":
		// Convert to GeoJSON - will handle empty vehicle positions gracefully
		outputData, err = rt.VehiclePositionsToGeoJSON(msg, cmd.Format == "geojsonl")
		if err != nil {
			return err
		}
	case "png":
		// Generate static map from vehicle positions
		if cmd.OutputFile == "" {
			return errors.New("output file (-o) is required for png format")
		}
		outputData, err = cmd.generateVehicleMap(msg)
		if err != nil {
			return err
		}
	default:
		return errors.New("unsupported format: " + cmd.Format)
	}

	// Write
	outf := os.Stdout
	if cmd.OutputFile != "" {
		var err error
		outf, err = os.Create(cmd.OutputFile)
		if err != nil {
			return err
		}
		defer outf.Close()
	}
	if _, err := outf.Write(outputData); err != nil {
		return err
	}
	return nil
}

// generateVehicleMap creates a static map image from vehicle positions
func (cmd *RTConvertCommand) generateVehicleMap(msg *pb.FeedMessage) ([]byte, error) {
	// Create a new static map
	ctx := sm.NewContext()
	ctx.SetSize(cmd.MapWidth, cmd.MapHeight)

	// Set tile provider based on user choice
	var tileProvider *sm.TileProvider
	switch cmd.TileProvider {
	case "carto-dark":
		tileProvider = sm.NewTileProviderCartoDark()
	case "carto-light":
		tileProvider = sm.NewTileProviderCartoLight()
	case "osm":
		tileProvider = sm.NewTileProviderOpenStreetMaps()
	case "none":
		tileProvider = sm.NewTileProviderNone()
	default:
		tileProvider = sm.NewTileProviderNone()
	}
	ctx.SetTileProvider(tileProvider)

	// Track if we have any valid vehicle positions
	hasValidPositions := false

	// Add vehicle positions as markers
	for _, entity := range msg.Entity {
		if entity.Vehicle == nil || entity.Vehicle.Position == nil {
			continue
		}

		// Skip if position coordinates are missing
		if entity.Vehicle.Position.Longitude == nil || entity.Vehicle.Position.Latitude == nil {
			continue
		}

		lat := float64(*entity.Vehicle.Position.Latitude)
		lng := float64(*entity.Vehicle.Position.Longitude)

		// Create a marker for this vehicle
		markerColor, err := sm.ParseColorString(cmd.MarkerColor)
		if err != nil {
			return nil, fmt.Errorf("invalid marker color '%s': %w", cmd.MarkerColor, err)
		}
		marker := sm.NewMarker(s2.LatLngFromDegrees(lat, lng), markerColor, 15.0)

		ctx.AddObject(marker)
		hasValidPositions = true
	}

	// If no valid positions, return an error
	if !hasValidPositions {
		return nil, errors.New("no valid vehicle positions found for map generation")
	}

	// Render the map with automatic bounds fitting
	img, _, err := ctx.RenderWithBounds()
	if err != nil {
		return nil, err
	}

	// Encode as PNG
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
