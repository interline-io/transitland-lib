package request

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/url"

	"github.com/interline-io/transitland-lib/tl"
	"github.com/jlaffaye/ftp"
)

type Ftp struct{}

func (Ftp) Download(ctx context.Context, ustr string, secret tl.Secret, auth tl.FeedAuthorization) (io.ReadCloser, int, error) {
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
		secret.Username = "anonymous"
		secret.Password = "anonymous"
	}
	err = c.Login(secret.Username, secret.Password)
	if err != nil {
		return nil, 0, errors.New("could not connect to server")
	}
	r, err := c.Retr(u.Path)
	if err != nil {
		// return error directly
		return nil, 0, err
	}
	return r, 0, nil
}
