package cmds

import (
	"context"
	"errors"
	"os"
	"strings"

	"github.com/interline-io/transitland-lib/request"
	"github.com/interline-io/transitland-lib/rt"
	"github.com/interline-io/transitland-lib/tlcli"
	"github.com/spf13/pflag"
	"google.golang.org/protobuf/encoding/protojson"
)

// RTConvertCommand
type RTConvertCommand struct {
	InputFile  string
	OutputFile string
	Format     string
}

func (cmd *RTConvertCommand) HelpDesc() (string, string) {
	return "Convert GTFS-Realtime to JSON.", "Convert GTFS-Realtime protocol buffer files to JSON format. Eases inspecting live GTFS Realtime feeds. Enables processing with JSON-based tools like jq. For vehicle position feeds, you can also convert to GeoJSON (FeatureCollection) or GeoJSONL (one feature per line) formats for visualization or geographic analysis. See https://www.interline.io/blog/geojsonl-extracts/ for more information about GeoJSONL."
}

func (cmd *RTConvertCommand) HelpExample() string {
	return `% {{.ParentCommand}} {{.Command}} "trips.pb"
% {{.ParentCommand}} {{.Command}} --format geojson "vehicle_positions.pb"
% {{.ParentCommand}} {{.Command}} --format geojsonl "vehicle_positions.pb"
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
	fl.StringVarP(&cmd.Format, "format", "f", "json", "Output format: json, geojson, geojsonl (geojson formats only work with vehicle positions)")
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
		// Check if this is a vehicle positions feed
		hasVehiclePositions := false
		for _, entity := range msg.Entity {
			if entity.Vehicle != nil {
				hasVehiclePositions = true
				break
			}
		}
		if !hasVehiclePositions {
			return errors.New("geojson format only supported for vehicle positions feeds")
		}

		// Convert to GeoJSON
		outputData, err = rt.VehiclePositionsToGeoJSON(msg, cmd.Format == "geojsonl")
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
