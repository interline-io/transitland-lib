package request

import (
	"context"
	"errors"
	"io"
	"net"
	"net/url"
	"strings"
	"unicode"

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
	if auth.Type == "basic_auth" {
		if err := checkSecretHost(r.secret, u.Hostname()); err != nil {
			return nil, 0, err
		}
	} else {
		r.secret.Username = "anonymous"
		r.secret.Password = "anonymous"
	}
	// A line break in the path would end the RETR command and start another.
	if strings.ContainsFunc(u.Path, unicode.IsControl) {
		return nil, 0, errors.New("ftp path contains a control character")
	}
	p := u.Port()
	if p == "" {
		p = "21"
	}
	// The data connections go through the same dial func, so the SSRF guard
	// also checks any address a PASV reply names.
	c, err := ftp.Dial(net.JoinHostPort(u.Hostname(), p), ftp.DialWithDialFunc(func(network, address string) (net.Conn, error) {
		return safeDialer.DialContext(ctx, network, address)
	}))
	if err != nil {
		return nil, 0, errors.New("could not connect to server")
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
