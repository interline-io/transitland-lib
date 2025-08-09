package valhalla

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
	fdir := testdata.Path("directions/valhalla")
	tcs := []dt.TestCase{
		{
			Name:     "ped",
			Req:      bt["ped"],
			Success:  true,
			Duration: 3130,
			Distance: 4.387,
			ResJson:  testdata.Path("directions/response/val_ped.json"),
		},
		{
			Name:     "bike",
			Req:      bt["bike"],
			Success:  true,
			Duration: 1132,
			Distance: 4.912,
			ResJson:  "",
		},
		{
			Name:     "auto",
			Req:      bt["auto"],
			Success:  true,
			Duration: 1037,
			Distance: 5.133,
			ResJson:  "",
		},
		{
			Name:     "transit",
			Req:      bt["transit"],
			Success:  false,
			Duration: 0,
			Distance: 0,
			ResJson:  "",
		},
		{
			Name:     "no_dest_fail",
			Req:      bt["no_dest_fail"],
			Success:  false,
			Duration: 0,
			Distance: 0,
			ResJson:  "",
		},
		{
			Name:     "no_routable_dest_fail",
			Req:      bt["no_routable_dest_fail"],
			Success:  false,
			Duration: 0,
			Distance: 0,
			ResJson:  "",
		},
	}
	for _, tc := range tcs {
		t.Run(tc.Name, func(t *testing.T) {
			recorder := testutil.NewRecorder(filepath.Join(fdir, tc.Name), "directions://valhalla")
			defer recorder.Stop()
			h, err := makeTestRouter(recorder)
			if err != nil {
				t.Fatal(err)
			}
			dt.HandlerTest(t, h, tc)
		})
	}
}

func makeTestRouter(tr http.RoundTripper) (*Router, error) {
	endpoint := os.Getenv("TL_TEST_VALHALLA_ENDPOINT")
	apikey := os.Getenv("TL_TEST_VALHALLA_API_KEY")
	client := &http.Client{
		Timeout:   10 * time.Second,
		Transport: tr,
	}
	return NewRouter(client, endpoint, apikey), nil
}
