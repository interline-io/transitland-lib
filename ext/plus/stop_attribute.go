package plus

import (
	"github.com/interline-io/transitland-lib/causes"
	"github.com/interline-io/transitland-lib/gtfs"
	"github.com/interline-io/transitland-lib/tt"
)

// StopAttribute stop_attributes.txt
type StopAttribute struct {
	StopID            string `csv:"stop_id"`
	AccessibilityID   int    `csv:"accessibility_id"`
	CardinalDirection string `csv:"cardinal_direction"`
	RelativePosition  string `csv:"relative_position"`
	StopCity          string `csv:"stop_city"`
	tt.BaseEntity
}

// Filename stop_attributes.txt
func (ent *StopAttribute) Filename() string {
	return "stop_attributes.txt"
}

// TableName ext_plus_stop_attributes
func (ent *StopAttribute) TableName() string {
	return "ext_plus_stop_attributes"
}

// UpdateKeys updates Entity references.
func (ent *StopAttribute) UpdateKeys(emap *tt.EntityMap) error {
	if fkid, ok := emap.GetEntity(&gtfs.Stop{StopID: tt.NewString(ent.StopID)}); ok {
		ent.StopID = fkid
	} else {
		return causes.NewInvalidReferenceError("stop_id", ent.StopID)
	}
	return nil
}
