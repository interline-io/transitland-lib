package gtfs

import (
	"fmt"

	"github.com/interline-io/transitland-lib/causes"
	"github.com/interline-io/transitland-lib/tt"
)

type Timeframe struct {
	TimeframeGroupID tt.String  `csv:",required" standardized_sort:"1"`
	StartTime        tt.Seconds `standardized_sort:"2"`
	EndTime          tt.Seconds `standardized_sort:"3"`
	ServiceID        tt.Key     `csv:",required" target:"calendar.txt" standardized_sort:"4"`
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
