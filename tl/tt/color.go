package tt

import (
	"fmt"
	"strings"
)

// Color represents a valid hex color
type Color struct {
	Option[string]
}

func (r Color) String() string {
	return r.Val
}

func (r *Color) Error() error {
	if !IsValidColor(r.Val) {
		return &InvalidColorError{r.Val}
	}
	return nil
}

// Errors, helpers

type InvalidColorError struct {
	Value string
}

func (e *InvalidColorError) Error() string {
	return fmt.Sprintf("invalid color: '%s'", e.Value)
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
