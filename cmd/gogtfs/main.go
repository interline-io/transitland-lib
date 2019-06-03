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
	"github.com/interline-io/gotransit/gtdb"
	"github.com/interline-io/gotransit/internal/enums"
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

func main() {
	// Validate
	validateCommand := flag.NewFlagSet("validate", flag.ExitOnError)
	validateExtensions := arrayFlags{}
	validateCommand.Var(&validateExtensions, "ext", "Include GTFS Extension")

	// Copy
	copyCommand := flag.NewFlagSet("copy", flag.ExitOnError)
	// Multiple flags e.g. "--ext a --ext b"
	copyExtensions := arrayFlags{}
	copyCommand.Var(&copyExtensions, "ext", "Include GTFS Extension")
	var copyFilters arrayFlags
	copyCommand.Var(&copyFilters, "filter", "Apply GTFS Filter")
	// Regular flags
	var copyCreate = copyCommand.Bool("create", false, "Create new database")
	var copyDelete = copyCommand.Bool("delete", false, "Delete data from Feed Version before copying")
	var copyNewfv = copyCommand.Bool("newfv", false, "Create new Feed Version")
	var copyFvid = copyCommand.Int("fv", 0, "Feed Version ID")
	var copyOnlyVisited = copyCommand.Bool("visited", false, "Copy only visited entities")
	var copyInterpolate = copyCommand.Bool("interpolate", false, "Interpolate missing StopTime values")
	var copyCreateShapes = copyCommand.Bool("createshapes", false, "Create any missing Trip Shapes")
	var copyResults = copyCommand.Bool("results", false, "Show entity counts in output")
	_ = copyResults
	var copyNormalizeServiceIDs = copyCommand.Bool("normalizeserviceids", false, "Normalize ServiceIDs")

	// Extract
	extractCommand := flag.NewFlagSet("extract", flag.ExitOnError)
	extractAgencies := arrayFlags{}
	extractCommand.Var(&extractAgencies, "agency", "Extract Agency")
	extractStops := arrayFlags{}
	extractCommand.Var(&extractStops, "stop", "Extract Stop")
	extractTrip := arrayFlags{}
	extractCommand.Var(&extractTrip, "trip", "Extract Trip")
	extractCalendar := arrayFlags{}
	extractCommand.Var(&extractCalendar, "calendar", "Extract Calendar")
	extractRouteType := arrayFlags{}
	extractCommand.Var(&extractRouteType, "route_type", "Extract Routes matching route_type")
	extractRouteTypeCategory := arrayFlags{}
	extractCommand.Var(&extractRouteTypeCategory, "route_type_category", "Extracr Routes matching this route_type category")
	//
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
		copyCommand.Parse(os.Args[2:])
	case "validate":
		validateCommand.Parse(os.Args[2:])
	case "extract":
		extractCommand.Parse(os.Args[2:])
	default:
		exit("%q is not valid command.", os.Args[1])
	}

	if validateCommand.Parsed() {
		reader := getReader(validateCommand.Arg(0))
		defer reader.Close()
		v, err := validator.NewValidator(reader)
		if err != nil {
			panic(err)
		}
		for _, ext := range validateExtensions {
			e, err := gotransit.GetExtension(ext)
			if err != nil {
				exit("No extension for: %s", ext)
			}
			v.Copier.AddExtension(e)
		}
		v.Validate()
	}

	if extractCommand.Parsed() {
		reader := getReader(extractCommand.Arg(0))
		defer reader.Close()
		writer := getWriter(extractCommand.Arg(1))
		defer writer.Close()
		//
		fm := map[string][]string{}
		if len(extractRouteTypeCategory) > 0 {
			for _, i := range extractRouteTypeCategory {
				for _, rt := range enums.GetRouteCategory(i) {
					extractRouteType = append(extractRouteType, strconv.Itoa(rt.Code))
				}
			}
		}
		if len(extractRouteType) > 0 {
			rthits := map[int]bool{}
			for _, i := range extractRouteType {
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
		}
		fmt.Printf("Extract filter: %#v\n", fm)

		// Load graph
		em := extract.NewExtractMarker()
		fmt.Println("Loading graph")
		em.Load(reader)
		// Apply filters
		fmt.Println("Searching graph to apply filters")
		em.Filter(fm)
		fmt.Println("Copying...")
		cp := copier.NewCopier(reader, writer)
		// Set copier options
		cp.Marker = &em
		cp.AllowEntityErrors = false
		cp.AllowReferenceErrors = false
		// Copy
		cp.Copy()
	}

	if copyCommand.Parsed() {
		inurl := copyCommand.Arg(0)
		// Reader
		reader := getReader(copyCommand.Arg(0))
		defer reader.Close()
		writer := getWriter(copyCommand.Arg(1))
		defer writer.Close()

		// If a DBWriter, create a FV
		if v, ok := writer.(*gtdb.Writer); ok {
			// If writing to database, we must normalize calendars
			nsids := true
			copyNormalizeServiceIDs = &nsids
			db := v.Adapter.DB()
			if *copyCreate == true {
				writer.Create()
			}
			if *copyNewfv == true {
				fv, err := gotransit.NewFeedVersion(reader)
				if err != nil {
					exit("Could not create FeedVersion: %s", err)
				}
				dberr := db.
					Where(gotransit.FeedVersion{URL: inurl, SHA1: fv.SHA1}).
					FirstOrCreate(&fv).
					Error
				if dberr != nil {
					exit("Could not create FeedVersion: %s", dberr)
				}
				copyFvid = &fv.ID
			}
			if *copyFvid > 0 {
				v.FeedVersionID = *copyFvid
			}
			if *copyDelete == true {
				v.Delete()
			}
		}

		// Create Copier
		// Add extensions
		cp := copier.NewCopier(reader, writer)
		for _, ext := range copyExtensions {
			e, err := gotransit.GetExtension(ext)
			if err != nil {
				exit("No extension for: %s", ext)
			}
			cp.AddExtension(e)
			e.Create(writer)
		}
		// Add filters
		for _, ext := range copyFilters {
			ef, err := gotransit.GetEntityFilter(ext)
			if err != nil {
				exit("No filter for '%s': %s", ext, err)
			}
			cp.AddEntityFilter(ef)
		}
		// Set options
		cp.AllowEntityErrors = true
		cp.AllowReferenceErrors = false
		cp.NormalizeServiceIDs = *copyNormalizeServiceIDs
		cp.InterpolateStopTimes = *copyInterpolate
		cp.CreateMissingShapes = *copyCreateShapes
		// Copy
		if *copyOnlyVisited == true {
			cp.CopyVisited()
		} else {
			cp.Copy()
		}
		// Copy Extensions
		cp.CopyExtensions()
	}
}
