package dmfr

import (
	"testing"
)

func TestSecrets(t *testing.T) {
	s := Secrets{}
	if err := s.Load("../test/data/dmfr/secrets.json"); err != nil {
		t.Error(err)
	}
	if len(s) != 4 {
		t.Errorf("got %d secrets, expected 4", len(s))
	}
}

func TestSecrets_MatchFilename(t *testing.T) {
	s := Secrets{}
	if err := s.Load("../test/data/dmfr/secrets.json"); err != nil {
		t.Error(err)
	}
	testcases := []struct {
		match     string
		expecterr bool
	}{
		{"test.dmfr.json", false},
		{"notfound", true},
		{"", true},
	}
	for _, tc := range testcases {
		t.Run(tc.match, func(t *testing.T) {
			found, err := s.MatchFilename(tc.match)
			if err != nil && tc.expecterr == false {
				t.Errorf("got unexpected error: %s", err)
			} else if err == nil && tc.expecterr == true {
				t.Errorf("expected error")
			} else if tc.expecterr == false && tc.match != found.Filename {
				t.Errorf("got %s, expected %s", found.Filename, tc.match)
			}
		})
	}
}

func TestSecrets_MatchFeed(t *testing.T) {
	s := Secrets{}
	if err := s.Load("../test/data/dmfr/secrets.json"); err != nil {
		t.Error(err)
	}
	testcases := []struct {
		match     string
		expecterr bool
	}{
		{"f-ok", false},
		{"not-found", true},
		{"", true},
		{"f-invalid", true},
	}
	for _, tc := range testcases {
		t.Run(tc.match, func(t *testing.T) {
			found, err := s.MatchFeed(tc.match)
			if err != nil && tc.expecterr == false {
				t.Errorf("got unexpected error: %s", err)
			} else if err == nil && tc.expecterr == true {
				t.Errorf("expected error")
			} else if tc.expecterr == false && tc.match != found.FeedID {
				t.Errorf("got %s, expected %s", found.FeedID, tc.match)
			}
		})
	}
}
