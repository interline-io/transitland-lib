package extract

import (
	"errors"
	"flag"
	"fmt"
	"strconv"
	"strings"

	"github.com/interline-io/transitland-lib/copier"
	"github.com/interline-io/transitland-lib/ext"
	_ "github.com/interline-io/transitland-lib/ext/plus"
	"github.com/interline-io/transitland-lib/filters"
	_ "github.com/interline-io/transitland-lib/filters"
	"github.com/interline-io/transitland-lib/internal/cli"
	"github.com/interline-io/transitland-lib/internal/xy"
	"github.com/interline-io/transitland-lib/log"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tldb"
)

// Command
type Command struct {
	// Default options
	copier.Options
	// Typical DMFR options
	fvid       int
	create     bool
	extensions cli.ArrayFlags
	// extract specific arguments
	Prefix            string
	extractAgencies   cli.ArrayFlags
	extractStops      cli.ArrayFlags
	extractTrips      cli.ArrayFlags
	extractCalendars  cli.ArrayFlags
	extractRoutes     cli.ArrayFlags
	extractRouteTypes cli.ArrayFlags
	extractSet        cli.ArrayFlags
	excludeAgencies   cli.ArrayFlags
	excludeStops      cli.ArrayFlags
	excludeTrips      cli.ArrayFlags
	excludeCalendars  cli.ArrayFlags
	excludeRoutes     cli.ArrayFlags
	excludeRouteTypes cli.ArrayFlags
	bbox              string
	writeExtraColumns bool
	readerPath        string
	writerPath        string
}

func (cmd *Command) Parse(args []string) error {
	fl := flag.NewFlagSet("extract", flag.ExitOnError)
	fl.Usage = func() {
		log.Print("Usage: extract <input> <output>")
		fl.PrintDefaults()
	}
	fl.Var(&cmd.extensions, "ext", "Include GTFS Extension")
	fl.IntVar(&cmd.fvid, "fvid", 0, "Specify FeedVersionID when writing to a database")
	fl.BoolVar(&cmd.create, "create", false, "Create a basic database schema if none exists")
	// Copy options
	fl.Float64Var(&cmd.SimplifyShapes, "simplify-shapes", 0.0, "Simplify shapes with this tolerance (ex. 0.000005)")
	fl.BoolVar(&cmd.AllowEntityErrors, "allow-entity-errors", false, "Allow entities with errors to be copied")
	fl.IntVar(&cmd.Options.ErrorLimit, "error-limit", 1000, "Max number of detailed errors per error group")
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
	fl.Var(&cmd.extractAgencies, "extract-agency", "Extract Agency")
	fl.Var(&cmd.extractStops, "extract-stop", "Extract Stop")
	fl.Var(&cmd.extractTrips, "extract-trip", "Extract Trip")
	fl.Var(&cmd.extractCalendars, "extract-calendar", "Extract Calendar")
	fl.Var(&cmd.extractRoutes, "extract-route", "Extract Route")
	fl.Var(&cmd.extractRouteTypes, "extract-route-type", "Extract Routes matching route_type")

	// Exclude options
	fl.Var(&cmd.excludeAgencies, "exclude-agency", "Exclude Agency")
	fl.Var(&cmd.excludeStops, "exclude-stop", "Exclude Stop")
	fl.Var(&cmd.excludeTrips, "exclude-trip", "Exclude Trip")
	fl.Var(&cmd.excludeCalendars, "exclude-calendar", "Exclude Calendar")
	fl.Var(&cmd.excludeRoutes, "exclude-route", "Exclude Route")
	fl.Var(&cmd.excludeRouteTypes, "exclude-route-type", "Exclude Routes matching route_type")

	fl.StringVar(&cmd.bbox, "bbox", "", "Extract bbox")

	fl.Var(&cmd.extractSet, "set", "Set values on output; format is filename,id,key,value")
	fl.StringVar(&cmd.Prefix, "prefix", "", "Prefix entities in this feed")

	// Entity selection options
	// fl.BoolVar(&cmd.onlyVisitedEntities, "only-visited-entities", false, "Only copy visited entities")
	// fl.BoolVar(&cmd.allEntities, "all-entities", false, "Copy all entities")
	fl.Parse(args)
	if fl.NArg() < 2 {
		fl.Usage()
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

	var bboxExcludeStops []string
	if cmd.bbox != "" {
		bbox, err := xy.ParseBbox(cmd.bbox)
		if err != nil {
			return err
		}
		for stop := range reader.Stops() {
			spt := xy.Point{
				Lon: stop.Geometry.Point.X(),
				Lat: stop.Geometry.Point.Y(),
			}
			if bbox.Contains(spt) {
				cmd.extractStops = append(cmd.extractStops, stop.StopID)
			} else {
				bboxExcludeStops = append(bboxExcludeStops, stop.StopID)
			}
		}
	}

	//
	fm := map[string][]string{}
	fm["trips.txt"] = cmd.extractTrips[:]
	fm["agency.txt"] = cmd.extractAgencies[:]
	fm["routes.txt"] = cmd.extractRoutes[:]
	fm["calendar.txt"] = cmd.extractCalendars[:]
	fm["stops.txt"] = cmd.extractStops[:]

	ex := map[string][]string{}
	ex["trips.txt"] = cmd.excludeTrips[:]
	ex["agency.txt"] = cmd.excludeAgencies[:]
	ex["routes.txt"] = cmd.excludeRoutes[:]
	ex["calendar.txt"] = cmd.excludeCalendars[:]
	ex["stops.txt"] = cmd.excludeStops[:]

	count := 0
	for _, v := range fm {
		count += len(v)
	}
	for _, v := range ex {
		count += len(v)
	}

	// Marker
	if count > 0 {
		log.Debugf("Extract filter:")
		for k, v := range fm {
			for _, i := range v {
				log.Debugf("\t%s: %s", k, i)
			}
		}
		em := NewMarker()
		log.Debugf("Loading graph")
		if err := em.Filter(reader, fm, ex); err != nil {
			return err
		}
		for _, sid := range bboxExcludeStops {
			em.Mark("stops.txt", sid, false)
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
