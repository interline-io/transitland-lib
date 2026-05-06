package gtfs

import (
	"github.com/interline-io/transitland-lib/tt"
)

// FareMedia fare_media.txt
type FareMedia struct {
	FareMediaID   tt.String `csv:",required" standardized_sort:"1"`
	FareMediaName tt.String `csv:",required"`
	FareMediaType tt.Int    `csv:",required" enum:"0,1,2,3,4"`
	tt.BaseEntity
}

func (ent *FareMedia) EntityKey() string {
	return ent.FareMediaID.Val
}

func (ent *FareMedia) EntityID() string {
	return ent.FareMediaID.Val
}

func (ent *FareMedia) Filename() string {
	return "fare_media.txt"
}

func (ent *FareMedia) TableName() string {
	return "gtfs_fare_media"
}
