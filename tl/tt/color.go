package tt

import (
	"fmt"
	"strings"

	"github.com/interline-io/transitland-lib/tl/causes"
)

type Outer struct {
	Val   string
	Valid bool
}

func (a *Outer) Test() {

}

type Color struct{ Outer }

func (c *Color) String() string {
	return ""
}

// CheckColor returns an error if the value is not a valid hex color
func CheckColor(field string, value string) (errs []error) {

	z := Color{}
	a := z.Valid
	_ = a
	z.Test()

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
