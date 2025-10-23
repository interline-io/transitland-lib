package rest

import (
	"context"
	_ "embed"
	"strconv"

	oa "github.com/getkin/kin-openapi/openapi3"
)

//go:embed operator_request.gql
var operatorQuery string

// OperatorRequest holds options for an Operator request
type OperatorRequest struct {
	OperatorKey   string    `json:"operator_key"`
	ID            int       `json:"id,string"`
	OnestopID     string    `json:"onestop_id"`
	FeedOnestopID string    `json:"feed_onestop_id"`
	Search        string    `json:"search"`
	TagKey        string    `json:"tag_key"`
	TagValue      string    `json:"tag_value"`
	Lon           float64   `json:"lon,string"`
	Lat           float64   `json:"lat,string"`
	Bbox          *restBbox `json:"bbox"`
	Radius        float64   `json:"radius,string"`
	Adm0Name      string    `json:"adm0_name"`
	Adm0Iso       string    `json:"adm0_iso"`
	Adm1Name      string    `json:"adm1_name"`
	Adm1Iso       string    `json:"adm1_iso"`
	CityName      string    `json:"city_name"`
	IncludeAlerts bool      `json:"include_alerts,string"`
	LicenseFilter
	WithCursor
}

func (r OperatorRequest) RequestInfo() RequestInfo {
	return RequestInfo{
		Path: "/operators",
		Get: &RequestOperation{
			Query: operatorQuery,
			Operation: &oa.Operation{
				Summary: `Search for operators`,
				Extensions: map[string]any{
					"x-alternates": []RequestAltPath{
						{"GET", "/operators.{format}", "Request operators in specified format"},
						{"GET", "/operators/{onestop_id}", "Request an operator by Onestop ID"},
						{"GET", "/operators/{onestop_id}.format", "Request an operator by Onestop ID in specified format"},
					},
				},
				Parameters: oa.Parameters{
					&pref{Value: &param{
						Name:        "tag_key",
						In:          "query",
						Description: `Search for operators with a tag. Combine with tag_value also query for the value of the tag.`,
						Schema:      newSRVal("string", "", nil),
						Extensions:  newExt("", "tag_key=us_ntd_id", ""),
					}},
					&pref{Value: &param{
						Name:        "tag_value",
						In:          "query",
						Description: `Search for feeds tagged with a given value. Must be combined with tag_key.`,
						Schema:      newSRVal("string", "", nil),
						Extensions:  newExt("", "tag_key=us_ntd_id&tag_value=40029", ""),
					}},
					newPRefExt("onestopParam", "", "onestop_id=o-9q9-caltrain", ""),
					newPRefExt("feedParam", "", "feed_onestop_id=f-sf~bay~area~rg", ""),
					newPRefExt("searchParam", "", "search=bart", ""),
					newPRef("includeAlertsParam"),
					newPRef("idParam"),
					newPRef("afterParam"),
					newPRef("limitParam"),
					newPRefExt("adm0NameParam", "", "adm0_name=Mexico", ""),
					newPRefExt("adm0IsoParam", "", "adm0_iso=US", ""),
					newPRefExt("adm1NameParam", "", "adm1_name=California", ""),
					newPRefExt("adm1IsoParam", "", "adm1_iso=US-CA", ""),
					newPRefExt("cityNameParam", "", "city_name=Oakland", ""),
					newPRefExt("radiusParam", "Search for operators geographically, based on stops at this location; radius is in meters, requires lon and lat", "lon=-122.3&lat=37.8&radius=1000", ""),
					newPRef("lonParam"),
					newPRef("latParam"),
					newPRefExt("bboxParam", "", "bbox=-122.269,37.807,-122.267,37.808", ""),
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

type OperatorKeyRequest struct {
	OperatorRequest
}

func (r OperatorKeyRequest) RequestInfo() RequestInfo {
	return RequestInfo{
		Path: "/operators/{operator_key}",
		Get: &RequestOperation{
			Query: operatorQuery,
			Operation: &oa.Operation{
				Summary: "Operators",
				Parameters: oa.Parameters{
					&pref{Value: &param{
						Name:        "operator_key",
						In:          "path",
						Required:    true,
						Description: `Operator lookup key; can be a Onestop ID or an internal database integer ID `,
						Schema:      newSRVal("string", "", nil),
						Extensions:  newExt("", "o-9q9-caltrain", "o-9q9-caltrain"),
					}},
					newPRef("includeAlertsParam"),
					newPRefExt("limitParam", "", "limit=1", ""),
					newPRefExt("formatParam", "", "format=geojson", ""),
				},
			},
		},
	}
}

// ResponseKey returns the GraphQL response entity key.
func (r OperatorRequest) ResponseKey() string { return "operators" }

// Query returns a GraphQL query string and variables.
func (r OperatorRequest) Query(ctx context.Context) (string, map[string]interface{}) {
	if r.OperatorKey == "" {
		// pass
	} else if v, err := strconv.Atoi(r.OperatorKey); err == nil {
		r.ID = v
	} else {
		r.OnestopID = r.OperatorKey
	}
	where := hw{}
	where["merged"] = true
	if r.FeedOnestopID != "" {
		where["feed_onestop_id"] = r.FeedOnestopID
	}
	if r.OnestopID != "" {
		where["onestop_id"] = r.OnestopID
	}
	if r.Search != "" {
		where["search"] = r.Search
	}
	if r.TagKey != "" {
		where["tags"] = hw{r.TagKey: r.TagValue}
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
	return operatorQuery, hw{
		"limit":          r.CheckLimit(),
		"after":          r.CheckAfter(),
		"ids":            checkIds(r.ID),
		"include_alerts": r.IncludeAlerts,
		"where":          where,
	}
}
