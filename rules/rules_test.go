package rules

import "testing"

func Test_checkAllowedChars(t *testing.T) {
	testcases := []struct {
		Value  string
		Expect bool
	}{
		{`ok`, true},
		{`a ok`, true},
		{`a OK ok`, true},
		{`a (OK)`, true},
		{`a - ok`, true},
		{`a <> ok`, true},
		{`a & ok`, true},
		{`a "ok"`, true},
		{`a _ ok`, false},
		{`a\ok`, false},
		{`a \ ok`, false},
		{`a $ ok`, false},
		{`a ^ ok`, false},
		{`你好`, true},
		{`你好 \ ok`, false},
	}
	for _, tc := range testcases {
		if v := checkAllowedChars(tc.Value); v != tc.Expect {
			t.Errorf("got %t, expected %t for '%s'", v, tc.Expect, tc.Value)
		}
	}
}
