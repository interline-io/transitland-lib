package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"

	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/lib/pq"
)

type Feed struct {
	SearchRank *string
	tl.Feed
}

// OnestopID is called FeedID in transitland-lib.
func (f *Feed) OnestopID() (string, error) {
	return f.FeedID, nil
}

type FeedLicense struct {
	tl.FeedLicense
}

type FeedUrls struct {
	tl.FeedUrls
}

type FeedAuthorization struct {
	tl.FeedAuthorization
}
type Agency struct {
	OnestopID       string `json:"onestop_id"`
	FeedOnestopID   string `json:"feed_onestop_id"`
	FeedVersionSHA1 string `json:"feed_version_sha1"`
	SearchRank      *string
	tl.Agency
}

type Calendar struct {
	tl.Calendar
}

type FeedState struct {
	dmfr.FeedState
}

type FeedVersion struct {
	tl.FeedVersion
}

type Operator struct {
	ID                      int
	AgencyID                *int             `json:"agency_id"`
	AgencyName              *string          `json:"agency_name"`
	AgencyOnestopID         *string          `json:"agency_onestop_id"`
	FeedID                  *int             `json:"feed_id"`
	FeedVersionID           *int             `json:"feed_version_id"`
	FeedVersionSha1         *string          `json:"feed_version_sha1"`
	FeedOnestopID           *string          `json:"feed_onestop_id"`
	FeedNamespaceID         *string          `json:"feed_namespace_id"`
	CityName                *string          `json:"city_name"`
	Adm1name                *string          `json:"adm1name"`
	Adm0name                *string          `json:"adm0name"`
	OnestopID               *string          `json:"onestop_id"`
	OperatorID              *int             `json:"operator_id"`
	OperatorOnestopID       *string          `json:"operator_onestop_id"`
	OperatorName            *string          `json:"operator_name"`
	OperatorShortName       *string          `json:"operator_short_name"`
	OperatorTags            *json.RawMessage `json:"operator_tags"` // json map[string]string
	OperatorAssociatedFeeds *json.RawMessage `json:"operator_associated_feeds"`
	PlacesCache             *pq.StringArray  `json:"places_cache"`
	SearchRank              *string
}

type Route struct {
	FeedOnestopID                string
	FeedVersionSHA1              string
	OnestopID                    *string
	HeadwaySecondsWeekdayMorning *int
	SearchRank                   *string
	Geometry                     tl.Geometry // is not read from database by default
	tl.Route
}

type Trip struct {
	tl.Trip
}

type StopTime struct {
	tl.StopTime
}

type Stop struct {
	FeedOnestopID   string
	FeedVersionSHA1 string
	OnestopID       *string
	SearchRank      *string
	tl.Stop
}

type Frequency struct {
	tl.Frequency
}

type CalendarDate struct {
	tl.CalendarDate
}

type Shape struct {
	tl.Shape
}

type Level struct {
	tl.Level
}

type FeedInfo struct {
	tl.FeedInfo
}

