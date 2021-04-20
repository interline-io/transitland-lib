package tl

import (
	"fmt"

	"github.com/interline-io/transitland-lib/tl/causes"
	"github.com/interline-io/transitland-lib/tl/enum"
)

// Stop stops.txt
type Stop struct {
	StopID             string `csv:",required" required:"true"`
	StopName           string
	StopCode           string
	StopDesc           string
	StopLat            float64 `db:"-"` // csv load to Geometry
	StopLon            float64 `db:"-"`
	ZoneID             string
	StopURL            string
	LocationType       int
	ParentStation      OKey
	StopTimezone       string
	WheelchairBoarding int
	LevelID            OKey
	Geometry           Point  `csv:"-" db:"geometry"`
	AreaID             string `db:"-"`
	BaseEntity
}

// SetCoordinates takes a [2]float64 and sets the Stop's lon,lat
func (ent *Stop) SetCoordinates(p [2]float64) {
	ent.Geometry = NewPoint(p[0], p[1])
}

// Coordinates returns the stop lon,lat as a [2]float64
func (ent *Stop) Coordinates() [2]float64 {
	ret := [2]float64{0, 0}
	c := ent.Geometry.FlatCoords()
	if len(c) != 2 {
		return ret
	}
	ret[0] = c[0]
	ret[1] = c[1]
	return ret
}

// EntityID returns the ID or StopID.
func (ent *Stop) EntityID() string {
	return entID(ent.ID, ent.StopID)
}

// EntityKey returns the GTFS identifier.
func (ent *Stop) EntityKey() string {
	return ent.StopID
}

// Errors for this Entity.
func (ent *Stop) Errors() (errs []error) {
	c := ent.Coordinates()
	lat := c[1]
	lon := c[0]
	errs = append(errs, ent.BaseEntity.Errors()...)
	errs = append(errs, enum.CheckPresent("stop_id", ent.StopID)...)
	errs = append(errs, enum.CheckInsideRange("stop_lat", lat, -90.0, 90.0)...)
	errs = append(errs, enum.CheckInsideRange("stop_lon", lon, -180.0, 180.0)...)
	errs = append(errs, enum.CheckURL("stop_url", ent.StopURL)...)
	errs = append(errs, enum.CheckInsideRangeInt("location_type", ent.LocationType, 0, 4)...)
	errs = append(errs, enum.CheckInsideRangeInt("wheelchair_boarding", ent.WheelchairBoarding, 0, 2)...)
	// TODO: This should be an enum for exhaustive search
	lt := ent.LocationType
	if (lt == 0 || lt == 1 || lt == 2) && len(ent.StopName) == 0 {
		errs = append(errs, causes.NewConditionallyRequiredFieldError("stop_name"))
	}
	// Check for "0" value...
	if lt == 1 && ent.ParentStation.Key != "" {
		errs = append(errs, causes.NewInvalidFieldError("parent_station", "", fmt.Errorf("station cannot have parent_station")))
	}
	if (lt == 2 || lt == 3 || lt == 4) && ent.ParentStation.Key == "" {
		errs = append(errs, causes.NewConditionallyRequiredFieldError("parent_station"))
	}
	return errs
}

// Filename stops.txt
func (ent *Stop) Filename() string {
	return "stops.txt"
}

// TableName gtfs_stops
func (ent *Stop) TableName() string {
	return "gtfs_stops"
}

// UpdateKeys updates Entity references.
func (ent *Stop) UpdateKeys(emap *EntityMap) error {
	// Pathway Level
	if ent.LevelID.Key != "" {
		if v, ok := emap.GetEntity(&Level{LevelID: ent.LevelID.Key}); ok {
			ent.LevelID = NewOKey(v)
		} else {
			return causes.NewInvalidReferenceError("level_id", ent.LevelID.Key)
		}
	}
	// Adjust ParentStation
	if ent.ParentStation.Key != "" {
		if parentID, ok := emap.GetEntity(&Stop{StopID: ent.ParentStation.Key}); ok {
			ent.ParentStation = NewOKey(parentID)
		} else {
			return causes.NewInvalidReferenceError("parent_station", ent.ParentStation.Key)
		}
	}
	return nil
}
