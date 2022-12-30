package request

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"testing"

	"github.com/interline-io/transitland-lib/tl"
)

func TestS3Request(t *testing.T) {
	s3Key := "test-s3-upload.txt"
	s3Uri := os.Getenv("TL_TEST_S3_STORAGE")
	testData := []byte("test s3 file upload")
	if s3Uri == "" {
		t.Skip("Set TL_TEST_S3_STORAGE for this test")
		return
	}
	t.Run("upload", func(t *testing.T) {
		rw, err := os.CreateTemp(t.TempDir(), s3Key)
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
		t.Log("uploading to:", s3Key)
		uploader, err := NewS3FromUrl(s3Uri)
		if err != nil {
			t.Fatal(err)
		}
		if err := uploader.Upload(context.Background(), s3Key, tl.Secret{}, r); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("download", func(t *testing.T) {
		// Download again
		t.Log("downloading from:", s3Uri)
		downloader, err := NewS3FromUrl(s3Uri)
		if err != nil {
			t.Fatal(err)
		}
		downloadReader, _, err := downloader.Download(context.Background(), s3Key, tl.Secret{}, tl.FeedAuthorization{})
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
	})
	t.Run("signed url", func(t *testing.T) {
		// Download again
		t.Log("creating signed url:", s3Uri)
		downloader, err := NewS3FromUrl(s3Uri)
		if err != nil {
			t.Fatal(err)
		}
		signedUrl, err := downloader.CreateSignedUrl(context.Background(), s3Key, tl.Secret{})
		if err != nil {
			t.Fatal(err)
		}
		fmt.Println("signed:", signedUrl)
		resp, err := http.Get(signedUrl)
		if err != nil {
			t.Error(err)
		}
		downloadData, err := ioutil.ReadAll(resp.Body)
		if string(downloadData) != string(testData) {
			t.Errorf("got data '%s', expected '%s'", string(downloadData), string(testData))
		}
	})

}
