package cmds

import (
	"errors"
	"os"

	"github.com/interline-io/log"
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
	return `% {{.ParentCommand}} {{.Command}} "trips.pb" "trips.pb.json"`
}

func (cmd *RTConvertCommand) HelpArgs() string {
	return "[flags] <input pb> <output json>"
}

func (cmd *RTConvertCommand) AddFlags(fl *pflag.FlagSet) {
}

func (cmd *RTConvertCommand) Parse(args []string) error {
	fl := tlcli.NewNArgs(args)
	if fl.NArg() < 1 {
		return errors.New("requires input pb")
	}
	if fl.NArg() < 2 {
		return errors.New("requires output json file")
	}
	cmd.InputFile = fl.Arg(0)
	cmd.OutputFile = fl.Arg(1)
	return nil
}

func (cmd *RTConvertCommand) Run() error {
	log.Info().Msgf("Converting '%s' to '%s'", cmd.InputFile, cmd.OutputFile)
	msg, err := rt.ReadURL(cmd.InputFile, request.WithAllowLocal)
	if err != nil {
		return err
	}
	mOpts := protojson.MarshalOptions{UseProtoNames: true}
	rtJson, err := mOpts.Marshal(msg)
	if err != nil {
		return err
	}
	outf, err := os.Create(cmd.OutputFile)
	if err != nil {
		return err
	}
	defer outf.Close()
	outf.Write(rtJson)
	return nil
}
