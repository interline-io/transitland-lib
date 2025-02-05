package cmds

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/adapters"
	"github.com/interline-io/transitland-lib/copier"
	"github.com/interline-io/transitland-lib/ext"
	_ "github.com/interline-io/transitland-lib/ext/plus"
	"github.com/interline-io/transitland-lib/extract"
	"github.com/interline-io/transitland-lib/filters"
	"github.com/interline-io/transitland-lib/tlcli"
	"github.com/spf13/pflag"
)

// ExtractCommand
type ExtractCommand struct {
	// Default options
	copier.Options
	// Typical DMFR options
	fvid       int
	create     bool
	extensions []string
	// extract specific arguments
	Prefix            string
	extractAgencies   []string
	extractStops      []string
	extractTrips      []string
	extractCalendars  []string
	extractRoutes     []string
	extractRouteTypes []string
	extractSet        []string
	excludeAgencies   []string
	excludeStops      []string
	excludeTrips      []string
	excludeCalendars  []string
	excludeRoutes     []string
	excludeRouteTypes []string
	bbox              string
	writeExtraColumns bool
	readerPath        string
	writerPath        string
}

func (cmd *ExtractCommand) HelpDesc() (string, string) {
	a := "Extract a subset of a GTFS feed"
	b := `The extract command extends the basic copy command with a number of additional options and transformations. It can be used to pull out a single route or trip, interpolate stop times, override a single value on an entity, etc. This is a separate command to keep the basic copy command simple while allowing the extract command to grow and add more features over time.`
	return a, b
}

func (cmd *ExtractCommand) HelpExample() string {
	return `
# Extract a single trip from the BART GTFS, and rename the agency to "test".
% {{.ParentCommand}} {{.Command}} --extract-trip "3050453" --set "agency.txt,BART,agency_id,test" "https://www.bart.gov/dev/schedules/google_transit.zip" output2.zip

# Note renamed agency
% unzip -p output2.zip agency.txt
agency_id,agency_name,agency_url,agency_timezone,agency_lang,agency_phone,agency_fare_url,agency_email
test,Bay Area Rapid Transit,https://www.bart.gov/,America/Los_Angeles,,510-464-6000,,

# Only entities related to the specified trip are included in the output.
% unzip -p output2.zip trips.txt
route_id,service_id,trip_id,trip_headsign,trip_short_name,direction_id,block_id,shape_id,wheelchair_accessible,bikes_allowed
1,2020_09_14-DX-MVS-Weekday-15,3050453,San Francisco International Airport,,1,,01_shp,0,0

$ unzip -p output2.zip routes.txt
route_id,agency_id,route_short_name,route_long_name,route_desc,route_type,route_url,route_color,route_text_color,route_sort_order
1,test,YL-S,Antioch to SFIA/Millbrae,,1,http://www.bart.gov/schedules/bylineresults?route=1,FFFF33,,0

% unzip -p output2.zip stop_times.txt
trip_id,arrival_time,departure_time,stop_id,stop_sequence,stop_headsign,pickup_type,drop_off_type,shape_dist_traveled,timepoint
3050453,04:53:00,04:53:00,CONC,0,,0,0,0.00000,0
3050453,04:58:00,04:58:00,PHIL,2,,0,0,4.06000,0
3050453,05:01:00,05:02:00,WCRK,3,,0,0,5.77000,0
3050453,05:06:00,05:07:00,LAFY,4,,0,0,9.23000,0
3050453,05:11:00,05:12:00,ORIN,5,,0,0,12.99000,0
3050453,05:17:00,05:18:00,ROCK,6,,0,0,17.38000,0
...
`
}

func (cmd *ExtractCommand) HelpArgs() string {
	return "[flags] <reader> <writer>"
}

