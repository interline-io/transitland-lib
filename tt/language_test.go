package tt

import (
	"strings"
	"testing"

	"github.com/interline-io/transitland-lib/causes"
)

func TestIsValidLanguage(t *testing.T) {
	// Values that name a language. The point of each group is noted because
	// several of these were rejected before and cost feeds their agency.txt or
	// translations.txt.
	valid := []string{
		// ISO 639-1, with and without the subtags BCP 47 allows after it.
		"en", "fr", "de", "ga",
		"en-US", "sr-Cyrl-ME", "es-419", "ca-ES-valencia",
		"de-DE-u-co-phonebk", "en-GB-oxendict",
		// Case is not significant in BCP 47.
		"EN", "DE",
		// ISO 639-2 and 639-3. "mul" is what the spec tells a multilingual
		// dataset to use, and Montenegrin and Halkomelem have no two-letter
		// code at all.
		"mul", "und", "zxx", "cnr", "hur", "asd",
		// A region, script or variant this library does not recognize still
		// leaves the language readable, so the value is accepted. "en-EN" is a
		// common typo for "en-US" and still says English.
		"en-EN", "en-blah", "en-Abcd", "es-ES-tradnl", "es-041", "und-EN",
		// Registered grandfathered tags keep their ISO 639 primary subtag.
		"no-NO-NY",
	}
	for _, v := range valid {
		t.Run("valid/"+v, func(t *testing.T) {
			if !IsValidLanguage(v) {
				t.Errorf("IsValidLanguage(%q) = false, expected true", v)
			}
		})
	}

	// Values that name no language. A consumer cannot pick capitalization or a
	// translation from any of these.
	invalid := []string{
		"",
		// Well formed but not in the registry.
		"xx", "xxx",
		// Not a language subtag at all.
		"english", "Not/Language",
		// BCP 47 separates subtags with hyphens; an underscore is not a
		// separator, so the whole value is read as one primary subtag.
		"en_US",
		// A one-character primary subtag is a singleton: it introduces an
		// extension or a private use tag rather than naming a language.
		"x-private", "i-klingon",
		// Four letters is the reserved space CLDR puts "root" in, which is an
		// alias for "und" and names no language of its own.
		"root", "abcd", "qaaa",
		// Trailing whitespace is part of the subtag.
		"en ",
	}
	for _, v := range invalid {
		t.Run("invalid/"+v, func(t *testing.T) {
			if IsValidLanguage(v) {
				t.Errorf("IsValidLanguage(%q) = true, expected false", v)
			}
		})
	}
}

func TestLanguage_Check(t *testing.T) {
	// A field that is not present is not judged. An empty string that was set
	// on purpose is a present value, and is reported like any other value that
	// names no language, which is how every other Option type behaves.
	t.Run("absent", func(t *testing.T) {
		var unset Language
		if err := unset.Check(); err != nil {
			t.Errorf("got %v, expected no error for an unset value", err)
		}
	})

	t.Run("recognized", func(t *testing.T) {
		if err := NewLanguage("en-EN").Check(); err != nil {
			t.Errorf("got %v, expected no error", err)
		}
	})

	// The cause carries the value and asks to be reported at warning level, so
	// a feed keeps the entity rather than dropping it and everything that
	// references it.
	t.Run("unrecognized is a warning naming the value", func(t *testing.T) {
		err := NewLanguage("xyz").Check()
		if err == nil {
			t.Fatal("got no error, expected one")
		}
		langErr, ok := err.(*causes.InvalidLanguageError)
		if !ok {
			t.Fatalf("got %T, expected *causes.InvalidLanguageError", err)
		}
		if lvl := langErr.ErrorLevel(); lvl != 1 {
			t.Errorf("got ErrorLevel %d, expected 1 (warning)", lvl)
		}
		if !strings.Contains(langErr.Error(), "xyz") {
			t.Errorf("got %q, expected the message to name the value", langErr.Error())
		}
	})
}
