package linerouter

import (
	"testing"

	dt "github.com/interline-io/transitland-lib/server/directions/directionstest"
	"github.com/interline-io/transitland-lib/testdata"
)

func TestRouter(t *testing.T) {
	bt := dt.MakeBasicTests()
	tcs := []dt.TestCase{
		{
			Name:     "ped",
			Req:      bt["ped"],
			Success:  true,
			Duration: 4116,
			Distance: 4.116,
			ResJson:  testdata.Path("server/directions/response/line_ped.json"),
		},
		{
			Name:     "bike",
			Req:      bt["bike"],
			Success:  true,
			Duration: 1029,
			Distance: 4.116,
			ResJson:  "",
		},
		{
			Name:     "auto",
			Req:      bt["auto"],
			Success:  true,
			Duration: 411,
			Distance: 4.116,
			ResJson:  "",
		},
		{
			Name:     "transit",
			Req:      bt["transit"],
			Success:  true,
			Duration: 823,
			Distance: 4.116,
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
	}
	for _, tc := range tcs {
		t.Run(tc.Name, func(t *testing.T) {
			h := &Router{}
			dt.HandlerTest(t, h, tc)
		})
	}
}
