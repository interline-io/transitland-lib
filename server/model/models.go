package model

import (
	"encoding/json"

	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/rt/pb"
	"github.com/interline-io/transitland-lib/service"
	"github.com/interline-io/transitland-lib/tlxy"
	"github.com/interline-io/transitland-lib/tt"
)

type StopPlaceParam struct {
	ID    int
	Point tlxy.Point
}

// PlaceGeometrySelect chooses which geometry columns a place query computes.
// Each is an aggregate over the Natural Earth tables, so it is only worth paying
// for when the caller asked for it.
type PlaceGeometrySelect struct {
	Bbox bool
}

// CensusGeographySelect chooses which geometry columns a census geography query
// returns. Each is expensive on every row, so a query returns only the ones its
// field selects.
type CensusGeographySelect struct {
	Geometry             bool
	GeometryArea         bool
	IntersectionArea     bool
	IntersectionGeometry bool
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
	// internal: set on a trip added in real time, which MakeTrip builds.
	AddedInRealtime bool
	// internal: the service dates of the requested runs, as given rather than
	// relocated into a fallback week. Realtime data is matched to these runs;
	// with none, to the trip's current run.
	RunDates []tt.Date
	// internal: the start of the requested run of a frequency-based trip, in
	// seconds into its service day, which realtime data naming a start_time
	// has to match, and the seconds the run's times are shifted from the
	// trip's stop times. Both invalid for any other run.
	RunStart  tt.Seconds
	RunOffset tt.Int
	// internal: set for a trip that stands for no run, such as a pattern's
	// representative trip, which realtime data never matches.
	NoRealtime bool
	// Every service date matched by a dates or service_dates query. Under
	// `dates` this reaches one day before the earliest requested date.
	ServiceDates []*tt.Date
	// Wire format for ServiceDates: one comma-separated column.
	ServiceDatesAgg tt.String `db:"service_dates_agg"`
	gtfs.Trip
}

// RouteStopPatternTimetable is a stop pattern's trips as grids by stop and trip.
type RouteStopPatternTimetable struct {
	StopIds        []int
	Trips          []*Trip
	DepartureTimes *RouteStopPatternTimetableTimeGrid
	ArrivalTimes   *RouteStopPatternTimetableTimeGrid
	PickupTypes    *RouteStopPatternTimetableGrid
	DropOffTypes   *RouteStopPatternTimetableGrid
	Timepoints     *RouteStopPatternTimetableGrid
}

// RouteStopPatternTimetableGrid is one of a timetable's grids of numbers, by stop
// and trip.
type RouteStopPatternTimetableGrid struct {
	Values [][]tt.Int
}

// RouteStopPatternTimetableTimeGrid is one of a timetable's grids of times, by
// stop and trip.
type RouteStopPatternTimetableTimeGrid struct {
	Values [][]tt.Seconds
}

// RTStopTimeUpdate is the realtime data matched to one stop time of a run.
type RTStopTimeUpdate struct {
	TripUpdate *pb.TripUpdate
	// The stop time update naming the stop time, or nil when none does.
	StopTimeUpdate *pb.TripUpdate_StopTimeUpdate
	// Estimated arrival and departure in unix seconds, or 0 for none: from the
	// stop time update's events, or else the delay carried from earlier stops.
	EstimatedArrival   int64
	EstimatedDeparture int64
	// The sources of the estimates, empty for none.
	ArrivalSource   EstimateSource
	DepartureSource EstimateSource
}

// EstimateSource names the source of an estimated arrival or departure.
type EstimateSource string

const (
	// EstimateTime is the time the stop time update's event gives.
	EstimateTime EstimateSource = "time"
	// EstimateDelay is the scheduled time plus a delay that the stop time
	// update's own events give or imply.
	EstimateDelay EstimateSource = "delay"
	// EstimateCarried is the scheduled time plus a delay carried from an
	// earlier stop time update, or the trip's own delay before the first.
	EstimateCarried EstimateSource = "carried"
)

// ArrivalEvent returns the stop time update's arrival event, or else its
// departure event: the event that estimates the arrival.
func (u *RTStopTimeUpdate) ArrivalEvent() *pb.TripUpdate_StopTimeEvent {
	if u.StopTimeUpdate == nil {
		return nil
	}
	if u.StopTimeUpdate.Arrival != nil {
		return u.StopTimeUpdate.Arrival
	}
	return u.StopTimeUpdate.Departure
}

// DepartureEvent returns the stop time update's departure event, or else its
// arrival event: the event that estimates the departure.
func (u *RTStopTimeUpdate) DepartureEvent() *pb.TripUpdate_StopTimeEvent {
	if u.StopTimeUpdate == nil {
		return nil
	}
	if u.StopTimeUpdate.Departure != nil {
		return u.StopTimeUpdate.Departure
	}
	return u.StopTimeUpdate.Arrival
}

