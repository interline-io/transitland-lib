package rest

import (
	"context"
	_ "embed"
	"strconv"

	oa "github.com/getkin/kin-openapi/openapi3"
)

//go:embed feed_version_request.gql
var feedVersionQuery string

// FeedVersionRequest holds options for a Feed Version request
type FeedVersionRequest struct {
	FeedVersionKey  string    `json:"feed_version_key"`
	FeedKey         string    `json:"feed_key"`
	ID              int       `json:"id,string"`
	FeedID          int       `json:"feed_id,string"`
	FeedOnestopID   string    `json:"feed_onestop_id"`
	Sha1            string    `json:"sha1"`
	FetchedBefore   string    `json:"fetched_before"`
	FetchedAfter    string    `json:"fetched_after"`
	CoversStartDate string    `json:"covers_start_date"`
	CoversEndDate   string    `json:"covers_end_date"`
	Lon             float64   `json:"lon,string"`
	Lat             float64   `json:"lat,string"`
	Radius          float64   `json:"radius,string"`
	Bbox            *restBbox `json:"bbox"`
	WithCursor
}

func (r FeedVersionRequest) RequestInfo() RequestInfo {
	return RequestInfo{
		Path: "/feed_versions",
		Get: &RequestOperation{
			Query: feedVersionQuery,
			Operation: &oa.Operation{
				Summary: `Search for feed versions`,
				Extensions: map[string]any{
					"x-alternates": []RequestAltPath{
						{"GET", "/feed_versions.{format}", "Request feed versions in specified format"},
						{"GET", "/feed_versions/{feed_version_key}", "Request a feed version by ID or SHA1"},
						{"GET", "/feed_versions/{feed_version_key}.{format}", "Request a feed version by ID or SHA1 in specified format"},
						{"GET", "/feeds/{feed_key}/feed_versions", "Request feed versions by feed ID or Onestop ID"},
					},
				},
				Parameters: oa.Parameters{
					&pref{Value: &param{
						Name:        "feed_version_key",
						In:          "query",
						Description: `Feed version lookup key; can be an integer ID or a SHA1 value`,
						Schema:      newSRVal("string", "", nil),
					}},
					&pref{Value: &param{
						Name:        "feed_key",
						In:          "query",
						Description: `Feed lookup key; can be an integer ID or Onestop ID`,
						Schema:      newSRVal("string", "", nil),
					}},
					&pref{Value: &param{
						Name:        "sha1",
						In:          "query",
						Description: `Feed version SHA1`,
						Schema:      newSRVal("string", "", nil),
						Extensions:  newExt("", "sha1=e535eb2b3...", "sha1=dd7aca4a8e4c90908fd3603c097fabee75fea907"),
					}},
					&pref{Value: &param{
						Name:        "feed_onestop_id",
						In:          "query",
						Description: `Feed OnestopID`,
						Schema:      newSRVal("string", "", nil),
						Extensions:  newExt("", "feed_onestop_id=f-sf~bay~area~rg", "feed_onestop_id=f-sf~bay~area~rg"),
					}},
					&pref{Value: &param{
						Name:        "fetched_before",
						In:          "query",
						Description: `Filter for feed versions fetched earlier than given date time in UTC`,
						Schema:      newSRVal("string", "datetime", nil),
						Extensions:  newExt("", "fetched_before=2023-01-01T00:00:00Z", "fetched_before=2023-01-01T00:00:00Z"),
					}},
					&pref{Value: &param{
						Name:        "fetched_after",
						In:          "query",
						Description: `Filter for feed versions fetched since given date time in UTC`,
						Schema:      newSRVal("string", "datetime", nil),
						Extensions:  newExt("", "fetched_after=2023-01-01T00:00:00Z", "fetched_after=2023-01-01T00:00:00Z"),
					}},
					newPRef("idParam"),
					newPRef("afterParam"),
					newPRefExt("limitParam", "", "limit=1", "limit=1"),
					newPRefExt("formatParam", "", "format=geojson", "format=geojson"),
					newPRefExt("radiusParam", "Search for feed versions geographically; radius is in meters, requires lon and lat", "lon=-122.3&lat=37.8&radius=1000", ""),
					newPRef("lonParam"),
					newPRef("latParam"),
					newPRefExt("bboxParam", "", "bbox=-122.269,37.807,-122.267,37.808", ""),
				},
			},
		},
	}
}

