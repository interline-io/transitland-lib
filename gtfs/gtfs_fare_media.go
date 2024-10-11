package gtfs

import (
	"github.com/interline-io/transitland-lib/tt"
)

// FareMedia fare_media.txt
type FareMedia struct {
	FareMediaID   tt.String
	FareMediaName tt.String
	FareMediaType tt.Int
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

func (ent *FareMedia) Errors() (errs []error) {
	errs = append(errs, ent.BaseEntity.Errors()...)
	errs = append(errs, tt.CheckPresent("fare_media_id", ent.FareMediaID.Val)...)
	errs = append(errs, tt.CheckPresent("fare_media_name", ent.FareMediaName.Val)...)
	errs = append(errs, tt.CheckInsideRangeInt("fare_media_type", ent.FareMediaType.Val, 0, 4)...)
	return errs
}
