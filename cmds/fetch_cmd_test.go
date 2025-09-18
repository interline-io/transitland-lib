package cmds

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/internal/testdb"
	"github.com/interline-io/transitland-lib/internal/testpath"
)

func TestFetchCommand(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf, err := os.ReadFile(testpath.RelPath(filepath.Join("testdata", r.URL.Path)))
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}
		w.Write(buf)
	}))
	defer ts.Close()

	// note - Spec==gtfs is required for fetch
	f200 := dmfr.Feed{FeedID: "f-200", Spec: "gtfs", URLs: dmfr.FeedUrls{StaticCurrent: fmt.Sprintf("%s/gtfs-examples/example.zip", ts.URL)}}
	f404 := dmfr.Feed{FeedID: "f-404", Spec: "gtfs", URLs: dmfr.FeedUrls{StaticCurrent: fmt.Sprintf("%s/404", ts.URL)}}
	fvErrorExample := dmfr.Feed{FeedID: "f-error", Spec: "gtfs", URLs: dmfr.FeedUrls{StaticCurrent: fmt.Sprintf("%s/gtfs-examples/example-errors.zip", ts.URL)}}
	cases := []struct {
		name               string
		fvcount            int
		fatalErrorContains string
		feeds              []dmfr.Feed
		command            []string
		fail               bool
		strict             bool
	}{
		{
			name:    "single fetch",
			fvcount: 1,
			feeds:   []dmfr.Feed{f200},
		},
		{
			name:    "multiple fetch",
			fvcount: 1,
			feeds:   []dmfr.Feed{f200, f404},
			command: []string{"f-200", "f-404"},
		},
		{
			name:    "fetch error",
			fvcount: 1,
			feeds:   []dmfr.Feed{f200, f404},
			command: []string{"f-200"},
		},
		{
			name:    "fetch error 2",
			fvcount: 0,
			feeds:   []dmfr.Feed{f200, f404},
			command: []string{"f-404"},
		},
		{
			name:               "fail",
			fvcount:            0,
			feeds:              []dmfr.Feed{f200, f404},
			command:            []string{"f-404"},
			fail:               true,
			fatalErrorContains: "404",
		},
		{
			name:    "non-strict validation",
			fvcount: 2,
			feeds:   []dmfr.Feed{f200, fvErrorExample},
		},
		{
			name:    "strict validation",
			fvcount: 1,
			feeds:   []dmfr.Feed{f200, fvErrorExample},
			strict:  true,
		},
		{
			name:               "strict validation with fail",
			fvcount:            1,
			feeds:              []dmfr.Feed{f200, fvErrorExample},
			fail:               true,
			strict:             true,
			fatalErrorContains: "strict validation failed",
		},
	}
	ctx := context.TODO()
	for _, exp := range cases {
		t.Run(exp.name, func(t *testing.T) {
			adapter := testdb.TempSqliteAdapter()
			for _, feed := range exp.feeds {
				testdb.ShouldInsert(t, adapter, &feed)
			}
			c := FetchCommand{}
			c.Adapter = adapter
			tmpDir := t.TempDir()
			c.Options.Storage = tmpDir
			c.Options.StrictValidation = exp.strict
			c.Fail = exp.fail
			if err := c.Parse(exp.command); err != nil {
				t.Fatal(err)
			}
			if err := c.Run(ctx); err != nil && exp.fatalErrorContains != "" {
				if !strings.Contains(err.Error(), exp.fatalErrorContains) {
					t.Errorf("got '%s' error, expected to contain '%s'", err.Error(), exp.fatalErrorContains)
				}
			} else if err != nil {
				t.Fatal(err)
			} else if exp.fatalErrorContains != "" {
				t.Fatalf("Did not get expected error match: %s", exp.fatalErrorContains)
			}
			// Test
			feeds := []dmfr.Feed{}
			testdb.ShouldSelect(t, adapter, &feeds, "SELECT * FROM current_feeds")
			if len(feeds) != len(exp.feeds) {
				t.Errorf("got %d feeds, expect %d", len(feeds), len(exp.feeds))
			}
			fvs := []dmfr.FeedVersion{}
			testdb.ShouldSelect(t, adapter, &fvs, "SELECT * FROM feed_versions")
			if len(fvs) != exp.fvcount {
				t.Errorf("got %d feed versions, expect %d", len(fvs), exp.fvcount)
			}
			for _, fv := range fvs {
				fn := filepath.Join(tmpDir, fv.File)
				// fn := fv.File
				st, err := os.Stat(fn)
				if err != nil {
					t.Errorf("got '%s', expected file '%s' to exist", err.Error(), fn)
				} else {
					// TODO: Check SHA1
					if st.Size() == 0 {
						t.Errorf("expected non-empty file")
					}
				}
			}
		})
	}
}