type StopTime struct {
	ServiceDate      tt.Date
	Date             tt.Date
	GtfsTripID       string            // internal: the trip's GTFS trip_id, for matching realtime
	RunStart         tt.Seconds        `db:"run_start"`  // internal: the start of the run, for an instance of a frequency-based trip
	RunOffset        tt.Int            `db:"run_offset"` // internal: the seconds the run's times are shifted from the trip's stop times, for such an instance
	RouteID          tt.Int            `db:"route_id"`   // internal: the trip's route, for a filter's route_limit
	RTTrip           *Trip             // internal: the trip of a stop time added in real time
	RTStopTimeUpdate *RTStopTimeUpdate // internal
	gtfs.StopTime
}

// SetServiceDate sets a stop time's service date and its calendar date, which
// is the next day for a time past midnight.
func (st *StopTime) SetServiceDate(d tt.Date) {
	st.ServiceDate = d
	if st.ArrivalTime.Val >= 24*60*60 {
		st.Date.Set(d.Val.AddDate(0, 0, 1))
	} else {
		st.Date = d
	}
}

type Stop struct {
	FeedOnestopID       string
	FeedVersionSHA1     string
	OnestopID           *string
	SearchRank          *string
	WithinFeatures      tt.Strings
	WithRouteID         tt.Int
	WithAgencyID        tt.Int
	WithLocationGroupID tt.Int // internal: grouping key set by StopsByLocationGroupIDs
	gtfs.Stop
}

type Frequency struct {
	FeedOnestopID   string
	FeedVersionSHA1 string
	gtfs.Frequency
}

// Transfer is a transfers.txt record.
type Transfer struct {
	FeedOnestopID   string
	FeedVersionSHA1 string
	gtfs.Transfer
}

type CalendarDate struct {
	gtfs.CalendarDate
}

type Shape struct {
	service.ShapeLine
}

type Level struct {
	FeedOnestopID   string
	FeedVersionSHA1 string
	Geometry        tt.MultiPolygon
	ParentStation   tt.Key
	gtfs.Level
}

type FeedInfo struct {
	gtfs.FeedInfo
}

// Translation is a translations.txt record.
type Translation struct {
	FeedOnestopID   string
	FeedVersionSHA1 string
	gtfs.Translation
}

// TranslationValue returns the translations.translation column, which the
// embedded gtfs.Translation shadows.
func (t *Translation) TranslationValue() tt.String {
	return t.Translation.Translation
}

// Attribution is an attributions.txt record.
type Attribution struct {
	FeedOnestopID   string
	FeedVersionSHA1 string
	gtfs.Attribution
}

type Pathway struct {
	FeedOnestopID   string
	FeedVersionSHA1 string
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

type Location struct {
	FeedOnestopID   string
	FeedVersionSHA1 string
	gtfs.Location
}

type BookingRule struct {
	FeedOnestopID   string
	FeedVersionSHA1 string
	gtfs.BookingRule
}

type FareAttribute struct {
	FeedOnestopID   string
	FeedVersionSHA1 string
	gtfs.FareAttribute
}

type FareRule struct {
	FeedOnestopID   string
	FeedVersionSHA1 string
	gtfs.FareRule
}

type FareMedia struct {
	FeedOnestopID   string
	FeedVersionSHA1 string
	gtfs.FareMedia
}

type FareProduct struct {
	FeedOnestopID   string
	FeedVersionSHA1 string
	gtfs.FareProduct
}

type FareLegRule struct {
	FeedOnestopID   string
	FeedVersionSHA1 string
	gtfs.FareLegRule
}

type FareLegJoinRule struct {
	FeedOnestopID   string
	FeedVersionSHA1 string
	gtfs.FareLegJoinRule
}

type FareTransferRule struct {
	FeedOnestopID   string
	FeedVersionSHA1 string
	gtfs.FareTransferRule
}

type RiderCategory struct {
	FeedOnestopID   string
	FeedVersionSHA1 string
	gtfs.RiderCategory
}

type Timeframe struct {
	FeedOnestopID   string
	FeedVersionSHA1 string
	gtfs.Timeframe
}

type Area struct {
	FeedOnestopID   string
	FeedVersionSHA1 string
	WithStopID      tt.Int
	gtfs.Area
}

type Network struct {
	FeedOnestopID   string
	FeedVersionSHA1 string
	WithRouteID     tt.Int
	gtfs.Network
}

type RouteNetwork struct {
	FeedOnestopID   string
	FeedVersionSHA1 string
	gtfs.RouteNetwork
}

type StopArea struct {
	FeedOnestopID   string
	FeedVersionSHA1 string
	gtfs.StopArea
}

type LocationGroup struct {
	FeedOnestopID   string
	FeedVersionSHA1 string
	WithStopID      tt.Int // internal: grouping key set by LocationGroupsByStopIDs
	gtfs.LocationGroup
}

type LocationGroupStop struct {
	FeedOnestopID   string
	FeedVersionSHA1 string
	gtfs.LocationGroupStop
}

// FlexStopTime is an alias for StopTime.
// Both types represent stop_times records and share the same underlying model (gtfs.StopTime).
// The separation exists only for GraphQL schema purposes where they have different resolvers.
type FlexStopTime = StopTime

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
