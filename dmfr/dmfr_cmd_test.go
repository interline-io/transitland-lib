package dmfr

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/interline-io/gotransit"
	"github.com/interline-io/gotransit/internal/testdb"
)

// Test some commands.
// This should only test high level command functionality.
// Use a disk-backed sqlite for simplicity.

func Test_dmfrImportCommand(t *testing.T) {

}

func Test_dmfrSyncCommand(t *testing.T) {
	cases := []struct {
		count       int
		errContains string
		command     []string
	}{
		{2, "", []string{"sync", "../testdata/dmfr/example.json"}},
		{4, "", []string{"sync", "../testdata/dmfr/example.json", "../testdata/dmfr/bayarea.dmfr.json"}},
		{0, "no such file", []string{"sync", "../testdaata/dmfr/does-not-exist.json"}},
	}
	_ = cases
	for _, exp := range cases {
		t.Run("", func(t *testing.T) {
			w := mustGetWriter("sqlite3://:memory:", true)
			c := dmfrSyncCommand{adapter: w.Adapter}
			err := c.Run(exp.command)
			if err != nil {
				if !strings.Contains(err.Error(), exp.errContains) {
					t.Errorf("got '%s' error, expected to contain '%s'", err.Error(), exp.errContains)
				}
			}
			// Test
			feeds := []Feed{}
			w.Adapter.Select(&feeds, "SELECT * FROM current_feeds")
			if len(feeds) != exp.count {
				t.Errorf("got %d feeds, expect %d", len(feeds), exp.count)
			}
		})

	}
}

func Test_dmfrFetchCommand(t *testing.T) {
	ts200 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf, err := ioutil.ReadFile(ExampleZip.URL)
		if err != nil {
			t.Error(err)
		}
		w.Write(buf)
	}))
	defer ts200.Close()
	ts404 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Status-Code", "404")
		w.Write([]byte("not found"))
	}))
	defer ts404.Close()
	// tempdir
	tmpdir, err := ioutil.TempDir("", "gtfs")
	if err != nil {
		t.Error(err)
		return
	}
	defer os.RemoveAll(tmpdir) // clean up
	// note - Spec==gtfs is required for fetch
	f200 := Feed{FeedID: "f--200", Spec: "gtfs", URLs: FeedUrls{StaticCurrent: ts200.URL}}
	f404 := Feed{FeedID: "f--404", Spec: "gtfs", URLs: FeedUrls{StaticCurrent: ts404.URL}}
	cases := []struct {
		fvcount     int
		errContains string
		feeds       []Feed
		gtfsdir     string
		command     []string
	}{
		{1, "", []Feed{f200}, "", []string{"fetch"}},
		{1, "", []Feed{f200, f404}, "", []string{"fetch", "f--200", "f--404"}},
		{1, "", []Feed{f200, f404}, tmpdir, []string{"fetch", "-gtfsdir", tmpdir, "f--200"}},
		{0, "", []Feed{f200, f404}, "", []string{"fetch", "f--404"}},
	}
	_ = cases
	for _, exp := range cases {
		t.Run("", func(t *testing.T) {
			adapter := mustGetWriter("sqlite3://:memory:", true).Adapter
			for _, feed := range exp.feeds {
				testdb.ShouldInsert(t, adapter, &feed)
			}
			c := dmfrFetchCommand{adapter: adapter}
			err := c.Run(exp.command)
			if err != nil {
				if !strings.Contains(err.Error(), exp.errContains) {
					t.Errorf("got '%s' error, expected to contain '%s'", err.Error(), exp.errContains)
				}
			}
			// Test
			feeds := []Feed{}
			testdb.ShouldSelect(t, adapter, &feeds, "SELECT * FROM current_feeds")
			if len(feeds) != len(exp.feeds) {
				t.Errorf("got %d feeds, expect %d", len(feeds), len(exp.feeds))
			}
			// for _, feed := range feeds {
			// 	fmt.Printf("Feed: %#v\n", feed)
			// }
			fvs := []gotransit.FeedVersion{}
			testdb.ShouldSelect(t, adapter, &fvs, "SELECT * FROM feed_versions")
			if len(fvs) != exp.fvcount {
				t.Errorf("got %d feed versions, expect %d", len(fvs), exp.fvcount)
			}
			if exp.gtfsdir != "" {
				for _, fv := range fvs {
					// fn := filepath.Join(exp.gtfsdir, fv.File)
					fn := fv.File
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
			}
		})
	}
}
