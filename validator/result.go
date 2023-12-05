package validator

import (
	"fmt"
	"time"

	"github.com/interline-io/transitland-lib/copier"
	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/rt"
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tl/tt"
	"github.com/interline-io/transitland-lib/tldb"
)

// Result contains a validation report result,
type Result struct {
	copier.Result                                       // add to copier result:
	Success              bool                           `json:"success"`
	FailureReason        string                         `json:"failure_reason"`
	SHA1                 string                         `json:"sha1"`
	EarliestCalendarDate tl.Date                        `json:"earliest_calendar_date"`
	LatestCalendarDate   tl.Date                        `json:"latest_calendar_date"`
	Timezone             string                         `json:"timezone"`
	Agencies             []tl.Agency                    `json:"agencies"`
	Routes               []tl.Route                     `json:"routes"`
	Stops                []tl.Stop                      `json:"stops"`
	FeedInfos            []tl.FeedInfo                  `json:"feed_infos"`
	Files                []dmfr.FeedVersionFileInfo     `json:"files"`
	ServiceLevels        []dmfr.FeedVersionServiceLevel `json:"service_levels"`
	Realtime             []RealtimeResult               `json:"realtime"`
}

type RealtimeResult struct {
	Url                  string                    `json:"url"`
	Json                 map[string]any            `json:"json"`
	EntityCounts         rt.EntityCounts           `json:"entity_counts"`
	TripUpdateStats      []rt.TripUpdateStats      `json:"trip_update_stats"`
	VehiclePositionStats []rt.VehiclePositionStats `json:"vehicle_position_stats"`
	Errors               []error
}

func SaveValidationReport(atx tldb.Adapter, result *Result, fvid int, saveStatic bool, saveRealtimeStats bool) error {
	// Save validation reports
	validationReport := ValidationReport{}
	validationReport.FeedVersionID = fvid
	validationReport.ReportedAt = tt.NewTime(time.Now())
	if _, err := atx.Insert(&validationReport); err != nil {
		return err
	}
	if saveRealtimeStats {
		for _, r := range result.Realtime {
			for _, s := range r.TripUpdateStats {
				tripReport := ValidationReportTripUpdateStat{
					ValidationReportID: validationReport.ID,
					AgencyID:           s.AgencyID,
					RouteID:            s.RouteID,
					TripScheduledCount: s.TripScheduledCount,
					TripMatchCount:     s.TripMatchCount,
				}
				if _, err := atx.Insert(&tripReport); err != nil {
					return err
				}
				fmt.Printf("tp: %#v\n", tripReport)
			}
			for _, s := range r.VehiclePositionStats {
				_ = s
			}
		}
	}

	return nil
}

type ValidationReport struct {
	ReportedAt tt.Time
	tl.BaseEntity
}

func (e *ValidationReport) TableName() string {
	return "tl_validation_reports"
}

type ValidationReportTripUpdateStat struct {
	ValidationReportID int
	AgencyID           string
	RouteID            string
	TripScheduledCount int
	TripMatchCount     int
	tl.BaseEntity
}

func (e *ValidationReportTripUpdateStat) TableName() string {
	return "tl_validation_trip_update_stats"
}
