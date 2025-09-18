package httpcache

import (
	"bytes"
	"io"
	"io/ioutil"
	"net/http"
)

type cacheResponse struct {
	Headers    map[string][]string
	Body       []byte
	StatusCode int
}

func newCacheResponse(res *http.Response) (*cacheResponse, error) {
	// Save and restore body
	var bodyB []byte
	if res.Body != nil {
		bodyB, _ = ioutil.ReadAll(res.Body)
		res.Body = ioutil.NopCloser(bytes.NewBuffer(bodyB))
	}

	c := cacheResponse{}
	c.Body = bodyB
	c.Headers = map[string][]string{}
	for k, v := range res.Header {
		c.Headers[k] = v
	}
	c.StatusCode = res.StatusCode
	return &c, nil
}

func fromCacheResponse(a *cacheResponse) (*http.Response, error) {
	rr := http.Response{}
	rr.Body = io.NopCloser(bytes.NewReader(a.Body))
	rr.ContentLength = int64(len(a.Body))
	rr.StatusCode = a.StatusCode
	rr.Header = http.Header{}
	for k, v := range a.Headers {
		for _, vv := range v {
			rr.Header.Add(k, vv)
		}
	}
	return &rr, nil
}
