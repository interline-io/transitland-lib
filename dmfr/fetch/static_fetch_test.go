package fetch

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/interline-io/transitland-lib/internal/testdb"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tldb"
)

func TestStaticFetch(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf, err := ioutil.ReadFile(ExampleZip.URL)
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
		fr, err := StaticFetch(atx, feed, Options{FeedURL: url, Directory: tmpdir})
		if err != nil {
			t.Error(err)
			return err
		}
		if fr.Found {
			t.Error("expected new feed")
			return nil
		}
		if fr.FeedVersion.SHA1 != ExampleZip.SHA1 {
			t.Errorf("got %s expect %s", fr.FeedVersion.SHA1, ExampleZip.SHA1)
			return nil
		}
		fv2 := tl.FeedVersion{ID: fr.FeedVersion.ID}
		testdb.ShouldFind(t, atx, &fv2)
		if fv2.URL != url {
			t.Errorf("got %s expect %s", fv2.URL, url)
		}
		if fv2.FeedID != feed.ID {
			t.Errorf("got %d expect %d", fv2.FeedID, feed.ID)
		}
		if fv2.SHA1 != ExampleZip.SHA1 {
			t.Errorf("got %s expect %s", fv2.SHA1, ExampleZip.SHA1)
		}
		return nil
	})
}

func TestStaticFetch_404(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Status-Code", "404")
		w.Write([]byte("not found"))
	}))
	defer ts.Close()
	testdb.WithAdapterRollback(func(atx tldb.Adapter) error {
		url := ts.URL
		feed := testdb.CreateTestFeed(atx, url)
		fr, err := StaticFetch(atx, feed, Options{FeedURL: url, Directory: ""})
		if err != nil {
			t.Error(err)
			return err
		}
		if fr.FetchError == nil {
			t.Error("expected error")
			return nil
		}
		if fr.FeedVersion.ID != 0 {
			t.Errorf("got %d expect %d", fr.FeedVersion.ID, 0)
		}
		errmsg := fr.FetchError.Error()
		experr := "file does not exist"
		if !strings.HasPrefix(errmsg, experr) {
			t.Errorf("got '%s' expected prefix '%s'", errmsg, experr)
		}
		return nil
	})
}

func TestStaticFetch_Exists(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf, err := ioutil.ReadFile(ExampleZip.URL)
		if err != nil {
			t.Error(err)
		}
		w.Write(buf)
	}))
	testdb.WithAdapterRollback(func(atx tldb.Adapter) error {
		url := ts.URL
		feed := testdb.CreateTestFeed(atx, url)
		fr, err := StaticFetch(atx, feed, Options{FeedURL: url, Directory: ""})
		if err != nil {
			t.Error(err)
		}
		if fr.Found {
			t.Error("expected new feed")
		}
		if fr.FeedVersion.SHA1 != ExampleZip.SHA1 {
			t.Errorf("got %s expect %s", fr.FeedVersion.SHA1, ExampleZip.SHA1)
		}
		fr2, err2 := StaticFetch(atx, feed, Options{FeedURL: url, Directory: ""})
		if err2 != nil {
			t.Error(err2)
			return err2
		}
		if !(fr2.Found) {
			t.Error("expected found feed")
		}
		if fr2.FeedVersion.SHA1 != ExampleZip.SHA1 {
			t.Errorf("got %s expect %s", fr2.FeedVersion.SHA1, ExampleZip.SHA1)
		}
		if fr2.FeedVersion.ID == 0 {
			t.Error("expected non-zero value")
		}
		if fr.FeedVersion.ID != fr2.FeedVersion.ID {
			t.Errorf("got %d expected %d", fr.FeedVersion.ID, fr2.FeedVersion.ID)
		}
		return nil
	})
}
