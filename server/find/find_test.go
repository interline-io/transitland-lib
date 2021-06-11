package find

import "testing"

func Test_alphanumeric(t *testing.T) {
	tcs := []struct {
		name   string
		value  string
		expect string
	}{
		{"ascii char", "a", "a"},
		{"ascii string", "abc", "abc"},
		{"ascii alphanumeric", "abc123", "abc123"},
		{"ascii space", "a b c", "a b c"},
		{"emdash remove", "a—b", "ab"},
		{"double space ok", "a  b", "a  b"},
		{"remove slash", "a/b", "ab"},
		{"remove single quote", "a'b", "ab"},
		{"remove dounle quote", "\"", ""},
		{"remove :", "a:b", "ab"},
		{"remove *", "a*b", "ab"},
		{"remove &", "a&b", "ab"},
		{"remove |", "a|b", "ab"},
		{"tab to space", "\t", " "},
		{"french", "Hôtel", "Hôtel"},
		{"chinese", "火车", "火车"},
		{"chinese with ascii", "abc 火车 123", "abc 火车 123"},
		{"japanese", "列車", "列車"},
		{"russian", "тренироваться", "тренироваться"},
		{"hebrew", "רכבת", "רכבת"},
		{"arabic", "قطار", "قطار"},
		{"arabic with ascii", "test قطار", "test قطار"},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			ret := alphanumeric(tc.value)
			if ret != tc.expect {
				t.Errorf("got '%s', expect '%s'", ret, tc.expect)
			}
		})
	}
}

func Test_az09(t *testing.T) {
	tcs := []struct {
		name   string
		value  string
		expect string
	}{
		{"plain", "hello", "hello"},
		{"underscore", "hello_world", "hello_world"},
		{"digits", "123", "123"},
		{"remove quotes", "a'b'\"c", "abc"},
		{"remove symbols", "a!b@c#d$e%f;g(h", "abcdefgh"},
	}
	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			ret := az09(tc.value)
			if ret != tc.expect {
				t.Errorf("got '%s', expect '%s'", ret, tc.expect)
			}
		})
	}
}
