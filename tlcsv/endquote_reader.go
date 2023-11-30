package tlcsv

import (
	"bytes"
	"io"
)

func newEndquoteReader(rd io.Reader) *endquoteReader {
	r := &endquoteReader{
		rd: rd,
	}
	return r
}

type endquoteReader struct {
	rd  io.Reader // reader provided by the client
	buf []byte    // buffered data
	err error     // last error
}

func (r *endquoteReader) Read(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}
	if len(p) > len(r.buf) {
		r.buf = make([]byte, len(p))
	}
	n, r.err = r.rd.Read(r.buf)
	if n == 0 || r.err != nil {
		return 0, r.err
	}
	r.buf = bytes.ReplaceAll(r.buf[:n], []byte("\" \n"), []byte("\"\n"))
	n = copy(p, r.buf)
	r.buf = nilIfEmpty(r.buf[n:])
	return n, r.err
}

func nilIfEmpty(buf []byte) (res []byte) {
	if len(buf) > 0 {
		res = buf
	}
	return
}
