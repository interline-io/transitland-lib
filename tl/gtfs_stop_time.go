package tl

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"

	"github.com/interline-io/transitland-lib/tl/causes"
	"github.com/interline-io/transitland-lib/tl/enum"
)

// StopTime stop_times.txt
type StopTime struct {
	TripID            string          `csv:"trip_id"`
	ArrivalTime       int             `csv:"arrival_time" `
	DepartureTime     int             `csv:"departure_time" `
	StopID            string          `csv:"stop_id" required:"true"`
	StopSequence      int             `csv:"stop_sequence" required:"true" min:"0"`
	StopHeadsign      sql.NullString  `csv:"stop_headsign"`
	PickupType        sql.NullInt32   `csv:"pickup_type" min:"0" max:"3"`
	DropOffType       sql.NullInt32   `csv:"drop_off_type" min:"0" max:"3"`
	ShapeDistTraveled sql.NullFloat64 `csv:"shape_dist_traveled" min:"0"`
	Timepoint         sql.NullInt32   `csv:"timepoint" min:"0" max:"1"`
	Interpolated      sql.NullInt32   // interpolated times: 0 for provided, 1 interpolated // TODO: 1 for shape, 2 for straight-line
	FeedVersionID     int
	extra             []string
	loadErrors        []error
	loadWarnings      []error
}

// SetID sets the integer ID.
func (ent *StopTime) SetID(id int) {
}

// SetFeedVersionID sets the Entity's FeedVersionID.
func (ent *StopTime) SetFeedVersionID(fvid int) {
	ent.FeedVersionID = fvid
}

// AddError adds a loading error to the entity, e.g. from a CSV parse failure
func (ent *StopTime) AddError(err error) {
	ent.loadErrors = append(ent.loadErrors, err)
}

// AddWarning .
func (ent *StopTime) AddWarning(err error) {
	ent.loadWarnings = append(ent.loadErrors, err)
}

// Extra provides any additional fields that were present.
func (ent *StopTime) Extra() map[string]string {
	ret := map[string]string{}
	for i := 0; i < len(ent.extra); i += 2 {
		ret[ent.extra[i]] = ent.extra[i+1]
	}
	return ret
}

// SetExtra adds a string key, value pair to the entity's extra fields.
func (ent *StopTime) SetExtra(key string, value string) {
	ent.extra = append(ent.extra, key, value)
}

// EntityID returns nothing.
func (ent *StopTime) EntityID() string {
	return ""
}

// Warnings returns validation warnings.
func (ent *StopTime) Warnings() []error { return ent.loadWarnings }

// Errors for this Entity.
func (ent *StopTime) Errors() []error {
	// No reflection
	errs := []error{}
	errs = append(errs, ent.loadErrors...)
	errs = append(errs, enum.CheckPresent("trip_id", ent.TripID)...)
	errs = append(errs, enum.CheckPresent("stop_id", ent.StopID)...)
	errs = append(errs, enum.CheckPositiveInt("stop_sequence", ent.StopSequence)...)
	errs = append(errs, enum.CheckInsideRangeInt("pickup_type", int(ent.PickupType.Int32), 0, 3)...)
	errs = append(errs, enum.CheckInsideRangeInt("drop_off_type", int(ent.DropOffType.Int32), 0, 3)...)
	errs = append(errs, enum.CheckPositive("shape_dist_traveled", ent.ShapeDistTraveled.Float64)...)
	errs = append(errs, enum.CheckInsideRangeInt("timepoint", int(ent.Timepoint.Int32), -1, 1)...)
	errs = append(errs, enum.CheckInsideRangeInt("arrival_time", ent.ArrivalTime, -1, 1<<31)...)
	errs = append(errs, enum.CheckInsideRangeInt("departure", ent.DepartureTime, -1, 1<<31)...)
	// Other errors
	at, dt := ent.ArrivalTime, ent.DepartureTime
	if at != 0 && dt != 0 && at > dt {
		errs = append(errs, causes.NewInvalidFieldError("departure_time", "", fmt.Errorf("departure_time '%d' must come after arrival_time '%d'", dt, at)))
	}
	return errs
}

// Filename stop_times.txt
func (ent *StopTime) Filename() string {
	return "stop_times.txt"
}

// TableName gtfs_stop_times
func (ent *StopTime) TableName() string {
	return "gtfs_stop_times"
}

// UpdateKeys updates Entity references.
func (ent *StopTime) UpdateKeys(emap *EntityMap) error {
	if tripID, ok := emap.GetEntity(&Trip{TripID: ent.TripID}); ok {
		ent.TripID = tripID
	} else {
		return causes.NewInvalidReferenceError("trip_id", ent.TripID)
	}
	if stopID, ok := emap.GetEntity(&Stop{StopID: ent.StopID}); ok {
		ent.StopID = stopID
	} else {
		return causes.NewInvalidReferenceError("stop_id", ent.StopID)
	}
	return nil
}

