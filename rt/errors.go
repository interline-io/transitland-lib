package rt

import "github.com/interline-io/transitland-lib/tl/tt"

// Errors
// https://github.com/CUTR-at-USF/gtfs-realtime-validator/blob/master/RULES.md
var (
	E001 = RealtimeError{msg: "Not in POSIX time", code: 1}
	E002 = RealtimeError{msg: "stop_time_updates not strictly sorted", code: 2}
	E003 = RealtimeError{msg: "GTFS-rt trip_id does not exist in GTFS data", code: 3}
	E004 = RealtimeError{msg: "GTFS-rt route_id does not exist in GTFS data", code: 4}
	// E006 = RealtimeError{msg: "Missing required trip field for frequency-based exact_times = 0", code: 6}
	E009 = RealtimeError{msg: "GTFS-rt stop_sequence isn't provided for trip that visits same stop_id more than once", code: 9}
	// E010 = RealtimeError{msg: "location_type not 0 in stops.txt (Note that this is implemented but not executed because it's specific to GTFS - see issue #126)", code: 10}
	E011 = RealtimeError{msg: "GTFS-rt stop_id does not exist in GTFS data", code: 11}
	// E012 = RealtimeError{msg: "Header timestamp should be greater than or equal to all other timestamps", code: 12}
	// E013 = RealtimeError{msg: "Frequency type 0 trip schedule_relationship should be UNSCHEDULED or empty", code: 13}
	E015 = RealtimeError{msg: "All stop_ids referenced in GTFS-rt feeds must have the location_type = 0", code: 15}
	// E016 = RealtimeError{msg: "trip_ids with schedule_relationship ADDED must not be in GTFS data", code: 16}
	// E017 = RealtimeError{msg: "GTFS-rt content changed but has the same header timestamp", code: 17}
	E018 = RealtimeError{msg: "GTFS-rt header timestamp decreased between two sequential iterations", code: 18} // same as E012?
	// E019 = RealtimeError{msg: "GTFS-rt frequency type 1 trip start_time must be a multiple of GTFS headway_secs later than GTFS start_time", code: 19}
	E020 = RealtimeError{msg: "Invalid start_time format", code: 20}
	E021 = RealtimeError{msg: "Invalid start_date format", code: 21}
	E022 = RealtimeError{msg: "Sequential stop_time_update times are not increasing", code: 22}
	// E023 = RealtimeError{msg: "trip start_time does not match first GTFS arrival_time", code: 23}
	E024 = RealtimeError{msg: "trip direction_id does not match GTFS data", code: 24}
	E025 = RealtimeError{msg: "stop_time_update departure time is before arrival time", code: 25}
	E026 = RealtimeError{msg: "Invalid vehicle position", code: 26}
	// E027 = RealtimeError{msg: "Invalid vehicle bearing", code: 27}
	// E028 = RealtimeError{msg: "Vehicle position outside agency coverage area", code: 28}
	E029 = RealtimeError{msg: "Vehicle position far from trip shape", code: 29}
	// E030 = RealtimeError{msg: "GTFS-rt alert trip_id does not belong to GTFS-rt alert route_id  in GTFS trips.txt", code: 30}
	// E031 = RealtimeError{msg: "Alert informed_entity.route_id does not match informed_entity.trip.route_id", code: 31}
	// E032 = RealtimeError{msg: "Alert does not have an informed_entity", code: 32}
	// E033 = RealtimeError{msg: "Alert informed_entity does not have any specifiers", code: 33}
	// E034 = RealtimeError{msg: "GTFS-rt agency_id does not exist in GTFS data", code: 34}
	// E035 = RealtimeError{msg: "GTFS-rt trip.trip_id does not belong to GTFS-rt trip.route_id in GTFS trips.txt", code: 35}
	E036 = RealtimeError{msg: "Sequential stop_time_updates have the same stop_sequence", code: 36}
	E037 = RealtimeError{msg: "Sequential stop_time_updates have the same stop_id", code: 37}
	E038 = RealtimeError{msg: "Invalid header.gtfs_realtime_version", code: 38}
	E039 = RealtimeError{msg: "FULL_DATASET feeds should not include entity.is_deleted", code: 39}
	E040 = RealtimeError{msg: "stop_time_update doesn't contain stop_id or stop_sequence", code: 40}
	E041 = RealtimeError{msg: "StopTimeUpdates are required unless the trip is canceled", code: 41}
	E042 = RealtimeError{msg: "arrival or departure provided for NO_DATA stop_time_update", code: 42}
	E043 = RealtimeError{msg: "stop_time_update doesn't have arrival or departure", code: 43}
	E044 = RealtimeError{msg: "stop_time_update arrival/departure doesn't have delay or time", code: 44}
	// E045 = RealtimeError{msg: "GTFS-rt stop_time_update stop_sequence and stop_id do not match GTFS", code: 45}
	// E046 = RealtimeError{msg: "GTFS-rt stop_time_update without time doesn't have arrival/departure time in GTFS", code: 46}
	// E047 = RealtimeError{msg: "VehiclePosition and TripUpdate ID pairing mismatch", code: 47}
	E048 = RealtimeError{msg: "header timestamp not populated (GTFS-rt v2.0 and higher)", code: 48}
	E049 = RealtimeError{msg: "header incrementality not populated (GTFS-rt v2.0 and higher)", code: 49}
	E050 = RealtimeError{msg: "timestamp is in the future", code: 50}
	// E051 = RealtimeError{msg: "GTFS-rt stop_sequence not found in GTFS data", code: 51}
	// E052 = RealtimeError{msg: "vehicle.id is not unique", code: 52}
)

// Warnings
var (
// W001 = RealtimeWarning{msg: "timestamps not populated", code: 1}
// W002 = RealtimeWarning{msg: "vehicle_id not populated", code: 2}
// W003 = RealtimeWarning{msg: "ID in one feed missing from the other", code: 3}
// W004 = RealtimeWarning{msg: "vehicle speed is unrealistic", code: 4}
// W005 = RealtimeWarning{msg: "Missing vehicle_id in trip_update for frequency-based exact_times = 0", code: 5}
// W006 = RealtimeWarning{msg: "trip_update missing trip_id", code: 6}
// W007 = RealtimeWarning{msg: "Refresh interval is more than 35 seconds", code: 7}
// W008 = RealtimeWarning{msg: "Header timestamp is older than 65 seconds", code: 8}
// W009 = RealtimeWarning{msg: "schedule_relationship not populated", code: 9}
)

func ne(msg string, field string) *RealtimeError {
	return &RealtimeError{
		field: field,
		msg:   msg,
	}
}

func ef(e RealtimeError, field string) *RealtimeError {
	e2 := e
	e2.field = field
	return &e2
}

// RealtimeError is a GTFS RealTime error.
type RealtimeError struct {
	code  int
	field string
	geoms []tt.Geometry
	msg   string
}

func (e RealtimeError) Error() string {
	return e.msg
}

func (e RealtimeError) Code() int {
	return e.code
}

func (e RealtimeError) Field() string {
	return e.field
}

func (e RealtimeError) Geometries() []tt.Geometry {
	return e.geoms
}

// RealtimeWarning is a GTFS RealTime warning.
type RealtimeWarning struct {
	RealtimeError
}
