package gtcsv

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/interline-io/gotransit/internal/testutil"
)

// Reader interface tests.

func NewExampleExpect() (*testutil.ExpectEntities, *Reader) {
	reader, err := NewReader("../testdata/example")
	if err != nil {
		panic(err)
	}
	fe := &testutil.ExpectEntities{
		AgencyCount:        1,
		RouteCount:         5,
		TripCount:          11,
		StopCount:          9,
		StopTimeCount:      28,
		ShapeCount:         3,
		CalendarCount:      2,
		CalendarDateCount:  2,
		FeedInfoCount:      1,
		FareRuleCount:      4,
		FareAttributeCount: 2,
		FrequencyCount:     11,
		TransferCount:      0,
		ExpectAgencyIDs:    []string{"DTA"},
		ExpectRouteIDs:     []string{"AB", "BFC", "STBA", "CITY", "AAMV"},
		ExpectTripIDs:      []string{"AB1", "AB2", "STBA", "CITY1", "CITY2", "BFC1", "BFC2", "AAMV1", "AAMV2", "AAMV3", "AAMV4"},
		ExpectStopIDs:      []string{"FUR_CREEK_RES", "BULLFROG"}, // partial
		ExpectShapeIDs:     []string{"ok", "a", "c"},
		ExpectCalendarIDs:  []string{"FULLW", "WE"},
		ExpectFareIDs:      []string{"p", "a"},
	}
	return fe, reader
}

func TestReader(t *testing.T) {
	t.Run("Dir", func(t *testing.T) {
		fe, r := NewExampleExpect()
		if err := r.Open(); err != nil {
			t.Error(err)
		}
		defer r.Close()
		testutil.TestExpectEntities(t, *fe, r)
	})
	t.Run("Zip", func(t *testing.T) {
		fe, _ := NewExampleExpect()
		reader, err := NewReader("../testdata/example.zip")
		if err != nil {
			t.Error(err)
		}
		if err := reader.Open(); err != nil {
			t.Error(err)
		}
		defer reader.Close()
		testutil.TestExpectEntities(t, *fe, reader)
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
		//
		fe, _ := NewExampleExpect()
		reader, err := NewReader(ts.URL)
		if err != nil {
			t.Error(err)
		}
		if err := reader.Open(); err != nil {
			t.Error(err)
		}
		defer reader.Close()
		testutil.TestExpectEntities(t, *fe, reader)
	})
}
