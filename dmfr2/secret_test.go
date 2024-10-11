package tl

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSecret_MatchFilename(t *testing.T) {
	testcases := []struct {
		secret Secret
		match  string
		expect bool
	}{
		{
			secret: Secret{Filename: "test.dmfr.json"},
			match:  "test.dmfr.json",
			expect: true,
		},
		{
			secret: Secret{Filename: "test.dmfr.json"},
			match:  "notfound",
			expect: false,
		},
		{
			secret: Secret{Filename: "test.dmfr.json"},
			match:  "",
			expect: false,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.match, func(t *testing.T) {
			if v := tc.secret.MatchFilename(tc.match); v != tc.expect {
				t.Errorf("got %t, expected %t", v, tc.expect)
			}
		})
	}
}

func TestSecret_MatchFeed(t *testing.T) {
	testcases := []struct {
		secret Secret
		match  string
		expect bool
	}{
		{
			secret: Secret{FeedID: "f-ok"},
			match:  "f-ok",
			expect: true,
		},
		{
			secret: Secret{FeedID: "f-ok"},
			match:  "notfound",
			expect: false,
		},
		{
			secret: Secret{FeedID: "f-ok"},
			match:  "",
			expect: false,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.match, func(t *testing.T) {
			if v := tc.secret.MatchFeed(tc.match); v != tc.expect {
				t.Errorf("got %t, expected %t", v, tc.expect)
			}
		})
	}
}

func TestFeed_MatchSecrets(t *testing.T) {
	testcases := []struct {
		name      string
		feed      Feed
		secrets   []Secret
		match     Secret
		expectErr bool
	}{
		{
			name:      "feed id",
			feed:      Feed{FeedID: "f-test", File: "def.json"},
			secrets:   []Secret{{Filename: "xyz.json"}, {Filename: "def.json"}},
			match:     Secret{Filename: "def.json"},
			expectErr: false,
		},
		{
			name:      "feed id not matched",
			feed:      Feed{FeedID: "f-test"},
			secrets:   []Secret{{FeedID: "f-abc"}},
			expectErr: true,
		},
		{
			name:      "feed id and filename",
			feed:      Feed{FeedID: "f-test", File: "abc.json"},
			secrets:   []Secret{{FeedID: "f-test", Filename: "abc.json"}},
			match:     Secret{FeedID: "f-test", Filename: "abc.json"},
			expectErr: false,
		},
		{
			name:      "filename",
			feed:      Feed{FeedID: "f-test", File: "abc.json"},
			secrets:   []Secret{{Filename: "abc.json"}},
			match:     Secret{Filename: "abc.json"},
			expectErr: false,
		},
		{
			name:      "filename not matched",
			feed:      Feed{FeedID: "f-test", File: "def.json"},
			secrets:   []Secret{{Filename: "abc.json"}},
			expectErr: true,
		},
		{
			name:      "ambiguous feed id match",
			feed:      Feed{FeedID: "f-test", File: "def.json"},
			secrets:   []Secret{{FeedID: "f-test"}, {FeedID: "f-test"}},
			expectErr: true,
		},
		{
			name:      "ambiguous filename match",
			feed:      Feed{FeedID: "f-test", File: "def.json"},
			secrets:   []Secret{{Filename: "def.json"}, {Filename: "def.json"}},
			expectErr: true,
		},
		{
			name:      "ambiguous both match",
			feed:      Feed{FeedID: "f-test", File: "def.json"},
			secrets:   []Secret{{FeedID: "f-test", Filename: "def.json"}, {FeedID: "f-test", Filename: "def.json"}},
			expectErr: true,
		},
		{
			name:      "no secrets",
			feed:      Feed{FeedID: "f-test", File: "def.json"},
			secrets:   []Secret{},
			expectErr: true,
		},
		{
			name:      "feed id",
			feed:      Feed{FeedID: "f-test", File: "def.json"},
			secrets:   []Secret{{Filename: "xyz.json"}, {Filename: "def.json"}},
			match:     Secret{Filename: "def.json"},
			expectErr: false,
		},
		{
			name:      "urltype match",
			feed:      Feed{FeedID: "f-test", File: "def.json"},
			secrets:   []Secret{{FeedID: "f-test", URLType: "static_current"}, {FeedID: "f-test", URLType: "realtime_alerts"}},
			match:     Secret{FeedID: "f-test", URLType: "static_current"},
			expectErr: false,
		},
		{
			name:      "urltype ambiguous match 1",
			feed:      Feed{FeedID: "f-test", File: "def.json"},
			secrets:   []Secret{{FeedID: "f-test", URLType: "static_current"}, {FeedID: "f-test", URLType: "static_current"}},
			expectErr: true,
		},
		{
			name:      "urltype ambiguous match 2",
			feed:      Feed{FeedID: "f-test", File: "def.json"},
			secrets:   []Secret{{FeedID: "f-test", URLType: "static_current"}, {FeedID: "f-test"}},
			expectErr: true,
		},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			s, err := tc.feed.MatchSecrets(tc.secrets, "static_current")
			if tc.expectErr && err == nil {
				t.Errorf("got no error, expected error")
			} else if !tc.expectErr && err != nil {
				t.Errorf("got unexpected error '%s', expected no error", err.Error())
			} else if err == nil {
				assert.Equal(t, tc.match, s)
			}
		})
	}
}
