package rest

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/internal/util"
	"github.com/interline-io/transitland-lib/model"
	"github.com/interline-io/transitland-lib/request"
	"github.com/interline-io/transitland-mw/meters"
	"github.com/tidwall/gjson"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

const latestFeedVersionQuery = `
query($feed_onestop_id: String!, $ids: [Int!]) {
	feeds(ids: $ids, where: { onestop_id: $feed_onestop_id }) {
	  onestop_id
	  license {
		redistribution_allowed
	  }
	  feed_versions(limit: 1) {
		sha1
	  }
	}
  }
`

func feedDownloadRtHelper(graphqlHandler http.Handler, w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	key := chi.URLParam(r, "feed_key")
	rtType := fmt.Sprintf("realtime_%s", chi.URLParam(r, "rt_type"))
	format := chi.URLParam(r, "format")
	gvars := hw{}
	if key == "" {
		util.WriteJsonError(w, "not found", http.StatusNotFound)
		return
	} else if v, err := strconv.Atoi(key); err == nil {
		gvars["ids"] = []int{v}
	} else {
		gvars["feed_onestop_id"] = key
	}

	// Check if we're allowed to redistribute feed and look up latest feed version
	feedResponse, err := makeGraphQLRequest(ctx, graphqlHandler, latestFeedVersionQuery, gvars)
	if err != nil {
		util.WriteJsonError(w, "server error", http.StatusInternalServerError)
		return
	}

	found := false
	allowed := false
	jj, err := json.Marshal(feedResponse)
	if err != nil {
		util.WriteJsonError(w, "server error", http.StatusInternalServerError)
		return
	}
	if gjson.Get(string(jj), "feeds.0.license.redistribution_allowed").String() != "no" {
		allowed = true
	}

	// Check if we have data
	rtf := model.ForContext(ctx).RTFinder
	rtMsg, ok := rtf.GetMessage(ctx, key, rtType)
	if ok && rtMsg != nil {
		found = true
	}

	// Errors if not allowed or no data
	if !found {
		util.WriteJsonError(w, "not found", http.StatusNotFound)
		return
	}
	if !allowed {
		util.WriteJsonError(w, "not authorized", http.StatusUnauthorized)
		return
	}

	var data []byte
	var marshalErr error
	if format == "json" {
		data, marshalErr = protojson.Marshal(rtMsg)
		w.Header().Add("Content-Type", "application/json")
	} else {
		data, marshalErr = proto.Marshal(rtMsg)
		w.Header().Add("Content-Type", "application/octet-stream")
	}
	if marshalErr != nil {
		util.WriteJsonError(w, "error processing result", http.StatusInternalServerError)
		return
	}
	w.Write(data)
}

// Query redirects user to download the given fv from S3 public URL
// assuming that redistribution is allowed for the feed.
func feedVersionDownloadLatestHandler(graphqlHandler http.Handler, w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	key := chi.URLParam(r, "feed_key")
	gvars := hw{}
	if key == "" {
		util.WriteJsonError(w, "not found", http.StatusNotFound)
		return
	} else if v, err := strconv.Atoi(key); err == nil {
		gvars["ids"] = []int{v}
	} else {
		gvars["feed_onestop_id"] = key
	}

	// Check if we're allowed to redistribute feed and look up latest feed version
	feedResponse, err := makeGraphQLRequest(ctx, graphqlHandler, latestFeedVersionQuery, gvars)
	if err != nil {
		util.WriteJsonError(w, "server error", http.StatusInternalServerError)
		return
	}
	found := false
	allowed := false
	json, err := json.Marshal(feedResponse)
	if err != nil {
		util.WriteJsonError(w, "server error", http.StatusInternalServerError)
		return
	}
	if gjson.Get(string(json), "feeds.0.feed_versions.0.sha1").Exists() {
		found = true
	}
	if gjson.Get(string(json), "feeds.0.license.redistribution_allowed").String() != "no" {
		allowed = true
	}
	fid := gjson.Get(string(json), "feeds.0.onestop_id").String()
	fvsha1 := gjson.Get(string(json), "feeds.0.feed_versions.0.sha1").String()
	if !found {
		util.WriteJsonError(w, "not found", http.StatusNotFound)
		return
	}
	if !allowed {
		util.WriteJsonError(w, "not authorized", http.StatusUnauthorized)
		return
	}

	downloadKey := fmt.Sprintf("%s-%s.zip", fid, fvsha1)
	cfg := model.ForContext(ctx)
	if err := serveFromStorage(w, r, cfg.Storage, fvsha1, downloadKey); err != nil {
		// Do not meter
		log.For(ctx).Error().Err(err).Msg("feed version download failed")
		return
	}
	// Send request to metering
	if apiMeter := meters.ForContext(ctx); apiMeter != nil {
		apiMeter.Meter(ctx, meters.MeterEvent{
			Name:  "feed-version-downloads",
			Value: 1.0,
			Dimensions: []meters.Dimension{
				{Key: "fv_sha1", Value: fvsha1},
				{Key: "feed_onestop_id", Value: fid},
				{Key: "is_latest_feed_version", Value: "true"},
			},
		})
	}

}

