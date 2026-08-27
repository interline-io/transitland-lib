package cmds

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDmfrFormatCommandParse(t *testing.T) {
	testCases := []struct {
		name        string
		args        []string
		errorSubstr string
	}{
		{
			name:        "no arguments",
			args:        []string{},
			errorSubstr: "must specify filename",
		},
		{
			name: "one filename",
			args: []string{"a.dmfr.json"},
		},
		{
			name:        "two filenames",
			args:        []string{"a.dmfr.json", "b.dmfr.json"},
			errorSubstr: "only one filename allowed",
		},
		{
			name:        "glob expansion",
			args:        []string{"a.dmfr.json", "b.dmfr.json", "c.dmfr.json"},
			errorSubstr: "only one filename allowed",
		},
	}
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			cmd := DmfrFormatCommand{}
			err := cmd.Parse(tc.args)
			if tc.errorSubstr == "" {
				assert.NoError(t, err)
				assert.Equal(t, tc.args[0], cmd.Filename)
				return
			}
			if assert.Error(t, err) {
				assert.Contains(t, err.Error(), tc.errorSubstr)
			}
		})
	}
}
