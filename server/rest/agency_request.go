package rest

import (
	"context"
	_ "embed"
	"strconv"
	"strings"

	oa "github.com/getkin/kin-openapi/openapi3"
)

//go:embed agency_request.gql
var agencyQuery string

// AgencyRequest holds options for an Agency request
type AgencyRequest struct {
	ID              int       `json:"id,string"`
	AgencyKey       string    `json:"agency_key"`
	AgencyID        string    `json:"agency_id"`
	AgencyName      string    `json:"agency_name"`
	OnestopID       string    `json:"onestop_id"`
	FeedVersionSHA1 string    `json:"feed_version_sha1"`
	FeedOnestopID   string    `json:"feed_onestop_id"`
	Search          string    `json:"search"`
	Lon             float64   `json:"lon,string"`
	Lat             float64   `json:"lat,string"`
	Bbox            *restBbox `json:"bbox"`
	Radius          float64   `json:"radius,string"`
	Adm0Name        string    `json:"adm0_name"`
	Adm0Iso         string    `json:"adm0_iso"`
	Adm1Name        string    `json:"adm1_name"`
	Adm1Iso         string    `json:"adm1_iso"`
	CityName        string    `json:"city_name"`
	IncludeAlerts   bool      `json:"include_alerts,string"`
	IncludeRoutes   bool      `json:"include_routes,string"`
	LicenseFilter
	WithCursor
}

func (r AgencyRequest) RequestInfo() RequestInfo {
	return RequestInfo{
		Path: "/agencies",
		Get: RequestOperation{
			Query: agencyQuery,
			Operation: &oa.Operation{
				Summary: `Search for agencies`,
				Extensions: map[string]any{
					"x-alternates": []RequestAltPath{
						{"GET", "/agencies.{format}", "Request agencies in specified format"},
						{"GET", "/agencies/{agency_key}", "Request an agency"},
						{"GET", "/agencies/{agency_key}.format", "Request an agency in a specified format"},
					},
				},
				Parameters: oa.Parameters{
					&pref{Value: &param{
						Name:        "agency_key",
						In:          "query",
						Description: `Agency lookup key; can be an integer ID, a '<feed onestop_id>:<gtfs agency_id>' key, or a Onestop ID`,
						Schema:      newSRVal("string", "", nil),
					}},
					&pref{Value: &param{
						Name:        "agency_id",
						In:          "query",
						Description: `Search for records with this GTFS agency_id (string)`,
						Schema:      newSRVal("string", "", nil),
						Extensions:  newExt("", "agency_id=BART", ""),
					}},
					&pref{Value: &param{
						Name:        "agency_name",
						In:          "query",
						Description: `Search for records with this GTFS agency_name`,
						Schema:      newSRVal("string", "", nil),
						Extensions:  newExt("", "agency_name=Caltrain", ""),
					}},
					newPRef("idParam"),
					newPRef("includeAlertsParam"),
					newPRef("afterParam"),
					newPRefExt("limitParam", "", "limit=1", ""),
					newPRefExt("formatParam", "", "format=geojson", ""),
					newPRefExt("searchParam", "", "search=bart", ""),
					newPRefExt("onestopParam", "", "onestop_id=o-9q9-caltrain", ""),
					newPRefExt("sha1Param", "", "feed_version_sha1=1c4721d4...", "feed_version_sha1=1c4721d4e0c9fae1e81f7c79660696e4280ed05b"),
					newPRefExt("feedParam", "", "feed_onestop_id=f-sf~bay~area~rg", ""),
					newPRefExt("radiusParam", "Search for agencies geographically, based on stops at this location; radius is in meters, requires lon and lat", "lon=-122.3&lat=37.8&radius=1000", ""),
					newPRef("lonParam"),
					newPRef("latParam"),
					newPRefExt("bboxParam", "", "bbox=-122.269,37.807,-122.267,37.808", ""),
					newPRefExt("adm0NameParam", "", "adm0_name=Mexico", ""),
					newPRefExt("adm0IsoParam", "", "adm0_iso=US", ""),
					newPRefExt("adm1NameParam", "", "adm1_name=California", ""),
					newPRefExt("adm1IsoParam", "", "adm1_iso=US-CA", ""),
					newPRefExt("cityNameParam", "", "city_name=Oakland", ""),
					newPRef("licenseCommercialUseAllowedParam"),
					newPRef("licenseShareAlikeOptionalParam"),
					newPRef("licenseCreateDerivedProductParam"),
					newPRef("licenseRedistributionAllowedParam"),
					newPRef("licenseUseWithoutAttributionParam"),
				},
			},
		},
	}
}

