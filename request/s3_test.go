package request

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"os"
	"testing"
)

func TestS3(t *testing.T) {
	ctx := context.TODO()
	s3Uri := os.Getenv("TL_TEST_S3_STORAGE")
	b, err := NewS3FromUrl(s3Uri)
	if err != nil {
		t.Fatal(err)
	}
	testBucket(t, ctx, b)
}

func TestS3CreateSignedUrl(t *testing.T) {
	ctx := context.TODO()
	s3Uri := os.Getenv("TL_TEST_S3_STORAGE")
	s3Key := "test-s3-upload.zip"
	testData := []byte("test s3 file upload")
	if s3Uri == "" {
		t.Skip("Set TL_TEST_S3_STORAGE for this test")
		return
	}
	///////
	b, err := NewS3FromUrl(s3Uri)
	if err != nil {
		t.Fatal(err)
	}
	// Upload file
	data := bytes.NewBuffer(testData)
	if err := b.Upload(ctx, s3Key, data); err != nil {
		t.Fatal(err)
	}

	// Download again
	t.Log("creating signed url:", s3Uri)
	signedUrl, err := b.CreateSignedUrl(ctx, s3Key, "download.zip")
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.Get(signedUrl)
	if err != nil {
		t.Error(err)
	}
	downloadData, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if string(downloadData) != string(testData) {
		t.Errorf("got data '%s', expected '%s'", string(downloadData), string(testData))
	}
}
