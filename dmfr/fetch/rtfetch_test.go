package fetch

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/interline-io/transitland-lib/internal/testdb"
	"github.com/interline-io/transitland-lib/internal/testutil"
	"github.com/interline-io/transitland-lib/tldb"
)

func TestRTFetch(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf, err := ioutil.ReadFile(testutil.RelPath("test/data/rt/example.pb"))
		if err != nil {
			t.Error(err)
		}
		w.Write(buf)
	}))
	defer ts.Close()
	testdb.WithAdapterRollback(func(atx tldb.Adapter) error {
		tmpdir, err := ioutil.TempDir("", "gtfs")
		if err != nil {
			t.Error(err)
			return nil
		}
		defer os.RemoveAll(tmpdir) // clean up
		url := ts.URL
		feed := testdb.CreateTestFeed(atx, url)
		fr, err := RTFetch(atx, feed, Options{FeedURL: url, Directory: tmpdir})
		if err != nil {
			t.Error(err)
			return err
		}
		if fr.Found {
			t.Fatal("expected new feed")
		}
		if fr.FeedVersion.SHA1 != ExampleZip.SHA1 {
			t.Fatalf("got %s expect %s", fr.FeedVersion.SHA1, ExampleZip.SHA1)
		}
		return nil
	})
}
