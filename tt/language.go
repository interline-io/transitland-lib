package tt

import (
	"strings"

	"github.com/interline-io/transitland-lib/causes"
	"golang.org/x/text/language"
)

type Language struct {
	Option[string]
}

func (r Language) Check() error {
	if r.Valid && !IsValidLanguage(r.Val) {
		return causes.NewInvalidLanguageError("", r.Val)
	}
	return nil
}

func NewLanguage(v string) Language {
	return Language{Option: NewOption(v)}
}

// IsValidLanguage reports whether the value names a language.
//
// GTFS specifies IETF BCP 47 for agency_lang, feed_lang, default_lang and
// translations.language. Only the primary subtag is judged, because that is
// the part that names the language: a consumer choosing capitalization or a
// translation needs to know "this is English", and "en-EN", a common typo for
// "en-US", still says so. What follows the language is left alone. Those
// registries gain entries over time, so a feed can be correct while this
// library is out of date, and naming a bad region or script is only worth
// doing in a message that says which subtag is at fault, which is more than
// this reports.
//
// Three-letter primary subtags are part of BCP 47 and were previously
// rejected. The spec directs a multilingual dataset to set feed_lang to "mul",
// and languages such as Montenegrin ("cnr") and Halkomelem ("hur") have no
// two-letter code at all.
func IsValidLanguage(value string) bool {
	// BCP 47 separates subtags with hyphens, so the primary subtag is
	// everything up to the first one. A value with no hyphen is its own primary
	// subtag, which is why an underscore separator needs no check of its own:
	// ParseBase is handed "en_US" whole and rejects it.
	primary, _, _ := strings.Cut(value, "-")
	_, err := language.ParseBase(primary)
	return err == nil
}