const feedVersionFileQuery = `
query($feed_version_sha1:String!, $ids: [Int!]) {
	feed_versions(limit:1, ids: $ids, where:{sha1:$feed_version_sha1}) {
	  sha1
	  feed {
		onestop_id
		license {
			redistribution_allowed
		}
	  }
	}
  }
`

// Query redirects user to download the given fv from S3 public URL
// assuming that redistribution is allowed for the feed.
func feedVersionDownloadHandler(graphqlHandler http.Handler, w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	gvars := hw{}
	key := chi.URLParam(r, "feed_version_key")
	if key == "" {
		util.WriteJsonError(w, "not found", http.StatusNotFound)
		return
	} else if v, err := strconv.Atoi(key); err == nil {
		gvars["ids"] = []int{v}
	} else {
		gvars["feed_version_sha1"] = key
	}
	// Check if we're allowed to redistribute feed
	checkfv, err := makeGraphQLRequest(ctx, graphqlHandler, feedVersionFileQuery, gvars)
	if err != nil {
		util.WriteJsonError(w, "server error", http.StatusInternalServerError)
		return
	}
	// todo: use gjson
	found := false
	allowed := false
	fid := ""
	fvsha1 := ""
	if v, ok := checkfv["feed_versions"].([]interface{}); len(v) > 0 && ok {
		if v2, ok := v[0].(hw); ok {
			fvsha1 = v2["sha1"].(string)
			if fvsha1 == key {
				found = true
			}
			if v3, ok := v2["feed"].(hw); ok {
				fid = v3["onestop_id"].(string)
				if v4, ok := v3["license"].(hw); ok {
					if v4["redistribution_allowed"] != "no" {
						allowed = true
					}
				}
			}
		}
	}
	if !found {
		util.WriteJsonError(w, "not found", http.StatusNotFound)
		return
	}
	if !allowed {
		util.WriteJsonError(w, "not authorized", http.StatusUnauthorized)
		return
	}

	downloadKey := fmt.Sprintf("%s-%s.zip", fid, fvsha1)
	cfg := model.ForContext(ctx)
	if err := serveFromStorage(w, r, cfg.Storage, fvsha1, downloadKey); err != nil {
		// Do not meter
		log.For(ctx).Error().Err(err).Msg("feed version download failed")
		return
	}
	// Send request to metering
	if apiMeter := meters.ForContext(ctx); apiMeter != nil {
		apiMeter.Meter(ctx, meters.MeterEvent{
			Name:  "feed-version-downloads",
			Value: 1.0,
			Dimensions: []meters.Dimension{
				{Key: "fv_sha1", Value: fvsha1},
				{Key: "feed_onestop_id", Value: fid},
				{Key: "is_latest_feed_version", Value: "false"},
			},
		})
	}
}

func serveFromStorage(w http.ResponseWriter, r *http.Request, storage string, fvsha1 string, downloadKey string) error {
	ctx := r.Context()
	store, err := request.GetStore(storage)
	if err != nil {
		util.WriteJsonError(w, "failed access file", http.StatusInternalServerError)
		return fmt.Errorf("failed to access file; could not get from storage: %w", err)
	}
	fvkey := fmt.Sprintf("%s.zip", fvsha1)
	if v, ok := store.(request.Presigner); ok {
		signedUrl, err := v.CreateSignedUrl(ctx, fvkey, downloadKey)
		if err != nil {
			util.WriteJsonError(w, "failed access file", http.StatusInternalServerError)
			return fmt.Errorf("failed to access file; could not presign: %w", err)
		}
		w.Header().Add("Location", signedUrl)
		w.WriteHeader(http.StatusFound)
	} else {
		rdr, _, err := store.Download(ctx, fvkey)
		if err != nil {
			util.WriteJsonError(w, "failed access file", http.StatusInternalServerError)
			return fmt.Errorf("failed to access file; not authorized: %w", err)
		}
		if _, err := io.Copy(w, rdr); err != nil {
			util.WriteJsonError(w, "failed access file", http.StatusInternalServerError)
			return fmt.Errorf("failed to access file; failed to copy to client: %w", err)
		}
	}
	return nil
}
