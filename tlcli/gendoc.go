// Package tlcli provides helper utilities for command line programs.
package tlcli

import (
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
	"github.com/spf13/pflag"
)

type GenDocCommand struct {
	Outpath string
	Delete  bool
	Command *cobra.Command
}

func (cmd *GenDocCommand) AddFlags(fl *pflag.FlagSet) {
	fl.BoolVar(&cmd.Delete, "delete", false, "Delete existing *.md files")
}

func (cmd *GenDocCommand) HelpDesc() (string, string) {
	return "Generate markdown documentation", ""
}

func (cmd *GenDocCommand) Parse(args []string) error {
	fl := NewNArgs(args)
	cmd.Outpath = fl.Arg(0)
	return nil
}

func (cmd *GenDocCommand) Run() error {
	if cmd.Delete {
		files, err := filepath.Glob(filepath.Join(cmd.Outpath, "*.md"))
		if err != nil {
			return err
		}
		for _, f := range files {
			if err := os.Remove(f); err != nil {
				return err
			}
		}
	}
	return doc.GenMarkdownTree(cmd.Command, cmd.Outpath)
}
