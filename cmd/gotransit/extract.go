package main

import (
	"flag"
	"fmt"
	"strconv"
	"strings"

	"github.com/interline-io/gotransit"
	"github.com/interline-io/gotransit/copier"
	"github.com/interline-io/gotransit/gtdb"
	"github.com/interline-io/gotransit/internal/extract"
	"github.com/interline-io/gotransit/internal/log"
)

// extractCommand
type extractCommand struct {
	basicCopyOptions
	// extract specific arguments
	onlyVisitedEntities  bool
	allEntities          bool
	interpolateStopTimes bool
	createMissingShapes  bool
	normalizeServiceIDs  bool
	useBasicRouteTypes   bool
	extractAgencies      arrayFlags
	extractStops         arrayFlags
	extractTrips         arrayFlags
	extractCalendars     arrayFlags
	extractRoutes        arrayFlags
	extractRouteTypes    arrayFlags
	extractSet           arrayFlags
}

func (cmd *extractCommand) Run(args []string) error {
	fl := flag.NewFlagSet("extract", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Println("Usage: extract <input> <output>")
		fl.PrintDefaults()
	}
	fl.Var(&cmd.extensions, "ext", "Include GTFS Extension")
	fl.IntVar(&cmd.fvid, "fvid", 0, "Specify FeedVersionID")
	fl.BoolVar(&cmd.newfv, "newfv", false, "Create a new FeedVersion from Reader")
	fl.BoolVar(&cmd.create, "create", false, "Create")
	fl.BoolVar(&cmd.allowEntityErrors, "allow-entity-errors", false, "Allow entity-level errors")
	fl.BoolVar(&cmd.allowReferenceErrors, "allow-reference-errors", false, "Allow reference errors")
	// Extract options
	fl.BoolVar(&cmd.interpolateStopTimes, "interpolate-stop-times", false, "Interpolate missing StopTime arrival/departure values")
	fl.BoolVar(&cmd.createMissingShapes, "create-missing-shapes", false, "Create missing Shapes from Trip stop-to-stop geometries")
	fl.BoolVar(&cmd.normalizeServiceIDs, "normalize-service-ids", false, "Create Calendar entities for CalendarDate service_id's")
	fl.BoolVar(&cmd.useBasicRouteTypes, "use-basic-route-types", false, "Collapse extended route_type's into basic GTFS values")
	// Entity selection options
	// fl.BoolVar(&cmd.onlyVisitedEntities, "only-visited-entities", false, "Only copy visited entities")
	// fl.BoolVar(&cmd.allEntities, "all-entities", false, "Copy all entities")
	fl.Var(&cmd.extractAgencies, "extract-agency", "Extract Agency")
	fl.Var(&cmd.extractStops, "extract-stop", "Extract Stop")
	fl.Var(&cmd.extractTrips, "extract-trip", "Extract Trip")
	fl.Var(&cmd.extractCalendars, "extract-calendar", "Extract Calendar")
	fl.Var(&cmd.extractRoutes, "extract-route", "Extract Route")
	fl.Var(&cmd.extractRouteTypes, "extract-route-type", "Extract Routes matching route_type")
	fl.Var(&cmd.extractSet, "set", "Set values on output; format is filename,id,key,value")
	fl.Parse(args)
	if fl.NArg() < 2 {
		fl.Usage()
		exit("requires input reader and output writer")
	}
	// Reader / Writer
	reader := MustGetReader(fl.Arg(0))
	defer reader.Close()
	writer := MustGetWriter(fl.Arg(1), cmd.create)
	defer writer.Close()
	// Setup copier
	cp := copier.NewCopier(reader, writer)
	cp.AllowEntityErrors = cmd.allowEntityErrors
	cp.AllowReferenceErrors = cmd.allowReferenceErrors
	cp.UseBasicRouteTypes = cmd.useBasicRouteTypes
	cp.InterpolateStopTimes = cmd.interpolateStopTimes
	cp.CreateMissingShapes = cmd.createMissingShapes
	cp.NormalizeServiceIDs = cmd.normalizeServiceIDs
	if dbw, ok := writer.(*gtdb.Writer); ok {
		if cmd.fvid != 0 {
			dbw.FeedVersionID = cmd.fvid
		} else if cmd.newfv {
			if _, err := dbw.CreateFeedVersion(reader); err != nil {
				exit("Error creating FeedVersion: %s", err)
			}
		}
		cp.NormalizeServiceIDs = true
	}
	for _, ext := range cmd.extensions {
		e, err := gotransit.GetExtension(ext)
		if err != nil {
			exit("No extension for: %s", ext)
		}
		cp.AddExtension(e)
		if cmd.create {
			if err := e.Create(writer); err != nil {
				exit("%s", err)
			}
		}
	}
	for _, ext := range cmd.filters {
		ef, err := gotransit.GetEntityFilter(ext)
		if err != nil {
			exit("No filter for '%s': %s", ext, err)
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
				exit("Invalid set argument")
			}
			tx.AddValue(setv[0], setv[1], setv[2], setv[3])
		}
		cp.AddEntityFilter(tx)
	}
	// Create Marker
	rthits := map[int]bool{}
	for _, i := range cmd.extractRouteTypes {
		// TODO: Use enums.GetRouteType
		if v, err := strconv.Atoi(i); err == nil {
			rthits[v] = true
		} else {
			exit("Invalid route_type: %s", i)
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
	cp.Copy()
	return nil
}
