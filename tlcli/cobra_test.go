package tlcli

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/spf13/pflag"
	"github.com/stretchr/testify/assert"
)

type testCommand struct {
	value string
	args  []string
	slice []string
	w     io.Writer
}

func (cmd *testCommand) AddFlags(fl *pflag.FlagSet) {
	fl.StringVar(&cmd.value, "value", "", "value")
	fl.StringSliceVar(&cmd.slice, "slice", nil, "slice")
}

func (cmd *testCommand) Parse(args []string) error {
	cmd.args = []string{}
	cmd.args = append(cmd.args, args...)
	return nil
}

func (cmd *testCommand) Run(ctx context.Context) error {
	fmt.Fprintf(
		cmd.w,
		"testCommand: flag: '%s' slice: '%s' args: '%s'",
		cmd.value,
		strings.Join(cmd.slice, " "),
		strings.Join(cmd.args, " "),
	)
	return nil
}

func TestRunWithArgs(t *testing.T) {
	r := &testCommand{}
	w := bytes.NewBuffer(nil)
	r.w = w
	RunWithArgs(r, []string{"--value=abc", "--slice=1", "--slice=2", "one", "two", "three"})
	expect := `testCommand: flag: 'abc' slice: '1 2' args: 'one two three'`
	assert.Equal(t, expect, w.String())
}
