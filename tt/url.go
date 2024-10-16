package tt

import (
	"fmt"
	"strings"

	"github.com/interline-io/transitland-lib/causes"
)

type Url struct {
	Option[string]
}

func (r Url) Check() error {
	if r.Valid && !IsValidURL(r.Val) {
		return causes.NewInvalidFieldError("", r.Val, fmt.Errorf("invalid url"))
	}
	return nil
}

func NewUrl(v string) Url {
	return Url{Option: NewOption(v)}
}

// Basic methods

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
	}
	return false
}
