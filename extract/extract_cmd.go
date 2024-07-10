package extract

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/cmd/tlcli"
	"github.com/interline-io/transitland-lib/copier"
	"github.com/interline-io/transitland-lib/ext"
	_ "github.com/interline-io/transitland-lib/ext/plus"
	"github.com/interline-io/transitland-lib/filters"
	_ "github.com/interline-io/transitland-lib/filters"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tldb"
	"github.com/spf13/pflag"
)

// Command
type Command struct {
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

func (cmd *Command) HelpDesc() (string, string) {
	return "Extract a subset of a GTFS feed", ""
}

func (cmd *Command) HelpArgs() string {
	return "[flags] <reader> <writer>"
}

func (cmd *Command) AddFlags(fl *pflag.FlagSet) {
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

func (cmd *Command) Parse(args []string) error {
	fl := tlcli.NewNArgs(args)
	if fl.NArg() < 2 {
		return errors.New("requires input reader and output writer")
	}
	cmd.readerPath = fl.Arg(0)
	cmd.writerPath = fl.Arg(1)
	return nil
}

func (cmd *Command) Run() error {
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
		if v, ok := writer.(tl.WriterWithExtraColumns); ok {
			v.WriteExtraColumns(true)
		} else {
			return errors.New("writer does not support extra output columns")
		}
	}

	// Create fv
	if dbw, ok := writer.(*tldb.Writer); ok {
		if cmd.fvid != 0 {
			dbw.FeedVersionID = cmd.fvid
		} else {
			fvid, err := dbw.CreateFeedVersion(reader)
			if err != nil {
				return fmt.Errorf("error creating feed version: %s", err.Error())
			}
			dbw.FeedVersionID = fvid
		}
		cmd.Options.NormalizeServiceIDs = true
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
		tx := NewSetterFilter()
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
		v, ok := rthits[ent.RouteType]
		if !ok {
			continue
		}
		if v {
			cmd.extractRoutes = append(cmd.extractRoutes, ent.RouteID)
		} else {
			cmd.excludeRoutes = append(cmd.excludeRoutes, ent.RouteID)
		}
	}

	em := NewMarker()
	// Includes
	em.bbox = cmd.bbox
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
		log.Debugf("Extract filter: loading graph")
		if err := em.Filter(reader); err != nil {
			return err
		}
		cp.Marker = &em
		log.Debugf("Graph loading complete")
	}

	// Copy
	result := cp.Copy()
	result.DisplaySummary()
	result.DisplayErrors()
	return nil
}
