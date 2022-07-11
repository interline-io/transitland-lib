package tt

import (
	"fmt"
	"strings"
)

type Url struct {
	Option[string]
}

func NewUrl(v string) Url {
	return Url{Option[string]{Valid: (v != ""), Val: v}}
}

func (r Url) String() string {
	return r.Val
}

func (r *Url) Error() error {
	if !IsValidURL(r.Val) {
		return &InvalidUrlError{r.Val}
	}
	return nil
}

// Errors, helpers

type InvalidUrlError struct {
	Value string
}

func (e *InvalidUrlError) Error() string {
	return fmt.Sprintf("invalid url")
}

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
