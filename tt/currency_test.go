package tt

import (
	"testing"
)

func TestCurrencyAmount(t *testing.T) {
	tests := []struct {
		name   string
		cur    CurrencyAmount
		setcur string
		tocsv  string
	}{
		{"none", NewCurrencyAmount(1.234), "", "1.234"},           // default to nearest precision
		{"unknown", NewCurrencyAmount(1.234), "unknown", "1.234"}, // default to nearest precision
		{"USD", NewCurrencyAmount(1.234), "USD", "1.23"},          // round to nearest cent
		{"usd", NewCurrencyAmount(1.234), "usd", "1.23"},          // case insensitive
		{"EUR", NewCurrencyAmount(1.234), "EUR", "1.23"},          // round to nearest euro cent
		{"JPY", NewCurrencyAmount(1.234), "JPY", "1"},             // no decimal
		{"JPY", NewCurrencyAmount(1.678), "JPY", "2"},             // round to nearest yen
		{"default", NewCurrencyAmount(1.67890), "", "1.6789"},     // nearest precision needed to represent value
		{"0.9999999", NewCurrencyAmount(0.9999999), "", "0.9999999"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			c := tc.cur
			if tc.setcur != "" {
				c.SetCurrency(tc.setcur)
			}
			s := c.ToCsv()
			if s != tc.tocsv {
				t.Errorf("got %s expected %s", s, tc.tocsv)
			}

		})
	}
}

func Test_IsValidCurrency(t *testing.T) {
	type args struct {
		val string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{"USD", args{"USD"}, true},
		{"usd", args{"usd"}, true},
		{"JPY", args{"JPY"}, true},
		{"XYZ", args{"XYZ"}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := IsValidCurrency(tc.args.val); got != tc.want {
				t.Errorf("IsValidCurrency() = %v, want %v", got, tc.want)
			}
		})
	}
}
