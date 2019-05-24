package pathways

import (
	"github.com/interline-io/gotransit"
)

// Pathway pathways.txt
type Pathway struct {
	PathwayID           string  `csv:"pathway_id" required:"true"`
	FromStopID          string  `csv:"from_stop_id" required:"true"`
	ToStopID            string  `csv:"to_stop_id" required:"true"`
	PathwayMode         string  `csv:"pathway_mode" required:"true" min:"1" max:"7"`
	IsBidirectional     int     `csv:"is_bidirectional" required:"true" min:"0" max:"1"`
	Length              float64 `csv:"length" min:"0"`
	TraversalTime       int     `csv:"traversal_time" min:"0"`
	StairCount          int     `csv:"stair_count"`
	MaxSlope            float64 `csv:"max_slope"`
	MinWidth            float64 `csv:"min_width"`
	SignpostedAs        string  `csv:"signposted_as"`
	ReverseSignpostedAs string  `csv:"reversed_signposted_as"`
	gotransit.BaseEntity
}

// Filename pathways.txt
func (ent *Pathway) Filename() string {
	return "pathways.txt"
}

// TableName ext_pathway_pathways
func (ent *Pathway) TableName() string {
	return "ext_pathway_pathways"
}
