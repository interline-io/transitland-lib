package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/interline-io/gotransit"
	"github.com/interline-io/gotransit/copier"
	_ "github.com/interline-io/gotransit/ext/pathways"
	_ "github.com/interline-io/gotransit/ext/plus"
	_ "github.com/interline-io/gotransit/gtcsv"
	"github.com/interline-io/gotransit/gtdb"
	"github.com/interline-io/gotransit/validator"
)

// Helpers
func exit(fmts string, args ...interface{}) {
	fmt.Printf(fmts+"\n", args...)
	os.Exit(1)
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
	validateCommand := flag.NewFlagSet("validate", flag.ExitOnError)
	validateExtensions := arrayFlags{}
	validateCommand.Var(&validateExtensions, "ext", "Include GTFS Extension")

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

	if len(os.Args) == 1 {
		fmt.Println("usage: gotransit <command> [<args>]")
		fmt.Println("Commands:")
		fmt.Println("  copy")
		fmt.Println("  validate")
		return
	}

	switch os.Args[1] {
	case "copy":
		copyCommand.Parse(os.Args[2:])
	case "validate":
		validateCommand.Parse(os.Args[2:])
	default:
		exit("%q is not valid command.", os.Args[1])
	}

	if validateCommand.Parsed() {
		inurl := validateCommand.Arg(0)
		if len(inurl) == 0 {
			exit("No reader specified")
		}
		reader, err := gotransit.NewReader(inurl)
		if err != nil {
			exit("No known reader for '%s': %s", inurl, err)
		}
		reader.Open()
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

	if copyCommand.Parsed() {
		inurl := copyCommand.Arg(0)
		if len(inurl) == 0 {
			exit("No reader specified")
		}
		outurl := copyCommand.Arg(1)
		if len(outurl) == 0 {
			exit("No writer specified")
		}
		// Reader
		reader, err := gotransit.NewReader(inurl)
		if err != nil {
			exit("No known reader for '%s': %s", inurl, err)
		}
		if err := reader.Open(); err != nil {
			exit("Could not open '%s': %s", inurl, err)
		}
		defer reader.Close()
		// Writer
		writer, err := gotransit.NewWriter(outurl)
		if err != nil {
			exit("No known writer for '%s': %s", outurl, err)
		}
		if err := writer.Open(); err != nil {
			exit("Could not open '%s': %s", inurl, err)
		}
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
