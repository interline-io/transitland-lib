package gql

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

// Fixture stops in feed versions that resolve 2018-05-29 differently: BA's
// service window covers it, so BA answers for the date as asked, while the
// others have no window covering it and relocate it into their own fallback
// week.
const serviceWindowDate = "2018-05-29"

var (
	serviceWindowInWindow  = [2]string{"BA", "MONT"}
	serviceWindowRelocates = [][2]string{{"CT", "70011"}, {"HA", "1014"}, {"WMATA", "ENT_A01_C01_E"}}
)

// One `stops` selection can span feed versions, and every stop in it shares a
// single filter. Each feed version resolves the requested date against its own
// service window, and that resolution must not follow the shared filter into
// the next feed version. The finder visits them in map order, so a leak makes
// BA's answer depend on whether it was visited first, and the same request
// returns different stop times from one call to the next.
func TestStopResolver_ServiceWindowIsPerFeedVersion(t *testing.T) {
	c, _ := newTestClient(t)

	query := func(q string) string {
		var resp map[string]interface{}
		if err := c.Post(q, &resp); err != nil {
			t.Fatal(err)
		}
		return toJson(resp)
	}
	// Resolved rather than hardcoded: these are unstable database ids.
	stopID := func(feed string, gtfsStopID string) string {
		j := query(`query{stops(where:{feed_onestop_id:"` + feed + `", stop_id:"` + gtfsStopID + `"}){id}}`)
		id := gjson.Get(j, "stops.0.id")
		if !id.Exists() {
			t.Fatalf("fixture stop %s:%s not found", feed, gtfsStopID)
		}
		return id.String()
	}
	inWindow := stopID(serviceWindowInWindow[0], serviceWindowInWindow[1])
	// Three more feed versions. Only the visit order decides whether a leak
	// shows, so one alone would leave a coin flip; with these the odds of
	// missing it across the runs below are about one in a million.
	var relocating []string
	for _, s := range serviceWindowRelocates {
		relocating = append(relocating, stopID(s[0], s[1]))
	}

	// stop_times for one stop out of a request covering all the given stops.
	departures := func(want string, ids ...string) string {
		j := query(`query{stops(ids:[` + strings.Join(ids, ",") + `]){
			id stop_times(where:{date:"` + serviceWindowDate + `", use_service_window:true}){ service_date departure_time }
		}}`)
		for _, s := range gjson.Get(j, "stops").Array() {
			if s.Get("id").String() == want {
				return s.Get("stop_times").Raw
			}
		}
		t.Fatalf("stop %s missing from response %s", want, j)
		return ""
	}

	// The answer when nothing else shares the request.
	alone := departures(inWindow, inWindow)
	assert.Contains(t, alone, `"service_date":"`+serviceWindowDate+`"`,
		"fixture should answer for the requested date, not a relocated one")

	// The premise: at least one of the others really does relocate the date.
	// Without this the test can keep passing while asserting nothing, if the
	// fixtures ever drift so that no feed version resolves the date differently.
	relocated := false
	for i, id := range relocating {
		sd := gjson.Get(departures(id, id), "0.service_date").String()
		if sd != "" && sd != serviceWindowDate {
			relocated = true
			break
		}
		t.Logf("%v did not relocate %s (service_date %q)", serviceWindowRelocates[i], serviceWindowDate, sd)
	}
	assert.True(t, relocated, "fixture should include a feed version that relocates the date")

	// Adding stops whose feed versions relocate the date must not change BA's
	// answer, and must not change it differently from one call to the next.
	all := append([]string{inWindow}, relocating...)
	for i := 0; i < 10; i++ {
		if !assert.JSONEq(t, alone, departures(inWindow, all...), "run %d differs from the single-stop query", i) {
			t.FailNow()
		}
	}
}
