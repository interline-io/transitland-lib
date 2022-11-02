package request

import (
	"context"
	"io"
	"os"
	"strings"

	"github.com/interline-io/transitland-lib/tl"
)

type Local struct{}

func (Local) Download(ctx context.Context, ustr string, secret tl.Secret, auth tl.FeedAuthorization) (io.ReadCloser, int, error) {
	r, reqErr := os.Open(strings.TrimPrefix(ustr, "file://"))
	return r, 0, reqErr
}

func (Local) Upload(ctx context.Context, ustr string, secret tl.Secret, uploadFile io.Reader) error {
	// Do not overwrite files
	// out, err := os.OpenFile(ustr, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0666)
	out, err := os.Create(ustr)
	if err != nil {
		return err
	}
	_, err = io.Copy(out, uploadFile)
	return err
}
