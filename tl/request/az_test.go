package request

import (
	"context"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"testing"

	"github.com/interline-io/transitland-lib/tl"
)

func TestAzRequest(t *testing.T) {
	azKey := "test-az-upload.txt"
	azUri := os.Getenv("TL_TEST_AZ_STORAGE")
	testData := []byte("test azure file upload")
	if azUri == "" {
		t.Skip("Set TL_TEST_AZ_STORAGE for this test")
		return
	}
	t.Run("upload", func(t *testing.T) {
		rw, err := os.CreateTemp(t.TempDir(), "test-az-upload.txt")
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
		uploader, err := NewAzFromUrl(azUri)
		if err != nil {
			t.Fatal(err)
		}
		if err := uploader.Upload(context.Background(), azKey, tl.Secret{}, r); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("download", func(t *testing.T) {
		// Download again
		t.Log("downloading from:", azUri)
		downloader, err := NewAzFromUrl(azUri)
		if err != nil {
			t.Fatal(err)
		}
		downloadReader, _, err := downloader.Download(context.Background(), azKey, tl.Secret{}, tl.FeedAuthorization{})
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
		t.Log("creating signed url:", azUri)
		downloader, err := NewAzFromUrl(azUri)
		if err != nil {
			t.Fatal(err)
		}
		signedUrl, err := downloader.CreateSignedUrl(context.Background(), azKey, "download.zip", tl.Secret{})
		if err != nil {
			t.Fatal(err)
		}
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
