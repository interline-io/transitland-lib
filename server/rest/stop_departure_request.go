package rest

import (
	"context"
	_ "embed"
	"strconv"
	"strings"

	oa "github.com/getkin/kin-openapi/openapi3"
)

//go:embed stop_departure_request.gql
var stopDepartureQuery string

// StopDepartureRequest holds options for a /stops/_/departures request
type StopDepartureRequest struct {
	StopKey          string `json:"stop_key"`
	ID               int    `json:"id,string"`
	StopID           string `json:"stop_id"`
	FeedOnestopID    string `json:"feed_onestop_id"`
	OnestopID        string `json:"onestop_id"`
	Next             int    `json:"next,string"`
	ServiceDate      string `json:"service_date"`
	Date             string `json:"date"`
	RelativeDate     string `json:"relative_date"`
	StartTime        string `json:"start_time"`
	EndTime          string `json:"end_time"`
	IncludeGeometry  bool   `json:"include_geometry,string"`
	IncludeAlerts    bool   `json:"include_alerts,string"`
	UseServiceWindow *bool  `json:"use_service_window,string"`
	WithCursor
}

func (r StopDepartureRequest) RequestInfo() RequestInfo {
	return RequestInfo{
		Path: "/stops/{stop_key}/departures",
		Get: RequestOperation{
			Query: stopDepartureQuery,
			Operation: &oa.Operation{
				Summary: `Departures from a given stop based on static and real-time data`,
				Extensions: map[string]any{
					"x-alternates": []RequestAltPath{},
				},
				Parameters: oa.Parameters{
					&pref{Value: &param{
						Name:        "stop_key",
						In:          "path",
						Description: `Stop lookup key; can be an integer ID, a '<feed onestop_id>:<gtfs stop_id'> key, a Onestop ID`,
						Required:    true,
						Schema:      newSRVal("string", "", nil),
						Extensions:  newExt("", "f-sf~bay~area~rg:LAKE", "/stops/f-sf~bay~area~rg:LAKE/departures"),
					}},
					newPRefExt("limitParam", "", "limit=1", "/stops/f-sf~bay~area~rg:LAKE/departures?limit=1"),
					&pref{Value: &param{
						Name:        "service_date",
						In:          "query",
						Description: `Search for departures on a specified GTFS service calendar date, in YYYY-MM-DD format`,
						Schema:      newSRVal("string", "date", nil),
						Extensions:  newExt("", "service_date=2022-09-28", "/stops/f-sf~bay~area~rg:LAKE/departures?service_date=2022-09-28"),
					}},
					&pref{Value: &param{
						Name:        "date",
						In:          "query",
						Description: `Search for departures on a specified calendar date, in YYYY-MM-DD format`,
						Schema:      newSRVal("string", "date", nil),
						Extensions:  newExt("", "date=2022-09-28", "/stops/f-sf~bay~area~rg:LAKE/departures?date=2022-09-28"),
					}},
					&pref{Value: &param{
						Name:        "next",
						In:          "query",
						Description: `Search for departures leaving within the next specified number of seconds in local time`,
						Schema:      newSRVal("integer", "", nil),
						Extensions:  newExt("", "next=600", "/stops/f-sf~bay~area~rg:LAKE/departures?next=600"),
					}},
					&pref{Value: &param{
						Name:        "start_time",
						In:          "query",
						Description: `Search for departures leaving after a specified local time, in HH:MM:SS format`,
						Schema:      newSRVal("string", "", nil),
						Extensions:  newExt("", "start_time=10:00:00", "/stops/f-sf~bay~area~rg:LAKE/departures?start_time=10:00:00&service_date=2022-09-28"),
					}},
					&pref{Value: &param{
						Name:        "end_time",
						In:          "query",
						Description: `Search for departures leaving before a specified local time, in HH:MM:SS format`,
						Schema:      newSRVal("string", "", nil),
						Extensions:  newExt("", "end_time=11:00:00", "/stops/f-sf~bay~area~rg:LAKE/departures?end_time=11:00:00&service_date=2022-09-28"),
					}},
					&pref{Value: &param{
						Name:        "include_geometry",
						In:          "query",
						Description: `Include route geometry`,
						Schema:      newSRVal("string", "", []any{"true", "false"}),
						Extensions:  newExt("", "include_geometry=true", "/stops/f-sf~bay~area~rg:LAKE/departures?include_geometry=true"),
					}},
					&pref{Value: &param{
						Name:        "use_service_window",
						In:          "query",
						Description: `Use a fall-back service date if the requested service_date is outside the active service period of the feed version. The fall-back date is selected as the matching day-of-week in the week which provides the best level of scheduled service in the feed version. This value defaults to true.`,
						Schema:      newSRVal("string", "", []any{"true", "false"}),
						Extensions:  newExt("", "use_service_window=false", "/stops/f-sf~bay~area~rg:LAKE/departures?use_service_window=false"),
					}},
					newPRef("idParam"),
					newPRefExt("relativeDateParam", "", "relative_date=NEXT_MONDAY", "/stops/f-sf~bay~area~rg:LAKE/departures?relative_date=NEXT_MONDAY"),
					newPRef("includeAlertsParam"),
					newPRef("afterParam"),
				},
			},
		},
	}
}

// ResponseKey returns the GraphQL response entity key.
func (r StopDepartureRequest) ResponseKey() string { return "stops" }

// IncludeNext
func (r StopDepartureRequest) IncludeNext() bool { return false }

// Query returns a GraphQL query string and variables.
func (r StopDepartureRequest) Query(ctx context.Context) (string, map[string]interface{}) {
	if r.StopKey == "" {
		// TODO: add a way to reject request as invalid
	} else if fsid, eid, ok := strings.Cut(r.StopKey, ":"); ok {
		r.FeedOnestopID = fsid
		r.StopID = eid
	} else if v, err := strconv.Atoi(r.StopKey); err == nil && v > 0 {
		// require an actual ID, not just 0
		r.ID = v
	} else {
		r.OnestopID = r.StopKey
	}
	where := hw{}
	if r.OnestopID != "" {
		where["onestop_id"] = r.OnestopID
	}
	if r.FeedOnestopID != "" {
		where["feed_onestop_id"] = r.FeedOnestopID
	}
	if r.StopID != "" {
		where["stop_id"] = r.StopID
	}
	//
	stwhere := hw{}
	if r.UseServiceWindow == nil || *r.UseServiceWindow {
		stwhere["use_service_window"] = true
	}
	// Restore previous default behavior
	// If no date is specified, use next hour
	if r.Date != "" {
		stwhere["date"] = r.Date
	} else if r.RelativeDate != "" {
		stwhere["relative_date"] = strings.ToUpper(r.RelativeDate)
	} else if r.ServiceDate != "" {
		stwhere["service_date"] = r.ServiceDate
	} else if r.Next == 0 && r.StartTime == "" && r.EndTime == "" {
		r.Next = 3600
	}
	// Use StartTime/EndTime OR Next
	if r.StartTime != "" || r.EndTime != "" {
		if r.StartTime != "" {
			stwhere["start"] = r.StartTime
		}
		if r.EndTime != "" {
			stwhere["end"] = r.EndTime
		}
	} else if r.Next > 0 {
		stwhere["next"] = r.Next
	}
	return stopDepartureQuery, hw{
		"include_geometry": r.IncludeGeometry,
		"include_alerts":   r.IncludeAlerts,
		"limit":            r.CheckLimit(),
		"ids":              checkIds(r.ID),
		"where":            where,
		"stop_time_where":  stwhere,
	}
}
