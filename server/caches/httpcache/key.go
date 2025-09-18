package httpcache

import (
	"bytes"
	"crypto/sha1"
	"fmt"
	"io/ioutil"
	"net/http"
)

type HTTPKey func(*http.Request) string

func DefaultKey(req *http.Request) string {
	// Save and restore body
	var bodyB []byte
	if req.Body != nil {
		bodyB, _ = ioutil.ReadAll(req.Body)
		req.Body = ioutil.NopCloser(bytes.NewBuffer(bodyB))
	}
	// Create key
	s := sha1.New()
	s.Write([]byte(req.Method))
	s.Write([]byte(req.URL.String()))
	s.Write(bodyB)
	for k, v := range req.Header {
		s.Write([]byte(k))
		for _, vv := range v {
			s.Write([]byte(vv))
		}
	}
	return fmt.Sprintf("%x", s.Sum(nil))
}

func NoHeadersKey(req *http.Request) string {
	// Save and restore body
	var bodyB []byte
	if req.Body != nil {
		bodyB, _ = ioutil.ReadAll(req.Body)
		req.Body = ioutil.NopCloser(bytes.NewBuffer(bodyB))
	}
	// Create key
	s := sha1.New()
	s.Write([]byte(req.Method))
	s.Write([]byte(req.URL.String()))
	s.Write(bodyB)
	return fmt.Sprintf("%x", s.Sum(nil))
}
