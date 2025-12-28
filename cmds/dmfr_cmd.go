package cmds

import (
	"github.com/interline-io/transitland-lib/tlcli"
	"github.com/spf13/cobra"
)

// DmfrCommand is the parent command for DMFR subcommands.
type DmfrCommand struct {
	cobraCmd *cobra.Command
}

func NewDmfrCommand(pc string) *cobra.Command {
	dmfrCmd := &cobra.Command{
		Use:   "dmfr",
		Short: "DMFR commands",
		Long:  "Commands for working with DMFR (Distributed Mobility Feed Registry) files.",
	}
	dmfrCmd.AddCommand(
		tlcli.CobraHelper(&DmfrLintCommand{}, pc+" dmfr", "lint"),
		tlcli.CobraHelper(&DmfrFormatCommand{}, pc+" dmfr", "format"),
		tlcli.CobraHelper(&DmfrFromDirCommand{}, pc+" dmfr", "from-dir"),
	)
	return dmfrCmd
}
