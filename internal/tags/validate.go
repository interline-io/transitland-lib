package tags

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/interline-io/gotransit/causes"
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
			if !IsValidTimezone(strv) {
				errs = append(errs, causes.NewInvalidFieldError(k.Csv, strv, fmt.Errorf("invalid timezone")))
			}
		case "color":
			if !IsValidColor(strv) {
				errs = append(errs, causes.NewInvalidFieldError(k.Csv, strv, fmt.Errorf("invalid color")))
			}
		case "email":
			if !IsValidEmail(strv) {
				errs = append(errs, causes.NewInvalidFieldError(k.Csv, strv, fmt.Errorf("invalid email")))
			}
		case "url":
			if !IsValidURL(strv) {
				errs = append(errs, causes.NewInvalidFieldError(k.Csv, strv, fmt.Errorf("invalid url")))
			}
		case "lang":
			if !IsValidLang(strv) {
				errs = append(errs, causes.NewInvalidFieldError(k.Csv, strv, fmt.Errorf("invalid language")))
			}
		case "currency":
			if !IsValidCurrency(strv) {
				errs = append(errs, causes.NewInvalidFieldError(k.Csv, strv, fmt.Errorf("invalid currency")))
			}
		}
	}
	return errs
}

/* Validation Helpers */

// IsValidLang check is valid language
func IsValidLang(value string) bool {
	if len(value) == 0 {
		return true
	}
	// Only check the prefix code
	code := strings.Split(value, "-")
	_, ok := langs[strings.ToLower(code[0])]
	return ok
}

// IsValidCurrency check is valid currency
func IsValidCurrency(value string) bool {
	if len(value) == 0 {
		return true
	}
	_, ok := currencies[strings.ToLower(value)]
	return ok
}

// IsValidTimezone check is valid timezone
func IsValidTimezone(value string) bool {
	if len(value) == 0 {
		return true
	}
	_, ok := timezones[strings.ToLower(value)]
	return ok
}

// IsValidEmail check if valid email
func IsValidEmail(email string) bool {
	if strings.Contains(email, "@") {
		return true
	} else if len(email) == 0 {
		return true
	}
	return false
}

// IsValidColor check is valid color
func IsValidColor(color string) bool {
	// todo: hex validation?
	if len(color) == 0 {
		return true
	} else if len(color) == 7 && strings.HasPrefix(color, "#") {
		return true
	} else if len(color) == 6 {
		return true
	}
	return false
}

// IsValidURL check is valid url
func IsValidURL(url string) bool {
	// todo: full validation?
	if strings.HasPrefix(url, "http://") {
		return true
	} else if strings.HasPrefix(url, "https://") {
		return true
	} else if strings.Contains(url, ".") {
		// allow bare hosts, e.g. "example.com"
		return true
	} else if len(url) == 0 {
		return true
	}
	return false
}
