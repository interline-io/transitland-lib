package rules

import (
	"regexp"

	"github.com/interline-io/transitland-lib/tl/causes"
)

///////////////

type bc = causes.Context

var allowedChars = regexp.MustCompile(`^[\.0-9\s\p{L}\(\)-/\&<>"']+$`)

func checkAllowedChars(s string) bool {
	if s == "" {
		return true
	}
	return allowedChars.MatchString(s)
}
