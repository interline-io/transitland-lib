package gtcsv

import (
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/interline-io/gotransit/internal/mock"
)

// Reader interface tests.

func NewCSVExampleExpect() *mock.Expect {
	reader, err := NewReader("../testdata/example")
	if err != nil {
		panic(err)
	}
	return &mock.Expect{
		AgencyCount:        1,
		RouteCount:         5,
		TripCount:          11,
		StopCount:          9,
		StopTimeCount:      28,
		ShapeCount:         9,
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
		Reader:             reader,
	}
}

func TestReader(t *testing.T) {
	t.Run("Dir", func(t *testing.T) {
		fe := NewCSVExampleExpect()
		fe.Reader.Open()
		defer fe.Reader.Close()
		mock.TestExpect(t, *fe, fe.Reader)
	})
	t.Run("Zip", func(t *testing.T) {
		fe := NewCSVExampleExpect()
		reader, _ := NewReader("../testdata/example.zip")
		fe.Reader = reader
		fe.Reader.Open()
		defer fe.Reader.Close()
		mock.TestExpect(t, *fe, fe.Reader)
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
		fe := NewCSVExampleExpect()
		reader, _ := NewReader(ts.URL)
		fe.Reader = reader
		fe.Reader.Open()
		defer fe.Reader.Close()
		mock.TestExpect(t, *fe, fe.Reader)
	})
}
