package gtfs

import (
	"testing"

	"github.com/interline-io/transitland-lib/tt"
)

func TestAgency_Errors(t *testing.T) {
	newAgency := func(fn func(*Agency)) *Agency {
		agency := &Agency{
			AgencyID:       tt.NewString("ok"),
			AgencyName:     tt.NewString("valid agency"),
			AgencyURL:      tt.NewUrl("http://google.com"),
			AgencyTimezone: tt.NewTimezone("America/Los_Angeles"),
			AgencyLang:     tt.NewLanguage("en"),
			AgencyPhone:    tt.NewString("515 555-5555"),
			AgencyFareURL:  tt.NewUrl("http://example.com"),
			AgencyEmail:    tt.NewEmail("info@example.com"),
		}
		if fn != nil {
			fn(agency)
		}
		return agency
	}

	tests := []struct {
		name           string
		agency         *Agency
		expectedErrors []ExpectError
	}{
		{
			name:           "Valid agency",
			agency:         newAgency(nil),
			expectedErrors: nil,
		},
		{
			name: "Invalid agency_url",
			agency: newAgency(func(a *Agency) {
				a.AgencyURL = tt.NewUrl("abcxyz")
			}),
			expectedErrors: ParseExpectErrors("InvalidFieldError:agency_url"),
		},
		{
			name: "Missing agency_timezone (required field)",
			agency: newAgency(func(a *Agency) {
				a.AgencyTimezone = tt.Timezone{}
			}),
			expectedErrors: ParseExpectErrors("RequiredFieldError:agency_timezone"),
		},
		{
			name: "Invalid agency_lang",
			agency: newAgency(func(a *Agency) {
				a.AgencyLang = tt.NewLanguage("xyz")
			}),
			expectedErrors: ParseExpectErrors("InvalidFieldError:agency_lang"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			errs := tt.CheckErrors(tc.agency)
			CheckErrors(tc.expectedErrors, errs, t)
		})
	}
}