// Query returns a GraphQL query string and variables.
func (r FeedVersionRequest) Query(ctx context.Context) (string, map[string]interface{}) {
	// Handle feed key
	if r.FeedKey == "" {
		// pass
	} else if v, err := strconv.Atoi(r.FeedKey); err == nil {
		r.FeedID = v
	} else {
		r.FeedOnestopID = r.FeedKey
	}
	// Handle feed version key
	if r.FeedVersionKey == "" {
		// pass
	} else if v, err := strconv.Atoi(r.FeedVersionKey); err == nil {
		r.ID = v
	} else {
		r.Sha1 = r.FeedVersionKey
	}
	where := hw{}
	if r.FeedID > 0 {
		where["feed_ids"] = []int{r.FeedID}
	}
	if r.FeedOnestopID != "" {
		where["feed_onestop_id"] = r.FeedOnestopID
	}
	if r.Sha1 != "" {
		where["sha1"] = r.Sha1
	}
	if r.Lat != 0.0 && r.Lon != 0.0 {
		where["near"] = hw{"lat": r.Lat, "lon": r.Lon, "radius": r.Radius}
	}
	if r.Bbox != nil {
		where["bbox"] = r.Bbox.AsJson()
	}
	whereCovers := hw{}
	if r.CoversStartDate != "" {
		whereCovers["start_date"] = r.CoversStartDate
	}
	if r.CoversEndDate != "" {
		whereCovers["end_date"] = r.CoversEndDate
	}
	if r.FetchedAfter != "" {
		whereCovers["fetched_after"] = r.FetchedAfter
	}
	if r.FetchedBefore != "" {
		whereCovers["fetched_before"] = r.FetchedBefore
	}
	if len(whereCovers) > 0 {
		where["covers"] = whereCovers
	}
	return feedVersionQuery, hw{"limit": r.CheckLimit(), "after": r.CheckAfter(), "ids": checkIds(r.ID), "where": where}
}

// ResponseKey .
func (r FeedVersionRequest) ResponseKey() string {
	return "feed_versions"
}

///////////

// Currently this exists only for OpenAPI documentation
type FeedVersionDownloadRequest struct {
}

type FeedVersionKeyRequest struct {
	FeedVersionRequest
}

func (r FeedVersionKeyRequest) RequestInfo() RequestInfo {
	return RequestInfo{
		Path: "/feed_versions/{feed_version_key}",
		Get: &RequestOperation{
			Query: feedVersionQuery,
			Operation: &oa.Operation{
				Summary: "Feed Versions",
				Parameters: oa.Parameters{
					&pref{Value: &param{
						Name:        "feed_version_key",
						In:          "path",
						Required:    true,
						Description: `Feed version lookup key; can be an integer ID or a SHA1 value`,
						Schema:      newSRVal("string", "", nil),
						Extensions:  newExt("", "dd7aca4a8e4c90908fd3603c097fabee75fea907", "dd7aca4a8e4c90908fd3603c097fabee75fea907"),
					}},
					newPRefExt("limitParam", "", "limit=1", ""),
					newPRefExt("formatParam", "", "format=geojson", ""),
				},
			},
		},
	}
}

func (r FeedVersionDownloadRequest) RequestInfo() RequestInfo {
	return RequestInfo{
		Path:        "/feed_versions/{feed_version_key}/download",
		Description: `Download this feed version GTFS zip for this feed, if redistribution is allowed by the source feed's license. Available only using Transitland professional or enterprise plan API keys.`,
		Get: &RequestOperation{
			Operation: &oa.Operation{
				Summary: "Download feed version",
				Extensions: map[string]any{
					"x-required-role": "tl_download_fv_historic",
				},
				Parameters: oa.Parameters{
					&pref{Value: &param{
						Name:        "feed_version_key",
						In:          "path",
						Required:    true,
						Description: `Feed version lookup key; can be an integer ID or a SHA1 value`,
						Schema:      newSRVal("string", "", nil),
					}},
				},
				Responses: oa.NewResponses(
					oa.WithStatus(200, &oa.ResponseRef{
						Value: &oa.Response{
							Description: toPtr("Success"),
							Content: oa.Content{
								"application/octet-stream": &oa.MediaType{
									Schema: newSRVal("string", "binary", nil),
								},
							},
						},
					}),
					oa.WithStatus(401, &oa.ResponseRef{
						Value: &oa.Response{
							Description: toPtr("Not authorized - feed redistribution not allowed"),
						},
					}),
					oa.WithStatus(404, &oa.ResponseRef{
						Value: &oa.Response{
							Description: toPtr("Not found - feed not found"),
						},
					}),
				),
			},
		},
	}
}

// Query returns a GraphQL query string and variables.
func (r FeedVersionDownloadRequest) Query(ctx context.Context) (string, map[string]interface{}) {
	return "", nil
}
