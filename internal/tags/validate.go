package tags

import (
	"fmt"
	"reflect"

	"github.com/interline-io/gotransit/causes"
	"github.com/interline-io/gotransit/enums"
)

// ValidateTags returns a validation report using validators defined in struct tags.
func ValidateTags(ent interface{}) (errs []error) {
	m := GetStructTagMap(ent)
	val := reflect.ValueOf(ent)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}
	// for fidx, k := range m {
	for _, k := range m {
		if len(k.Csv) == 0 {
			continue
		}
		valueField := val.Field(k.Index)
		strv := valueField.String()
		// range validators
		switch valueField.Interface().(type) {
		case string:
			v := valueField.String()
			if k.Required && len(v) == 0 {
				errs = append(errs, causes.NewRequiredFieldError(k.Csv))
			}
		case int:
			v := float64(valueField.Int())
			if v < k.Min || v > k.Max {
				errs = append(errs, causes.NewInvalidFieldError(k.Csv, fmt.Sprintf("%d", valueField.Int()), fmt.Errorf("value %f out of bounds, min %f max %f", v, k.Min, k.Max)))
			}
		case float64:
			v := valueField.Float()
			if v < k.Min || v > k.Max {
				errs = append(errs, causes.NewInvalidFieldError(k.Csv, fmt.Sprintf("%f", v), fmt.Errorf("value %f out of bounds, min %f max %f", v, k.Min, k.Max)))
			}
		}
		// named validators
		switch k.Validator {
		case "timezone":
			if !enums.IsValidTimezone(strv) {
				errs = append(errs, causes.NewInvalidFieldError(k.Csv, strv, fmt.Errorf("invalid timezone")))
			}
		case "color":
			if !enums.IsValidColor(strv) {
				errs = append(errs, causes.NewInvalidFieldError(k.Csv, strv, fmt.Errorf("invalid color")))
			}
		case "email":
			if !enums.IsValidEmail(strv) {
				errs = append(errs, causes.NewInvalidFieldError(k.Csv, strv, fmt.Errorf("invalid email")))
			}
		case "url":
			if !enums.IsValidURL(strv) {
				errs = append(errs, causes.NewInvalidFieldError(k.Csv, strv, fmt.Errorf("invalid url")))
			}
		case "lang":
			if !enums.IsValidLang(strv) {
				errs = append(errs, causes.NewInvalidFieldError(k.Csv, strv, fmt.Errorf("invalid language")))
			}
		case "currency":
			if !enums.IsValidCurrency(strv) {
				errs = append(errs, causes.NewInvalidFieldError(k.Csv, strv, fmt.Errorf("invalid currency")))
			}
		}
	}
	return errs
}
