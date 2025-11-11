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
	Geometry      tt.MultiPolygon
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

type JobRun struct {
	ID           int          `json:"id"`
	JobType      string       `json:"job_type"`
	JobArgs      tt.Map       `json:"job_args"`
	Status       JobRunStatus `json:"status"`
	StartedAt    *time.Time   `json:"started_at,omitempty"`
	CompletedAt  *time.Time   `json:"completed_at,omitempty"`
	Metadata     tt.Map       `json:"metadata"`
	Metrics      tt.Map       `json:"metrics"`
	LogSummary   *string      `json:"log_summary,omitempty"`
	ErrorMessage *string      `json:"error_message,omitempty"`
	CreatedBy    *string      `json:"created_by,omitempty"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
	Artifacts    []*Artifact  `json:"artifacts,omitempty"`
}

type Artifact struct {
	ID             int            `json:"id"`
	Name           string         `json:"name"`
	ArtifactType   string         `json:"artifact_type"`
	StorageType    StorageType    `json:"storage_type"`
	InlineJsonData *tt.Map        `json:"inline_json_data,omitempty"`
	StorageURL     *string        `json:"storage_url,omitempty"`
	ContentType    *string        `json:"content_type,omitempty"`
	SizeBytes      *int64         `json:"size_bytes,omitempty"`
	Metadata       tt.Map         `json:"metadata"`
	JobRunID       *int           `json:"job_run_id,omitempty"`
	CreatedBy      *string        `json:"created_by,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	JobRun         *JobRun        `json:"job_run,omitempty"`
	FeedVersions   []*FeedVersion `json:"feed_versions,omitempty"`
}

// JobRunStatus represents the status of a job run
type JobRunStatus string

const (
	JobRunStatusPending   JobRunStatus = "pending"
	JobRunStatusRunning   JobRunStatus = "running"
	JobRunStatusSuccess   JobRunStatus = "success"
	JobRunStatusFailed    JobRunStatus = "failed"
	JobRunStatusCancelled JobRunStatus = "cancelled"
)

var AllJobRunStatus = []JobRunStatus{
	JobRunStatusPending,
	JobRunStatusRunning,
	JobRunStatusSuccess,
	JobRunStatusFailed,
	JobRunStatusCancelled,
}

func (e JobRunStatus) IsValid() bool {
	switch e {
	case JobRunStatusPending, JobRunStatusRunning, JobRunStatusSuccess, JobRunStatusFailed, JobRunStatusCancelled:
		return true
	}
	return false
}

func (e JobRunStatus) String() string {
	return string(e)
}

// StorageType represents where an artifact is stored
type StorageType string

const (
	StorageTypeInline StorageType = "inline"
	StorageTypeS3     StorageType = "s3"
	StorageTypeAzure  StorageType = "azure"
)

var AllStorageType = []StorageType{
	StorageTypeInline,
	StorageTypeS3,
	StorageTypeAzure,
}

func (e StorageType) IsValid() bool {
	switch e {
	case StorageTypeInline, StorageTypeS3, StorageTypeAzure:
		return true
	}
	return false
}

func (e StorageType) String() string {
	return string(e)
}

// ArtifactRelationshipType represents the relationship between an artifact and feed version
type ArtifactRelationshipType string

const (
	ArtifactRelationshipTypeInput  ArtifactRelationshipType = "input"
	ArtifactRelationshipTypeOutput ArtifactRelationshipType = "output"
)

var AllArtifactRelationshipType = []ArtifactRelationshipType{
	ArtifactRelationshipTypeInput,
	ArtifactRelationshipTypeOutput,
}

func (e ArtifactRelationshipType) IsValid() bool {
	switch e {
	case ArtifactRelationshipTypeInput, ArtifactRelationshipTypeOutput:
		return true
	}
	return false
}

func (e ArtifactRelationshipType) String() string {
	return string(e)
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
