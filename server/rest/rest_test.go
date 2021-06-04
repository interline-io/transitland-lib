package rest

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

const LON = 37.803613
const LAT = -122.271556

type testCase struct {
	Name             string
	Key              string
	Request          apiHandler
	Format           string
	ExpectCount      int
	ExpectOnestopIDs []string
}

func checkTestCase(t *testing.T, tc testCase) {
	tm := time.Now()
	data, err := makeRequest("http://localhost:8080", tc.Request, tc.Format)
	response := hw{}
	json.Unmarshal(data, &response)
	if err != nil {
		t.Error(err)
		return
	}
	osids := map[string]bool{}
	if err, ok := response["error"]; ok {
		t.Error(err)
		return
	}
	features, ok := response[tc.Key].([]interface{})
	if !ok {
		t.Error("no values for key")
		return
	}
	for _, f := range features {
		f2, ok := f.(map[string]interface{})
		if ok {
			osid, ok2 := f2["onestop_id"].(string)
			if ok2 {
				osids[osid] = true
			}
		}
	}

	for _, v := range tc.ExpectOnestopIDs {
		if !osids[v] {
			t.Errorf("did not find expected entity %s", v)
		}
	}
	if len(features) != tc.ExpectCount {
		t.Errorf("got %d expect %d", len(features), tc.ExpectCount)
	}
	fmt.Println("time:", (time.Now().UnixNano()-tm.UnixNano())/1e6, "ms")
}

func TestFeedRequest(t *testing.T) {
	bartosid := []string{"f-9q9-bart"}
	cases := []testCase{
		{"none", "feeds", &FeedRequest{}, "", 4, nil},
		{"onestop_id", "feeds", &FeedRequest{OnestopID: "f-9q9-bart"}, "", 1, bartosid},
		// {"lat,lon,r:100,limit:100", "feeds", &FeedRequest{Limit: 100, Lon: LON, Lat: LAT, Radius: 100.0}, "", 2, bartosid},
		// {"lat,lon,r:1000,limit:100", "feeds", &FeedRequest{Limit: 100, Lon: LON, Lat: LAT, Radius: 1000.0}, "", 5, bartosid},
	}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			checkTestCase(t, tc)
		})
	}
}

func TestFeedVersionRequest(t *testing.T) {
	bartosid := []string{}
	cases := []testCase{
		{"none", "feed_versions", &FeedVersionRequest{}, "", 4, nil},
		{"limit:1", "feed_versions", &FeedVersionRequest{Limit: 1}, "", 1, nil},
		{"limit:2", "feed_versions", &FeedVersionRequest{Limit: 2}, "", 2, nil},
		// {"sha1", "feed_versions", &FeedVersionRequest{FeedVersionSHA1: "bd3c5cb000c28124d47b7f7d49c7067a10b772c9"}, "", 1, nil},
		{"feed_onestop_id,limit:100", "feed_versions", &FeedVersionRequest{Limit: 100, FeedOnestopID: "f-9q9-bart"}, "", 1, bartosid},
	}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			checkTestCase(t, tc)
		})
	}
}

func TestAgencyRequest(t *testing.T) {
	bartosid := []string{"o-9q9-bayarearapidtransit"}
	cases := []testCase{
		{"none", "agencies", AgencyRequest{}, "", 4, nil},
		{"limit:1", "agencies", AgencyRequest{Limit: 1}, "", 1, nil},
		{"limit:100", "agencies", AgencyRequest{Limit: 100}, "", 4, nil},
		{"feed_version_sha1", "agencies", AgencyRequest{FeedVersionSHA1: "bd3c5cb000c28124d47b7f7d49c7067a10b772c9"}, "", 1, bartosid},
		{"feed_onestop_id", "agencies", AgencyRequest{FeedOnestopID: "f-9q9-bart"}, "", 1, bartosid},
		{"feed_onestop_id,agency_id", "agencies", AgencyRequest{FeedOnestopID: "f-9q9-bart", AgencyID: "BA"}, "", 1, bartosid},
		{"agency_id", "agencies", AgencyRequest{AgencyID: "BA"}, "", 1, bartosid},
		{"agency_name", "agencies", AgencyRequest{AgencyName: "Bay Area Rapid Transit"}, "", 1, bartosid},
		{"onestop_id", "agencies", AgencyRequest{OnestopID: "o-9q9-bayarearapidtransit"}, "", 1, bartosid},
		{"onestop_id,feed_version_sha1", "agencies", AgencyRequest{OnestopID: "o-9q9-bayarearapidtransit", FeedVersionSHA1: "bd3c5cb000c28124d47b7f7d49c7067a10b772c9"}, "", 1, bartosid},
		// {"lat,lon,r:100,limit:100", "agencies", AgencyRequest{Limit: 100, Lon: LON, Lat: LAT, Radius: 100.0}, "", 2, bartosid},
		// {"lat,lon,r:1000,limit:100", "agencies", AgencyRequest{Limit: 100, Lon: LON, Lat: LAT, Radius: 1000.0}, "", 3, bartosid},
		// {"feed_version_sha1,lat,lon,r:1000", "agencies", AgencyRequest{FeedVersionSHA1: "8aafa62ec25558004b74202202e78a7bb67a5747", Lon: LON, Lat: LAT, Radius: 1000.0}, "", 1, bartosid},
		// {"feed_onestop_id,lat,lon,r:1000", "agencies", AgencyRequest{FeedOnestopID: "f-9q9-bart", Lon: LON, Lat: LAT, Radius: 1000.0}, "", 1, bartosid},
		// {"feed_onestop_id2,lat,lon,r:1000,limit:100", "agencies", AgencyRequest{Limit: 100, FeedOnestopID: "f-9q9-actransit", Lon: LON, Lat: LAT, Radius: 1000.0}, "", 1, []string{"o-9q9-actransit"}},
	}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			checkTestCase(t, tc)
		})
	}
}

