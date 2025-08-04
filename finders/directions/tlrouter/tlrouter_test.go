package tlrouter

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"
	"time"

	dt "github.com/interline-io/transitland-lib/finders/directions/directionstest"
	"github.com/interline-io/transitland-lib/testdata"
	"github.com/interline-io/transitland-mw/testutil"
)

func TestRouter(t *testing.T) {
	bt := dt.MakeBasicTests()
	baseTime := time.Unix(1738200531, 0).In(time.UTC)
	fdir := testdata.Path("directions/tlrouter")
	tcs := []dt.TestCase{
		{
			Name:     "ped",
			Req:      bt["ped"],
			Success:  true,
			Duration: 4116,
			Distance: 4.4618,
		},
		{
			Name:    "auto",
			Req:     bt["auto"],
			Success: false,
		},
		{
			Name:     "transit",
			Req:      bt["transit"],
			Success:  true,
			Duration: 1480,
			Distance: 4.4618,
		},
		{
			Name:    "no_dest_fail",
			Req:     bt["no_dest_fail"],
			Success: false,
		},
	}
	for _, tc := range tcs {
		t.Run(tc.Name, func(t *testing.T) {
			recorder := testutil.NewRecorder(filepath.Join(fdir, tc.Name), "directions://tlrouter")
			defer recorder.Stop()
			h, err := makeTestRouter(recorder)
			if err != nil {
				t.Fatal(err)
			}
			tc.Req.DepartAt = &baseTime
			dt.HandlerTest(t, h, tc)
		})
	}
}

func makeTestRouter(tr http.RoundTripper) (*Router, error) {
	endpoint := os.Getenv("TL_TEST_TLROUTER_ENDPOINT")
	apikey := os.Getenv("TL_TEST_TLROUTER_APIKEY")
	client := &http.Client{
		Timeout:   10 * time.Second,
		Transport: tr,
	}
	return NewRouter(client, endpoint, apikey), nil
}
