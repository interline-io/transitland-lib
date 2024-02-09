package rt

import (
	"encoding/json"
	"fmt"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/tl/causes"
	"github.com/interline-io/transitland-lib/tl/tt"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"
)

// Errors
// https://github.com/CUTR-at-USF/gtfs-realtime-validator/blob/master/RULES.md
var (
	E001 = nec("Not in POSIX time", "E001")
	E002 = nec("stop_time_updates not strictly sorted", "E002")
	E003 = nec("GTFS-rt trip_id does not exist in GTFS data", "E003")
	E004 = nec("GTFS-rt route_id does not exist in GTFS data", "E004")
	// E006 = nec("Missing required trip field for frequency-based exact_times = 0", "E006")
	E009 = nec("GTFS-rt stop_sequence isn't provided for trip that visits same stop_id more than once", "E009")
	// E010 = nec("location_type not 0 in stops.txt (Note that this is implemented but not executed because it's specific to GTFS - see issue #1"E026")", "E010")
	E011 = nec("GTFS-rt stop_id does not exist in GTFS data", "E011")
	// E012 = nec("Header timestamp should be greater than or equal to all other timestamps", "E012")
	// E013 = nec("Frequency type 0 trip schedule_relationship should be UNSCHEDULED or empty", "E013")
	E015 = nec("All stop_ids referenced in GTFS-rt feeds must have the location_type = 0", "E015")
	// E016 = nec("trip_ids with schedule_relationship ADDED must not be in GTFS data", "E016")
	// E017 = nec("GTFS-rt content changed but has the same header timestamp", "E017")
	E018 = nec("GTFS-rt header timestamp decreased between two sequential iterations", "E018") // same as E012?
	// E019 = nec("GTFS-rt frequency type 1 trip start_time must be a multiple of GTFS headway_secs later than GTFS start_time", "E019")
	E020 = nec("Invalid start_time format", "E020")
	E021 = nec("Invalid start_date format", "E021")
	E022 = nec("Sequential stop_time_update times are not increasing", "E022")
	// E023 = nec("trip start_time does not match first GTFS arrival_time", "E023")
	E024 = nec("trip direction_id does not match GTFS data", "E024")
	E025 = nec("stop_time_update arrival time is after departure time", "E025")
	E026 = nec("Invalid vehicle position", "E026")
	// E027 = nec("Invalid vehicle bearing", "E027")
	// E028 = nec("Vehicle position outside agency coverage area", "E028")
	E029 = nec("Vehicle position far from trip shape", "E029")
	// E030 = nec("GTFS-rt alert trip_id does not belong to GTFS-rt alert route_id  in GTFS trips.txt", "E030")
	// E031 = nec("Alert informed_entity.route_id does not match informed_entity.trip.route_id", "E031")
	// E032 = nec("Alert does not have an informed_entity", "E032")
	// E033 = nec("Alert informed_entity does not have any specifiers", "E033")
	// E034 = nec("GTFS-rt agency_id does not exist in GTFS data", "E034")
	// E035 = nec("GTFS-rt trip.trip_id does not belong to GTFS-rt trip.route_id in GTFS trips.txt", "E035")
	E036 = nec("Sequential stop_time_updates have the same stop_sequence", "E036")
	E037 = nec("Sequential stop_time_updates have the same stop_id", "E037")
	E038 = nec("Invalid header.gtfs_realtime_version", "E038")
	E039 = nec("FULL_DATASET feeds should not include entity.is_deleted", "E039")
	E040 = nec("stop_time_update doesn't contain stop_id or stop_sequence", "E040")
	E041 = nec("StopTimeUpdates are required unless the trip is canceled", "E041")
	E042 = nec("arrival or departure provided for NO_DATA stop_time_update", "E042")
	E043 = nec("stop_time_update doesn't have arrival or departure", "E043")
	E044 = nec("stop_time_update arrival/departure doesn't have delay or time", "E044")
	// E045 = nec("GTFS-rt stop_time_update stop_sequence and stop_id do not match GTFS", "E045")
	// E046 = nec("GTFS-rt stop_time_update without time doesn't have arrival/departure time in GTFS", "E046")
	// E047 = nec("VehiclePosition and TripUpdate ID pairing mismatch", "E047")
	E048 = nec("header timestamp not populated (GTFS-rt v2.0 and higher)", "E048")
	E049 = nec("header incrementality not populated (GTFS-rt v2.0 and higher)", "E049")
	E050 = nec("timestamp is in the future", "E050")
	// E051 = nec("GTFS-rt stop_sequence not found in GTFS data", "E051")
	// E052 = nec("vehicle.id is not unique", "E052")
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

type bc = causes.Context

func nec(msg string, errorCode string) RealtimeError {
	return RealtimeError{
		bc: causes.Context{
			Message:   msg,
			ErrorCode: errorCode,
		},
	}
}

func newError(msg string, field string) *RealtimeError {
	return &RealtimeError{
		bc: causes.Context{
			Field:   field,
			Message: msg,
		},
	}
}

func withField(e RealtimeError, field string) *RealtimeError {
	e2 := e
	e2.Field = field
	return &e2
}

func withFieldAndJson(e RealtimeError, field string, value any, ent protoreflect.ProtoMessage, msg string, msgArgs ...any) *RealtimeError {
	e2 := e
	e2.Field = field
	if value != nil {
		var err error
		e2.Value, err = tt.ToCsv(value)
		if err != nil {
			log.Error().Err(err).Msgf("could not convert value of type %T to string", value)
		}
	}
	if msg != "" {
		e2.Message = fmt.Sprintf(msg, msgArgs...)
	}
	e2.entityJson = pbEntityToMap(ent)
	return &e2
}

func pbEntityToMap(ent protoreflect.ProtoMessage) tt.Map {
	mOpts := protojson.MarshalOptions{UseProtoNames: true}
	entityJsonBytes, _ := mOpts.Marshal(ent)
	entityJson := map[string]any{}
	if err := json.Unmarshal(entityJsonBytes, &entityJson); err != nil {
		panic(err)
	}
	return tt.NewMap(entityJson)
}

// RealtimeError is a GTFS RealTime error.
type RealtimeError struct {
	bc
	geom       tt.Geometry
	entityJson tt.Map
}

func (e RealtimeError) Geometry() tt.Geometry {
	return e.geom
}

// Return as tt.Map, not map[string]any
func (e RealtimeError) EntityJson() tt.Map {
	return e.entityJson
}

// RealtimeWarning is a GTFS RealTime warning.
type RealtimeWarning struct {
	RealtimeError
}
