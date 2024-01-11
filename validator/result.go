package validator

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/interline-io/transitland-lib/copier"
	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/dmfr/store"
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

func (r *Result) Key() string {
	return fmt.Sprintf("report-%s-%d.json", r.SHA1, time.Now().In(time.UTC).Unix())
}

func SaveValidationReport(atx tldb.Adapter, result *Result, reportedAt time.Time, fvid int, reportStorage string) error {
	// Save validation reports
	validationReport := ValidationReport{}
	validationReport.FeedVersionID = fvid
	validationReport.ReportedAt = tt.NewTime(reportedAt)

	// Save JSON
	if reportStorage != "" {
		validationReport.File = tt.NewString(result.Key())
		store, err := store.GetStore(reportStorage)
		if err != nil {
			return err
		}
		jj, err := json.Marshal(result)
		if err != nil {
			return err
		}
		jb := bytes.NewReader(jj)
		if err := store.Upload(context.Background(), validationReport.File.Val, tl.Secret{}, jb); err != nil {
			return err
		}
	}

	// Save record
	if _, err := atx.Insert(&validationReport); err != nil {
		return err
	}

	// Save error groups
	for _, eg := range result.Errors {
		egEnt := ValidationReportErrorGroup{
			ValidationReportID: validationReport.ID,
			Filename:           eg.Filename,
			Field:              eg.Field,
			ErrorType:          eg.ErrorType,
			ErrorCode:          eg.ErrorCode,
			Count:              eg.Count,
		}
		if _, err := atx.Insert(&egEnt); err != nil {
			return err
		}
		for _, egErr := range eg.Errors {
			if _, err := atx.Insert(&ValidationReportErrorExemplar{
				ValidationReportErrorGroupID: egEnt.ID,
				Line:                         egErr.Line,
				Message:                      egErr.Message,
				EntityID:                     egErr.EntityID,
				Value:                        egErr.Value,
			}); err != nil {
				return err
			}
		}
	}

	// Save additional stats
	for _, r := range result.Realtime {
		for _, s := range r.TripUpdateStats {
			tripReport := ValidationReportTripUpdateStat{
				ValidationReportID: validationReport.ID,
				AgencyID:           s.AgencyID,
				RouteID:            s.RouteID,
				TripScheduledCount: s.TripScheduledCount,
				TripMatchCount:     s.TripMatchCount,
				TripScheduledIDs:   tt.NewStrings(s.TripScheduledIDs),
			}
			if _, err := atx.Insert(&tripReport); err != nil {
				return err
			}
		}
		for _, s := range r.VehiclePositionStats {
			vpReport := ValidationReportTripUpdateStat{
				ValidationReportID: validationReport.ID,
				AgencyID:           s.AgencyID,
				RouteID:            s.RouteID,
				TripScheduledCount: s.TripScheduledCount,
				TripMatchCount:     s.TripMatchCount,
				TripScheduledIDs:   tt.NewStrings(s.TripScheduledIDs),
			}
			if _, err := atx.Insert(&vpReport); err != nil {
				return err
			}
		}
	}
	return nil
}

//////

type ValidationReport struct {
	ReportedAt tt.Time
	File       tt.String
	tl.BaseEntity
}

func (e *ValidationReport) TableName() string {
	return "tl_validation_reports"
}

//////

type ValidationReportErrorGroup struct {
	ValidationReportID int
	Filename           string
	Field              string
	ErrorType          string
	ErrorCode          string
	Count              int
	tl.DatabaseEntity
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
	// Geometries                   []tt.Geometry
	tl.DatabaseEntity
}

func (e *ValidationReportErrorExemplar) TableName() string {
	return "tl_validation_report_error_exemplars"
}

//////

type ValidationReportTripUpdateStat struct {
	ValidationReportID int
	AgencyID           string
	RouteID            string
	TripScheduledCount int
	TripMatchCount     int
	TripScheduledIDs   tt.Strings `db:"trip_scheduled_ids"`
	tl.DatabaseEntity
}

func (e *ValidationReportTripUpdateStat) TableName() string {
	return "tl_validation_trip_update_stats"
}

//////

type ValidationReportVehiclePositionStat struct {
	ValidationReportID int
	AgencyID           string
	RouteID            string
	TripScheduledCount int
	TripMatchCount     int
	TripScheduledIDs   tt.Strings `db:"trip_scheduled_ids"`
	tl.DatabaseEntity
}

func (e *ValidationReportVehiclePositionStat) TableName() string {
	return "tl_validation_vehicle_position_stats"
}
