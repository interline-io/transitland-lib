package tt

import (
	"fmt"
	"strings"

	"github.com/interline-io/transitland-lib/causes"
)

type Email struct {
	Option[string]
}

// CheckEmail returns an error if the value is not a reasonably valid email address
func CheckEmail(field string, value string) (errs []error) {
	if !IsValidEmail(value) {
		errs = append(errs, causes.NewInvalidFieldError(field, value, fmt.Errorf("invalid email")))
	}
	return errs
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
