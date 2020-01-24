package gotransit

import (
	"github.com/interline-io/gotransit/causes"
)

// Pathway pathways.txt
type Pathway struct {
	PathwayID           string  `csv:"pathway_id" required:"true"`
	FromStopID          string  `csv:"from_stop_id" required:"true"`
	ToStopID            string  `csv:"to_stop_id" required:"true"`
	PathwayMode         int     `csv:"pathway_mode" required:"true" min:"1" max:"7"`
	IsBidirectional     int     `csv:"is_bidirectional" required:"true" min:"0" max:"1"`
	Length              float64 `csv:"length" min:"0"`
	TraversalTime       int     `csv:"traversal_time" min:"0"`
	StairCount          int     `csv:"stair_count"`
	MaxSlope            float64 `csv:"max_slope"`
	MinWidth            float64 `csv:"min_width"`
	SignpostedAs        string  `csv:"signposted_as"`
	ReverseSignpostedAs string  `csv:"reversed_signposted_as"`
	BaseEntity
}

// EntityID returns the ID or StopID.
func (ent *Pathway) EntityID() string {
	return entID(ent.ID, ent.PathwayID)
}

// Filename pathways.txt
func (ent *Pathway) Filename() string {
	return "pathways.txt"
}

// TableName ext_pathway_pathways
func (ent *Pathway) TableName() string {
	return "gtfs_pathways"
}

// UpdateKeys updates Entity references.
func (ent *Pathway) UpdateKeys(emap *EntityMap) error {
	if fkid, ok := emap.GetEntity(&Stop{StopID: ent.FromStopID}); ok {
		ent.FromStopID = fkid
	} else {
		return causes.NewInvalidReferenceError("from_stop_id", ent.FromStopID)
	}
	if fkid, ok := emap.GetEntity(&Stop{StopID: ent.ToStopID}); ok {
		ent.ToStopID = fkid
	} else {
		return causes.NewInvalidReferenceError("to_stop_id", ent.ToStopID)
	}
	return nil
}
