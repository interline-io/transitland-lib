package gtfs

import (
	"fmt"

	"github.com/interline-io/transitland-lib/causes"
	"github.com/interline-io/transitland-lib/tlxy"
	"github.com/interline-io/transitland-lib/tt"
)

// Stop stops.txt
type Stop struct {
	StopID             tt.String `csv:",required" required:"true"`
	StopName           tt.String
	StopCode           tt.String
	StopDesc           tt.String
	StopLat            tt.Float `db:"-" range:"-90,90"` // csv load to Geometry
	StopLon            tt.Float `db:"-" range:"-180,180"`
	ZoneID             tt.String
	StopURL            tt.Url
	TtsStopName        tt.String
	PlatformCode       tt.String
	LocationType       tt.Int `enum:"0,1,2,3,4"`
	ParentStation      tt.Key `target:"stops.txt"`
	StopTimezone       tt.Timezone
	WheelchairBoarding tt.Int   `enum:"0,1,2"`
	LevelID            tt.Key   `target:"levels.txt"`
	Geometry           tt.Point `csv:"-" db:"geometry"`
	tt.BaseEntity
}

// EntityID returns the ID or StopID.
func (ent *Stop) EntityID() string {
	return entID(ent.ID, ent.StopID.Val)
}

// EntityKey returns the GTFS identifier.
func (ent *Stop) EntityKey() string {
	return ent.StopID.Val
}

// Filename stops.txt
func (ent *Stop) Filename() string {
	return "stops.txt"
}

// TableName gtfs_stops
func (ent *Stop) TableName() string {
	return "gtfs_stops"
}

// SetCoordinates takes a [2]float64 and sets the Stop's lon,lat
func (ent *Stop) SetCoordinates(p [2]float64) {
	ent.Geometry = tt.NewPoint(p[0], p[1])
}

// Coordinates returns the stop lon,lat as a [2]float64
func (ent *Stop) Coordinates() [2]float64 {
	ret := [2]float64{0, 0}
	if ent.Geometry.Val == nil {
		return ret
	}
	c := ent.Geometry.FlatCoords()
	if len(c) != 2 {
		return ret
	}
	ret[0] = c[0]
	ret[1] = c[1]
	return ret
}

func (ent *Stop) ToPoint() tlxy.Point {
	return ent.Geometry.ToPoint()
}

func (ent *Stop) ConditionalErrors() []error {
	var errs []error
	// TODO: This should be an enum for exhaustive search
	lt := ent.LocationType.Val
	if lt == 0 || lt == 1 || lt == 2 {
		if len(ent.StopName.Val) == 0 {
			errs = append(errs, causes.NewConditionallyRequiredFieldError("stop_name"))
		}
		if !ent.StopLon.Valid {
			errs = append(errs, causes.NewConditionallyRequiredFieldError("stop_lon"))
		}
		if !ent.StopLat.Valid {
			errs = append(errs, causes.NewConditionallyRequiredFieldError("stop_lat"))
		}
	}

	// Check for "0" value...
	if lt == 1 && ent.ParentStation.Val != "" {
		errs = append(errs, causes.NewInvalidFieldError("parent_station", ent.ParentStation.Val, fmt.Errorf("station cannot have parent_station")))
	}
	if (lt == 2 || lt == 3 || lt == 4) && ent.ParentStation.Val == "" {
		errs = append(errs, causes.NewConditionallyRequiredFieldError("parent_station"))
	}
	return errs
}
