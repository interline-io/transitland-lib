package gtfs

import (
	"testing"

	"github.com/interline-io/transitland-lib/tt"
)

func TestFareMedia_Errors(t *testing.T) {
	tests := []struct {
		name           string
		fareMedia      *FareMedia
		expectedErrors []ExpectError
	}{
		{
			name: "Valid: basic fare media",
			fareMedia: &FareMedia{
				FareMediaID:   tt.NewString("media1"),
				FareMediaName: tt.NewString("Clipper"),
				FareMediaType: tt.NewInt(2),
			},
			expectedErrors: nil,
		},
		{
			name: "Invalid: missing fare_media_id",
			fareMedia: &FareMedia{
				FareMediaName: tt.NewString("Clipper"),
				FareMediaType: tt.NewInt(2),
			},
			expectedErrors: ParseExpectErrors("RequiredFieldError:fare_media_id"),
		},
		{
			name: "Invalid: missing fare_media_name",
			fareMedia: &FareMedia{
				FareMediaID:   tt.NewString("media1"),
				FareMediaType: tt.NewInt(2),
			},
			expectedErrors: ParseExpectErrors("RequiredFieldError:fare_media_name"),
		},
		{
			name: "Invalid: missing fare_media_type",
			fareMedia: &FareMedia{
				FareMediaID:   tt.NewString("media1"),
				FareMediaName: tt.NewString("Clipper"),
			},
			expectedErrors: ParseExpectErrors("RequiredFieldError:fare_media_type"),
		},
		{
			name: "Invalid: invalid fare_media_type",
			fareMedia: &FareMedia{
				FareMediaID:   tt.NewString("media1"),
				FareMediaName: tt.NewString("Clipper"),
				FareMediaType: tt.NewInt(99),
			},
			expectedErrors: ParseExpectErrors("InvalidFieldError:fare_media_type"),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			errs := tt.CheckErrors(tc.fareMedia)
			CheckErrors(tc.expectedErrors, errs, t)
		})
	}
}
