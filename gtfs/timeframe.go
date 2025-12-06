package gtfs

import (
	"fmt"

	"github.com/interline-io/transitland-lib/causes"
	"github.com/interline-io/transitland-lib/tt"
)

type Timeframe struct {
	TimeframeGroupID tt.String `csv:",required"`
	StartTime        tt.Seconds
	EndTime          tt.Seconds
	ServiceID        tt.Key `csv:",required" target:"calendar.txt"`
	tt.BaseEntity
}

func (ent *Timeframe) GroupKey() (string, string) {
	return "timeframe_group_id", ent.TimeframeGroupID.Val
}

func (ent *Timeframe) Filename() string {
	return "timeframes.txt"
}

func (ent *Timeframe) TableName() string {
	return "gtfs_timeframes"
}

// Errors for this Entity.
func (ent *Timeframe) ConditionalErrors() (errs []error) {
	if ent.StartTime.IsValid() && ent.EndTime.IsValid() && ent.StartTime.Int() > ent.EndTime.Int() {
		errs = append(errs, causes.NewInvalidFieldError("end_time", fmt.Sprintf("%d", ent.EndTime.Int()), fmt.Errorf("end_time must be greater than or equal to start_time")))
	}
	return errs
}
