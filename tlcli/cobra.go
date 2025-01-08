package tlcli

import (
	"bytes"
	"context"
	"fmt"
	"text/template"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

type hasHelpDesc interface {
	HelpDesc() (string, string)
}

type hasHelpExample interface {
	HelpExample() string
}

type hasHelpArgs interface {
	HelpArgs() string
}

type Runner interface {
	AddFlags(*pflag.FlagSet)
	Parse([]string) error
	Run(context.Context) error
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

func CobraHelper(r Runner, pc string, subc string) *cobra.Command {
	cobraCommand := &cobra.Command{
		Use: subc,
	}
	if v, ok := r.(hasHelpArgs); ok {
		cobraCommand.Use = fmt.Sprintf("%s %s", subc, v.HelpArgs())
	}
	if v, ok := r.(hasHelpDesc); ok {
		short, long := v.HelpDesc()
		cobraCommand.Short = short
		cobraCommand.Long = fmt.Sprintf("%s\n\n%s", short, long)
	}
	if v, ok := r.(hasHelpExample); ok {
		calledAs := subc
		type helpValues struct {
			ParentCommand string
			Command       string
		}
		w := bytes.NewBuffer(nil)
		helpTemplate := v.HelpExample()
		t := template.Must(template.New(subc).Parse(helpTemplate))
		t.Execute(w, helpValues{Command: calledAs, ParentCommand: pc})
		cobraCommand.Example = w.String()
	}
	cobraCommand.PreRunE = func(cmd *cobra.Command, args []string) error {
		return r.Parse(args)
	}
	cobraCommand.RunE = func(cmd *cobra.Command, args []string) error {
		cmd.SilenceUsage = true
		return r.Run(cmd.Context())
	}
	r.AddFlags(cobraCommand.Flags())
	return cobraCommand
}

func RunWithArgs(r Runner, args []string) error {
	c := CobraHelper(r, "", "")
	c.SetArgs(args)
	return c.ExecuteContext(context.TODO())
}
