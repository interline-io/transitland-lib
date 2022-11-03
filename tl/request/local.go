package request

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/interline-io/transitland-lib/tl"
)

type Local struct {
	Directory string
}

func (r Local) Download(ctx context.Context, ustr string, secret tl.Secret, auth tl.FeedAuthorization) (io.ReadCloser, int, error) {
	fmt.Println("opening:", r.Directory, "-", ustr)
	rd, err := os.Open(strings.TrimPrefix(filepath.Join(r.Directory, ustr), "file://"))
	return rd, 0, err
}

func (r Local) Upload(ctx context.Context, key string, secret tl.Secret, uploadFile io.Reader) error {
	// Do not overwrite files
	fn := filepath.Join(r.Directory, key)
	out, err := os.OpenFile(fn, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0666)
	if err != nil {
		return err
	}
	_, err = io.Copy(out, uploadFile)
	return err
}
