package gtfs

import (
	"testing"

	"github.com/interline-io/transitland-lib/internal/testutil"
	"github.com/interline-io/transitland-lib/tt"
)

func TestTranslation_Errors(t *testing.T) {
	newTranslation := func(fn func(*Translation)) *Translation {
		translation := &Translation{
			TableNameValue: tt.NewString("agency"),
			FieldName:      tt.NewString("agency_name"),
			Language:       tt.NewLanguage("en"),
			Translation:    tt.NewString("hello"),
			RecordID:       tt.NewString("ok"),
		}
		if fn != nil {
			fn(translation)
		}
		return translation
	}

	testcases := []struct {
		name           string
		entity         *Translation
		expectedErrors []testutil.ExpectError
	}{
		{
			name:           "Valid translation",
			entity:         newTranslation(nil),
			expectedErrors: nil,
		},
		{
			name: "Invalid table_name",
			entity: newTranslation(func(t *Translation) {
				t.TableNameValue = tt.NewString("xyz")
			}),
			expectedErrors: ParseExpectErrors("InvalidFieldError:table_name"),
		},
		{
			name: "Invalid language",
			entity: newTranslation(func(t *Translation) {
				t.Language = tt.NewLanguage("xyz")
			}),
			expectedErrors: ParseExpectErrors("InvalidFieldError:language"),
		},
		{
			name: "record_id required when field_value empty",
			entity: newTranslation(func(t *Translation) {
				t.RecordID = tt.String{}
				t.FieldValue = tt.String{}
			}),
			expectedErrors: ParseExpectErrors("ConditionallyRequiredFieldError:record_id"),
		},
		{
			name: "record_sub_id required for stop_times",
			entity: newTranslation(func(t *Translation) {
				t.TableNameValue = tt.NewString("stop_times")
				t.FieldName = tt.NewString("stop_headsign")
				t.RecordID = tt.NewString("ok")
				t.RecordSubID = tt.String{}
			}),
			expectedErrors: ParseExpectErrors("ConditionallyRequiredFieldError:record_sub_id"),
		},
		{
			name: "record_id forbidden for feed_info",
			entity: newTranslation(func(t *Translation) {
				t.TableNameValue = tt.NewString("feed_info")
				t.FieldName = tt.NewString("feed_publisher_name")
				t.RecordID = tt.NewString("ok")
			}),
			expectedErrors: ParseExpectErrors("ConditionallyForbiddenFieldError:record_id"),
		},
		{
			name: "record_id and record_sub_id forbidden for feed_info",
			entity: newTranslation(func(t *Translation) {
				t.TableNameValue = tt.NewString("feed_info")
				t.FieldName = tt.NewString("feed_publisher_name")
				t.RecordID = tt.NewString("ok")
				t.RecordSubID = tt.NewString("ok")
			}),
			expectedErrors: ParseExpectErrors("ConditionallyForbiddenFieldError:record_id", "ConditionallyForbiddenFieldError:record_sub_id"),
		},
		{
			name: "field_value forbidden for feed_info",
			entity: newTranslation(func(t *Translation) {
				t.TableNameValue = tt.NewString("feed_info")
				t.FieldName = tt.NewString("feed_publisher_name")
				t.RecordID = tt.String{}
				t.FieldValue = tt.NewString("asd")
			}),
			expectedErrors: ParseExpectErrors("ConditionallyForbiddenFieldError:field_value"),
		},
		{
			name: "record_id exclusive with field_value",
			entity: newTranslation(func(t *Translation) {
				t.RecordID = tt.NewString("ok")
				t.FieldValue = tt.NewString("asd")
			}),
			expectedErrors: ParseExpectErrors("ConditionallyForbiddenFieldError:field_value"),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			errs := tt.CheckErrors(tc.entity)
			testutil.CheckErrors(tc.expectedErrors, errs, t)
		})
	}
}
