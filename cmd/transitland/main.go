package main

import (
	"flag"
	"os"

	"github.com/interline-io/transitland-lib/copier"
	"github.com/interline-io/transitland-lib/dmfr/fetch"
	"github.com/interline-io/transitland-lib/dmfr/importer"
	"github.com/interline-io/transitland-lib/dmfr/sync"
	"github.com/interline-io/transitland-lib/dmfr/unimporter"
	_ "github.com/interline-io/transitland-lib/ext/plus"
	"github.com/interline-io/transitland-lib/extract"
	"github.com/interline-io/transitland-lib/internal/log"
	"github.com/interline-io/transitland-lib/tl"
	_ "github.com/interline-io/transitland-lib/tlcsv"
	_ "github.com/interline-io/transitland-lib/tldb"
	"github.com/interline-io/transitland-lib/validator"
)

type runner interface {
	Parse([]string) error
	Run() error
}

func main() {
	log.SetLevel(log.INFO)
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
		log.SetLevel(log.ERROR)
	}
	if debugFlag {
		log.SetLevel(log.DEBUG)
	}
	if traceFlag {
		log.SetLevel(log.TRACE)
		log.SetQueryLog(true)
	}
	args := flag.Args()
	subc := flag.Arg(0)
	if subc == "" {
		flag.Usage()
		log.Exit("")
	}
	var r runner
	switch subc {
	case "copy":
		r = &copier.Command{}
	case "validate":
		r = &validator.Command{}
	case "extract":
		r = &extract.Command{}
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
		log.Exit("%q is not valid command.", subc)
	}
	if err := r.Parse(args[1:]); err != nil {
		log.Exit("Erorr: %s", err.Error())
	}
	if err := r.Run(); err != nil {
		log.Exit("Error: %s", err.Error())
	}
}
