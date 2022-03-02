package tl

import (
	"testing"
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
