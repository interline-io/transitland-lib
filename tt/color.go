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

// IsValidColor check is valid color
func IsValidColor(color string) bool {
	// todo: hex validation?
	if len(color) == 7 && strings.HasPrefix(color, "#") {
		return true
	} else if len(color) == 6 {
		return true
	}
	return false
}
