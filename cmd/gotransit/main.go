package main

import (
	"flag"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/interline-io/gotransit"
	"github.com/interline-io/gotransit/copier"
	_ "github.com/interline-io/gotransit/ext/pathways"
	_ "github.com/interline-io/gotransit/ext/plus"
	"github.com/interline-io/gotransit/extract"
	_ "github.com/interline-io/gotransit/gtcsv"
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

func getWriter(outurl string) gotransit.Writer {
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

// copyCommand
type copyCommand struct {
	create               bool
	allowEntityErrors    bool
	allowReferenceErrors bool
	extensions           arrayFlags
	filters              arrayFlags
	args                 []string
}

func (cmd *copyCommand) run(args []string) {
	fl := flag.NewFlagSet("copy", flag.ExitOnError)
	fl.Var(&cmd.extensions, "ext", "Include GTFS Extension")
	fl.BoolVar(&cmd.create, "create", false, "Create")
	fl.BoolVar(&cmd.allowEntityErrors, "allow-entity-errors", false, "")
	fl.BoolVar(&cmd.allowReferenceErrors, "allow-reference-errors", false, "")
	// fl.BoolVar(&cmd.copyNewfv, "newfv", false, "Create new Feed Version")
	// fl.BoolVar(&cmd.copyResults, "results", false, "Show entity counts in output")
	// fl.IntVar(&cmd.copyFvid, "fv", 0, "Feed Version ID")
	fl.Parse(args)
	cmd.args = fl.Args()
	if len(cmd.args) < 2 {
		exit("requires input and output")
	}
	// Reader
	reader := getReader(cmd.args[0])
	defer reader.Close()
	writer := getWriter(cmd.args[1])
	defer writer.Close()
	if cmd.create {
		if err := writer.Create(); err != nil {
			exit("%s", err)
		}
	}
	// Add extensions
	cp := copier.NewCopier(reader, writer)
	for _, ext := range cmd.extensions {
		e, err := gotransit.GetExtension(ext)
		if err != nil {
			exit("no extension for: %s", ext)
		}
		cp.AddExtension(e)
		e.Create(writer)
	}
	// Add filters
	for _, ext := range cmd.filters {
		ef, err := gotransit.GetEntityFilter(ext)
		if err != nil {
			exit("no filter for '%s': %s", ext, err)
		}
		cp.AddEntityFilter(ef)
	}
	// Copy
	cp.Copy()
	cp.CopyExtensions()
}

// extractCommand
type extractCommand struct {
	create               bool
	allowEntityErrors    bool
	allowReferenceErrors bool
	extensions           arrayFlags
	filters              arrayFlags
	args                 []string
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
	fl.BoolVar(&cmd.create, "create", false, "Create")
	fl.BoolVar(&cmd.allowEntityErrors, "allow-entity-errors", false, "")
	fl.BoolVar(&cmd.allowReferenceErrors, "allow-reference-errors", false, "")
	// Extract options
	fl.BoolVar(&cmd.interpolateStopTimes, "interpolate-stop-times", false, "")
	fl.BoolVar(&cmd.createMissingShapes, "create-missing-shapes", false, "")
	fl.BoolVar(&cmd.normalizeServiceIDs, "normalize-service-ids", false, "")
	fl.BoolVar(&cmd.useBasicRouteTypes, "use-basic-route-types", false, "")
	// Entity selection options
	fl.BoolVar(&cmd.onlyVisitedEntities, "only-visited-entities", false, "")
	fl.BoolVar(&cmd.allEntities, "all-entities", false, "")
	fl.Var(&cmd.extractAgencies, "extract-agency", "Extract Agency")
	fl.Var(&cmd.extractStops, "extract-stop", "Extract Stop")
	fl.Var(&cmd.extractTrips, "extract-trip", "Extract Trip")
	fl.Var(&cmd.extractCalendars, "extract-calendar", "Extract Calendar")
	fl.Var(&cmd.extractRoutes, "extract-route", "Extract Route")
	fl.Var(&cmd.extractRouteTypes, "extract-route-type", "Extract Routes matching route_type")
	fl.Var(&cmd.extractSet, "set", "Set values on output; format is filename,id,key,value")
	fl.Parse(args)
	cmd.args = fl.Args()
	//
	reader := getReader(cmd.args[0])
	defer reader.Close()
	writer := getWriter(cmd.args[1])
	defer writer.Close()
	// Setup copier
	cp := copier.NewCopier(reader, writer)
	// Set copier options
	cp.AllowEntityErrors = cmd.allowEntityErrors
	cp.AllowReferenceErrors = cmd.allowReferenceErrors
	cp.UseBasicRouteTypes = cmd.useBasicRouteTypes
	// Set values
	setvalues := [][]string{}
	for _, setv := range cmd.extractSet {
		setvalues = append(setvalues, strings.Split(setv, ","))
	}
	if len(setvalues) > 0 {
		tx := extract.NewSetterFilter()
		for _, setv := range setvalues {
			if len(setv) != 4 {
				fmt.Println("invalid set argument")
				continue
			}
			tx.AddValue(setv[0], setv[1], setv[2], setv[3])
		}
		cp.AddEntityFilter(tx)
	}
	// Extract entities
	fm := map[string][]string{}
	rthits := map[int]bool{}
	for _, i := range cmd.extractRouteTypes {
		if v, err := strconv.Atoi(i); err == nil {
			rthits[v] = true
		} else {
			fmt.Println("invalid route_type:", i)
		}
	}
	for ent := range reader.Routes() {
		if _, ok := rthits[ent.RouteType]; ok {
			fm["routes.txt"] = append(fm["routes.txt"], ent.RouteID)
		}
	}
	// Regular IDs
	for _, i := range cmd.extractTrips {
		fm["trips.txt"] = append(fm["trips.txt"], i)
	}
	for _, i := range cmd.extractAgencies {
		fm["agency.txt"] = append(fm["agency.txt"], i)
	}
	for _, i := range cmd.extractRoutes {
		fm["routes.txt"] = append(fm["routes.txt"], i)
	}
	for _, i := range cmd.extractCalendars {
		fm["calendar.txt"] = append(fm["calendar.txt"], i)
	}
	for _, i := range cmd.extractStops {
		fm["stops.txt"] = append(fm["stops.txt"], i)
	}
	log.Debug("Extract filter:")
	for k, v := range fm {
		for _, i := range v {
			log.Debug("\t%s: %s", k, i)
		}
	}
	// Marker
	em := extract.NewExtractMarker()
	em.Load(reader)
	em.Filter(fm)
	cp.Marker = &em
	// Copy
	cp.Copy()
	cp.CopyExtensions()
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

func main() {
	if len(os.Args) == 1 {
		fmt.Println("usage: gotransit <command> [<args>]")
		fmt.Println("Commands:")
		fmt.Println("  copy")
		fmt.Println("  extract")
		fmt.Println("  validate")
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
	default:
		exit("%q is not valid command.", os.Args[1])
	}
}

// if v, ok := writer.(*gtdb.Writer); ok {
// 	// If writing to database, we must normalize calendars
// 	nsids := true
// 	copyNormalizeServiceIDs = &nsids
// 	db := v.Adapter.DB()
// 	if *copyCreate == true {
// 		writer.Create()
// 	}
// 	if *copyNewfv == true {
// 		fv, err := gotransit.NewFeedVersion(reader)
// 		if err != nil {
// 			exit("Could not create FeedVersion: %s", err)
// 		}
// 		dberr := db.
// 			Where(gotransit.FeedVersion{URL: inurl, SHA1: fv.SHA1}).
// 			FirstOrCreate(&fv).
// 			Error
// 		if dberr != nil {
// 			exit("Could not create FeedVersion: %s", dberr)
// 		}
// 		copyFvid = &fv.ID
// 	}
// 	if *copyFvid > 0 {
// 		v.FeedVersionID = *copyFvid
// 	}
// 	if *copyDelete == true {
// 		v.Delete()
// 	}
// }
