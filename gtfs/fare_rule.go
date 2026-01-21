package gtfs

import (
	"fmt"

	"github.com/interline-io/transitland-lib/tt"
)

// FareRule fare_rules.txt
type FareRule struct {
	FareID        tt.String `csv:",required" target:"fare_attributes.txt" standardized_sort:"1"`
	RouteID       tt.Key    `target:"routes.txt" standardized_sort:"2"`
	OriginID      tt.String `standardized_sort:"3"`
	DestinationID tt.String `standardized_sort:"4"`
	ContainsID    tt.String `standardized_sort:"5"`
	tt.BaseEntity
}

// Filename fare_rules.txt
func (ent *FareRule) Filename() string {
	return "fare_rules.txt"
}

// TableName gtfs_fare_Rules
func (ent *FareRule) TableName() string {
	return "gtfs_fare_rules"
}

func (ent *FareRule) DuplicateKey() string {
	return fmt.Sprintf(
		"fare_id:'%s' route_id:'%s' origin_id:'%s' destination_id:'%s' contains_id:'%s'",
		ent.FareID.Val,
		ent.RouteID.Val,
		ent.OriginID.Val,
		ent.DestinationID.Val,
		ent.ContainsID.Val,
	)
}
