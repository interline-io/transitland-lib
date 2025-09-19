package awsrouter

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"path/filepath"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/location"
	dt "github.com/interline-io/transitland-lib/server/directions/directionstest"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/server/testutil"
	"github.com/interline-io/transitland-lib/testdata"
)

func TestRouter(t *testing.T) {
	bt := dt.MakeBasicTests()

	tcs := []dt.TestCase{
		{
			Name:     "ped",
			Req:      bt["ped"],
			Success:  true,
			Duration: 4215,
			Distance: 4.100,
			ResJson:  testdata.Path("server/directions/response/aws_ped.json"),
		},
		{
			Name:     "bike",
			Req:      bt["bike"],
			Success:  false,
			Duration: 0,
			Distance: 0,
			ResJson:  "",
		},
		{
			Name:     "auto",
			Req:      bt["auto"],
			Success:  true,
			Duration: 671,
			Distance: 5.452,
			ResJson:  "",
		},
		{
			Name:     "depart_now",
			Req:      model.DirectionRequest{Mode: model.StepModeAuto, From: &dt.BaseFrom, To: &dt.BaseTo, DepartAt: nil},
			Success:  true,
			Duration: 936,
			Distance: 4.1,
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
			recorder := testutil.NewRecorder(filepath.Join(testdata.Path("server/directions/aws/location"), tc.Name), "directions://aws")
			defer recorder.Stop()
			h, err := makeTestMockRouter(recorder)
			if err != nil {
				t.Fatal(err)
			}
			dt.HandlerTest(t, h, tc)
		})
	}
}

// Mock reader
func makeTestMockRouter(tr http.RoundTripper) (*Router, error) {
	// Use custom client/transport
	cn := ""
	lc := &mockLocationClient{
		Client: &http.Client{
			Transport: tr,
		},
	}
	return NewRouter(lc, cn), nil
}

// Regenerate results
// func makeTestAwsRouter(tr http.RoundTripper) (*awsRouter, error) {
// 	cn := os.Getenv("TL_AWS_LOCATION_CALCULATOR")
// 	cfg, err := awsconfig.LoadDefaultConfig(context.Background())
// 	if err != nil {
// 		return nil, err
// 	}
// 	cfg.HTTPClient = &http.Client{
// 		Transport: tr,
// 	}
// 	lc := location.NewFromConfig(cfg)
// 	return newAWSRouter(lc, cn), nil
// }

// We need to mock out the location services client
type mockLocationClient struct {
	Client *http.Client
}

func (mc *mockLocationClient) CalculateRoute(ctx context.Context, params *location.CalculateRouteInput, opts ...func(*location.Options)) (*location.CalculateRouteOutput, error) {
	reqBody, err := json.Marshal(params)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest("POST", "directions://aws", bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	resp, err := mc.Client.Do(req)
	if err != nil {
		return nil, err
	}
	b, _ := io.ReadAll(resp.Body)
	a := location.CalculateRouteOutput{}
	if err := json.Unmarshal(b, &a); err != nil {
		return nil, err
	}
	return &a, nil
}
