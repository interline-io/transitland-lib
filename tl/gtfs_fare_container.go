package tl

import (
	"github.com/interline-io/transitland-lib/tl/causes"
	"github.com/interline-io/transitland-lib/tl/tt"
)

// FareContainer fare_containers.txt
type FareContainer struct {
	FareContainerID        String
	FareContainerName      String
	Amount                 Float
	MinimumInitialPurchase Float
	Currency               String
	BaseEntity
}

func (ent *FareContainer) EntityKey() string {
	return ent.FareContainerID.Val
}

func (ent *FareContainer) EntityID() string {
	return ent.FareContainerID.Val
}

func (ent *FareContainer) Filename() string {
	return "fare_containers.txt"
}

func (ent *FareContainer) TableName() string {
	return "gtfs_fare_containers"
}

func (ent *FareContainer) Errors() (errs []error) {
	errs = append(errs, ent.BaseEntity.Errors()...)
	errs = append(errs, tt.CheckPositive("amount", ent.Amount.Val)...)
	errs = append(errs, tt.CheckPresent("fare_container_id", ent.FareContainerID.Val)...)
	errs = append(errs, tt.CheckPresent("fare_container_name", ent.FareContainerName.Val)...)
	if ent.Currency.Val == "" && !ent.Amount.Valid {
		errs = append(errs, causes.NewConditionallyRequiredFieldError("currency"))
	}
	if ent.Currency.Val != "" && ent.Amount.Valid && ent.MinimumInitialPurchase.Valid {
		errs = append(errs, causes.NewConditionallyForbiddenFieldError("currency", ""))
	}
	return errs
}