func (cmd *ExtractCommand) AddFlags(fl *pflag.FlagSet) {
	fl.StringArrayVar(&cmd.extensions, "ext", nil, "Include GTFS Extension")
	fl.IntVar(&cmd.fvid, "fvid", 0, "Specify FeedVersionID when writing to a database")
	fl.BoolVar(&cmd.create, "create", false, "Create a basic database schema if none exists")
	// Copy options
	fl.Float64Var(&cmd.SimplifyShapes, "simplify-shapes", 0.0, "Simplify shapes with this tolerance (ex. 0.000005)")
	fl.BoolVar(&cmd.AllowEntityErrors, "allow-entity-errors", false, "Allow entities with errors to be copied")
	fl.IntVar(&cmd.Options.ErrorLimit, "error-limit", 10, "Max number of detailed errors per error group")
	fl.BoolVar(&cmd.AllowReferenceErrors, "allow-reference-errors", false, "Allow entities with reference errors to be copied")
	fl.BoolVar(&cmd.InterpolateStopTimes, "interpolate-stop-times", false, "Interpolate missing StopTime arrival/departure values")
	fl.BoolVar(&cmd.CreateMissingShapes, "create-missing-shapes", false, "Create missing Shapes from Trip stop-to-stop geometries")
	fl.BoolVar(&cmd.NormalizeServiceIDs, "normalize-service-ids", false, "Create any missing Calendar entities for CalendarDate service_id's")
	fl.BoolVar(&cmd.Options.DeduplicateJourneyPatterns, "deduplicate-stop-times", false, "Deduplicate StopTimes using Journey Patterns")
	fl.BoolVar(&cmd.SimplifyCalendars, "simplify-calendars", false, "Attempt to simplify CalendarDates into regular Calendars")
	fl.BoolVar(&cmd.Options.NormalizeTimezones, "normalize-timezones", false, "Normalize timezones and apply default stop timezones based on agency and parent stops")
	fl.BoolVar(&cmd.UseBasicRouteTypes, "use-basic-route-types", false, "Collapse extended route_type's into basic GTFS values")
	fl.BoolVar(&cmd.CopyExtraFiles, "write-extra-files", false, "Copy additional files found in source to destination")
	fl.BoolVar(&cmd.writeExtraColumns, "write-extra-columns", false, "Include extra columns in output")

	// Extract options
	fl.StringArrayVar(&cmd.extractAgencies, "extract-agency", nil, "Extract Agency")
	fl.StringArrayVar(&cmd.extractStops, "extract-stop", nil, "Extract Stop")
	fl.StringArrayVar(&cmd.extractTrips, "extract-trip", nil, "Extract Trip")
	fl.StringArrayVar(&cmd.extractCalendars, "extract-calendar", nil, "Extract Calendar")
	fl.StringArrayVar(&cmd.extractRoutes, "extract-route", nil, "Extract Route")
	fl.StringArrayVar(&cmd.extractRouteTypes, "extract-route-type", nil, "Extract Routes matching route_type")

	// Exclude options
	fl.StringArrayVar(&cmd.excludeAgencies, "exclude-agency", nil, "Exclude Agency")
	fl.StringArrayVar(&cmd.excludeStops, "exclude-stop", nil, "Exclude Stop")
	fl.StringArrayVar(&cmd.excludeTrips, "exclude-trip", nil, "Exclude Trip")
	fl.StringArrayVar(&cmd.excludeCalendars, "exclude-calendar", nil, "Exclude Calendar")
	fl.StringArrayVar(&cmd.excludeRoutes, "exclude-route", nil, "Exclude Route")
	fl.StringArrayVar(&cmd.excludeRouteTypes, "exclude-route-type", nil, "Exclude Routes matching route_type")

	fl.StringVar(&cmd.bbox, "bbox", "", "Extract bbox as (min lon, min lat, max lon, max lat), e.g. -122.276,37.794,-122.259,37.834")

	fl.StringArrayVar(&cmd.extractSet, "set", nil, "Set values on output; format is filename,id,key,value")
	fl.StringVar(&cmd.Prefix, "prefix", "", "Prefix entities in this feed")

}

func (cmd *ExtractCommand) Parse(args []string) error {
	fl := tlcli.NewNArgs(args)
	if fl.NArg() < 2 {
		return errors.New("requires input reader and output writer")
	}
	cmd.readerPath = fl.Arg(0)
	cmd.writerPath = fl.Arg(1)
	return nil
}

