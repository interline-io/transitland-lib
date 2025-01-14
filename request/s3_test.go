package request

import (
	"context"
	"os"
	"testing"
)

func TestS3(t *testing.T) {
	ctx := context.TODO()
	s3Uri := os.Getenv("TL_TEST_S3_STORAGE")
	if s3Uri == "" {
		t.Skip("Set TL_TEST_S3_STORAGE for this test")
		return
	}
	b, err := NewS3FromUrl(s3Uri)
	if err != nil {
		t.Fatal(err)
	}
	testBucket(t, ctx, b)
}
