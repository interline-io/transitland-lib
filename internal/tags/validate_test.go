package tags

import (
	"testing"

	"github.com/interline-io/gotransit/causes"
)

func TestValidateTags_validators(t *testing.T) {
	for k := range structTagMapCache {
		delete(structTagMapCache, k)
	}
	type tVE1 struct {
		TestInt      int     `csv:"int" min:"-10" max:"10"`
		TestFloat    float64 `csv:"float64" min:"-10" max:"10"`
		TestMin      float64 `csv:"min" min:"-10"`
		TestMax      float64 `csv:"max" max:"10"`
		TestURL      string  `csv:"url" validator:"url"`
		TestTimezone string  `csv:"timezone" validator:"timezone"`
		TestLang     string  `csv:"lang" validator:"lang"`
		TestEmail    string  `csv:"email" validator:"email"`
		TestColor    string  `csv:"color" validator:"color"`
		TestCurrency string  `csv:"currency" validator:"currency"`
	}
	type exp struct {
		name  string
		ent   tVE1
		count int
	}
	expect := []exp{
		exp{"int1", tVE1{TestInt: 5}, 0},
		exp{"int2", tVE1{TestInt: -20}, 1},
		exp{"int3", tVE1{TestInt: 20}, 1},
		exp{"float1", tVE1{TestFloat: -20.0}, 1},
		exp{"float2", tVE1{TestFloat: 20.0}, 1},
		exp{"float3", tVE1{TestMin: -100.0}, 1},
		exp{"bounds1", tVE1{TestMin: 10000.0}, 0},
		exp{"bounds2", tVE1{TestMax: 100.0}, 1},
		exp{"bounds3", tVE1{TestMax: -10000.0}, 0},
		exp{"url0", tVE1{TestURL: "http://example.com"}, 0},
		exp{"url1", tVE1{TestURL: "https://example.com"}, 0},
		exp{"url2", tVE1{TestURL: "example.com"}, 0},
		exp{"url3", tVE1{TestURL: "asdxyz"}, 1},
		exp{"tz0", tVE1{TestTimezone: "America/Los_Angeles"}, 0},
		exp{"lang0", tVE1{TestLang: "en"}, 0},
		exp{"lang1", tVE1{TestLang: "en-US"}, 0},
		exp{"lang2", tVE1{TestLang: "asdXYZ"}, 1},
		exp{"email0", tVE1{TestEmail: "info@example.com"}, 0},
		exp{"email1", tVE1{TestEmail: "example.com"}, 1},
		exp{"color0", tVE1{TestColor: "#ffffff"}, 0},
		exp{"color1", tVE1{TestColor: "ffffff"}, 0},
		exp{"color2", tVE1{TestColor: "axyz123"}, 1},
		exp{"currency", tVE1{TestCurrency: "asd"}, 1},
		exp{"currency", tVE1{TestCurrency: "usd"}, 0},
	}
	for _, v := range expect {
		t.Run(v.name, func(t *testing.T) {
			errs := ValidateTags(&v.ent)
			if len(errs) != v.count {
				t.Error("expected", v.count, "errors, got", len(errs))
			}
			if len(errs) > 0 && v.count > 0 {
				if e, ok := errs[0].(*causes.InvalidFieldError); !ok {
					t.Error("expected InvalidFieldError, got", e)
				}
			}
		})
	}
}

func TestValidateTags_required(t *testing.T) {
	for k := range structTagMapCache {
		delete(structTagMapCache, k)
	}
	type tVE3 struct {
		Test string `csv:"test" required:"true"`
	}
	if errs := ValidateTags(&tVE3{Test: "ok"}); errs != nil {
		t.Error("expected no errors")
	}
	// required
	if errs := ValidateTags(&tVE3{Test: ""}); len(errs) == 0 {
		t.Error("expected 1 error")
	} else if _, ok := errs[0].(*causes.RequiredFieldError); !ok {
		t.Error("expected RequiredFieldError")
	}
}

func Test_IsValidURL(t *testing.T) {
	type args struct {
		url string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"empty", args{""}, true},
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
		{"empty", args{""}, true},
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
		{"empty", args{""}, true},
		// {"invalid", args{"Not/Timezone"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidTimezone(tt.args.tz); got != tt.want {
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
		{"en", args{"en"}, true},
		{"empty", args{""}, true},
		// {"invalid", args{"Not/Timezone"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidLang(tt.args.lang); got != tt.want {
				t.Errorf("IsValidLang() = %v, want %v", got, tt.want)
			}
		})
	}
}
