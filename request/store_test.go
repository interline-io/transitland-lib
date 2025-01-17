package request

import (
	"bytes"
	"context"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/interline-io/transitland-lib/internal/testpath"
	"github.com/stretchr/testify/assert"
)

func testBucket(t *testing.T, ctx context.Context, bucket Store) {
	uploadKey := "ok.md"
	testFullkey := testpath.RelPath("testdata/request/readme.md")
	checkfunc := func(b string) bool {
		return strings.HasSuffix(b, ".txt")
	}
	checkRtFiles, err := findFiles(testpath.RelPath("testdata/request"), checkfunc)
	if err != nil {
		t.Fatal(err)
	}
	srcDir := testpath.RelPath("testdata/request")
	srcDirPrefix := "test-upload-all"

	////////
	localCheckDir, err := os.MkdirTemp("", "testBucketDownload")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(localCheckDir)

	///////
	t.Run("Upload", func(t *testing.T) {
		// Upload file
		if err := Upload(ctx, bucket, testFullkey, uploadKey); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("Download", func(t *testing.T) {
		// Now check the uploaded file
		checkfn := filepath.Join(localCheckDir, uploadKey+".download")
		if err := Download(ctx, bucket, uploadKey, checkfn); err != nil {
			t.Fatal(err)
		}
		if checkf, err := filesEqual(testFullkey, checkfn); err != nil {
			t.Fatal(err)
		} else if !checkf {
			t.Error("expected files to be equal")
		}
	})
	t.Run("UploadAll", func(t *testing.T) {
		// Upload several files
		if err := UploadAll(ctx, bucket, srcDir, srcDirPrefix, checkfunc); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("ListKeys", func(t *testing.T) {
		keys, err := bucket.ListKeys(ctx, srcDirPrefix)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, len(checkRtFiles), len(keys), "expected same number of keys in ListKeys")
	})
	t.Run("DownloadAll", func(t *testing.T) {
		// Now download and check the uploaded files
		downloadDir := filepath.Join(localCheckDir, "downloadAll")
		fns, err := DownloadAll(ctx, bucket, downloadDir, srcDirPrefix, checkfunc)
		if err != nil {
			t.Fatal(err)
		}
		assert.Equal(t, len(checkRtFiles), len(fns), "expected number of downloaded files")
		for _, checkRtFn := range checkRtFiles {
			checkDownloadFn := filepath.Join(
				downloadDir,
				stripDir(testpath.RelPath("testdata/request"), checkRtFn),
			)
			if checkRelKey, err := filesEqual(checkRtFn, checkDownloadFn); err != nil {
				t.Fatal(err)
			} else if !checkRelKey {
				t.Error("expeced files to be equal")
			}

		}
	})
	if bucketSign, ok := bucket.(Presigner); ok {
		t.Run("CreateSignedUrl", func(t *testing.T) {
			// Upload file
			signKey := "test-upload.zip"
			testData := []byte("test file upload")
			data := bytes.NewBuffer(testData)
			if err := bucket.Upload(ctx, signKey, data); err != nil {
				t.Fatal(err)
			}
			// Download again
			signedUrl, err := bucketSign.CreateSignedUrl(ctx, signKey, "download.zip")
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
		})
	}
}

func filesEqual(a string, b string) (bool, error) {
	adata, err := os.ReadFile(a)
	if err != nil {
		return false, err
	}
	bdata, err := os.ReadFile(b)
	if err != nil {
		return false, err
	}
	return slices.Equal(adata, bdata), nil
}
