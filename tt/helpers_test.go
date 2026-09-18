package tt

import (
	"testing"
)

func Test_IsValidURL(t *testing.T) {
	type args struct {
		url string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"empty", args{""}, false},
		{"http", args{"http://example.com"}, true},
		{"https", args{"https://example.com"}, true},
		{"fail1", args{"fail://example.com"}, true},
		{"fail1", args{"example.com"}, true},
		{"fail2", args{"asdf"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidURL(tt.args.url); got != tt.want {
				t.Errorf("IsValidURL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_IsValidColor(t *testing.T) {
	type args struct {
		color string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"with#", args{"#ffffff"}, true},
		{"without#", args{"ffffff"}, true},
		{"empty", args{""}, false},
		{"wronglen", args{"#ffff"}, false},
		{"len#", args{"xffffff"}, false},
		//{"nothex", args{"xyzxyz"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidColor(tt.args.color); got != tt.want {
				t.Errorf("IsValidColor() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_IsValidEmail(t *testing.T) {
	type args struct {
		email string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"with@", args{"info@example.com"}, true},
		{"empty", args{"info@example.com"}, true},
		{"without@", args{"example.com"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidEmail(tt.args.email); got != tt.want {
				t.Errorf("IsValidEmail() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_IsValidTimezone(t *testing.T) {
	type args struct {
		tz string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"America/Los_Angeles", args{"America/Los_Angeles"}, true},
		{"empty", args{""}, false},
		{"asd", args{"asd"}, false},
		// {"invalid", args{"Not/Timezone"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, got := IsValidTimezone(tt.args.tz); got != tt.want {
				t.Errorf("IsValidTimezone() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_IsValidLang(t *testing.T) {
	type args struct {
		lang string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"two letter", args{"en"}, true},
		{"region", args{"en-US"}, true},
		{"script and region", args{"sr-Cyrl-ME"}, true},
		// Three-letter primary subtags are valid BCP 47. "mul" is what the
		// spec tells a multilingual dataset to put in feed_lang, and "cnr"
		// (Montenegrin) has no two-letter code.
		{"three letter", args{"cnr"}, true},
		// Asas, an ISO 639-3 language. This case previously asserted false,
		// which encoded the bug rather than the spec.
		{"iso 639-3", args{"asd"}, true},
		{"multiple languages", args{"mul"}, true},
		{"undetermined", args{"und"}, true},
		{"no linguistic content", args{"zxx"}, true},
		{"empty", args{""}, false},
		{"unknown subtag", args{"xx"}, false},
		{"not a tag", args{"english"}, false},
		{"underscore separator", args{"en_US"}, false},
		{"cldr root alias", args{"root"}, false},
		{"cldr root alias mixed case", args{"Root"}, false},
		{"trailing space", args{"en "}, false},
		{"invalid", args{"Not/Timezone"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidLanguage(tt.args.lang); got != tt.want {
				t.Errorf("IsValidLanguage() = %v, want %v", got, tt.want)
			}
		})
	}
}
