package gtfs

import (
	"fmt"

	"github.com/interline-io/transitland-lib/causes"
	"github.com/interline-io/transitland-lib/tt"
)

// FareLegJoinRule fare_leg_join_rules.txt
type FareLegJoinRule struct {
	FromNetworkID tt.String `csv:",required" target:"networks.txt" standardized_sort:"1"`
	ToNetworkID   tt.String `csv:",required" target:"networks.txt" standardized_sort:"2"`
	FromStopID    tt.String `target:"stops.txt" standardized_sort:"3"`
	ToStopID      tt.String `target:"stops.txt" standardized_sort:"4"`
	tt.BaseEntity
}

func (ent *FareLegJoinRule) Filename() string {
	return "fare_leg_join_rules.txt"
}

func (ent *FareLegJoinRule) TableName() string {
	return "gtfs_fare_leg_join_rules"
}

// ConditionalErrors enforces that from_stop_id and to_stop_id are present together.
func (ent *FareLegJoinRule) ConditionalErrors() (errs []error) {
	if ent.FromStopID.Valid && !ent.ToStopID.Valid {
		errs = append(errs, causes.NewConditionallyRequiredFieldError("to_stop_id"))
	} else if ent.ToStopID.Valid && !ent.FromStopID.Valid {
		errs = append(errs, causes.NewConditionallyRequiredFieldError("from_stop_id"))
	}
	return errs
}

// DuplicateKey returns the composite primary key for duplicate detection.
func (ent *FareLegJoinRule) DuplicateKey() string {
	return fmt.Sprintf(
		"from_network_id:'%s' to_network_id:'%s' from_stop_id:'%s' to_stop_id:'%s'",
		ent.FromNetworkID.Val,
		ent.ToNetworkID.Val,
		ent.FromStopID.Val,
		ent.ToStopID.Val,
	)
}
