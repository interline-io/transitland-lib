package main

import (
	"flag"
	"os"

	"github.com/interline-io/transitland-lib/copier"
	"github.com/interline-io/transitland-lib/diff"
	"github.com/interline-io/transitland-lib/dmfr/fetch"
	"github.com/interline-io/transitland-lib/dmfr/importer"
	"github.com/interline-io/transitland-lib/dmfr/sync"
	"github.com/interline-io/transitland-lib/dmfr/unimporter"
	_ "github.com/interline-io/transitland-lib/ext/plus"
	_ "github.com/interline-io/transitland-lib/ext/redate"
	"github.com/interline-io/transitland-lib/extract"
	"github.com/interline-io/transitland-lib/log"
	"github.com/interline-io/transitland-lib/tl"
	_ "github.com/interline-io/transitland-lib/tlcsv"
	_ "github.com/interline-io/transitland-lib/tldb"
	"github.com/interline-io/transitland-lib/validator"
	"github.com/rs/zerolog"
)

type runner interface {
	Parse([]string) error
	Run() error
}

func main() {
	quietFlag := false
	debugFlag := false
	traceFlag := false
	versionFlag := false
	flag.BoolVar(&quietFlag, "q", false, "Only send critical errors to stderr")
	flag.BoolVar(&debugFlag, "v", false, "Enable verbose output")
	flag.BoolVar(&traceFlag, "vv", false, "Enable more verbose/query output")
	flag.BoolVar(&versionFlag, "version", false, "Show version and GTFS spec information")
	flag.Usage = func() {
		log.Print("Usage of %s:", os.Args[0])
		log.Print("Commands:")
		log.Print("  copy")
		log.Print("  validate")
		log.Print("  extract")
		log.Print("  fetch")
		log.Print("  import")
		log.Print("  unimport")
		log.Print("  sync")
		log.Print("  dmfr")

	}
	flag.Parse()
	if versionFlag {
		log.Print("transitland-lib version: %s", tl.VERSION)
		log.Print("GTFS specification version: https://github.com/google/transit/blob/%s/gtfs/spec/en/reference.md", tl.GTFSVERSION)
		log.Print("GTFS Realtime specification version: https://github.com/google/transit/blob/%s/gtfs-realtime/proto/gtfs-realtime.proto", tl.GTFSRTVERSION)
		return
	}
	if quietFlag {
		log.SetLevel(zerolog.Disabled)
	} else if debugFlag {
		log.SetLevel(zerolog.DebugLevel)
	} else if traceFlag {
		log.SetLevel(zerolog.TraceLevel)
	}

	args := flag.Args()
	subc := flag.Arg(0)
	if subc == "" {
		flag.Usage()
		os.Exit(1)
	}
	var r runner
	switch subc {
	case "copy":
		r = &copier.Command{}
	case "validate":
		r = &validator.Command{}
	case "extract":
		r = &extract.Command{}
	case "diff":
		r = &diff.Command{}
	case "fetch":
		r = &fetch.Command{}
	case "import":
		r = &importer.Command{}
	case "unimport":
		r = &unimporter.Command{}
	case "sync":
		r = &sync.Command{}
	case "dmfr": // backwards compat
		r = &dmfrCommand{}
	default:
		log.Errorf("%q is not valid command.", subc)
		os.Exit(1)
	}
	if err := r.Parse(args[1:]); err != nil {
		log.Errorf(err.Error())
		os.Exit(1)
	}
	if err := r.Run(); err != nil {
		log.Errorf(err.Error())
		os.Exit(1)
	}
}
