package request

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"

	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/jlaffaye/ftp"
)

type Ftp struct {
	secret dmfr.Secret
}

func (r *Ftp) SetSecret(secret dmfr.Secret) error {
	r.secret = secret
	return nil
}

func (r Ftp) Download(ctx context.Context, ustr string) (io.ReadCloser, int, error) {
	return r.DownloadAuth(ctx, ustr, dmfr.FeedAuthorization{})
}

func (r Ftp) DownloadAuth(ctx context.Context, ustr string, auth dmfr.FeedAuthorization) (io.ReadCloser, int, error) {
	// Download FTP
	u, err := url.Parse(ustr)
	if err != nil {
		return nil, 0, errors.New("could not parse url")
	}
	p := u.Port()
	if p == "" {
		p = "21"
	}
	c, err := ftp.Dial(fmt.Sprintf("%s:%s", u.Hostname(), p), ftp.DialWithContext(ctx))
	if err != nil {
		return nil, 0, errors.New("could not connect to server")
	}
	if auth.Type != "basic_auth" {
		r.secret.Username = "anonymous"
		r.secret.Password = "anonymous"
	}
	err = c.Login(r.secret.Username, r.secret.Password)
	if err != nil {
		return nil, 0, errors.New("could not connect to server")
	}
	rio, err := c.Retr(u.Path)
	if err != nil {
		// return error directly
		return nil, 0, err
	}
	return rio, 0, nil
}
