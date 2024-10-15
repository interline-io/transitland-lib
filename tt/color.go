package tt

import (
	"fmt"
	"strings"

	"github.com/interline-io/transitland-lib/causes"
)

type Color struct {
	Option[string]
}

func NewColor(v string) Color {
	return Color{Option: NewOption(v)}
}

func (r Color) Check() error {
	if r.Valid && !IsValidColor(r.Val) {
		return causes.NewInvalidFieldError("", r.Val, fmt.Errorf("invalid color"))
	}
	return nil
}

// CheckColor returns an error if the value is not a valid hex color
func CheckColor(field string, value string) (errs []error) {
	if !IsValidColor(value) {
		errs = append(errs, causes.NewInvalidFieldError(field, value, fmt.Errorf("invalid color")))
	}
	return errs
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
