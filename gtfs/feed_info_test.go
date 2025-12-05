package gtfs

import (
	"testing"

	"github.com/interline-io/transitland-lib/internal/testutil"
	"github.com/interline-io/transitland-lib/tt"
)

func TestFeedInfo_Errors(t *testing.T) {
	newFeedInfo := func(fn func(*FeedInfo)) *FeedInfo {
		startDate, _ := tt.ParseDate("20100101")
		endDate, _ := tt.ParseDate("21001231")
		feedInfo := &FeedInfo{
			FeedPublisherName: tt.NewString("ok"),
			FeedPublisherURL:  tt.NewUrl("http://example.com"),
			FeedLang:          tt.NewLanguage("en"),
			FeedStartDate:     startDate,
			FeedEndDate:       endDate,
			FeedVersion:       tt.NewString("1.0"),
			DefaultLang:       tt.NewLanguage("en"),
			FeedContactEmail:  tt.NewEmail("info@interline.io"),
			FeedContactURL:    tt.NewUrl("https://interline.io"),
		}
		if fn != nil {
			fn(feedInfo)
		}
		return feedInfo
	}

	testcases := []struct {
		name           string
		entity         *FeedInfo
		expectedErrors []testutil.ExpectError
	}{
		{
			name:           "Valid feed_info",
			entity:         newFeedInfo(nil),
			expectedErrors: nil,
		},
		{
			name: "Missing feed_publisher_name",
			entity: newFeedInfo(func(f *FeedInfo) {
				f.FeedPublisherName = tt.String{}
			}),
			expectedErrors: PE("RequiredFieldError:feed_publisher_name"),
		},
		{
			name: "Missing feed_publisher_url",
			entity: newFeedInfo(func(f *FeedInfo) {
				f.FeedPublisherURL = tt.Url{}
			}),
			expectedErrors: PE("RequiredFieldError:feed_publisher_url"),
		},
		{
			name: "Missing feed_lang",
			entity: newFeedInfo(func(f *FeedInfo) {
				f.FeedLang = tt.Language{}
			}),
			expectedErrors: PE("RequiredFieldError:feed_lang"),
		},
		{
			name: "Invalid feed_publisher_url",
			entity: newFeedInfo(func(f *FeedInfo) {
				f.FeedPublisherURL = tt.NewUrl("abcxyz")
			}),
			expectedErrors: PE("InvalidFieldError:feed_publisher_url"),
		},
		{
			name: "Invalid feed_lang",
			entity: newFeedInfo(func(f *FeedInfo) {
				f.FeedLang = tt.NewLanguage("xyz")
			}),
			expectedErrors: PE("InvalidFieldError:feed_lang"),
		},
		{
			name: "feed_end_date before feed_start_date",
			entity: newFeedInfo(func(f *FeedInfo) {
				startDate, _ := tt.ParseDate("20100101")
				endDate, _ := tt.ParseDate("20090101")
				f.FeedStartDate = startDate
				f.FeedEndDate = endDate
			}),
			expectedErrors: PE("InvalidFieldError:feed_end_date"),
		},
		{
			name: "Invalid default_lang",
			entity: newFeedInfo(func(f *FeedInfo) {
				f.DefaultLang = tt.NewLanguage("xyz")
			}),
			expectedErrors: PE("InvalidFieldError:default_lang"),
		},
		{
			name: "Invalid feed_contact_email",
			entity: newFeedInfo(func(f *FeedInfo) {
				f.FeedContactEmail = tt.NewEmail("xyz")
			}),
			expectedErrors: PE("InvalidFieldError:feed_contact_email"),
		},
		{
			name: "Invalid feed_contact_url",
			entity: newFeedInfo(func(f *FeedInfo) {
				f.FeedContactURL = tt.NewUrl("xyz")
			}),
			expectedErrors: PE("InvalidFieldError:feed_contact_url"),
		},
	}

	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			errs := tt.CheckErrors(tc.entity)
			testutil.CheckErrors(tc.expectedErrors, errs, t)
		})
	}
}
