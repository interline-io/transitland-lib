// Code generated by github.com/99designs/gqlgen, DO NOT EDIT.

package model

import (
	"fmt"
	"io"
	"strconv"

	"github.com/interline-io/transitland-lib/tl"
)

type AgencyFilter struct {
	OnestopID       *string `json:"onestop_id"`
	FeedVersionSha1 *string `json:"feed_version_sha1"`
	FeedOnestopID   *string `json:"feed_onestop_id"`
	AgencyID        *string `json:"agency_id"`
	AgencyName      *string `json:"agency_name"`
	Search          *string `json:"search"`
}

type AgencyPlaceFilter struct {
	MinRank *float64 `json:"min_rank"`
	Search  *string  `json:"search"`
}

type CalendarDateFilter struct {
	Date          *tl.ODate `json:"date"`
	ExceptionType *int      `json:"exception_type"`
}

type FeedFilter struct {
	OnestopID    *string       `json:"onestop_id"`
	Spec         []string      `json:"spec"`
	FetchError   *bool         `json:"fetch_error"`
	ImportStatus *ImportStatus `json:"import_status"`
	Search       *string       `json:"search"`
}

type FeedVersionDeleteResult struct {
	Success bool `json:"success"`
}

type FeedVersionFilter struct {
	FeedOnestopID *string `json:"feed_onestop_id"`
	Sha1          *string `json:"sha1"`
}

type FeedVersionServiceLevelFilter struct {
	StartDate  *tl.ODate `json:"start_date"`
	EndDate    *tl.ODate `json:"end_date"`
	AllRoutes  *bool     `json:"all_routes"`
	DistinctOn *string   `json:"distinct_on"`
	RouteIds   []string  `json:"route_ids"`
}

type FeedVersionSetInput struct {
	Name        *string `json:"name"`
	Description *string `json:"description"`
}

type FeedVersionUnimportResult struct {
	Success bool `json:"success"`
}

type OperatorFilter struct {
	Merged          *bool   `json:"merged"`
	OnestopID       *string `json:"onestop_id"`
	FeedVersionSha1 *string `json:"feed_version_sha1"`
	FeedOnestopID   *string `json:"feed_onestop_id"`
	AgencyID        *string `json:"agency_id"`
	Search          *string `json:"search"`
}

type PathwayFilter struct {
	PathwayMode *int `json:"pathway_mode"`
}

type RouteFilter struct {
	OnestopID         *string `json:"onestop_id"`
	FeedVersionSha1   *string `json:"feed_version_sha1"`
	FeedOnestopID     *string `json:"feed_onestop_id"`
	RouteID           *string `json:"route_id"`
	RouteType         *int    `json:"route_type"`
	OperatorOnestopID *string `json:"operator_onestop_id"`
	Search            *string `json:"search"`
	AgencyIds         []int   `json:"agency_ids"`
}

type StopFilter struct {
	OnestopID       *string     `json:"onestop_id"`
	FeedVersionSha1 *string     `json:"feed_version_sha1"`
	FeedOnestopID   *string     `json:"feed_onestop_id"`
	StopID          *string     `json:"stop_id"`
	AgencyIds       []int       `json:"agency_ids"`
	Geometry        *tl.Polygon `json:"geometry"`
	Search          *string     `json:"search"`
}

type StopTimeFilter struct {
	ServiceDate *tl.ODate `json:"service_date"`
	StartTime   *int      `json:"start_time"`
	EndTime     *int      `json:"end_time"`
}

type TripFilter struct {
	TripID          *string `json:"trip_id"`
	RouteID         *int    `json:"route_id"`
	FeedVersionSha1 *string `json:"feed_version_sha1"`
	FeedOnestopID   *string `json:"feed_onestop_id"`
}

type UserProfileInput struct {
	Name  *string `json:"name"`
	Email *string `json:"email"`
}

type ImportStatus string

const (
	ImportStatusSuccess    ImportStatus = "success"
	ImportStatusError      ImportStatus = "error"
	ImportStatusInProgress ImportStatus = "in_progress"
)

var AllImportStatus = []ImportStatus{
	ImportStatusSuccess,
	ImportStatusError,
	ImportStatusInProgress,
}

func (e ImportStatus) IsValid() bool {
	switch e {
	case ImportStatusSuccess, ImportStatusError, ImportStatusInProgress:
		return true
	}
	return false
}

func (e ImportStatus) String() string {
	return string(e)
}

func (e *ImportStatus) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = ImportStatus(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid ImportStatus", str)
	}
	return nil
}

func (e ImportStatus) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}

type Role string

const (
	RoleAnon  Role = "ANON"
	RoleAdmin Role = "ADMIN"
	RoleUser  Role = "USER"
)

var AllRole = []Role{
	RoleAnon,
	RoleAdmin,
	RoleUser,
}

func (e Role) IsValid() bool {
	switch e {
	case RoleAnon, RoleAdmin, RoleUser:
		return true
	}
	return false
}

func (e Role) String() string {
	return string(e)
}

func (e *Role) UnmarshalGQL(v interface{}) error {
	str, ok := v.(string)
	if !ok {
		return fmt.Errorf("enums must be strings")
	}

	*e = Role(str)
	if !e.IsValid() {
		return fmt.Errorf("%s is not a valid Role", str)
	}
	return nil
}

func (e Role) MarshalGQL(w io.Writer) {
	fmt.Fprint(w, strconv.Quote(e.String()))
}
