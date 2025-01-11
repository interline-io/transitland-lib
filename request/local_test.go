package request

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/interline-io/transitland-lib/internal/testpath"
)

func TestLocal(t *testing.T) {
	ctx := context.Background()
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
	if err != nil {
		t.Fatal(err)
	}
	// Upload file
	localUri := filepath.Join(t.TempDir(), "test.txt")
	downloader := Local{}
	t.Run("Upload", func(t *testing.T) {
		t.Log("uploading to:", localUri)
		uploader := Local{}
		if err := uploader.Upload(ctx, localUri, dmfr.Secret{}, r); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("Download", func(t *testing.T) {
		// Download again
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
	})
	t.Run("DownloadAll", func(t *testing.T) {
		baseDir := testpath.RelPath("testdata")
		d := Local{Directory: baseDir}
		fns, err := d.DownloadAll(ctx, t.TempDir(), dmfr.Secret{}, func(key string) bool {
			return strings.HasSuffix(key, ".pb")
		})
		if err != nil {
			t.Fatal(err)
		}
		if len(fns) == 0 {
			t.Fatal("did not copy any files")
		}
		for _, fn := range fns {
			if st, err := os.Stat(fn); err != nil {
				t.Fatal(err)
			} else {
				fmt.Printf("%#v\n", st)
			}
		}
	})
	t.Run("UploadAll", func(t *testing.T) {

	})
}