// GetString returns the string representation of an field.
func (ent *StopTime) GetString(key string) (string, error) {
	v := ""
	switch key {
	case "trip_id":
		v = ent.TripID
	case "stop_headsign":
		v = ent.StopHeadsign.String
	case "stop_id":
		v = ent.StopID
	case "arrival_time":
		v = SecondsToString(ent.ArrivalTime)
	case "departure_time":
		v = SecondsToString(ent.DepartureTime)
	case "stop_sequence":
		v = strconv.Itoa(ent.StopSequence)
	case "pickup_type":
		v = strconv.Itoa(int(ent.PickupType.Int32))
	case "drop_off_type":
		v = strconv.Itoa(int(ent.DropOffType.Int32))
	case "shape_dist_traveled":
		if ent.ShapeDistTraveled.Valid {
			v = fmt.Sprintf("%0.5f", ent.ShapeDistTraveled.Float64)
		}
	case "timepoint":
		if ent.Timepoint.Valid {
			v = strconv.Itoa(int(ent.Timepoint.Int32))
		}
	default:
		return v, errors.New("unknown key")
	}
	return v, nil
}

// SetString provides a fast, non-reflect loading path.
func (ent *StopTime) SetString(key, value string) error {
	var perr error
	hi := value
	switch key {
	case "trip_id":
		ent.TripID = hi
	case "stop_headsign":
		ent.StopHeadsign = sql.NullString{Valid: true, String: hi}
	case "stop_id":
		ent.StopID = hi
	case "arrival_time":
		if hi == "" {
			ent.ArrivalTime = -1
		} else if s, err := StringToSeconds(hi); err != nil {
			perr = causes.NewFieldParseError("arrival_time", hi)
		} else {
			ent.ArrivalTime = s
		}
	case "departure_time":
		if hi == "" {
			ent.DepartureTime = -1
		} else if s, err := StringToSeconds(hi); err != nil {
			perr = causes.NewFieldParseError("departure_time", hi)
		} else {
			ent.DepartureTime = s
		}
	case "stop_sequence":
		if a, err := strconv.Atoi(hi); err != nil {
			perr = causes.NewFieldParseError("stop_sequence", hi)
		} else {
			ent.StopSequence = a
		}
	case "pickup_type":
		if len(hi) == 0 {
			ent.PickupType = sql.NullInt32{Valid: false}
		} else if a, err := strconv.Atoi(hi); err != nil {
			perr = causes.NewFieldParseError("pickup_type", hi)
		} else {
			ent.PickupType = sql.NullInt32{Valid: true, Int32: int32(a)}
		}
	case "drop_off_type":
		if len(hi) == 0 {
			ent.DropOffType = sql.NullInt32{Valid: false}
		} else if a, err := strconv.Atoi(hi); err != nil {
			perr = causes.NewFieldParseError("drop_off_type", hi)
		} else {
			ent.DropOffType = sql.NullInt32{Valid: true, Int32: int32(a)}
		}
	case "shape_dist_traveled":
		if len(hi) == 0 {
			ent.ShapeDistTraveled = sql.NullFloat64{Float64: 0, Valid: false}
		} else if a, err := strconv.ParseFloat(hi, 64); err != nil {
			perr = causes.NewFieldParseError("shape_dist_traveled", hi)
		} else {
			ent.ShapeDistTraveled = sql.NullFloat64{Float64: a, Valid: true}
		}
	case "timepoint":
		// special use -1 for empty timepoint value
		if len(hi) == 0 {
			ent.Timepoint = sql.NullInt32{Valid: false}
		} else if a, err := strconv.Atoi(hi); err != nil {
			perr = causes.NewFieldParseError("timepoint", hi)
		} else {
			ent.Timepoint = sql.NullInt32{Valid: true, Int32: int32(a)}
		}
	default:
		ent.SetExtra(key, hi)
	}
	return perr
}

// ValidateStopTimes checks if the trip follows GTFS rules.
func ValidateStopTimes(stoptimes []StopTime) []error {
	errs := []error{}
	if len(stoptimes) == 0 {
		errs = append(errs, causes.NewEmptyTripError(len(stoptimes)))
		return errs // assumes >= 1 below
	}
	if len(stoptimes) < 2 {
		errs = append(errs, causes.NewEmptyTripError(len(stoptimes)))
	}
	if stoptimes[len(stoptimes)-1].ArrivalTime <= 0 {
		errs = append(errs, causes.NewSequenceError("arrival_time", ""))
	}
	lastDist := stoptimes[0].ShapeDistTraveled
	lastTime := stoptimes[0].DepartureTime
	lastSequence := stoptimes[0].StopSequence
	for _, st := range stoptimes[1:] {
		// Ensure we do not have duplicate StopSequennce
		if st.StopSequence == lastSequence {
			errs = append(errs, causes.NewSequenceError("stop_sequence", strconv.Itoa(st.StopSequence)))
		} else {
			lastSequence = st.StopSequence
		}
		// Ensure the arrows of time are pointing towards the future.
		if st.ArrivalTime > 0 && st.ArrivalTime < lastTime {
			errs = append(errs, causes.NewSequenceError("arrival_time", strconv.Itoa(st.ArrivalTime)))
		} else if st.DepartureTime > 0 && st.DepartureTime < st.ArrivalTime {
			errs = append(errs, causes.NewSequenceError("departure_time", strconv.Itoa(st.DepartureTime)))
		} else if st.DepartureTime > 0 {
			lastTime = st.DepartureTime
		}
		if st.ShapeDistTraveled.Float64 > 0 && st.ShapeDistTraveled.Float64 < lastDist.Float64 {
			errs = append(errs, causes.NewSequenceError("shape_dist_traveled", fmt.Sprintf("%f", st.ShapeDistTraveled.Float64)))
		} else if st.ShapeDistTraveled.Float64 > 0 {
			lastDist = st.ShapeDistTraveled
		}
	}
	return errs
}
