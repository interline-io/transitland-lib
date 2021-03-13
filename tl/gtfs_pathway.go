package tl

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/interline-io/transitland-lib/tl/causes"
)

// Pathway pathways.txt
type Pathway struct {
	PathwayID           string  `csv:"pathway_id,required"`
	FromStopID          string  `csv:"from_stop_id,required"`
	ToStopID            string  `csv:"to_stop_id,required"`
	PathwayMode         int     `csv:"pathway_mode,required"`
	IsBidirectional     int     `csv:"is_bidirectional,required"`
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

// EntityKey returns the GTFS identifier.
func (ent *Pathway) EntityKey() string {
	return ent.PathwayID
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

// GetString returns the string representation of an field.
func (ent *Pathway) GetString(key string) (string, error) {
	v := ""
	switch key {
	case "pathway_id":
		v = ent.PathwayID
	case "from_stop_id":
		v = ent.FromStopID
	case "to_stop_id":
		v = ent.ToStopID
	case "pathway_mode":
		v = strconv.Itoa(ent.PathwayMode)
	case "is_bidirectional":
		v = strconv.Itoa(ent.IsBidirectional)
	case "length":
		if ent.Length > 0 {
			v = fmt.Sprintf("%0.5f", ent.Length)
		}
	case "traversal_time":
		if ent.TraversalTime > 0 {
			v = strconv.Itoa(ent.TraversalTime)
		}
	case "stair_count":
		if ent.StairCount != 0 && ent.StairCount != -1 {
			v = strconv.Itoa(ent.StairCount)
		}
	case "max_slope":
		if ent.MaxSlope != 0 {
			v = fmt.Sprintf("%0.2f", ent.MaxSlope)
		}
	case "min_width":
		if ent.MinWidth != 0 {
			v = fmt.Sprintf("%0.2f", ent.MinWidth)
		}
	case "signposted_as":
		v = ent.SignpostedAs
	case "reversed_signposted_as":
		v = ent.ReverseSignpostedAs
	default:
		return v, errors.New("unknown key")
	}
	return v, nil
}
