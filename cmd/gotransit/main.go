package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/interline-io/gotransit"
	"github.com/interline-io/gotransit/copier"
	"github.com/interline-io/gotransit/dmfr"
	_ "github.com/interline-io/gotransit/ext/pathways"
	_ "github.com/interline-io/gotransit/ext/plus"
	_ "github.com/interline-io/gotransit/gtcsv"
	"github.com/interline-io/gotransit/gtdb"
	"github.com/interline-io/gotransit/internal/extract"
	"github.com/interline-io/gotransit/internal/log"
	"github.com/interline-io/gotransit/validator"
)

// Helpers
func exit(fmts string, args ...interface{}) {
	fmt.Printf(fmts+"\n", args...)
	os.Exit(1)
}

func getReader(inurl string) gotransit.Reader {
	if len(inurl) == 0 {
		exit("No reader specified")
	}
	// Reader
	reader, err := gotransit.NewReader(inurl)
	if err != nil {
		exit("No known reader for '%s': %s", inurl, err)
	}
	if err := reader.Open(); err != nil {
		exit("Could not open '%s': %s", inurl, err)
	}
	return reader
}

func getWriter(outurl string, create bool) gotransit.Writer {
	if len(outurl) == 0 {
		exit("No writer specified")
	}
	// Writer
	writer, err := gotransit.NewWriter(outurl)
	if err != nil {
		exit("No known writer for '%s': %s", outurl, err)
	}
	if err := writer.Open(); err != nil {
		exit("Could not open '%s': %s", outurl, err)
	}
	if create {
		if err := writer.Create(); err != nil {
			exit("%s", err)
		}
	}
	return writer
}

func btos(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

// https://stackoverflow.com/questions/28322997/how-to-get-a-list-of-values-into-a-flag-in-golang/28323276#28323276
type arrayFlags []string

func (i *arrayFlags) String() string {
	return strings.Join(*i, ",")
}

func (i *arrayFlags) Set(value string) error {
	*i = append(*i, value)
	return nil
}

// basicCopyOptions
type basicCopyOptions struct {
	fvid                 int
	newfv                bool
	create               bool
	allowEntityErrors    bool
	allowReferenceErrors bool
	extensions           arrayFlags
	filters              arrayFlags
	args                 []string
}

// copyCommand
type copyCommand struct {
	basicCopyOptions
}

func (cmd *copyCommand) run(args []string) {
	fl := flag.NewFlagSet("copy", flag.ExitOnError)
	fl.Var(&cmd.extensions, "ext", "Include GTFS Extension")
	fl.IntVar(&cmd.fvid, "fvid", 0, "Specify FeedVersionID")
	fl.BoolVar(&cmd.newfv, "newfv", false, "Create a new FeedVersion from Reader")
	fl.BoolVar(&cmd.create, "create", false, "Create")
	fl.BoolVar(&cmd.allowEntityErrors, "allow-entity-errors", false, "Allow entity-level errors")
	fl.BoolVar(&cmd.allowReferenceErrors, "allow-reference-errors", false, "Allow reference errors")
	fl.Parse(args)
	cmd.args = fl.Args()
	if len(cmd.args) < 2 {
		exit("Requires input and output")
	}
	// Reader / Writer
	reader := getReader(cmd.args[0])
	defer reader.Close()
	writer := getWriter(cmd.args[1], cmd.create)
	defer writer.Close()
	// Setup copier
	cp := copier.NewCopier(reader, writer)
	cp.AllowEntityErrors = cmd.allowEntityErrors
	cp.AllowReferenceErrors = cmd.allowReferenceErrors
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
	// Add filters
	for _, ext := range cmd.filters {
		ef, err := gotransit.GetEntityFilter(ext)
		if err != nil {
			exit("No filter for '%s': %s", ext, err)
		}
		cp.AddEntityFilter(ef)
	}
	cp.Copy()
}

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

func (cmd *extractCommand) run(args []string) {
	fl := flag.NewFlagSet("extract", flag.ExitOnError)
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
	cmd.args = fl.Args()
	// Reader / Writer
	reader := getReader(cmd.args[0])
	defer reader.Close()
	writer := getWriter(cmd.args[1], cmd.create)
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
}

// validateCommand
type validateCommand struct {
	validateExtensions arrayFlags
	args               []string
}

func (cmd *validateCommand) run(args []string) {
	fl := flag.NewFlagSet("validate", flag.ExitOnError)
	fl.Var(&cmd.validateExtensions, "ext", "Include GTFS Extension")
	fl.Parse(args)
	cmd.args = fl.Args()
	//
	reader := getReader(cmd.args[0])
	defer reader.Close()
	v, err := validator.NewValidator(reader)
	if err != nil {
		panic(err)
	}
	for _, ext := range cmd.validateExtensions {
		e, err := gotransit.GetExtension(ext)
		if err != nil {
			exit("No extension for: %s", ext)
		}
		v.Copier.AddExtension(e)
	}
	v.Validate()
}

type dmfrCommand struct {
	args []string
}

func (cmd *dmfrCommand) run(args []string) {
	fl := flag.NewFlagSet("dmfr", flag.ExitOnError)
	fl.Parse(args)
	cmd.args = fl.Args()
	switch os.Args[2] {
	case "validate":
		// TODO
	case "merge":
		// TODO
	default:
		exit("%q is not valid subcommand.", os.Args[2])
	}
	for _, arg := range os.Args[2:] {
		log.Info("Loading DMFR: %s", arg)
		registry, err := dmfr.LoadAndParseRegistry(arg)
		if err != nil {
			exit("Error when loading DMFR: %s", err)
		}
		log.Info("Success loading DMFR with %d feeds", len(registry.Feeds))
	}
}

func main() {
	if len(os.Args) == 1 {
		fmt.Println("usage: gotransit <command> [<args>]")
		fmt.Println("Commands:")
		fmt.Println("  copy")
		fmt.Println("  extract")
		fmt.Println("  validate")
		fmt.Println("  dmfr")
		return
	}
	switch os.Args[1] {
	case "copy":
		cmd := copyCommand{}
		cmd.run(os.Args[2:])
	case "validate":
		cmd := validateCommand{}
		cmd.run(os.Args[2:])
	case "extract":
		cmd := extractCommand{}
		cmd.run(os.Args[2:])
	case "dmfr":
		cmd := dmfrCommand{}
		cmd.run(os.Args[2:])
	default:
		exit("%q is not valid command.", os.Args[1])
	}
}
