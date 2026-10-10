package dmfr

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

func TestSecret_MatchHost(t *testing.T) {
	testcases := []struct {
		name      string
		scope     string
		match     string
		expect    bool
		expectErr bool
	}{
		{name: "unscoped", scope: "", match: "abc.com", expect: true},
		{name: "exact", scope: "mta.info", match: "mta.info", expect: true},
		{name: "exact ignores case", scope: "mta.info", match: "MTA.info", expect: true},
		{name: "scope ignores case", scope: "MTA.info", match: "mta.info", expect: true},
		{name: "other host", scope: "mta.info", match: "abc.com"},
		{name: "exact excludes subdomains", scope: "mta.info", match: "api-endpoint.mta.info"},
		{name: "wildcard subdomain", scope: "*.mta.info", match: "api-endpoint.mta.info", expect: true},
		{name: "wildcard nested subdomain", scope: "*.mta.info", match: "a.b.mta.info", expect: true},
		{name: "wildcard excludes the domain itself", scope: "*.mta.info", match: "mta.info"},
		{name: "wildcard needs a label boundary", scope: "*.mta.info", match: "evilmta.info"},
		{name: "wildcard anchors at the end", scope: "*.mta.info", match: "mta.info.abc.com"},
		{name: "IPv4 address", scope: "203.0.113.5", match: "203.0.113.5", expect: true},
		// strings.ToLower folds U+0130 to "i", but net/http dials another domain.
		{name: "dotted capital I", scope: "gtfs.trimet.org", match: "gtfs.trİmet.org"},
		{name: "unicode host matches its ASCII form", scope: "xn--mnchen-3ya.de", match: "münchen.de", expect: true},
		{name: "unicode scope matches its ASCII form", scope: "münchen.de", match: "xn--mnchen-3ya.de", expect: true},
		{name: "IPv6 zone is not a subdomain", scope: "*.mta.info", match: "2a01:4f8::1%.mta.info"},
		{name: "port", scope: "mta.info:443", match: "mta.info", expectErr: true},
		{name: "scheme", scope: "https://mta.info", match: "mta.info", expectErr: true},
		{name: "trailing dot", scope: "mta.info.", match: "mta.info", expectErr: true},
		{name: "whitespace", scope: " mta.info", match: "mta.info", expectErr: true},
		{name: "IPv6 brackets", scope: "[::1]", match: "::1", expectErr: true},
		{name: "bare wildcard", scope: "*", match: "mta.info", expectErr: true},
		{name: "wildcard without a domain", scope: "*.", match: "mta.info.", expectErr: true},
		{name: "wildcard without a dot", scope: "*mta.info", match: "evilmta.info", expectErr: true},
	}
	for _, tc := range testcases {
		t.Run(tc.name, func(t *testing.T) {
			v, err := (Secret{Host: tc.scope}).MatchHost(tc.match)
			if tc.expectErr {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tc.expect, v)
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
