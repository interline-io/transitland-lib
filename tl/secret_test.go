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
		{Secret{Filename: "test.dmfr.json"}, "test.dmfr.json", true},
		{Secret{Filename: "test.dmfr.json"}, "notfound", false},
		{Secret{Filename: "test.dmfr.json"}, "", false},
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
		{Secret{FeedID: "f-ok"}, "f-ok", true},
		{Secret{FeedID: "f-ok"}, "notfound", false},
		{Secret{FeedID: "f-ok"}, "", false},
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
		{"feed id", Feed{FeedID: "f-test", File: "def.json"}, []Secret{{Filename: "xyz.json"}, {Filename: "def.json"}}, Secret{Filename: "def.json"}, false},
		{"feed id not matched", Feed{FeedID: "f-test"}, []Secret{{FeedID: "f-abc"}}, Secret{}, true},
		{"feed id and filename", Feed{FeedID: "f-test", File: "abc.json"}, []Secret{{FeedID: "f-test", Filename: "abc.json"}}, Secret{FeedID: "f-test", Filename: "abc.json"}, false},
		{"filename", Feed{FeedID: "f-test", File: "abc.json"}, []Secret{{Filename: "abc.json"}}, Secret{Filename: "abc.json"}, false},
		{"filename not matched", Feed{FeedID: "f-test", File: "def.json"}, []Secret{{Filename: "abc.json"}}, Secret{}, true},
		{"ambiguous feed id match", Feed{FeedID: "f-test", File: "def.json"}, []Secret{{FeedID: "f-test"}, {FeedID: "f-test"}}, Secret{}, true},
		{"ambiguous filename match", Feed{FeedID: "f-test", File: "def.json"}, []Secret{{Filename: "def.json"}, {Filename: "def.json"}}, Secret{Filename: "def.json"}, true},
		{"ambiguous both match", Feed{FeedID: "f-test", File: "def.json"}, []Secret{{FeedID: "f-test", Filename: "def.json"}, {FeedID: "f-test", Filename: "def.json"}}, Secret{}, true},
		{"no secrets", Feed{FeedID: "f-test", File: "def.json"}, []Secret{}, Secret{}, true},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			s, err := tc.feed.MatchSecrets(tc.secrets)
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
