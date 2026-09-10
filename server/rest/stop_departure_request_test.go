package rest

import (
	"slices"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tidwall/gjson"
)

func TestStopDepartureRequest(t *testing.T) {
	bp := func(v bool) *bool {
		return &v
	}
	sid := "s-9q9nfsxn67-fruitvale"
	testcases := []testCase{
		{
			name:         "basic",
			h:            StopDepartureRequest{StopKey: sid},
			format:       "",
			selector:     "stops.#.stop_id",
			expectSelect: nil,
			expectLength: 1,
		},
		{
			name:         "departure 10:00:00",
			h:            StopDepartureRequest{StopKey: sid, ServiceDate: "2018-06-04", StartTime: "10:00:00", WithCursor: WithCursor{Limit: 5}},
			format:       "",
			selector:     "stops.0.departures.#.departure_time",
			expectSelect: []string{"10:02:00", "10:02:00", "10:05:00", "10:09:00", "10:12:00"},
		},
		{
			name:         "departure 10:00:00 to 10:10:00",
			h:            StopDepartureRequest{StopKey: sid, ServiceDate: "2018-06-04", StartTime: "10:00:00", EndTime: "10:10:00"},
			format:       "",
			selector:     "stops.0.departures.#.departure_time",
			expectSelect: []string{"10:02:00", "10:02:00", "10:05:00", "10:09:00"},
		},
		{
			name:         "include_geometry=true",
			h:            StopDepartureRequest{StopKey: sid, ServiceDate: "2018-06-04", StartTime: "10:00:00", EndTime: "10:10:00", IncludeGeometry: true},
			format:       "",
			selector:     "stops.0.departures.0.trip.shape.geometry.type",
			expectSelect: []string{"LineString"},
		},
		{
			name:         "include_geometry=false",
			h:            StopDepartureRequest{StopKey: sid, ServiceDate: "2018-06-04", StartTime: "10:00:00", EndTime: "10:10:00", IncludeGeometry: false},
			format:       "",
			selector:     "stops.0.departures.0.trip.shape.geometry.type",
			expectSelect: []string{},
		},
		{
			name: "next=4 hours",
			h:    StopDepartureRequest{StopKey: sid, Next: 4 * 3600, IncludeGeometry: false, WithCursor: WithCursor{Limit: 1000}},
			f: func(t *testing.T, jj string) {
				a := gjson.Get(jj, "stops.0.departures").Array()
				dates := map[string]int{}
				serviceDates := map[string]int{}
				var departures []string
				for _, st := range a {
					dates[st.Get("date").String()] += 1
					serviceDates[st.Get("service_date").String()] += 1
					departures = append(departures, st.Get("departure.scheduled_local").String())
				}
				slices.Sort(departures)
				assert.Equal(t, map[string]int{"2018-05-31": 75}, dates, "dates")
				assert.Equal(t, map[string]int{"2018-05-31": 75}, serviceDates, "service_dates")
				assert.Equal(t, "2018-05-31T17:02:00-07:00", slices.Min(departures), "departure min")
				assert.Equal(t, "2018-05-31T20:48:00-07:00", slices.Max(departures), "departure max")
			},
		},
		{
			name: "next=24 hours",
			h:    StopDepartureRequest{StopKey: sid, Next: 24 * 3600, IncludeGeometry: false, WithCursor: WithCursor{Limit: 1000}},
			f: func(t *testing.T, jj string) {
				a := gjson.Get(jj, "stops.0.departures").Array()
				dates := map[string]int{}
				serviceDates := map[string]int{}
				var departures []string
				for _, st := range a {
					dates[st.Get("date").String()] += 1
					serviceDates[st.Get("service_date").String()] += 1
					departures = append(departures, st.Get("departure.scheduled_local").String())
				}
				slices.Sort(departures)
				assert.Equal(t, map[string]int{"2018-05-31": 111, "2018-06-01": 303}, dates, "dates")
				assert.Equal(t, map[string]int{"2018-05-31": 120, "2018-06-01": 294}, serviceDates, "service_dates")
				assert.Equal(t, "2018-05-31T17:02:00-07:00", slices.Min(departures), "departure min")
				assert.Equal(t, "2018-06-01T16:57:00-07:00", slices.Max(departures), "departure max")
			},
		},
		{
			name: "next=4 hours relative_date=next saturday",
			h:    StopDepartureRequest{StopKey: sid, Next: 4 * 3600, RelativeDate: "NEXT_SATURDAY", IncludeGeometry: false, WithCursor: WithCursor{Limit: 1000}},
			f: func(t *testing.T, jj string) {
				a := gjson.Get(jj, "stops.0.departures").Array()
				dates := map[string]int{}
				serviceDates := map[string]int{}
				var departures []string
				for _, st := range a {
					dates[st.Get("date").String()] += 1
					serviceDates[st.Get("service_date").String()] += 1
					departures = append(departures, st.Get("departure.scheduled_local").String())
				}
				slices.Sort(departures)
				assert.Equal(t, map[string]int{"2018-06-02": 60}, dates, "dates")
				assert.Equal(t, map[string]int{"2018-06-02": 60}, serviceDates, "service_dates")
				assert.Equal(t, "2018-06-02T17:02:00-07:00", slices.Min(departures), "departure min")
				assert.Equal(t, "2018-06-02T20:49:00-07:00", slices.Max(departures), "departure max")
			},
		},
		{
			name: "service_date 2018-06-06 22:00 to 26:00",
			h:    StopDepartureRequest{StopKey: sid, ServiceDate: "2018-06-05", StartTime: "22:00:00", EndTime: "26:00:00", WithCursor: WithCursor{Limit: 1000}},
			f: func(t *testing.T, jj string) {
				a := gjson.Get(jj, "stops.0.departures").Array()
				dates := map[string]int{}
				serviceDates := map[string]int{}
				var departures []string
				for _, st := range a {
					dates[st.Get("date").String()] += 1
					serviceDates[st.Get("service_date").String()] += 1
					departures = append(departures, st.Get("departure.scheduled_local").String())
				}
				slices.Sort(departures)
				assert.Equal(t, map[string]int{"2018-06-05": 24, "2018-06-06": 9}, dates, "dates")
				assert.Equal(t, map[string]int{"2018-06-05": 33}, serviceDates, "service_dates")
				assert.Equal(t, "2018-06-05T22:02:00-07:00", slices.Min(departures), "departure min")
				assert.Equal(t, "2018-06-06T01:00:00-07:00", slices.Max(departures), "departure max")
			},
		},
		{
			name:         "relative_date=today",
			h:            StopDepartureRequest{StopKey: sid, RelativeDate: "today", StartTime: "10:00:00", EndTime: "10:10:00", UseServiceWindow: bp(true)},
			format:       "",
			selector:     "stops.0.departures.#.date",
			expectSelect: []string{"2018-05-31", "2018-05-31", "2018-05-31", "2018-05-31"},
		},
		{
			name:         "relative_date=next wednesday",
			h:            StopDepartureRequest{StopKey: sid, RelativeDate: "next_wednesday", StartTime: "10:00:00", EndTime: "10:10:00", UseServiceWindow: bp(true)},
			format:       "",
			selector:     "stops.0.departures.#.date",
			expectSelect: []string{"2018-06-06", "2018-06-06", "2018-06-06", "2018-06-06"},
		},
		{
			name:         "use_service_window=true",
			h:            StopDepartureRequest{StopKey: sid, ServiceDate: "2022-05-30", StartTime: "10:00:00", EndTime: "10:10:00", UseServiceWindow: bp(true)},
			format:       "",
			selector:     "stops.0.departures.#.service_date",
			expectSelect: []string{"2018-06-04", "2018-06-04", "2018-06-04", "2018-06-04"},
		},
		{
			name:         "use_service_window=false",
			h:            StopDepartureRequest{StopKey: sid, ServiceDate: "2022-05-30", StartTime: "10:00:00", EndTime: "10:10:00", UseServiceWindow: bp(false)},
			format:       "",
			selector:     "stops.0.departures.#.service_date",
			expectSelect: []string{},
		},
		{
			name:         "use_service_window=false good date",
			h:            StopDepartureRequest{StopKey: sid, ServiceDate: "2018-06-04", StartTime: "10:00:00", EndTime: "10:10:00", UseServiceWindow: bp(false)},
			format:       "",
			selector:     "stops.0.departures.#.service_date",
			expectSelect: []string{"2018-06-04", "2018-06-04", "2018-06-04", "2018-06-04"},
		},
		{
			name:         "selects best service window date",
			h:            StopDepartureRequest{StopKey: sid, ServiceDate: "2022-05-30", StartTime: "10:00:00", EndTime: "10:10:00"},
			format:       "",
			selector:     "stops.0.departures.#.service_date",
			expectSelect: []string{"2018-06-04", "2018-06-04", "2018-06-04", "2018-06-04"},
		},
		{
			name:         "no pagination",
			h:            StopDepartureRequest{StopKey: sid, ServiceDate: "2018-06-04", WithCursor: WithCursor{Limit: 1}},
			format:       "",
			selector:     "meta.next",
			expectSelect: []string{},
		},
		{
			name:         "requires valid stop key",
			h:            StopDepartureRequest{StopKey: "0"},
			format:       "",
			selector:     "stops.0.onestop_id",
			expectSelect: []string{},
		},
		{
			name:         "requires valid stop key 2",
			h:            StopDepartureRequest{StopKey: "-1"},
			format:       "",
			selector:     "stops.0.onestop_id",
			expectSelect: []string{},
		},
		{
			name:         "feed_key",
			h:            StopDepartureRequest{StopKey: "BA:FTVL"},
			format:       "",
			selector:     "stops.0.stop_id",
			expectSelect: []string{"FTVL"},
		},
		//
		{
			name: "include_alerts:true",
			h:    StopDepartureRequest{StopKey: "BA:FTVL", ServiceDate: "2018-05-30", IncludeAlerts: true},
			f: func(t *testing.T, jj string) {
				a := gjson.Get(jj, "stops.0.alerts").Array()
				assert.Equal(t, 2, len(a), "alert count")
			},
		},
		{
			name: "include_alerts:false",
			h:    StopDepartureRequest{StopKey: "BA:FTVL", ServiceDate: "2018-05-30", IncludeAlerts: false},
			f: func(t *testing.T, jj string) {
				a := gjson.Get(jj, "stops.0.alerts").Array()
				assert.Equal(t, 0, len(a), "alert count")
			},
		},
		// TODO
		// {
		// 	"requires valid stop key 3",
		// 	StopDepartureRequest{StopKey: ""},
		// 	"",
		// 	"stops.0.onestop_id",
		// 	[]string{},
		// 	0,
		// },
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			checkTestCase(t, tc)
		})
	}
}
