package main

import (
	"flag"
	"strconv"
	"strings"

	"github.com/interline-io/transitland-lib/copier"
	"github.com/interline-io/transitland-lib/ext"
	"github.com/interline-io/transitland-lib/extract"
	"github.com/interline-io/transitland-lib/internal/cli"
	"github.com/interline-io/transitland-lib/internal/log"
	"github.com/interline-io/transitland-lib/tldb"
)

// extractCommand
type extractCommand struct {
	// Default options
	copier.Options
	// Typical DMFR options
	fvid       int
	create     bool
	extensions cli.ArrayFlags
	filters    cli.ArrayFlags
	// extract specific arguments
	onlyVisitedEntities bool
	allEntities         bool
	extractAgencies     cli.ArrayFlags
	extractStops        cli.ArrayFlags
	extractTrips        cli.ArrayFlags
	extractCalendars    cli.ArrayFlags
	extractRoutes       cli.ArrayFlags
	extractRouteTypes   cli.ArrayFlags
	extractSet          cli.ArrayFlags
}

func (cmd *extractCommand) Run(args []string) error {
	fl := flag.NewFlagSet("extract", flag.ExitOnError)
	fl.Usage = func() {
		log.Print("Usage: extract <input> <output>")
		fl.PrintDefaults()
	}
	fl.Var(&cmd.extensions, "ext", "Include GTFS Extension")
	fl.IntVar(&cmd.fvid, "fvid", 0, "Specify FeedVersionID when writing to a database")
	fl.BoolVar(&cmd.create, "create", false, "Create a basic database schema if none exists")
	// Copy options
	fl.BoolVar(&cmd.AllowEntityErrors, "allow-entity-errors", false, "Allow entities with errors to be copied")
	fl.BoolVar(&cmd.AllowReferenceErrors, "allow-reference-errors", false, "Allow entities with reference errors to be copied")
	fl.BoolVar(&cmd.InterpolateStopTimes, "interpolate-stop-times", false, "Interpolate missing StopTime arrival/departure values")
	fl.BoolVar(&cmd.CreateMissingShapes, "create-missing-shapes", false, "Create missing Shapes from Trip stop-to-stop geometries")
	fl.BoolVar(&cmd.NormalizeServiceIDs, "normalize-service-ids", false, "Create any missing Calendar entities for CalendarDate service_id's")
	fl.BoolVar(&cmd.SimplifyCalendars, "simplify-calendars", false, "Attempt to simplify CalendarDates into regular Calendars")
	fl.BoolVar(&cmd.UseBasicRouteTypes, "use-basic-route-types", false, "Collapse extended route_type's into basic GTFS values")
	// Extract options
	fl.Var(&cmd.extractAgencies, "extract-agency", "Extract Agency")
	fl.Var(&cmd.extractStops, "extract-stop", "Extract Stop")
	fl.Var(&cmd.extractTrips, "extract-trip", "Extract Trip")
	fl.Var(&cmd.extractCalendars, "extract-calendar", "Extract Calendar")
	fl.Var(&cmd.extractRoutes, "extract-route", "Extract Route")
	fl.Var(&cmd.extractRouteTypes, "extract-route-type", "Extract Routes matching route_type")
	fl.Var(&cmd.extractSet, "set", "Set values on output; format is filename,id,key,value")
	// Entity selection options
	// fl.BoolVar(&cmd.onlyVisitedEntities, "only-visited-entities", false, "Only copy visited entities")
	// fl.BoolVar(&cmd.allEntities, "all-entities", false, "Copy all entities")
	fl.Parse(args)
	if fl.NArg() < 2 {
		fl.Usage()
		log.Exit("Requires input reader and output writer")
	}
	// Reader / Writer
	reader := ext.MustGetReader(fl.Arg(0))
	defer reader.Close()
	writer := ext.MustGetWriter(fl.Arg(1), cmd.create)
	defer writer.Close()
	// Setup copier
	cp := copier.NewCopier(reader, writer, cmd.Options)
	if dbw, ok := writer.(*tldb.Writer); ok {
		if cmd.fvid != 0 {
			dbw.FeedVersionID = cmd.fvid
		} else {
			fvid, err := dbw.CreateFeedVersion(reader)
			if err != nil {
				log.Exit("Error creating FeedVersion: %s", err)
			}
			dbw.FeedVersionID = fvid
		}
		cp.NormalizeServiceIDs = true
	}
	// for _, extName := range cmd.extensions {
	// 	e, err := ext.GetExtension(extName)
	// 	if err != nil {
	// 		log.Exit("No extension for: %s", extName)
	// 	}
	// 	cp.AddExtension(e)
	// 	if cmd.create {
	// 		if err := e.Create(writer); err != nil {
	// 			log.Exit("Could not create writer: %s", err)
	// 		}
	// 	}
	// }
	for _, extName := range cmd.filters {
		ef, err := ext.GetEntityFilter(extName)
		if err != nil {
			log.Exit("No filter for '%s': %s", extName, err)
		}
		cp.AddEntityFilter(ef)
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
				log.Exit("Invalid set argument")
			}
			tx.AddValue(setv[0], setv[1], setv[2], setv[3])
		}
		cp.AddEntityFilter(tx)
	}
	// Create Marker
	rthits := map[int]bool{}
	for _, i := range cmd.extractRouteTypes {
		// TODO: Use enum.GetRouteType
		if v, err := strconv.Atoi(i); err == nil {
			rthits[v] = true
		} else {
			log.Exit("Invalid route_type: %s", i)
		}
	}
	for ent := range reader.Routes() {
		if _, ok := rthits[ent.RouteType]; ok {
			cmd.extractRoutes = append(cmd.extractRoutes, ent.RouteID)
		}
	}
	//
	fm := map[string][]string{}
	fm["trips.txt"] = cmd.extractTrips[:]
	fm["agency.txt"] = cmd.extractAgencies[:]
	fm["routes.txt"] = cmd.extractRoutes[:]
	fm["calendar.txt"] = cmd.extractCalendars[:]
	fm["stops.txt"] = cmd.extractStops[:]
	count := 0
	for _, v := range fm {
		count += len(v)
	}
	// Marker
	if count > 0 {
		log.Debug("Extract filter:")
		for k, v := range fm {
			for _, i := range v {
				log.Debug("\t%s: %s", k, i)
			}
		}
		em := extract.NewMarker()
		log.Debug("Loading graph")
		em.Filter(reader, fm)
		cp.Marker = &em
		log.Debug("Graph loading complete")
	}
	// Copy
	result := cp.Copy()
	result.DisplaySummary()
	return nil
}
