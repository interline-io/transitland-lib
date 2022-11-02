package request

import (
	"context"
	"crypto/sha1"
	"fmt"
	"io"
	"os"
	"testing"

	"github.com/interline-io/transitland-lib/tl"
)

func TestAzRequestDownload(t *testing.T) {
	azUri := os.Getenv("TL_TEST_AZ_URI")
	azSha1 := os.Getenv("TL_TEST_AZ_SHA1")
	if azUri == "" || azSha1 == "" {
		t.Skip("Set TL_TEST_AZ_URI and TL_TEST_AZ_SHA1 for this test")
		return
	}
	downloader := Az{}
	r, _, err := downloader.Download(context.Background(), azUri, tl.Secret{}, tl.FeedAuthorization{})
	if err != nil {
		t.Fatal(err)
	}
	h := sha1.New()
	io.Copy(h, r)
	sha1 := fmt.Sprintf("%x", h.Sum(nil))
	if sha1 != azSha1 {
		t.Errorf("got sha1 value '%s', expected '%s'", sha1, azSha1)
	}
}

func TestAzRequestUpload(t *testing.T) {
	azUri := os.Getenv("TL_TEST_AZ_UPLOAD")
	if azUri == "" {
		t.Skip("Set TL_TEST_AZ_UPLOAD for this test")
		return
	}
	testData := []byte("test azure file upload")
	rw, err := os.CreateTemp(t.TempDir(), "az-upload.txt")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := rw.Write(testData); err != nil {
		t.Fatal(err)
	}
	if err := rw.Close(); err != nil {
		t.Fatal(err)
	}
	r, err := os.Open(rw.Name())
	// Upload file
	t.Log("uploading to:", azUri)
	uploader := Az{}
	if err := uploader.Upload(context.Background(), azUri, tl.Secret{}, r); err != nil {
		t.Fatal(err)
	}
	// Download again
	t.Log("downloading from:", azUri)
	downloader := Az{}
	downloadReader, _, err := downloader.Download(context.Background(), azUri, tl.Secret{}, tl.FeedAuthorization{})
	if err != nil {
		t.Fatal(err)
	}
	downloadData, err := io.ReadAll(downloadReader)
	if err != nil {
		t.Fatal(err)
	}
	if string(downloadData) != string(testData) {
		t.Errorf("got data '%s', expected '%s'", string(downloadData), string(testData))
	}
}
