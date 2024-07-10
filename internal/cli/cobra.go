package cli

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type Runner interface {
	AddFlags(*pflag.FlagSet)
	Parse([]string) error
	Run() error
}

type NArgs struct {
	v []string
}

func (fl *NArgs) NArg() int {
	return len(fl.v)
}

func (fl *NArgs) Arg(i int) string {
	if i >= len(fl.v) {
		return ""
	}
	return fl.v[i]
}

func (fl *NArgs) Args() []string {
	return fl.v
}

func NewNArgs(v []string) *NArgs {
	return &NArgs{v: v}
}

func CobraHelper(r Runner, subc string) *cobra.Command {
	cobraCommand := &cobra.Command{
		Use: subc,
	}
	cobraCommand.PreRunE = func(cmd *cobra.Command, args []string) error {
		return r.Parse(args)
	}
	cobraCommand.RunE = func(cmd *cobra.Command, args []string) error {
		return r.Run()
	}
	r.AddFlags(cobraCommand.Flags())
	return cobraCommand
}
