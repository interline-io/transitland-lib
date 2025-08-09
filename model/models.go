package model

import (
	"encoding/json"
	"time"

	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/rt/pb"
	"github.com/interline-io/transitland-lib/service"
	"github.com/interline-io/transitland-lib/tlxy"
	"github.com/interline-io/transitland-lib/tt"
)

type ServiceWindow struct {
	NowLocal     time.Time
	StartDate    time.Time
	EndDate      time.Time
	FallbackWeek time.Time
}

type StopPlaceParam struct {
	ID    int
	Point tlxy.Point
}

//////////

type Feed struct {
	WithOperatorOnestopID tt.String
	SearchRank            *string
	dmfr.Feed
}

type FeedLicense struct {
	dmfr.FeedLicense
}

type FeedUrls struct {
	dmfr.FeedUrls
}

type FeedAuthorization struct {
	dmfr.FeedAuthorization
}

type StopExternalReference struct {
	dmfr.StopExternalReference
}

type Agency struct {
	OnestopID       string      `json:"onestop_id"`
	FeedOnestopID   string      `json:"feed_onestop_id"`
	FeedVersionSHA1 string      `json:"feed_version_sha1"`
	Geometry        *tt.Polygon `json:"geometry"`
	SearchRank      *string
	CoifID          *int
	gtfs.Agency
}

type Calendar struct {
	gtfs.Calendar
}

type FeedState struct {
	dmfr.FeedState
}

type FeedFetch struct {
	ResponseSha1 tt.String // confusing but easier than alternative fixes
	dmfr.FeedFetch
}

type FeedVersion struct {
	SHA1Dir tt.String `json:"sha1_dir"`
	dmfr.FeedVersion
}

type Operator struct {
	ID            int
	Generated     bool
	FeedID        int
	FeedOnestopID *string
	SearchRank    *string // internal
	AgencyID      int     // internal
	dmfr.Operator
}

type Route struct {
	FeedOnestopID                string
	FeedVersionSHA1              string
	OnestopID                    *string
	HeadwaySecondsWeekdayMorning *int
	SearchRank                   *string
	gtfs.Route
}

type Trip struct {
	RTTripID string // internal: for ADDED trips
	gtfs.Trip
}

type RTStopTimeUpdate struct {
	LastDelay      *int32
	StopTimeUpdate *pb.TripUpdate_StopTimeUpdate
	TripUpdate     *pb.TripUpdate
}

type StopTime struct {
	ServiceDate      tt.Date
	Date             tt.Date
	RTTripID         string            // internal: for ADDED trips
	RTStopTimeUpdate *RTStopTimeUpdate // internal
	gtfs.StopTime
}

type Stop struct {
	FeedOnestopID   string
	FeedVersionSHA1 string
	OnestopID       *string
	SearchRank      *string
	WithinFeatures  tt.Strings
	WithRouteID     tt.Int
	gtfs.Stop
}

type Frequency struct {
	gtfs.Frequency
}

type CalendarDate struct {
	gtfs.CalendarDate
}

type Shape struct {
	service.ShapeLine
}

type Level struct {
	Geometry      tt.Polygon
	ParentStation tt.Key
	gtfs.Level
}

type FeedInfo struct {
	gtfs.FeedInfo
}

type Pathway struct {
	gtfs.Pathway
}

type FeedVersionFileInfo struct {
	dmfr.FeedVersionFileInfo
}

type FeedVersionGtfsImport struct {
	WarningCount             *json.RawMessage `json:"warning_count"`
	EntityCount              *json.RawMessage `json:"entity_count"`
	SkipEntityErrorCount     *json.RawMessage `json:"skip_entity_error_count"`
	SkipEntityReferenceCount *json.RawMessage `json:"skip_entity_reference_count"`
	SkipEntityFilterCount    *json.RawMessage `json:"skip_entity_filter_count"`
	SkipEntityMarkedCount    *json.RawMessage `json:"skip_entity_marked_count"`
	dmfr.FeedVersionImport
}

type FeedVersionServiceLevel struct {
	dmfr.FeedVersionServiceLevel
}

// Some enum helpers

var specTypeMap = map[string]FeedSpecTypes{
	"gtfs":    FeedSpecTypesGtfs,
	"gtfs-rt": FeedSpecTypesGtfsRt,
	"gbfs":    FeedSpecTypesGbfs,
	"mds":     FeedSpecTypesMds,
}

func (f FeedSpecTypes) ToDBString() string {
	for k, v := range specTypeMap {
		if f == v {
			return k
		}
	}
	return ""
}

func (f FeedSpecTypes) FromDBString(s string) *FeedSpecTypes {
	a, ok := specTypeMap[s]
	if !ok {
		return nil
	}
	return &a
}
