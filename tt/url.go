package tt

import (
	"fmt"
	"strings"

	"github.com/interline-io/transitland-lib/causes"
)

type Url struct {
	Option[string]
}

func NewUrl(v string) Url {
	return Url{Option: NewOption(v)}
}

// CheckURL returns an error if the value is not a reasonably valid url
func CheckURL(field string, value string) (errs []error) {
	if !IsValidURL(value) {
		errs = append(errs, causes.NewInvalidFieldError(field, value, fmt.Errorf("invalid url")))
	}
	return errs
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
	} else if len(url) == 0 {
		return true
	}
	return false
}
