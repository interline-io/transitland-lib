package gtcsv

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/interline-io/gotransit/internal/testutil"
)

func compareMap(t *testing.T, result map[string]int, expect map[string]int) {
	for k, v := range expect {
		if i := result[k]; v != i {
			t.Error("expeced", k, "=", i)
		}
	}
}

// Reader interface tests.

func TestReader(t *testing.T) {
	t.Run("Dir", func(t *testing.T) {
		reader, _ := NewReader("../testdata/example")
		reader.Open()
		defer reader.Close()
		testutil.ReaderTester(reader, t)
	})
	t.Run("Zip", func(t *testing.T) {
		reader, _ := NewReader("../testdata/example.zip")
		reader.Open()
		defer reader.Close()
		testutil.ReaderTester(reader, t)
	})
	t.Run("URL", func(t *testing.T) {
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			buf, err := ioutil.ReadFile("../testdata/example.zip")
			if err != nil {
				t.Error(err)
			}
			w.Write(buf)
		}))
		defer ts.Close()
		reader, _ := NewReader(ts.URL)
		reader.Open()
		defer reader.Close()
		testutil.ReaderTester(reader, t)
	})
}
