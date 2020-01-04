package gotransit

import (
	"fmt"

	"github.com/interline-io/gotransit/causes"
	"github.com/interline-io/gotransit/internal/tags"
)

// Stop stops.txt
type Stop struct {
	StopID             string               `csv:"stop_id" required:"true"`
	StopName           string               `csv:"stop_name"` // conditionally required
	StopCode           string               `csv:"stop_code"`
	StopDesc           string               `csv:"stop_desc"`
	StopLat            float64              `db:"-" csv:"stop_lat" min:"-90" max:"90"` // required handled below
	StopLon            float64              `db:"-" csv:"stop_lon" min:"-180" max:"180"`
	ZoneID             string               `csv:"zone_id"`
	StopURL            string               `csv:"stop_url" validator:"url"`
	LocationType       int                  `csv:"location_type" min:"0" max:"4"`
	ParentStation      OptionalRelationship `csv:"parent_station"`
	StopTimezone       string               `csv:"stop_timezone"`
	WheelchairBoarding int                  `csv:"wheelchair_boarding" min:"0" max:"2"`
	LevelID            OptionalRelationship `csv:"level_id"`
	Geometry           Point                `db:"geometry"`
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

// Warnings for this Entity.
func (ent *Stop) Warnings() (errs []error) {
	lat := ent.StopLat
	lon := ent.StopLon
	if ent.Geometry.Valid {
		c := ent.Coordinates()
		lat = c[1]
		lon = c[0]
	}
	if ent.LocationType < 3 {
		if lat == 0 {
			errs = append(errs, causes.NewValidationWarning("stop_lat", "required field stop_lat is 0.0"))
		}
		if lon == 0 {
			errs = append(errs, causes.NewValidationWarning("stop_lon", "required field stop_lon is 0.0"))
		}
	}
	if len(ent.StopDesc) > 0 && ent.StopName == ent.StopDesc {
		errs = append(errs, causes.NewValidationWarning("stop_desc", "stop_desc is the same as stop_name"))
	}
	if !tags.IsValidTimezone(ent.StopTimezone) {
		errs = append(errs, causes.NewValidationWarning("stop_timezone", "stop_timezone is not a valid timezone"))
	}
	return errs
}

// Errors for this Entity.
func (ent *Stop) Errors() (errs []error) {
	errs = ValidateTags(ent)
	errs = append(errs, ent.BaseEntity.loadErrors...)
	if ent.LocationType < 3 && len(ent.StopName) == 0 {
		errs = append(errs, causes.NewConditionallyRequiredFieldError("stop_name"))
	}
	// Check for "0" value...
	if ent.LocationType == 1 && ent.ParentStation.Key != "" {
		errs = append(errs, causes.NewInvalidFieldError("parent_station", "", fmt.Errorf("station cannot have parent_station")))
	}
	if ent.LocationType > 1 && ent.ParentStation.Key == "" {
		errs = append(errs, causes.NewInvalidFieldError("parent_station", "", fmt.Errorf("must have parent_station"))) // ConditionallyRequiredFieldError
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
			ent.LevelID = OptionalRelationship{v, true}
		} else {
			return causes.NewInvalidReferenceError("level_id", ent.LevelID.Key)
		}
	}
	// Adjust ParentStation
	if ent.ParentStation.Key != "" {
		if parentID, ok := emap.GetEntity(&Stop{StopID: ent.ParentStation.Key}); ok {
			ent.ParentStation = OptionalRelationship{parentID, true}
		} else {
			return causes.NewInvalidReferenceError("parent_station", ent.ParentStation.Key)
		}
	}
	return nil
}
