package tt

import (
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
func IsValidLanguage(value string) bool {
	// BCP 47 separates subtags with hyphens; x/text also accepts underscores,
	// so reject those here rather than let "en_US" through as well-formed.
	if strings.ContainsRune(value, '_') {
		return false
	}
	// "root" is CLDR's name for the undetermined locale, and x/text accepts it
	// as an alias for "und". BCP 47 reserves four-letter primary subtags, so it
	// is not something a feed can name a language with.
	if strings.EqualFold(value, "root") {
		return false
	}
	_, err := language.Parse(value)
	return err == nil
}
