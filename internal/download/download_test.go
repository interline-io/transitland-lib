package download

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/interline-io/transitland-lib/internal/testutil"
)

func TestTemporaryDownload(t *testing.T) {
	expectFile := testutil.ExampleZip.URL
	expectBytes := int64(0)
	if fi, err := os.Stat(expectFile); err != nil {
		t.Error(err)
		t.FailNow()
	} else {
		expectBytes = fi.Size()
	}
	ts200 := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf, err := ioutil.ReadFile(expectFile)
		if err != nil {
			t.Error(err)
		}
		w.Write(buf)
	}))
	defer ts200.Close()
	td := TemporaryDownload{URL: ts200.URL}
	if td.File != nil {
		t.Errorf("expected nil")
	}
	if err := td.Open(); err != nil {
		t.Error(err)
	}
	if td.File == nil {
		t.Errorf("expected non nil")
	}
	fp := td.File.Name()
	if fi, err := os.Stat(fp); err != nil {
		t.Error(err)
	} else if fi.Size() != expectBytes {
		t.Errorf("got %d bytes, expected %d", fi.Size(), expectBytes)
	}
	if err := td.Close(); err != nil {
		t.Error(err)
	}
	if _, err := os.Stat(fp); err == nil {
		t.Errorf("file still exists; expected to be deleted")
	}
}
