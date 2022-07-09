package enum

import (
	"fmt"
	"strings"
)

type Email struct {
	Option[string]
}

func (r Email) String() string {
	return r.Val
}

func (r *Email) Error() error {
	if !IsValidEmail(r.Val) {
		return fmt.Errorf("invalid email")
	}
	return nil
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