func (cmd *ExtractCommand) Run(ctx context.Context) error {
	// Reader / Writer
	reader, err := ext.OpenReader(cmd.readerPath)
	if err != nil {
		return err
	}
	defer reader.Close()
	writer, err := ext.OpenWriter(cmd.writerPath, cmd.create)
	if err != nil {
		return err
	}
	defer writer.Close()

	if cmd.writeExtraColumns {
		if v, ok := writer.(adapters.WriterWithExtraColumns); ok {
			v.WriteExtraColumns(true)
		} else {
			return errors.New("writer does not support extra output columns")
		}
	}

	// Setup copier
	cmd.Options.Extensions = cmd.extensions
	cp, err := copier.NewCopier(reader, writer, cmd.Options)
	if err != nil {
		return err
	}

	if cmd.Prefix != "" {
		pfx, _ := filters.NewPrefixFilter()
		pfx.PrefixAll = true
		pfx.SetPrefix(0, cmd.Prefix)
		cp.AddExtension(pfx)
	}

	// Create SetterFilter
	setvalues := [][]string{}
	for _, setv := range cmd.extractSet {
		setvalues = append(setvalues, strings.Split(setv, ","))
	}
	if len(setvalues) > 0 {
		tx := extract.NewSetterFilter()
		for _, setv := range setvalues {
			if len(setv) != 4 {
				return errors.New("invalid set argument")
			}
			tx.AddValue(setv[0], setv[1], setv[2], setv[3])
		}
		cp.AddExtension(tx)
	}

	// Create Marker
	rthits := map[int]bool{}
	for _, i := range cmd.extractRouteTypes {
		// TODO: Use tt.GetRouteType
		if v, err := strconv.Atoi(i); err == nil {
			rthits[v] = true
		} else {
			return fmt.Errorf("invalid route_type: %s", i)
		}
	}
	for _, i := range cmd.excludeRouteTypes {
		if v, err := strconv.Atoi(i); err == nil {
			rthits[v] = false
		} else {
			return fmt.Errorf("invalid route_type: %s", i)
		}
	}
	for ent := range reader.Routes() {
		v, ok := rthits[ent.RouteType.Int()]
		if !ok {
			continue
		}
		if v {
			cmd.extractRoutes = append(cmd.extractRoutes, ent.RouteID.Val)
		} else {
			cmd.excludeRoutes = append(cmd.excludeRoutes, ent.RouteID.Val)
		}
	}

	em := extract.NewMarker()
	// Includes
	em.SetBbox(cmd.bbox)
	for _, eid := range cmd.extractTrips {
		em.AddInclude("trips.txt", eid)
	}
	for _, eid := range cmd.extractAgencies {
		em.AddInclude("agency.txt", eid)
	}
	for _, eid := range cmd.extractRoutes {
		em.AddInclude("routes.txt", eid)
	}
	for _, eid := range cmd.extractCalendars {
		em.AddInclude("calendar.txt", eid)
	}
	for _, eid := range cmd.extractStops {
		em.AddInclude("stops.txt", eid)
	}
	// Excludes
	for _, eid := range cmd.excludeTrips {
		em.AddExclude("trips.txt", eid)
	}
	for _, eid := range cmd.excludeAgencies {
		em.AddExclude("agency.txt", eid)
	}
	for _, eid := range cmd.excludeRoutes {
		em.AddExclude("routes.txt", eid)
	}
	for _, eid := range cmd.excludeCalendars {
		em.AddExclude("calendar.txt", eid)
	}
	for _, eid := range cmd.excludeStops {
		em.AddExclude("stops.txt", eid)
	}

	// Marker
	if em.Count() > 0 {
		log.For(ctx).Debug().Msgf("Extract filter: loading graph")
		if err := em.Filter(reader); err != nil {
			return err
		}
		cp.Marker = &em
		log.For(ctx).Debug().Msgf("Graph loading complete")
	}

	// Copy
	result := cp.Copy()
	result.DisplaySummary()
	result.DisplayErrors()
	return nil
}
