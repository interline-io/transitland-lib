package gtfs

import (
	"github.com/interline-io/transitland-lib/tt"
)

// Pathway pathways.txt
type Pathway struct {
	PathwayID           tt.String `csv:",required"`
	FromStopID          tt.String `csv:",required" target:"stops.txt"`
	ToStopID            tt.String `csv:",required" target:"stops.txt"`
	PathwayMode         tt.Int    `csv:",required"`
	IsBidirectional     tt.Int    `csv:",required"`
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
