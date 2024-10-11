package plus

import (
	"github.com/interline-io/transitland-lib/tl"
	"github.com/interline-io/transitland-lib/tl/causes"
	"github.com/interline-io/transitland-lib/tl/tt"
)

// Timepoint timepoints.txt
type Timepoint struct {
	TripID string `csv:"trip_id"`
	StopID string `csv:"stop_id"`
	tt.BaseEntity
}

// Filename timepoints.txt
func (ent *Timepoint) Filename() string {
	return "timepoints.txt"
}

// TableName ext_plus_timepoints
func (ent *Timepoint) TableName() string {
	return "ext_plus_timepoints"
}

// UpdateKeys updates Entity references.
func (ent *Timepoint) UpdateKeys(emap *tt.EntityMap) error {
	if fkid, ok := emap.GetEntity(&tl.Stop{StopID: ent.StopID}); ok {
		ent.StopID = fkid
	} else {
		return causes.NewInvalidReferenceError("stop_id", ent.StopID)
	}
	return nil
}
