package main

import (
	"flag"
	"os"

	dmfr "github.com/interline-io/transitland-lib/dmfr/cmd"
	_ "github.com/interline-io/transitland-lib/ext/plus"
	"github.com/interline-io/transitland-lib/internal/log"
	"github.com/interline-io/transitland-lib/tl"
	_ "github.com/interline-io/transitland-lib/tlcsv"
	_ "github.com/interline-io/transitland-lib/tldb"
)

///////////////

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
		log.Print("  extract")
		log.Print("  validate")
		log.Print("  dmfr")
		log.Print("  server")
	}
	flag.Parse()
	if versionFlag {
		log.Print("transitland-lib version: %s", tl.VERSION)
		log.Print("gtfs spec version: https://github.com/google/transit/blob/%s/gtfs/spec/en/reference.md", tl.GTFSVERSION)
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
	type runnable interface {
		Run([]string) error
	}
	var r runnable
	var err error
	switch subc {
	case "copy":
		r = &copyCommand{}
	case "validate":
		r = &validateCommand{}
	case "extract":
		r = &extractCommand{}
	case "dmfr":
		r = &dmfr.Command{}
	default:
		log.Exit("%q is not valid command.", subc)
	}
	err = r.Run(args[1:]) // consume first arg
	if err != nil {
		log.Exit("Error: %s", err.Error())
	}
}
