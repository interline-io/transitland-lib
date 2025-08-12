package gtfs

import (
	"fmt"

	"github.com/interline-io/transitland-lib/causes"
	"github.com/interline-io/transitland-lib/tt"
)

// Pathway pathways.txt
type Pathway struct {
	PathwayID           tt.String `csv:",required"`
	FromStopID          tt.String `csv:",required" target:"stops.txt"`
	ToStopID            tt.String `csv:",required" target:"stops.txt"`
	PathwayMode         tt.Int    `csv:",required" enum:"1,2,3,4,5,6,7"`
	IsBidirectional     tt.Int    `csv:",required" enum:"0,1"`
	Length              tt.Float
	TraversalTime       tt.Int
	StairCount          tt.Int
	MaxSlope            tt.Float
	MinWidth            tt.Float
	SignpostedAs        tt.String
	ReverseSignpostedAs tt.String
	tt.BaseEntity
}

// EntityID returns the ID or StopID.
func (ent *Pathway) EntityID() string {
	return entID(ent.ID, ent.PathwayID.Val)
}

// EntityKey returns the GTFS identifier.
func (ent *Pathway) EntityKey() string {
	return ent.PathwayID.Val
}

// Filename pathways.txt
func (ent *Pathway) Filename() string {
	return "pathways.txt"
}

// TableName ext_pathway_pathways
func (ent *Pathway) TableName() string {
	return "gtfs_pathways"
}

// ConditionalErrors returns validation errors for the Pathway entity.
func (ent *Pathway) ConditionalErrors() []error {
	var errs []error
	if ent.Length.Valid && ent.Length.Float() < 0 {
		errs = append(errs, causes.NewInvalidFieldError("length", ent.Length.String(), fmt.Errorf("must be non-negative when specified")))
	}
	if ent.MinWidth.Valid && ent.MinWidth.Float() <= 0 {
		errs = append(errs, causes.NewInvalidFieldError("min_width", ent.MinWidth.String(), fmt.Errorf("must be positive when specified")))
	}
	if ent.StairCount.Valid && ent.StairCount.Int() < 0 {
		errs = append(errs, causes.NewInvalidFieldError("stair_count", ent.StairCount.String(), fmt.Errorf("must be non-negative when specified")))
	}
	if ent.MaxSlope.Valid && ent.PathwayMode.Int() != 1 && ent.PathwayMode.Int() != 3 {
		errs = append(errs, causes.NewInvalidFieldError("max_slope", ent.MaxSlope.String(), fmt.Errorf("should only be used with walkways (pathway_mode=1) and moving sidewalks (pathway_mode=3)")))
	}
	if ent.PathwayMode.Int() == 7 && ent.IsBidirectional.Int() == 1 {
		errs = append(errs, causes.NewInvalidFieldError("is_bidirectional", ent.IsBidirectional.String(), fmt.Errorf("exit gates (pathway_mode=7) must not be bidirectional")))
	}
	return errs
}
