package gtfs

import "github.com/interline-io/transitland-lib/tt"

type Timeframe struct {
	TimeframeGroupID tt.String
	StartTime        tt.Seconds
	EndTime          tt.Seconds
	ServiceID        tt.Key `target:"calendar.txt"`
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
