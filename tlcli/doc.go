// Package tlcli provides helper utilities for command line programs.
package tlcli

import (
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
	"github.com/spf13/pflag"
)

type GenDocCommand struct {
	OutputPath string
	Command    *cobra.Command
}

func (cmd *GenDocCommand) AddFlags(fl *pflag.FlagSet) {}

func (cmd *GenDocCommand) HelpDesc() (string, string) {
	return "Generate markdown documentation", ""
}

func (cmd *GenDocCommand) Parse(args []string) error {
	fl := NewNArgs(args)
	cmd.OutputPath = fl.Arg(0)
	return nil
}

func (cmd *GenDocCommand) Run() error {
	return doc.GenMarkdownTree(cmd.Command, cmd.OutputPath)
}
