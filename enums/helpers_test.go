package enums

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
