package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/interline-io/gotransit"
	_ "github.com/interline-io/gotransit/ext/pathways"
	_ "github.com/interline-io/gotransit/ext/plus"
	_ "github.com/interline-io/gotransit/gtcsv"
	"github.com/interline-io/gotransit/gtdb"
	"github.com/interline-io/gotransit/internal/log"
)

// Helpers
func exit(fmts string, args ...interface{}) {
	fmt.Printf(fmts+"\n", args...)
	os.Exit(1)
}

// MustGetReader or exits.
func MustGetReader(inurl string) gotransit.Reader {
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

// MustGetWriter or exits.
func MustGetWriter(outurl string, create bool) gotransit.Writer {
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

// MustGetDBWriter opens a database or exits.
func MustGetDBWriter(dburl string, create bool) *gtdb.Writer {
	writer := MustGetWriter(dburl, true)
	w, ok := writer.(*gtdb.Writer)
	if !ok {
		exit("Writer is not a database")
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

///////////////

func main() {
	log.SetLevel(log.TRACE)
	flag.Usage = func() {
		fmt.Printf("Usage of %s:\n", os.Args[0])
		fmt.Println("Commands:")
		fmt.Println("  copy")
		fmt.Println("  extract")
		fmt.Println("  validate")
		fmt.Println("  dmfr")
		return
	}
	flag.Parse()
	args := flag.Args()
	subc := flag.Arg(0)
	if subc == "" {
		flag.Usage()
		exit("")
	}
	args = flag.Args()
	switch subc {
	case "copy":
		cmd := copyCommand{}
		cmd.run(args[1:])
	case "validate":
		cmd := validateCommand{}
		cmd.run(args[1:])
	case "extract":
		cmd := extractCommand{}
		cmd.run(args[1:])
	case "dmfr":
		cmd := dmfrCommand{}
		cmd.run(args[1:])
	default:
		exit("%q is not valid command.", subc)
	}
}
