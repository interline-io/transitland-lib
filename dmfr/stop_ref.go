package dmfr

import "github.com/interline-io/transitland-lib/tt"

type StopExternalReference struct {
	StopID              tt.Key
	TargetFeedOnestopID tt.String
	TargetStopID        tt.String
	Inactive            tt.Bool
	tt.BaseEntity
}

func (ent *StopExternalReference) TableName() string {
	return "tl_stop_external_references"
}
