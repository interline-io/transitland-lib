package metrics

import (
	"net/http"
	"time"

	"github.com/interline-io/log"
)

func WithMetric(m ApiMetric) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			t := time.Now()
			sw := newstatusResponseWriter(w)
			next.ServeHTTP(sw, r)
			td := float64(time.Since(t).Milliseconds()) / 1000.0
			log.Trace().
				Str("method", r.Method).
				Int("code", sw.statusCode).
				Int64("http_request_size_bytes", r.ContentLength).
				Int64("http_response_size_bytes", sw.bytesWritten).
				Float64("http_request_duration_seconds", td).
				Msgf("metrics")
			m.AddResponse(r.Method, sw.statusCode, r.ContentLength, sw.bytesWritten, td)
		})
	}
}

// statusResponseWriter adapted from
// https://www.alexedwards.net/blog/how-to-use-the-http-responsecontroller-type
type statusResponseWriter struct {
	http.ResponseWriter // Embed a http.ResponseWriter
	statusCode          int
	headerWritten       bool
	bytesWritten        int64
}

func newstatusResponseWriter(w http.ResponseWriter) *statusResponseWriter {
	return &statusResponseWriter{
		ResponseWriter: w,
		statusCode:     http.StatusOK,
	}
}

func (mw *statusResponseWriter) WriteHeader(statusCode int) {
	mw.ResponseWriter.WriteHeader(statusCode)
	if !mw.headerWritten {
		mw.statusCode = statusCode
		mw.headerWritten = true
	}
}

func (mw *statusResponseWriter) Write(b []byte) (int, error) {
	mw.headerWritten = true
	mw.bytesWritten += int64(len(b))
	return mw.ResponseWriter.Write(b)
}

func (mw *statusResponseWriter) Unwrap() http.ResponseWriter {
	return mw.ResponseWriter
}
