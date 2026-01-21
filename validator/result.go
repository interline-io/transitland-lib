package validator

import (
	"fmt"
	"time"

	tl "github.com/interline-io/transitland-lib"
	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/rt"
	"github.com/interline-io/transitland-lib/stats"
	"github.com/interline-io/transitland-lib/tt"
)

// Result contains a validation report result,
type Result struct {
	Validator               tt.String                              `json:"validator"`
	ValidatorVersion        tt.String                              `json:"validator_version"`
	Success                 tt.Bool                                `json:"success"`
	FailureReason           tt.String                              `json:"failure_reason"`
	ReportedAt              tt.Time                                `json:"reported_at"`
	ReportedAtLocal         tt.Time                                `json:"reported_at_local"`
	ReportedAtLocalTimezone tt.String                              `json:"reported_at_local_timezone"`
	File                    tt.String                              `json:"file"`
	IncludesStatic          tt.Bool                                `json:"includes_static"`
	IncludesRT              tt.Bool                                `json:"includes_rt"`
	Details                 ResultDetails                          `json:"details" db:"-"`
	Errors                  map[string]*ValidationReportErrorGroup `json:"errors" db:"-"`
	Warnings                map[string]*ValidationReportErrorGroup `json:"warnings" db:"-"`
	tt.BaseEntity
}

func (e *Result) TableName() string {
	return "tl_validation_reports"
}

func NewResult(evaluateAt time.Time, evaluateAtLocal time.Time) *Result {
	return &Result{
		Validator:               tt.NewString("transitland-lib"),
		ValidatorVersion:        tt.NewString(tl.Version.Tag),
		ReportedAt:              tt.NewTime(evaluateAt),
		ReportedAtLocal:         tt.NewTime(evaluateAtLocal),
		ReportedAtLocalTimezone: tt.NewString(evaluateAtLocal.Location().String()),
		IncludesStatic:          tt.NewBool(false),
		IncludesRT:              tt.NewBool(false),
		Success:                 tt.NewBool(false),
		Errors:                  map[string]*ValidationReportErrorGroup{},
		Warnings:                map[string]*ValidationReportErrorGroup{},
	}
}

type ResultDetails struct {
	SHA1                 tt.String        `json:"sha1"`
	Timezone             tt.String        `json:"timezone"`
	EarliestCalendarDate tt.Date          `json:"earliest_calendar_date"`
	LatestCalendarDate   tt.Date          `json:"latest_calendar_date"`
	Agencies             []gtfs.Agency    `json:"agencies"`
	Routes               []gtfs.Route     `json:"routes"`
	Stops                []gtfs.Stop      `json:"stops"`
	FeedInfos            []gtfs.FeedInfo  `json:"feed_infos"`
	Realtime             []RealtimeResult `json:"realtime"`
	stats.FeedVersionStats
}

type RealtimeResult struct {
	Url                  string          `json:"url"`
	Json                 map[string]any  `json:"json"`
	EntityCounts         rt.EntityCounts `json:"entity_counts"`
	TripUpdateStats      []rt.RTTripStat `json:"trip_update_stats"`
	VehiclePositionStats []rt.RTTripStat `json:"vehicle_position_stats"`
	Errors               []error
}

func (r *Result) Key() string {
	return fmt.Sprintf("report-%s-%d.json", r.Details.SHA1, time.Now().In(time.UTC).Unix())
}

type ValidationReportErrorGroup struct {
	ValidationReportID int
	Filename           string
	Field              string
	ErrorType          string
	ErrorCode          string
	GroupKey           string
	Level              int
	Count              int
	Errors             []ValidationReportErrorExemplar `db:"-"`
	tt.DatabaseEntity
}

func (e *ValidationReportErrorGroup) TableName() string {
	return "tl_validation_report_error_groups"
}

//////

type ValidationReportErrorExemplar struct {
	ValidationReportErrorGroupID int
	Line                         int
	Message                      string
	EntityID                     string
	Value                        string
	Geometry                     tt.Geometry
	EntityJson                   tt.Map
	tt.DatabaseEntity
}

func (e *ValidationReportErrorExemplar) TableName() string {
	return "tl_validation_report_error_exemplars"
}

//////

type ValidationReportTripUpdateStat struct {
	ValidationReportID      int
	AgencyID                string
	RouteID                 string
	TripScheduledIDs        tt.Strings `db:"trip_scheduled_ids"`
	TripRtIDs               tt.Strings `db:"trip_rt_ids"`
	TripScheduledCount      int
	TripScheduledMatched    int `db:"trip_match_count"`
	TripScheduledNotMatched int
	TripRtCount             int
	TripRtMatched           int
	TripRtNotMatched        int
	TripRtAddedIDs          tt.Strings `db:"trip_rt_added_ids"`
	TripRtAddedCount        int
	TripRtNotFoundIDs       tt.Strings `db:"trip_rt_not_found_ids"`
	TripRtNotFoundCount     int
	tt.DatabaseEntity
}

func (e *ValidationReportTripUpdateStat) TableName() string {
	return "tl_validation_trip_update_stats"
}

//////

type ValidationReportVehiclePositionStat struct {
	ValidationReportID      int
	AgencyID                string
	RouteID                 string
	TripScheduledIDs        tt.Strings `db:"trip_scheduled_ids"`
	TripRtIDs               tt.Strings `db:"trip_rt_ids"`
	TripScheduledCount      int
	TripScheduledMatched    int `db:"trip_match_count"`
	TripScheduledNotMatched int
	TripRtCount             int
	TripRtMatched           int
	TripRtNotMatched        int
	TripRtAddedIDs          tt.Strings `db:"trip_rt_added_ids"`
	TripRtAddedCount        int
	TripRtNotFoundIDs       tt.Strings `db:"trip_rt_not_found_ids"`
	TripRtNotFoundCount     int
	tt.DatabaseEntity
}

func (e *ValidationReportVehiclePositionStat) TableName() string {
	return "tl_validation_vehicle_position_stats"
}
