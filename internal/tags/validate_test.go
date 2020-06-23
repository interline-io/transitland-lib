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
		{"int1", tVE1{TestInt: 5}, 0},
		{"int2", tVE1{TestInt: -20}, 1},
		{"int3", tVE1{TestInt: 20}, 1},
		{"float1", tVE1{TestFloat: -20.0}, 1},
		{"float2", tVE1{TestFloat: 20.0}, 1},
		{"float3", tVE1{TestMin: -100.0}, 1},
		{"bounds1", tVE1{TestMin: 10000.0}, 0},
		{"bounds2", tVE1{TestMax: 100.0}, 1},
		{"bounds3", tVE1{TestMax: -10000.0}, 0},
		{"url0", tVE1{TestURL: "http://example.com"}, 0},
		{"url1", tVE1{TestURL: "https://example.com"}, 0},
		{"url2", tVE1{TestURL: "example.com"}, 0},
		{"url3", tVE1{TestURL: "asdxyz"}, 1},
		{"tz0", tVE1{TestTimezone: "America/Los_Angeles"}, 0},
		{"lang0", tVE1{TestLang: "en"}, 0},
		{"lang1", tVE1{TestLang: "en-US"}, 0},
		{"lang2", tVE1{TestLang: "asdXYZ"}, 1},
		{"email0", tVE1{TestEmail: "info@example.com"}, 0},
		{"email1", tVE1{TestEmail: "example.com"}, 1},
		{"color0", tVE1{TestColor: "#ffffff"}, 0},
		{"color1", tVE1{TestColor: "ffffff"}, 0},
		{"color2", tVE1{TestColor: "axyz123"}, 1},
		{"currency", tVE1{TestCurrency: "asd"}, 1},
		{"currency", tVE1{TestCurrency: "usd"}, 0},
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
