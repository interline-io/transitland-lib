package plus

import (
	tl "github.com/interline-io/transitland-lib"
	"github.com/interline-io/transitland-lib/causes"
)

// StopAttribute stop_attributes.txt
type StopAttribute struct {
	StopID            string `csv:"stop_id"`
	AccessibilityID   int    `csv:"accessibility_id"`
	CardinalDirection string `csv:"cardinal_direction"`
	RelativePosition  string `csv:"relative_position"`
	StopCity          string `csv:"stop_city"`
	tl.BaseEntity
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
func (ent *StopAttribute) UpdateKeys(emap *tl.EntityMap) error {
	if fkid, ok := emap.GetEntity(&tl.Stop{StopID: ent.StopID}); ok {
		ent.StopID = fkid
	} else {
		return causes.NewInvalidReferenceError("stop_id", ent.StopID)
	}
	return nil
}
