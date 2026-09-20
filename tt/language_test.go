package tt

import (
	"testing"

	"github.com/interline-io/transitland-lib/causes"
	"github.com/stretchr/testify/assert"
)

// Check reports three outcomes, not two. A tag naming a language the library
// knows is accepted silently; one that names no language is rejected; and one
// that names a language but carries a subtag the library does not know is
// accepted with an advisory naming that subtag, because the language is still
// usable and the value is most often a typo.
func TestLanguage_Check(t *testing.T) {
	t.Run("recognized, nothing reported", func(t *testing.T) {
		for _, v := range []string{
			"en", "en-US", "sr-Cyrl-ME", "es-419", "ca-ES-valencia",
			"de-DE-u-co-phonebk", "en-GB-oxendict",
			// Three-letter codes, including ones with no two-letter form.
			"hur", "cnr", "mul", "und", "zxx",
		} {
			assert.NoError(t, NewLanguage(v).Check(), "expected %q to be accepted with nothing reported", v)
		}
	})
	t.Run("names no language", func(t *testing.T) {
		for _, v := range []string{
			"xx", "xxx", "english", "en_US", "root", "x-private",
			"en-12", "fr-CA-fr", "en-x", "en ",
		} {
			assert.IsType(t, &causes.InvalidFieldError{}, NewLanguage(v).Check(), "expected %q to be rejected", v)
		}
	})
	t.Run("unrecognized subtag is named", func(t *testing.T) {
		for _, tc := range []struct{ value, subtag string }{
			{"en-EN", "EN"},
			{"en-blah", "Blah"},
			{"en-Abcd", "Abcd"},
			{"es-ES-tradnl", "tradnl"},
		} {
			err := NewLanguage(tc.value).Check()
			advisory, ok := err.(*causes.UnknownLanguageSubtagError)
			if !assert.True(t, ok, "expected %q to report an unrecognized subtag, got %v", tc.value, err) {
				continue
			}
			// Naming the subtag is the point: "unrecognized subtag 'EN'" tells
			// a producer where to look, "invalid language" does not.
			assert.Equal(t, tc.subtag, advisory.Subtag, "for %q", tc.value)
			assert.Equal(t, tc.value, advisory.Value, "for %q", tc.value)
			assert.Contains(t, advisory.Error(), tc.subtag)
		}
	})
	// An absent value is not a bad one. Whether it has to be there at all is
	// the required tag's business, not this method's.
	t.Run("absent", func(t *testing.T) {
		assert.NoError(t, Language{}.Check())
	})
}
