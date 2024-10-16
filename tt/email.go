package tt

import (
	"fmt"
	"strings"

	"github.com/interline-io/transitland-lib/causes"
)

type Email struct {
	Option[string]
}

func NewEmail(v string) Email {
	return Email{Option: NewOption(v)}
}

func (r Email) Check() error {
	if r.Valid && !IsValidEmail(r.Val) {
		return causes.NewInvalidFieldError("", r.Val, fmt.Errorf("invalid email"))
	}
	return nil
}

// IsValidEmail check if valid email
func IsValidEmail(email string) bool {
	return strings.Contains(email, "@")
}