// ResponseKey returns the GraphQL response entity key.
func (r AgencyRequest) ResponseKey() string { return "agencies" }

// Query returns a GraphQL query string and variables.
func (r AgencyRequest) Query(ctx context.Context) (string, map[string]interface{}) {
	if r.AgencyKey == "" {
		// pass
	} else if fsid, eid, ok := strings.Cut(r.AgencyKey, ":"); ok {
		r.FeedOnestopID = fsid
		r.AgencyID = eid
		r.IncludeRoutes = true
	} else if v, err := strconv.Atoi(r.AgencyKey); err == nil {
		r.ID = v
		r.IncludeRoutes = true
	} else {
		r.OnestopID = r.AgencyKey
		r.IncludeRoutes = true
	}

	where := hw{}
	if r.FeedVersionSHA1 != "" {
		where["feed_version_sha1"] = r.FeedVersionSHA1
	}
	if r.FeedOnestopID != "" {
		where["feed_onestop_id"] = r.FeedOnestopID
	}
	if r.OnestopID != "" {
		where["onestop_id"] = r.OnestopID
	}
	if r.AgencyID != "" {
		where["agency_id"] = r.AgencyID
	}
	if r.AgencyName != "" {
		where["agency_name"] = r.AgencyName
	}
	if r.Search != "" {
		where["search"] = r.Search
	}
	if r.Lat != 0.0 && r.Lon != 0.0 {
		where["near"] = hw{"lat": r.Lat, "lon": r.Lon, "radius": r.Radius}
	}
	if r.Bbox != nil {
		where["bbox"] = r.Bbox.AsJson()
	}
	if r.Adm0Name != "" {
		where["adm0_name"] = r.Adm0Name
	}
	if r.Adm1Name != "" {
		where["adm1_name"] = r.Adm1Name
	}
	if r.Adm0Iso != "" {
		where["adm0_iso"] = r.Adm0Iso
	}
	if r.Adm1Iso != "" {
		where["adm1_iso"] = r.Adm1Iso
	}
	if r.CityName != "" {
		where["city_name"] = r.CityName
	}
	where["license"] = checkLicenseFilter(r.LicenseFilter)
	return agencyQuery, hw{
		"limit":          r.CheckLimit(),
		"after":          r.CheckAfter(),
		"ids":            checkIds(r.ID),
		"include_alerts": r.IncludeAlerts,
		"include_routes": r.IncludeRoutes,
		"where":          where,
	}
}

///////////////

type AgencyKeyRequest struct {
	AgencyRequest
}

func (r AgencyKeyRequest) RequestInfo() RequestInfo {
	return RequestInfo{
		Path: "/agencies/{agency_key}",
		Get: RequestOperation{
			Query: agencyQuery,
			Operation: &oa.Operation{
				Summary: "Agencies",
				Parameters: oa.Parameters{
					&pref{Value: &param{
						Name:        "agency_key",
						In:          "path",
						Description: `Agency lookup key; can be an integer ID, a '<feed onestop_id>:<gtfs agency_id>' key, or a Onestop ID`,
						Schema:      newSRVal("string", "", nil),
					}},
					newPRef("includeAlertsParam"),
					newPRefExt("limitParam", "", "limit=1", ""),
					newPRefExt("formatParam", "", "format=geojson", ""),
				},
			},
		},
	}
}
