package rest

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
)

func TestFeedVersionDownloadRequest(t *testing.T) {
	g := os.Getenv("TL_TEST_GTFSDIR")
	if g == "" {
		t.Skip("TL_TEST_GTFSDIR not set - skipping")
	}
	cfg := testRestConfig()
	cfg.GtfsDir = g
	restSrv := cfg.srv
	t.Run("ok", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/feed_versions/e535eb2b3b9ac3ef15d82c56575e914575e732e0/download", nil)
		rr := httptest.NewRecorder()
		restSrv.ServeHTTP(rr, req)
		if sc := rr.Result().StatusCode; sc != 200 {
			t.Errorf("got status code %d, expected 200", sc)
		}
		if sc := rr.Result().ContentLength; sc != 456139 {
			t.Errorf("got %d bytes, expected 456139", sc)
		}
	})
	t.Run("not authorized", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/feed_versions/d2813c293bcfd7a97dde599527ae6c62c98e66c6/download", nil)
		rr := httptest.NewRecorder()
		restSrv.ServeHTTP(rr, req)
		if sc := rr.Result().StatusCode; sc != 401 {
			t.Errorf("got status code %d, expected 401", sc)
		}
	})
	t.Run("not found", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/feed_versions/asdxyz/download", nil)
		rr := httptest.NewRecorder()
		restSrv.ServeHTTP(rr, req)
		if sc := rr.Result().StatusCode; sc != 404 {
			t.Errorf("got status code %d, expected 404", sc)
		}
	})
}

func TestFeedDownloadLatestRequest(t *testing.T) {
	g := os.Getenv("TL_TEST_GTFSDIR")
	if g == "" {
		t.Skip("TL_TEST_GTFSDIR not set - skipping")
	}
	cfg := testRestConfig()
	cfg.GtfsDir = g
	restSrv := cfg.srv
	t.Run("ok", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/feeds/BA/download_latest_feed_version", nil)
		rr := httptest.NewRecorder()
		restSrv.ServeHTTP(rr, req)
		if sc := rr.Result().StatusCode; sc != 200 {
			t.Errorf("got status code %d, expected 200", sc)
		}
		if sc := rr.Result().ContentLength; sc != 456139 {
			t.Errorf("got %d bytes, expected 456139", sc)
		}
	})
	t.Run("not authorized", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/feeds/CT/download_latest_feed_version", nil)
		rr := httptest.NewRecorder()
		restSrv.ServeHTTP(rr, req)
		if sc := rr.Result().StatusCode; sc != 401 {
			t.Errorf("got status code %d, expected 401", sc)
		}
	})
	t.Run("not found", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/feeds/asdxyz/download_latest_feed_version", nil)
		rr := httptest.NewRecorder()
		restSrv.ServeHTTP(rr, req)
		if sc := rr.Result().StatusCode; sc != 404 {
			t.Errorf("got status code %d, expected 404", sc)
		}
	})
}