func TestStopRequest(t *testing.T) {
	bartosid := []string{}
	cases := []testCase{
		{"none", "stops", StopRequest{}, "", 20, bartosid},
		{"limit:1", "stops", StopRequest{Limit: 1}, "", 1, bartosid},
		{"limit:100", "stops", StopRequest{Limit: 100}, "", 100, bartosid},
		{"feed_onestop_id", "stops", StopRequest{FeedOnestopID: "f-9q9-bart"}, "", 20, bartosid},
		{"feed_onestop_id,stop_id", "stops", StopRequest{FeedOnestopID: "f-9q9-bart", StopID: "12TH"}, "", 1, bartosid},
		{"feed_version_sha1", "stops", StopRequest{FeedVersionSHA1: "bd3c5cb000c28124d47b7f7d49c7067a10b772c9"}, "", 20, bartosid},
		// {"lat,lon,r:100,limit:100", "stops", StopRequest{Limit: 100, Lon: LON, Lat: LAT, Radius: 100.0}, "", 13, bartosid},
		// {"lat,lon,r:1000,limit:100", "stops", StopRequest{Limit: 100, Lon: LON, Lat: LAT, Radius: 1000.0}, "", 100, bartosid},
		// {"feed_version_sha1,lat,lon,r:1000", "stops", StopRequest{FeedVersionSHA1: "8aafa62ec25558004b74202202e78a7bb67a5747", Lon: LON, Lat: LAT, Radius: 1000.0}, "", 17, bartosid},
		// {"feed_onestop_id,lat,lon,r:1000", "stops", StopRequest{FeedOnestopID: "f-9q9-bart", Lon: LON, Lat: LAT, Radius: 1000.0}, "", 17, bartosid},
		// {"feed_onestop_id,lat,lon,r:1000,limit:100", "stops", StopRequest{Limit: 100, FeedOnestopID: "f-9q9-actransit", Lon: LON, Lat: LAT, Radius: 1000.0}, "", 100, bartosid},
	}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			checkTestCase(t, tc)
		})
	}
}

func TestRouteRequest(t *testing.T) {
	bartosid := []string{}
	cases := []testCase{
		{"none", "routes", RouteRequest{}, "", 20, bartosid},
		{"limit:1", "routes", RouteRequest{Limit: 1}, "", 1, bartosid},
		{"limit:100", "routes", RouteRequest{Limit: 100}, "", 100, bartosid},
		{"feed_onestop_id", "routes", RouteRequest{FeedOnestopID: "f-9q9-bart"}, "", 14, bartosid},
		{"route_type", "routes", RouteRequest{RouteType: "2"}, "", 19, bartosid},
		{"feed_onestop_id,route_id", "routes", RouteRequest{FeedOnestopID: "f-9q9-bart", RouteID: "BG-S"}, "", 1, bartosid},
		{"feed_version_sha1", "routes", RouteRequest{FeedVersionSHA1: "bd3c5cb000c28124d47b7f7d49c7067a10b772c9"}, "", 14, bartosid},
		{"operator_onestop_id", "routes", RouteRequest{OperatorOnestopID: "o-9q9-bayarearapidtransit"}, "", 14, bartosid},
	}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			checkTestCase(t, tc)
		})
	}
}

func TestTripRequest(t *testing.T) {
	bartosid := []string{}
	cases := []testCase{
		{"none", "trips", TripRequest{}, "", 20, bartosid},
		{"limit:1", "trips", TripRequest{Limit: 1}, "", 1, bartosid},
		{"limit:100", "trips", TripRequest{Limit: 100}, "", 100, bartosid},
		{"feed_onestop_id", "trips", TripRequest{FeedOnestopID: "f-9q9-bart"}, "", 20, bartosid},
		{"feed_version_sha1", "routes", RouteRequest{FeedVersionSHA1: "bd3c5cb000c28124d47b7f7d49c7067a10b772c9"}, "", 14, bartosid},
		// {"route_id", "trips", TripRequest{Limit: 100, RouteID: 271175}, "", 100, bartosid},
		// {"feed_onestop_id,route_id", "trips", TripRequest{FeedOnestopID: "f-9q9-bart", RouteID: 271175, Limit: 1}, "", 1, bartosid},
	}
	for _, tc := range cases {
		t.Run(tc.Name, func(t *testing.T) {
			checkTestCase(t, tc)
		})
	}
}
