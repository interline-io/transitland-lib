package request

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/interline-io/transitland-lib/dmfr"
)

func TestLocalUpload(t *testing.T) {
	ctx := context.TODO()
	testData := []byte("test local file upload")
	rw, err := os.CreateTemp(t.TempDir(), "local-upload.txt")
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
	localUri := filepath.Join(t.TempDir(), "test.txt")
	t.Log("uploading to:", localUri)
	uploader := Local{}
	if err := uploader.Upload(ctx, localUri, dmfr.Secret{}, r); err != nil {
		t.Fatal(err)
	}
	// Download again
	downloader := Local{}
	t.Log("downloading from:", localUri)
	downloadReader, _, err := downloader.Download(ctx, localUri, dmfr.Secret{}, dmfr.FeedAuthorization{})
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
