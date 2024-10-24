package gtfs

import (
	"github.com/interline-io/transitland-lib/causes"
	"github.com/interline-io/transitland-lib/tt"
)

// FareProduct fare_products.txt
type FareProduct struct {
	FareProductID   tt.String `csv:",required" `
	FareProductName tt.String
	Amount          tt.CurrencyAmount `csv:",required"`
	Currency        tt.Currency       `csv:",required"`
	DurationStart   tt.Int            `enum:"0,1"`                                      // proposed extension
	DurationAmount  tt.Float          `range:"0,"`                                      // proposed extension
	DurationUnit    tt.Int            `enum:"0,1,2,3,4,5,6"`                            // proposed extension
	DurationType    tt.Int            `enum:"1,2"`                                      // proposed extension
	RiderCategoryID tt.Key            `target:"rider_categories.txt:rider_category_id"` // proposed extension
	FareMediaID     tt.Key            `target:"fare_media.txt"`                         // proposed extension
	tt.BaseEntity
}

func (ent *FareProduct) EntityID() string {
	return ent.FareProductID.Val
}

func (ent *FareProduct) Filename() string {
	return "fare_products.txt"
}

func (ent *FareProduct) TableName() string {
	return "gtfs_fare_products"
}

func (ent *FareProduct) GroupKey() (string, string) {
	return "fare_product_id", ent.FareProductID.Val
}

func (ent *FareProduct) GetValue(key string) (any, bool) {
	switch key {
	case "amount":
		ent.Amount.SetCurrency(ent.Currency.Val)
		return ent.Amount, true
	}
	return nil, false
}

func (ent *FareProduct) ConditionalErrors() (errs []error) {
	if ent.DurationAmount.Valid && !ent.DurationType.Valid {
		errs = append(errs, causes.NewConditionallyRequiredFieldError("duration_type"))
	}
	if ent.DurationType.Valid && !ent.DurationAmount.Valid {
		errs = append(errs, causes.NewConditionallyRequiredFieldError("duration_amount"))
	}
	return errs
}
