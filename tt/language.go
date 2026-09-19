package tt

import (
	"errors"
	"fmt"
	"strings"

	"github.com/interline-io/transitland-lib/causes"
	"golang.org/x/text/language"
)

type Language struct {
	Option[string]
}

func (r Language) Check() error {
	if r.Valid && !IsValidLanguage(r.Val) {
		return causes.NewInvalidFieldError("", r.Val, fmt.Errorf("invalid language"))
	}
	return nil
}

func NewLanguage(v string) Language {
	return Language{Option: NewOption(v)}
}

// IsValidLanguage check is valid language
//
// GTFS specifies IETF BCP 47 for agency_lang, feed_lang, default_lang and
// translations.language. Three-letter primary subtags are part of that: the
// spec directs a multilingual dataset to set feed_lang to "mul", and languages
// such as Montenegrin ("cnr") have no two-letter code at all.
//
// The check is deliberately uneven. The tag has to be well formed and it has
// to name a language the library recognizes, because a value that does not is
// no use to a consumer choosing capitalization or a translation. Everything
// after the language is only checked for shape: an unrecognized region, script
// or variant leaves the language itself intact, and those registries gain
// entries over time, so a feed can be right while the library is out of date.
// "en-EN", a common typo for "en-US", still says the language is English.
func IsValidLanguage(value string) bool {
	// BCP 47 separates subtags with hyphens; x/text also accepts underscores,
	// so reject those here rather than let "en_US" through as well-formed.
	if strings.ContainsRune(value, '_') {
		return false
	}
	// BCP 47 writes the primary subtag as 2*3ALPHA (ISO 639) or 5*8ALPHA
	// (reserved for future use). A four-letter primary subtag is reserved, and
	// a one-character one is a singleton, which introduces an extension or a
	// private use tag rather than naming a language. That turns away "root"
	// (CLDR's name for the undetermined locale, which x/text accepts as an
	// alias for "und"), "x-anything", and the deprecated "i-" tags: none of
	// them tells a consumer what language to expect.
	primary, _, _ := strings.Cut(value, "-")
	if n := len(primary); n < 2 || n == 4 || n > 8 {
		return false
	}
	tag, err := language.Parse(value)
	if err == nil {
		return true
	}
	// x/text returns a ValueError when the tag is well formed but one of its
	// subtags is not in the registry the library was built with; anything else
	// is a syntax error.
	var unknownSubtag language.ValueError
	if !errors.As(err, &unknownSubtag) {
		return false
	}
	// Parse strips the unrecognized subtag and keeps the rest, so the tag
	// still carries a language unless the language was the unknown part.
	base, _, _ := tag.Raw()
	return base.String() != "und"
}
