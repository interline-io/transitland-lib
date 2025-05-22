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
