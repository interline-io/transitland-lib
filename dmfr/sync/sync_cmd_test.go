package sync

import (
	"strings"
	"testing"

	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tldb"
)

func TestCommand(t *testing.T) {
	cases := []struct {
		count       int
		errContains string
		command     []string
	}{
		{2, "", []string{"../../test/data/dmfr/example.json"}},
		{4, "", []string{"../../test/data/dmfr/example.json", "../../test/data/dmfr/bayarea-local.dmfr.json"}},
		{0, "no such file", []string{"../../test/data/dmfr/does-not-exist.json"}},
	}
	_ = cases
	for _, exp := range cases {
		t.Run("", func(t *testing.T) {
			w := tldb.MustGetWriter("sqlite3://:memory:", true)
			c := Command{Adapter: w.Adapter}
			if err := c.Parse(exp.command); err != nil {
				t.Error(err)
			}
			err := c.Run()
			if err != nil {
				if !strings.Contains(err.Error(), exp.errContains) {
					t.Errorf("got '%s' error, expected to contain '%s'", err.Error(), exp.errContains)
				}
			}
			// Test
			feeds := []tl.Feed{}
			w.Adapter.Select(&feeds, "SELECT * FROM current_feeds")
			if len(feeds) != exp.count {
				t.Errorf("got %d feeds, expect %d", len(feeds), exp.count)
			}
		})

	}
}
