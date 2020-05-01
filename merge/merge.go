package merge

import (
	"flag"
	"fmt"
	"os"

	"github.com/interline-io/gotransit"
	"github.com/interline-io/gotransit/copier"
	"github.com/interline-io/gotransit/gtcsv"
	"github.com/interline-io/gotransit/internal/extract"
)

// Command merges feeds
type Command struct {
	DBURL   string
	DryRun  bool
	Readers []string
	Writer  string
}

// Parse sets options from command line flags.
func (cmd *Command) Parse(args []string) error {
	fl := flag.NewFlagSet("fetch", flag.ExitOnError)
	fl.Usage = func() {
		fmt.Println("Usage: merge <writer> [readers...]")
		fl.PrintDefaults()
	}
	fl.Parse(args)
	if cmd.DBURL == "" {
		cmd.DBURL = os.Getenv("DMFR_DATABASE_URL")
	}
	a := fl.Args()
	if len(a) < 2 {
		fmt.Println("writer and readers required")
		os.Exit(1)
	}
	cmd.Writer = a[0]
	cmd.Readers = a[1:]
	return nil
}

// Run executes this command.
func (cmd *Command) Run() error {
	writer, err := gtcsv.NewWriter(cmd.Writer)
	if err != nil {
		panic(err)
	}
	if err := writer.Open(); err != nil {
		panic(err)
	}
	// Get non-overlapping date ranges
	dateranges := map[string]daterange{}
	lastdr := daterange{}
	for _, url := range cmd.Readers {
		reader, err := gotransit.NewReader(url)
		if err != nil {
			panic(err)
		}
		if err := reader.Open(); err != nil {
			panic(err)
		}
		defer reader.Close()
		// Get the date range
		dr := daterange{}
		count := 0
		for ent := range reader.FeedInfos() {
			dr.prefix = ent.FeedVersion
			dr.start = ent.FeedStartDate.Time
			dr.end = ent.FeedEndDate.Time
			count++
		}
		fmt.Println("input:", dr.start, dr.end)
		if count != 1 {
			fmt.Println("Warning: zero or more than one feed infos:", count)
		}
		if !lastdr.end.IsZero() && dr.end.After(lastdr.start) {
			fmt.Println("clipping to ", dr.start)
			dr.end = lastdr.start
		}
		dateranges[url] = dr
		fmt.Println("dr:", dr.start, dr.end)
		lastdr = dr
	}

	for _, url := range cmd.Readers {
		fmt.Printf("\n\n========== %s  ==========\n", url)
		reader, err := gotransit.NewReader(url)
		if err != nil {
			panic(err)
		}
		if err := reader.Open(); err != nil {
			panic(err)
		}
		dr := dateranges[url]
		ef := NewFilter()
		ef.daterange = dr
		ef.prefix = dr.prefix
		// Reroot with filtered calendars
		fm := map[string][]string{}
		serviceIDs := map[string]bool{}
		for c := range reader.Calendars() {
			serviceIDs[c.ServiceID] = false
			if err := dr.clipCalendar(&c); err == nil {
				serviceIDs[c.ServiceID] = true
				fm["calendar.txt"] = append(fm["calendar.txt"], c.ServiceID)
			}
		}
		for cd := range reader.CalendarDates() {
			check, ok := serviceIDs[cd.ServiceID]
			if ok && check == false {
				// skip
			} else if dr.check(cd.Date) {
				serviceIDs[cd.ServiceID] = true
				fm["calendar.txt"] = append(fm["calendar.txt"], cd.ServiceID)
			}
		}
		fmt.Printf("service: %#v\n", fm)
		// filter
		gf := extract.NewMarker()
		if err := gf.Filter(reader, fm); err != nil {
			panic(err)
		}
		// Copy
		copier := copier.NewCopier(reader, writer)
		copier.Marker = &gf
		copier.AddEntityFilter(ef)
		copier.Copy()
	}
	return nil
}
