package inspect

import (
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"

	"github.com/interline-io/transitland-lib/ext"
	"github.com/interline-io/transitland-lib/internal/cli"
	"github.com/interline-io/transitland-lib/log"
	"github.com/interline-io/transitland-lib/rt/pb"
	"github.com/interline-io/transitland-lib/validator"
	"google.golang.org/protobuf/proto"
)

// Command
type Command struct {
	extensions cli.ArrayFlags
	spec       string
	readerPath string
}

func (cmd *Command) Parse(args []string) error {
	fl := flag.NewFlagSet("inspect", flag.ExitOnError)
	fl.Usage = func() {
		log.Print("Usage: inspect <spec> <file path or URL>")
		fl.PrintDefaults()
	}
	fl.Var(&cmd.extensions, "ext", "Include GTFS Extension")

	fl.Parse(args)
	if fl.NArg() < 2 {
		fl.Usage()
		return errors.New("requires spec and a file path or URL")
	}
	cmd.spec = fl.Arg(0)
	cmd.readerPath = fl.Arg(1)
	return nil
}

func (cmd *Command) Run() error {
	if cmd.spec == "gtfs" || cmd.spec == "GTFS" {
		reader, err := ext.OpenReader(cmd.readerPath)
		if err != nil {
			return err
		}
		defer reader.Close()

		var options validator.Options
		options.BestPractices = true
		options.IncludeEntities = true
		options.IncludeRouteGeometries = true

		v, _ := validator.NewValidator(reader, options)

		result, _ := v.Validate()

		entityCountTable := table.NewWriter()
		entityCountTable.SetOutputMirror(os.Stdout)
		entityCountTable.AppendHeader(table.Row{"GTFS File", "Entity Count"})
		for k, v := range result.EntityCount {
			entityCountTable.AppendRow(table.Row{k, v})
		}
		entityCountTable.SortBy([]table.SortBy{
			{Name: "GTFS File", Mode: table.Asc},
		})
		entityCountTable.Render()

		errorCountTable := table.NewWriter()
		errorCountTable.SetOutputMirror(os.Stdout)
		errorCountTable.AppendHeader(table.Row{"Entity Issue", "Severity", "Count"})
		for k, v := range result.Errors {
			errorCountTable.AppendRow(table.Row{k, "Error", v.Count})
		}
		for k, v := range result.Warnings {
			errorCountTable.AppendRow(table.Row{k, "Warning", v.Count})
		}
		errorCountTable.Render()

		feedInfoTable := table.NewWriter()
		feedInfoTable.SetOutputMirror(os.Stdout)
		feedInfoTable.AppendHeader(table.Row{
			"FeedPublisherName",
			"FeedPublisherURL",
			"FeedLang",
			"FeedVersion",
			"FeedStartDate",
			"FeedEndDate",
			"DefaultLang",
			"FeedContactEmail",
			"FeedContactURL"})
		for _, v := range result.FeedInfos {
			feedInfoTable.AppendRow(table.Row{
				v.FeedPublisherName,
				v.FeedPublisherURL,
				v.FeedLang,
				v.FeedVersion,
				v.FeedStartDate,
				v.FeedEndDate,
				v.DefaultLang,
				v.FeedContactEmail,
				v.FeedContactURL,
			})
		}
		feedInfoTable.Render()

		agencyTable := table.NewWriter()
		agencyTable.SetOutputMirror(os.Stdout)
		agencyTable.AppendHeader(table.Row{"AgencyID", "AgencyName"})
		for _, v := range result.Agencies {
			agencyTable.AppendRow(table.Row{v.AgencyID, v.AgencyName})
		}
		agencyTable.Render()
	} else if cmd.spec == "gtfs-rt" || cmd.spec == "gtfs-realtime" || cmd.spec == "rt" {
    client := &http.Client{}
    req, err := http.NewRequest("GET", cmd.readerPath, nil)
    resp, err := client.Do(req)
    defer resp.Body.Close()
    if err != nil {
        log.Errorf(err.Error())
    }
    body, err := ioutil.ReadAll(resp.Body)
    if err != nil {
			log.Errorf(err.Error())
    }
		feedMessage := pb.FeedMessage{}
    err = proto.Unmarshal(body, &feedMessage)
    if err != nil {
			log.Errorf(err.Error())
    }

		var jsonTransformer = text.NewJSONTransformer("", "  ")
		fmt.Print(jsonTransformer(feedMessage))
	}

	return nil
}
