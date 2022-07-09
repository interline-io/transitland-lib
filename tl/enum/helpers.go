package enum

import (
	"fmt"
	"strings"

	"github.com/interline-io/transitland-lib/tl/causes"
)

type bc = causes.Context

// Error wrapping helpers

type isEnum interface {
	String() string
	Error() error
	Present() bool
}

func CheckError(a []error, v error) []error {
	if v != nil {
		a = append(a, v)
	}
	return a
}

func CheckFieldError(field string, value isEnum) error {
	err := value.Error()
	if err == nil {
		return nil
	}
	err = causes.NewInvalidFieldError(field, value.String(), err)
	return err
}

func CheckFieldPresentError(field string, value isEnum) error {
	err := value.Error()
	if err != nil {
		err = causes.NewInvalidFieldError(field, value.String(), err)
	} else if value.String() == "" || !value.Present() {
		err = causes.NewRequiredFieldError(field)
	}
	return err
}

// CheckInArray returns an error if the value is not in the set of provided values.
func CheckInArray(field string, value string, values ...string) []error {
	for _, v := range values {
		if value == v {
			return nil
		}
	}
	return []error{causes.NewInvalidFieldError(field, value, fmt.Errorf("must be one of %s", strings.Join(values, ", ")))}
}

// CheckInArrayInt returns an error if the value is not in the set of provided values.
func CheckInArrayInt(field string, value int, values ...int) []error {
	for _, v := range values {
		if value == v {
			return nil
		}
	}
	var valueStrs []string
	for _, v := range values {
		valueStrs = append(valueStrs, fmt.Sprintf("%d", v))
	}
	return []error{causes.NewInvalidFieldError(field, fmt.Sprintf("%d", value), fmt.Errorf("must be one of %s", strings.Join(valueStrs, ", ")))}
}

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

// CheckEmail returns an error if the value is not a reasonably valid email address
func CheckEmail(field string, value string) (errs []error) {
	if !IsValidEmail(value) {
		errs = append(errs, causes.NewInvalidFieldError(field, value, fmt.Errorf("invalid email")))
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
