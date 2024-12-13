package cmds

import (
	"errors"
	"os"

	"github.com/interline-io/transitland-lib/request"
	"github.com/interline-io/transitland-lib/rt"
	"github.com/interline-io/transitland-lib/tlcli"
	"github.com/spf13/pflag"
	"google.golang.org/protobuf/encoding/protojson"
)

// RTConvertCommand
type RTConvertCommand struct {
	InputFile  string
	OutputFile string
}

func (cmd *RTConvertCommand) HelpDesc() (string, string) {
	return "Convert GTFS-RealTime to JSON", ""
}

func (cmd *RTConvertCommand) HelpExample() string {
	return `% {{.ParentCommand}} {{.Command}} "trips.pb"`
}

func (cmd *RTConvertCommand) HelpArgs() string {
	return "[flags] <input pb>"
}

func (cmd *RTConvertCommand) AddFlags(fl *pflag.FlagSet) {
	fl.StringVarP(&cmd.OutputFile, "out", "o", "", "Write JSON to file; defaults to stdout")
}

func (cmd *RTConvertCommand) Parse(args []string) error {
	fl := tlcli.NewNArgs(args)
	if fl.NArg() < 1 {
		return errors.New("requires input pb")
	}
	cmd.InputFile = fl.Arg(0)
	return nil
}

func (cmd *RTConvertCommand) Run() error {
	// Fetch
	msg, err := rt.ReadURL(cmd.InputFile, request.WithAllowLocal)
	if err != nil {
		return err
	}
	// Create json
	mOpts := protojson.MarshalOptions{UseProtoNames: true, Indent: "  "}
	rtJson, err := mOpts.Marshal(msg)
	if err != nil {
		return err
	}
	// Write
	outf := os.Stdout
	if cmd.OutputFile != "" {
		var err error
		outf, err = os.Create(cmd.OutputFile)
		if err != nil {
			return err
		}
		defer outf.Close()
	}
	if _, err := outf.Write(rtJson); err != nil {
		return err
	}
	return nil
}
