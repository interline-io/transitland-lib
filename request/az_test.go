package request

import (
	"context"
	"os"
	"testing"
)

func TestAz(t *testing.T) {
	ctx := context.TODO()
	azUri := os.Getenv("TL_TEST_AZ_STORAGE")
	if azUri == "" {
		t.Skip("Set TL_TEST_AZ_STORAGE for this test")
		return
	}
	b, err := NewAzFromUrl(azUri)
	if err != nil {
		t.Fatal(err)
	}
	testBucket(t, ctx, b)
}

func TestNewAzFromUrl(t *testing.T) {
	tests := []struct {
		name          string
		url           string
		wantAccount   string
		wantContainer string
		wantKeyPrefix string
		wantVerify    bool
		wantErr       bool
	}{
		{
			name:          "basic url",
			url:           "az://myaccount.blob.core.windows.net/mycontainer",
			wantAccount:   "myaccount.blob.core.windows.net",
			wantContainer: "mycontainer",
			wantKeyPrefix: "",
			wantVerify:    false,
		},
		{
			name:          "with key prefix",
			url:           "az://myaccount.blob.core.windows.net/mycontainer/some/prefix",
			wantAccount:   "myaccount.blob.core.windows.net",
			wantContainer: "mycontainer",
			wantKeyPrefix: "some/prefix",
			wantVerify:    false,
		},
		{
			name:          "verify_upload=true",
			url:           "az://myaccount.blob.core.windows.net/mycontainer?verify_upload=true",
			wantAccount:   "myaccount.blob.core.windows.net",
			wantContainer: "mycontainer",
			wantKeyPrefix: "",
			wantVerify:    true,
		},
		{
			name:          "verify_upload=false",
			url:           "az://myaccount.blob.core.windows.net/mycontainer?verify_upload=false",
			wantAccount:   "myaccount.blob.core.windows.net",
			wantContainer: "mycontainer",
			wantKeyPrefix: "",
			wantVerify:    false,
		},
		{
			name:          "verify_upload with prefix",
			url:           "az://myaccount.blob.core.windows.net/mycontainer/prefix?verify_upload=true",
			wantAccount:   "myaccount.blob.core.windows.net",
			wantContainer: "mycontainer",
			wantKeyPrefix: "prefix",
			wantVerify:    true,
		},
		{
			name:          "other query params ignored",
			url:           "az://myaccount.blob.core.windows.net/mycontainer?other=value&verify_upload=true",
			wantAccount:   "myaccount.blob.core.windows.net",
			wantContainer: "mycontainer",
			wantKeyPrefix: "",
			wantVerify:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			az, err := NewAzFromUrl(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewAzFromUrl() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if err != nil {
				return
			}
			if az.Account != tt.wantAccount {
				t.Errorf("Account = %v, want %v", az.Account, tt.wantAccount)
			}
			if az.Container != tt.wantContainer {
				t.Errorf("Container = %v, want %v", az.Container, tt.wantContainer)
			}
			if az.KeyPrefix != tt.wantKeyPrefix {
				t.Errorf("KeyPrefix = %v, want %v", az.KeyPrefix, tt.wantKeyPrefix)
			}
			if az.VerifyUpload != tt.wantVerify {
				t.Errorf("VerifyUpload = %v, want %v", az.VerifyUpload, tt.wantVerify)
			}
		})
	}
}
