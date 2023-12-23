package validator

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/interline-io/transitland-lib/internal/testdb"
	"github.com/interline-io/transitland-lib/internal/testutil"
	"github.com/interline-io/transitland-lib/tlcsv"
	"github.com/interline-io/transitland-lib/tldb"
)

func TestSaveValidationReport(t *testing.T) {
	reader, err := tlcsv.NewReader(testutil.RelPath("test/data/rt/ct.zip"))
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf, err := os.ReadFile(testutil.RelPath(filepath.Join("test/data/rt", r.URL.Path)))
		if err != nil {
			t.Error(err)
		}
		w.Write(buf)
	}))

	tz, _ := time.LoadLocation("America/Los_Angeles")
	now := time.Date(2023, 11, 7, 17, 05, 0, 0, tz)
	opts := Options{
		IncludeRealtimeJson: true,
		EvaluateAt:          now,
		ValidateRealtimeMessages: []string{
			ts.URL + "/ct-trip-updates.pb.json",
			ts.URL + "/ct-vehicle-positions.pb.json",
		},
	}

	v, _ := NewValidator(reader, opts)
	result, err := v.Validate()
	if err != nil {
		t.Fatal(err)
	}
	testdb.TempSqlite(func(atx tldb.Adapter) error {
		if err := SaveValidationReport(atx, result, now, 1, true, true, ""); err != nil {
			t.Fatal(err)
		}
		return nil
	})
}
