package cmds

import (
	"context"
	"strings"
	"testing"

	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/internal/testdb"
	"github.com/interline-io/transitland-lib/internal/testpath"
)

func TestSyncCommand(t *testing.T) {
	cases := []struct {
		count       int
		errContains string
		command     []string
	}{
		{2, "", []string{testpath.RelPath("testdata/dmfr/example.json")}},
		{4, "", []string{testpath.RelPath("testdata/dmfr/example.json"), testpath.RelPath("testdata/dmfr/bayarea-local.dmfr.json")}},
		{0, "no such file", []string{testpath.RelPath("testdata/dmfr/does-not-exist.json")}},
	}
	ctx := context.TODO()
	for _, exp := range cases {
		t.Run("", func(t *testing.T) {
			w := testdb.TempSqliteAdapter()
			c := SyncCommand{Adapter: w}
			if err := c.Parse(exp.command); err != nil {
				t.Error(err)
			}
			err := c.Run(ctx)
			if err != nil {
				if !strings.Contains(err.Error(), exp.errContains) {
					t.Errorf("got '%s' error, expected to contain '%s'", err.Error(), exp.errContains)
				}
			}
			// Test
			feeds := []dmfr.Feed{}
			w.Select(ctx, &feeds, "SELECT * FROM current_feeds")
			if len(feeds) != exp.count {
				t.Errorf("got %d feeds, expect %d", len(feeds), exp.count)
			}
		})

	}
}
