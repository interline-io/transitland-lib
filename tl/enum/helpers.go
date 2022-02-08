package enum

import (
	"fmt"
	"strings"

	"github.com/interline-io/transitland-lib/tl/causes"
)

// Error wrapping helpers

// CheckPositive returns an error if the value is non-negative
func CheckPositive(field string, value float64) (errs []error) {
	if value < 0 {
		errs = append(errs, causes.NewInvalidFieldError(field, fmt.Sprintf("%f", value), fmt.Errorf("must be non-negative")))
	}
	return errs
}

// CheckPositiveInt returns an error if the value is non-negative
func CheckPositiveInt(field string, value int) (errs []error) {
	if value < 0 {
		errs = append(errs, causes.NewInvalidFieldError(field, fmt.Sprintf("%d", value), fmt.Errorf("must be non-negative")))
	}
	return errs
}

// CheckInsideRange returns an error if the value is outside of the specified range
func CheckInsideRange(field string, value float64, min float64, max float64) (errs []error) {
	if value < min || value > max {
		errs = append(errs, causes.NewInvalidFieldError(field, fmt.Sprintf("%f", value), fmt.Errorf("out of bounds, min %f max %f", min, max)))
	}
	return errs
}

// CheckInsideRangeInt returns an error if the value is outside of the specified range
func CheckInsideRangeInt(field string, value int, min int, max int) (errs []error) {
	if value < min || value > max {
		errs = append(errs, causes.NewInvalidFieldError(field, fmt.Sprintf("%d", value), fmt.Errorf("out of bounds, min %d max %d", min, max)))
	}
	return errs
}

// CheckPresent returns an error if a string is empty
func CheckPresent(field string, value string) (errs []error) {
	if value == "" {
		errs = append(errs, causes.NewRequiredFieldError(field))
	}
	return errs
}

// CheckLanguage returns an error if the value is not a known language
func CheckLanguage(field string, value string) (errs []error) {
	if !IsValidLang(value) {
		errs = append(errs, causes.NewInvalidFieldError(field, value, fmt.Errorf("invalid language")))
	}
	return errs
}

// CheckCurrency returns an error if the value is not a known currency
func CheckCurrency(field string, value string) (errs []error) {
	if !IsValidCurrency(value) {
		errs = append(errs, causes.NewInvalidFieldError(field, value, fmt.Errorf("invalid currency")))
	}
	return errs
}

// CheckTimezone returns an error if the value is not a known timezone
func CheckTimezone(field string, value string) (errs []error) {
	if _, ok := IsValidTimezone(value); !ok {
		errs = append(errs, causes.NewInvalidTimezoneError(field, value))
	}
	return errs
}

// CheckEmail returns an error if the value is not a reasonably valid email address
func CheckEmail(field string, value string) (errs []error) {
	if !IsValidEmail(value) {
		errs = append(errs, causes.NewInvalidFieldError(field, value, fmt.Errorf("invalid email")))
	}
	return errs
}

// CheckColor returns an error if the value is not a valid hex color
func CheckColor(field string, value string) (errs []error) {
	if !IsValidColor(value) {
		errs = append(errs, causes.NewInvalidFieldError(field, value, fmt.Errorf("invalid color")))
	}
	return errs
}

// CheckURL returns an error if the value is not a reasonably valid url
func CheckURL(field string, value string) (errs []error) {
	if !IsValidURL(value) {
		errs = append(errs, causes.NewInvalidFieldError(field, value, fmt.Errorf("invalid url")))
	}
	return errs
}

// Basic methods

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
func IsValidTimezone(value string) (string, bool) {
	if len(value) == 0 {
		return "", true
	}
	nornmalized, ok := timezones[strings.ToLower(value)]
	return nornmalized, ok
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
