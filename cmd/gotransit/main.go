package main

import (
	"flag"
	"os"
	"strings"

	"github.com/interline-io/transitland-lib/dmfr"
	_ "github.com/interline-io/transitland-lib/ext/plus"
	_ "github.com/interline-io/transitland-lib/gtcsv"
	"github.com/interline-io/transitland-lib/gtdb"
	"github.com/interline-io/transitland-lib/internal/log"
	"github.com/interline-io/transitland-lib/tl"
)

// MustGetReader or exits.
func MustGetReader(inurl string) tl.Reader {
	if len(inurl) == 0 {
		log.Exit("No reader specified")
	}
	// Reader
	reader, err := tl.NewReader(inurl)
	if err != nil {
		log.Exit("No known reader for '%s': %s", inurl, err)
	}
	if err := reader.Open(); err != nil {
		log.Exit("Could not open '%s': %s", inurl, err)
	}
	return reader
}

// MustGetWriter or exits.
func MustGetWriter(outurl string, create bool) tl.Writer {
	if len(outurl) == 0 {
		log.Exit("No writer specified")
	}
	// Writer
	writer, err := tl.NewWriter(outurl)
	if err != nil {
		log.Exit("No known writer for '%s': %s", outurl, err)
	}
	if err := writer.Open(); err != nil {
		log.Exit("Could not open '%s': %s", outurl, err)
	}
	if create {
		if err := writer.Create(); err != nil {
			log.Exit("Could not create writer: %s", err)
		}
	}
	return writer
}

// MustGetDBWriter opens a database or exits.
func MustGetDBWriter(dburl string, create bool) *gtdb.Writer {
	writer := MustGetWriter(dburl, true)
	w, ok := writer.(*gtdb.Writer)
	if !ok {
		log.Exit("Writer is not a database")
	}
	return w
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

// Submodule
type runner interface {
	run([]string) error
}

///////////////

func main() {
	log.SetLevel(log.INFO)
	quietFlag := false
	debugFlag := false
	traceFlag := false
	flag.BoolVar(&quietFlag, "q", false, "Only send critical errors to stderr")
	flag.BoolVar(&debugFlag, "v", false, "Enable verbose output")
	flag.BoolVar(&traceFlag, "vv", false, "Enable more verbose/query output")
	flag.Usage = func() {
		log.Print("Usage of %s:", os.Args[0])
		log.Print("Commands:")
		log.Print("  copy")
		log.Print("  extract")
		log.Print("  validate")
		log.Print("  dmfr")
		return
	}
	flag.Parse()
	if quietFlag == true {
		log.SetLevel(log.ERROR)
	}
	if debugFlag == true {
		log.SetLevel(log.DEBUG)
	}
	if traceFlag == true {
		log.SetLevel(log.QUERY)
	}
	args := flag.Args()
	subc := flag.Arg(0)
	if subc == "" {
		flag.Usage()
		log.Exit("")
	}
	args = flag.Args()
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
