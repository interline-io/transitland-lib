package merge

import (
	"testing"

	"github.com/interline-io/transitland-lib/ext"
	"github.com/interline-io/transitland-lib/internal/testutil"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/stretchr/testify/assert"
)

func TestMerge(t *testing.T) {
	t.Run("merge", func(t *testing.T) {
		f1 := testutil.ExampleFeedBART
		f2 := testutil.ExampleFeedCaltrain
		cmd := Command{}
		tdir := t.TempDir()
		if err := cmd.Parse([]string{tdir, f1.URL, f2.URL}); err != nil {
			t.Fatal(err)
		}
		if err := cmd.Run(); err != nil {
			t.Fatal(err)
		}
		outReader, err := ext.OpenReader(tdir)
		if err != nil {
			t.Fatal(err)
		}
		entCount := map[string]int{}
		testutil.AllEntities(outReader, func(ent tl.Entity) {
			entCount[ent.Filename()] += 1
		})
		expectCount := map[string]int{}
		for k, v := range f1.Counts {
			expectCount[k] = v + f2.Counts[k]
		}
		// Adjustments
		expectCount["calendar.txt"] = 30
		delete(expectCount, "fare_attributes.txt")
		delete(expectCount, "fare_rules.txt")
		checked := 0
		for k, exp := range expectCount {
			assert.Equal(t, exp, entCount[k], k)
			checked += 1
		}
		if checked == 0 {
			t.Fatal("no checks were performed - make sure both example feeds in test_feeds.go have entity counts set")
		}
	})
}