type Pathway struct {
	tl.Pathway
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

// Support models that don't exist in transitland-lib

type RouteStop struct {
	ID       int `json:"id"`
	RouteID  int
	StopID   int
	AgencyID int
}

type RouteHeadway struct {
	ID                           int      `json:"id"`
	RouteID                      int      `json:"route_id"`
	SelectedStopID               int      `json:"selected_stop_id"`
	DirectionID                  int      `json:"direction_id"`
	HeadwaySecs                  *int     `json:"headway_secs"`
	DowCategory                  *int     `json:"dow_category"`
	ServiceDate                  tl.ODate `json:"service_date"`
	ServiceSeconds               *int     `json:"service_seconds"`
	StopTripCount                *int     `json:"stop_trip_count"`
	HeadwaySecondsMorningCount   *int     `json:"headway_seconds_morning_count"`
	HeadwaySecondsMorningMin     *int     `json:"headway_seconds_morning_min"`
	HeadwaySecondsMorningMid     *int     `json:"headway_seconds_morning_mid"`
	HeadwaySecondsMorningMax     *int     `json:"headway_seconds_morning_max"`
	HeadwaySecondsMiddayCount    *int     `json:"headway_seconds_midday_count"`
	HeadwaySecondsMiddayMin      *int     `json:"headway_seconds_midday_min"`
	HeadwaySecondsMiddayMid      *int     `json:"headway_seconds_midday_mid"`
	HeadwaySecondsMiddayMax      *int     `json:"headway_seconds_midday_max"`
	HeadwaySecondsAfternoonCount *int     `json:"headway_seconds_afternoon_count"`
	HeadwaySecondsAfternoonMin   *int     `json:"headway_seconds_afternoon_min"`
	HeadwaySecondsAfternoonMid   *int     `json:"headway_seconds_afternoon_mid"`
	HeadwaySecondsAfternoonMax   *int     `json:"headway_seconds_afternoon_max"`
	HeadwaySecondsNightCount     *int     `json:"headway_seconds_night_count"`
	HeadwaySecondsNightMin       *int     `json:"headway_seconds_night_min"`
	HeadwaySecondsNightMid       *int     `json:"headway_seconds_night_mid"`
	HeadwaySecondsNightMax       *int     `json:"headway_seconds_night_max"`
}

type RouteStopBuffer struct {
	StopPoints     *tl.Geometry `json:"stop_points"`
	StopBuffer     *tl.Geometry `json:"stop_buffer"`
	StopConvexhull *tl.Polygon  `json:"stop_convexhull"`
}

type RouteGeometry struct {
	RouteID     int           `json:"route_id"`
	DirectionID int           `json:"direction_id"`
	Generated   bool          `json:"generated"`
	Geometry    tl.LineString `json:"geometry"`
}

type AgencyPlace struct {
	AgencyID int      `json:"agency_id"`
	Name     *string  `json:"name"`
	Adm0name *string  `json:"adm0name"`
	Adm1name *string  `json:"adm1name"`
	Rank     *float64 `json:"rank"`
}

// Census models

type CensusGeography struct {
	ID            int         `json:"id"`
	LayerName     string      `json:"layer_name"`
	Geoid         *string     `json:"geoid"`
	Name          *string     `json:"name"`
	Aland         *float64    `json:"aland"`
	Awater        *float64    `json:"awater"`
	Geometry      *tl.Polygon `json:"geometry"`
	MatchEntityID int         // for matching to a stop, route, agency in query
}

type CensusTable struct {
	ID         int
	TableName  string
	TableTitle string
	TableGroup string
}

type CensusValue struct {
	GeographyID int
	TableID     int
	TableValues ValueMap
}

// ValueMap is just a JSONB map[string]interface{}
type ValueMap map[string]interface{}

// Value dump
func (a ValueMap) Value() (driver.Value, error) {
	return json.Marshal(a)
}

// Scan load
func (a *ValueMap) Scan(value interface{}) error {
	b, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}
	return json.Unmarshal(b, &a)
}

///////////////// Validation

// ValidationResult .
type ValidationResult struct {
	Success              bool                         `json:"success"`
	FailureReason        string                       `json:"failure_reason"`
	Errors               []ValidationResultErrorGroup `json:"errors"`
	Warnings             []ValidationResultErrorGroup `json:"warnings"`
	Sha1                 string                       `json:"sha1"`
	EarliestCalendarDate tl.ODate                     `json:"earliest_calendar_date"`
	LatestCalendarDate   tl.ODate                     `json:"latest_calendar_date"`
	Files                []FeedVersionFileInfo        `json:"files"`
	ServiceLevels        []FeedVersionServiceLevel    `json:"service_levels"`
	Agencies             []Agency                     `json:"agencies"`
	Routes               []Route                      `json:"routes"`
	Stops                []Stop                       `json:"stops"`
	FeedInfos            []FeedInfo                   `json:"feed_infos"`
}

type ValidationResultError struct {
	Filename  string `json:"filename"`
	ErrorType string `json:"error_type"`
	EntityID  string `json:"entity_id"`
	Field     string `json:"field"`
	Value     string `json:"value"`
	Message   string `json:"message"`
}

type ValidationResultErrorGroup struct {
	Filename  string                   `json:"filename"`
	ErrorType string                   `json:"error_type"`
	Count     int                      `json:"count"`
	Limit     int                      `json:"limit"`
	Errors    []*ValidationResultError `json:"errors"`
}

///////////////////// Fetch

type FeedVersionFetchResult struct {
	FeedVersion  *FeedVersion
	FetchError   *string
	FoundSHA1    bool
	FoundDirSHA1 bool
}

///////////////////// Import

type FeedVersionImportResult struct {
	Success bool
}
