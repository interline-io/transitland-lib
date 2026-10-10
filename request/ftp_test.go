package request

import (
	"context"
	"net"
	"testing"

	"github.com/interline-io/transitland-lib/dmfr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFtp_DownloadAuth_SecretHost(t *testing.T) {
	r := &Ftp{}
	r.SetSecret(dmfr.Secret{Username: "user", Password: "secret123", Host: "ftp.example.com"})
	_, _, err := r.DownloadAuth(context.Background(), "ftp://ftp.example.org/feed.zip", dmfr.FeedAuthorization{Type: "basic_auth"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not allowed for host")
}

func TestFtp_DownloadAuth_SSRFRejectsLoopback(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	defer ln.Close()
	accepted := make(chan struct{}, 1)
	go func() {
		if c, err := ln.Accept(); err == nil {
			accepted <- struct{}{}
			c.Close()
		}
	}()
	r := &Ftp{}
	_, _, err = r.DownloadAuth(context.Background(), "ftp://"+ln.Addr().String()+"/feed.zip", dmfr.FeedAuthorization{})
	assert.Error(t, err)
	assert.Empty(t, accepted, "the connection reached the loopback listener")
}

func TestFtp_DownloadAuth_ControlCharacters(t *testing.T) {
	r := &Ftp{}
	_, _, err := r.DownloadAuth(context.Background(), "ftp://ftp.example.com/feed.zip%0d%0aDELE%20/important", dmfr.FeedAuthorization{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "control character")
}
