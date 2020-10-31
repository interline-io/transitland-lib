package dmfr

import (
	"encoding/csv"
	"os"
	"strconv"
	"testing"

	"github.com/interline-io/transitland-lib/tlcsv"
)

func TestNewFeedVersionServiceInfosFromReader(t *testing.T) {
	url := "/Users/irees/src/interline-io/tlv2/testdata/gtfs/f-9q9-actransit/ac.zip"
	// url := "../test/data/external/bart.zip" // ExampleZip.URL
	// url := "/Users/irees/data/gtfs/bdb13d3afdcda9c5a2367a4cb6c2d1137a1ca322.zip"
	reader, err := tlcsv.NewReader(url)
	if err != nil {
		panic(err)
	}
	results, err := NewFeedVersionServiceInfosFromReader(reader)
	if err != nil {
		t.Error(err)
	}

	file, err := os.Create("/Users/irees/test/tl.csv")
	if err != nil {
		panic(err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	writer.Write([]string{"route_id", "input_date", "service_seconds"})
	defer writer.Flush()

	for _, result := range results {
		wk := []int{result.Monday, result.Tuesday, result.Wednesday, result.Thursday, result.Friday, result.Saturday, result.Sunday}
		start := result.StartDate
		for start.Before(result.EndDate) {
			for _, v := range wk {
				err := writer.Write([]string{
					result.RouteID,
					start.Format("2006-01-02"),
					strconv.Itoa(v),
				})
				if err != nil {
					panic(err)
				}
				start = start.AddDate(0, 0, 1)
			}
		}
	}
}
