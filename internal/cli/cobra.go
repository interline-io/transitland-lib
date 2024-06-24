package cli

import (
	"github.com/spf13/cobra"
)

type Runner interface {
	PreRunE([]string) error
	Run() error
}

func CobraHelper(r Runner) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		if err := r.PreRunE(args); err != nil {
			return err
		}
		return r.Run()
	}
}
