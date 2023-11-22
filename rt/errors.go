package rt

// Errors
// https://github.com/CUTR-at-USF/gtfs-realtime-validator/blob/master/RULES.md
var (
	E001 = RealtimeError{msg: "Not in POSIX time", Code: 1}
	E002 = RealtimeError{msg: "stop_time_updates not strictly sorted", Code: 2}
	E003 = RealtimeError{msg: "GTFS-rt trip_id does not exist in GTFS data", Code: 3}
	E004 = RealtimeError{msg: "GTFS-rt route_id does not exist in GTFS data", Code: 4}
	// E006 = RealtimeError{msg: "Missing required trip field for frequency-based exact_times = 0", Code: 6}
	E009 = RealtimeError{msg: "GTFS-rt stop_sequence isn't provided for trip that visits same stop_id more than once", Code: 9}
	// E010 = RealtimeError{msg: "location_type not 0 in stops.txt (Note that this is implemented but not executed because it's specific to GTFS - see issue #126)", Code: 10}
	E011 = RealtimeError{msg: "GTFS-rt stop_id does not exist in GTFS data", Code: 11}
	// E012 = RealtimeError{msg: "Header timestamp should be greater than or equal to all other timestamps", Code: 12}
	// E013 = RealtimeError{msg: "Frequency type 0 trip schedule_relationship should be UNSCHEDULED or empty", Code: 13}
	E015 = RealtimeError{msg: "All stop_ids referenced in GTFS-rt feeds must have the location_type = 0", Code: 15}
	// E016 = RealtimeError{msg: "trip_ids with schedule_relationship ADDED must not be in GTFS data", Code: 16}
	// E017 = RealtimeError{msg: "GTFS-rt content changed but has the same header timestamp", Code: 17}
	E018 = RealtimeError{msg: "GTFS-rt header timestamp decreased between two sequential iterations", Code: 18} // same as E012?
	// E019 = RealtimeError{msg: "GTFS-rt frequency type 1 trip start_time must be a multiple of GTFS headway_secs later than GTFS start_time", Code: 19}
	E020 = RealtimeError{msg: "Invalid start_time format", Code: 20}
	E021 = RealtimeError{msg: "Invalid start_date format", Code: 21}
	E022 = RealtimeError{msg: "Sequential stop_time_update times are not increasing", Code: 22}
	// E023 = RealtimeError{msg: "trip start_time does not match first GTFS arrival_time", Code: 23}
	E024 = RealtimeError{msg: "trip direction_id does not match GTFS data", Code: 24}
	E025 = RealtimeError{msg: "stop_time_update departure time is before arrival time", Code: 25}
	E026 = RealtimeError{msg: "Invalid vehicle position", Code: 26}
	// E027 = RealtimeError{msg: "Invalid vehicle bearing", Code: 27}
	// E028 = RealtimeError{msg: "Vehicle position outside agency coverage area", Code: 28}
	E029 = RealtimeError{msg: "Vehicle position far from trip shape", Code: 29}
	// E030 = RealtimeError{msg: "GTFS-rt alert trip_id does not belong to GTFS-rt alert route_id  in GTFS trips.txt", Code: 30}
	// E031 = RealtimeError{msg: "Alert informed_entity.route_id does not match informed_entity.trip.route_id", Code: 31}
	// E032 = RealtimeError{msg: "Alert does not have an informed_entity", Code: 32}
	// E033 = RealtimeError{msg: "Alert informed_entity does not have any specifiers", Code: 33}
	// E034 = RealtimeError{msg: "GTFS-rt agency_id does not exist in GTFS data", Code: 34}
	// E035 = RealtimeError{msg: "GTFS-rt trip.trip_id does not belong to GTFS-rt trip.route_id in GTFS trips.txt", Code: 35}
	E036 = RealtimeError{msg: "Sequential stop_time_updates have the same stop_sequence", Code: 36}
	E037 = RealtimeError{msg: "Sequential stop_time_updates have the same stop_id", Code: 37}
	E038 = RealtimeError{msg: "Invalid header.gtfs_realtime_version", Code: 38}
	E039 = RealtimeError{msg: "FULL_DATASET feeds should not include entity.is_deleted", Code: 39}
	E040 = RealtimeError{msg: "stop_time_update doesn't contain stop_id or stop_sequence", Code: 40}
	E041 = RealtimeError{msg: "StopTimeUpdates are required unless the trip is canceled", Code: 41}
	E042 = RealtimeError{msg: "arrival or departure provided for NO_DATA stop_time_update", Code: 42}
	E043 = RealtimeError{msg: "stop_time_update doesn't have arrival or departure", Code: 43}
	E044 = RealtimeError{msg: "stop_time_update arrival/departure doesn't have delay or time", Code: 44}
	// E045 = RealtimeError{msg: "GTFS-rt stop_time_update stop_sequence and stop_id do not match GTFS", Code: 45}
	// E046 = RealtimeError{msg: "GTFS-rt stop_time_update without time doesn't have arrival/departure time in GTFS", Code: 46}
	// E047 = RealtimeError{msg: "VehiclePosition and TripUpdate ID pairing mismatch", Code: 47}
	E048 = RealtimeError{msg: "header timestamp not populated (GTFS-rt v2.0 and higher)", Code: 48}
	E049 = RealtimeError{msg: "header incrementality not populated (GTFS-rt v2.0 and higher)", Code: 49}
	E050 = RealtimeError{msg: "timestamp is in the future", Code: 50}
	// E051 = RealtimeError{msg: "GTFS-rt stop_sequence not found in GTFS data", Code: 51}
	// E052 = RealtimeError{msg: "vehicle.id is not unique", Code: 52}
)

// Warnings
var (
// W001 = RealtimeWarning{msg: "timestamps not populated", Code: 1}
// W002 = RealtimeWarning{msg: "vehicle_id not populated", Code: 2}
// W003 = RealtimeWarning{msg: "ID in one feed missing from the other", Code: 3}
// W004 = RealtimeWarning{msg: "vehicle speed is unrealistic", Code: 4}
// W005 = RealtimeWarning{msg: "Missing vehicle_id in trip_update for frequency-based exact_times = 0", Code: 5}
// W006 = RealtimeWarning{msg: "trip_update missing trip_id", Code: 6}
// W007 = RealtimeWarning{msg: "Refresh interval is more than 35 seconds", Code: 7}
// W008 = RealtimeWarning{msg: "Header timestamp is older than 65 seconds", Code: 8}
// W009 = RealtimeWarning{msg: "schedule_relationship not populated", Code: 9}
)

func ne(msg string, field string) *RealtimeError {
	return &RealtimeError{
		Field: field,
		msg:   msg,
	}
}

func ef(e RealtimeError, field string) *RealtimeError {
	e2 := e
	e2.Field = field
	return &e2
}

// RealtimeError is a GTFS RealTime error.
type RealtimeError struct {
	Code  int
	Field string
	msg   string
}

func (err RealtimeError) Error() string {
	return err.msg
}

// RealtimeWarning is a GTFS RealTime warning.
type RealtimeWarning struct {
	Code  int
	Field string
	msg   string
}

func (err RealtimeWarning) Error() string {
	return err.msg
}
