package rest

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/internal/util"
	"github.com/interline-io/transitland-lib/request"
	"github.com/interline-io/transitland-lib/rt"
	"github.com/interline-io/transitland-lib/server/meters"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/tidwall/gjson"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

const latestFeedVersionQuery = `
query($feed_onestop_id: String, $ids: [Int!]) {
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

const feedVersionFileQuery = `
query($feed_version_sha1: String, $ids: [Int!]) {
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

func feedDownloadRtHelper(graphqlHandler http.Handler, w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	key := chi.URLParam(r, "feed_key")
	rtType := fmt.Sprintf("realtime_%s", chi.URLParam(r, "rt_type"))
	format := chi.URLParam(r, "format")
	if key == "" {
		util.WriteJsonError(w, "not found", http.StatusNotFound)
		return
	}

	// This endpoint serves realtime messages rather than a feed version file, so
	// the feed version is not required; an RT-only feed has no feed versions.
	//
	// The feed itself is required. The RT message store is a cache keyed by feed,
	// with no permission check of its own, so this lookup is the only thing that
	// establishes the caller may see this feed at all — and a feed they cannot see
	// resolves with an empty license, which reads as redistributable.
	d, err := LookupLatestFeedVersionDownload(ctx, graphqlHandler, key)
	if err != nil {
		util.WriteJsonError(w, "server error", http.StatusInternalServerError)
		return
	}
	if d.FeedOnestopID == "" {
		util.WriteJsonError(w, "not found", http.StatusNotFound)
		return
	}
	found := false
	allowed := d.RedistributionAllowed

	// Check if we have data
	rtf := model.ForContext(ctx).RTFinder
	rtMsg, ok := rtf.GetMessage(ctx, d.FeedOnestopID, rtType)
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
	switch format {
	case "json":
		data, marshalErr = protojson.Marshal(rtMsg)
		w.Header().Add("Content-Type", "application/json")
	case "geojson", "geojsonl":
		// Only support GeoJSON formats for vehicle positions
		if rtType != "realtime_vehicle_positions" {
			util.WriteJsonError(w, "geojson format only supported for vehicle_positions", http.StatusBadRequest)
			return
		}

		if format == "geojsonl" {
			// Use streaming for GeoJSONL to reduce memory usage
			w.Header().Add("Content-Type", "application/geo+json-seq")
			marshalErr = rt.VehiclePositionsToGeoJSONLStream(rtMsg, w)
			if marshalErr != nil {
				util.WriteJsonError(w, "error processing result", http.StatusInternalServerError)
				return
			}
			return // Already written to response
		} else {
			// Use non-streaming for standard GeoJSON
			data, marshalErr = rt.VehiclePositionsToGeoJSON(rtMsg, false)
			w.Header().Add("Content-Type", "application/geo+json")
		}
	default:
		data, marshalErr = proto.Marshal(rtMsg)
		w.Header().Add("Content-Type", "application/octet-stream")
	}
	if marshalErr != nil {
		util.WriteJsonError(w, "error processing result", http.StatusInternalServerError)
		return
	}
	w.Write(data)
}

// feedVersionDownloadMeter is the meter for feed version zip downloads: the
// quota checked before serving, and the usage recorded after.
const feedVersionDownloadMeter = "feed-version-downloads"

// downloadDimensions describes one feed version download for metering.
//
// The same dimensions gate the quota and record the usage. A limit applies
// only when its own dimensions are a subset of these, so checking with fewer
// dimensions than are recorded would silently skip dimension-scoped limits.
func downloadDimensions(d FeedVersionDownload) meters.Dimensions {
	return meters.Dimensions{
		{Key: "fv_sha1", Value: d.FeedVersionSHA1},
		{Key: "feed_onestop_id", Value: d.FeedOnestopID},
		{Key: "is_latest_feed_version", Value: strconv.FormatBool(d.IsLatestFeedVersion)},
	}
}

// checkDownloadQuota reports whether this download is within the caller's
// quota. It allows the download when no meter is configured, which is the
// case outside a metered deployment and in tests.
func checkDownloadQuota(ctx context.Context, dims meters.Dimensions) bool {
	// The context holds a full Meterer; ForContext narrows it to the
	// recording half, so reading a quota needs the reader back.
	meterReader, ok := meters.ForContext(ctx).(meters.MeterReader)
	if !ok {
		return true
	}
	allowed, err := meterReader.Check(ctx, feedVersionDownloadMeter, 1.0, dims)
	if err != nil {
		log.For(ctx).Error().Err(err).Msg("feed version download quota check failed")
	}
	return allowed
}

// recordDownload records one served feed version download against the quota
// that admitted it.
//
// The event carries a unique id, which is the delivery's idempotency key: the
// meter transport retries a failed batch, and without one a retry lands as a
// second indistinguishable usage record that cannot afterwards be told apart
// from a real second download.
func recordDownload(ctx context.Context, dims meters.Dimensions) {
	apiMeter := meters.ForContext(ctx)
	if apiMeter == nil {
		return
	}
	if err := apiMeter.Meter(ctx, meters.NewMeterEvent(feedVersionDownloadMeter, 1.0, dims)); err != nil {
		log.For(ctx).Error().Err(err).Msg("feed version download metering failed")
	}
}

// FeedVersionDownload identifies one feed version file and says whether its
// feed's license permits redistributing it.
//
// Zero values mean the lookup matched nothing: either the key names no feed
// version, or the caller may not see it. Redistribution is reported, not
// enforced.
type FeedVersionDownload struct {
	FeedOnestopID         string
	FeedVersionSHA1       string
	RedistributionAllowed bool
	// IsLatestFeedVersion records that this was resolved as a feed's current
	// version rather than requested by key. Download quotas are scoped on it.
	IsLatestFeedVersion bool
}

// LookupFeedVersionDownload resolves an integer id or a sha1 to the feed version
// file it names.
func LookupFeedVersionDownload(ctx context.Context, graphqlHandler http.Handler, key string) (FeedVersionDownload, error) {
	var d FeedVersionDownload
	vars, ok := downloadVars(key, "feed_version_sha1")
	if !ok {
		return d, nil
	}
	body, err := downloadLookup(ctx, graphqlHandler, feedVersionFileQuery, vars)
	if err != nil {
		return d, err
	}
	d.FeedVersionSHA1 = gjson.Get(body, "feed_versions.0.sha1").String()
	d.FeedOnestopID = gjson.Get(body, "feed_versions.0.feed.onestop_id").String()
	d.RedistributionAllowed = gjson.Get(body, "feed_versions.0.feed.license.redistribution_allowed").String() != "no"
	return d, nil
}

// LookupLatestFeedVersionDownload resolves an integer id or a feed Onestop ID to
// that feed's current version.
//
// A feed with no versions is not an error: RT-only feeds have none, and the
// redistribution answer is still meaningful for them.
func LookupLatestFeedVersionDownload(ctx context.Context, graphqlHandler http.Handler, feedKey string) (FeedVersionDownload, error) {
	d := FeedVersionDownload{IsLatestFeedVersion: true}
	vars, ok := downloadVars(feedKey, "feed_onestop_id")
	if !ok {
		return d, nil
	}
	body, err := downloadLookup(ctx, graphqlHandler, latestFeedVersionQuery, vars)
	if err != nil {
		return d, err
	}
	d.FeedVersionSHA1 = gjson.Get(body, "feeds.0.feed_versions.0.sha1").String()
	d.FeedOnestopID = gjson.Get(body, "feeds.0.onestop_id").String()
	d.RedistributionAllowed = gjson.Get(body, "feeds.0.license.redistribution_allowed").String() != "no"
	return d, nil
}

// ServeFeedVersion writes the feed version file, redirecting to a signed URL
// when the store supports one.
//
// It writes an error response itself before returning a non-nil error, so a
// caller should log the error rather than answer again.
func ServeFeedVersion(ctx context.Context, w http.ResponseWriter, storage string, d FeedVersionDownload) error {
	downloadKey := fmt.Sprintf("%s-%s.zip", d.FeedOnestopID, d.FeedVersionSHA1)
	return serveFromStorage(ctx, w, storage, d.FeedVersionSHA1, downloadKey)
}

// downloadVars builds the query variables for a key that is either an integer
// id or the string identifier named by nameVar. An empty key matches nothing.
func downloadVars(key string, nameVar string) (hw, bool) {
	if key == "" {
		return nil, false
	}
	if v, err := strconv.Atoi(key); err == nil {
		return hw{"ids": []int{v}}, true
	}
	return hw{nameVar: key}, true
}

// downloadLookup runs one lookup query and returns the response as JSON for
// gjson to read.
func downloadLookup(ctx context.Context, graphqlHandler http.Handler, query string, vars hw) (string, error) {
	response, err := makeGraphQLRequest(ctx, graphqlHandler, query, vars)
	if err != nil {
		return "", err
	}
	body, err := json.Marshal(response)
	if err != nil {
		return "", err
	}
	return string(body), nil
}

// Query redirects user to download the given fv from S3 public URL
// assuming that redistribution is allowed for the feed.
func feedVersionDownloadLatestHandler(graphqlHandler http.Handler, w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	d, err := LookupLatestFeedVersionDownload(ctx, graphqlHandler, chi.URLParam(r, "feed_key"))
	if err != nil {
		util.WriteJsonError(w, "server error", http.StatusInternalServerError)
		return
	}
	if d.FeedVersionSHA1 == "" || d.FeedOnestopID == "" {
		util.WriteJsonError(w, "not found", http.StatusNotFound)
		return
	}
	if !d.RedistributionAllowed {
		util.WriteJsonError(w, "not authorized", http.StatusUnauthorized)
		return
	}

	dims := downloadDimensions(d)
	if !checkDownloadQuota(ctx, dims) {
		util.WriteJsonError(w, "too many requests", http.StatusTooManyRequests)
		return
	}
	if err := ServeFeedVersion(ctx, w, model.ForContext(ctx).Storage, d); err != nil {
		// Do not meter
		log.For(ctx).Error().Err(err).Msg("feed version download failed")
		return
	}
	recordDownload(ctx, dims)
}

func feedVersionDownloadHandler(graphqlHandler http.Handler, w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	d, err := LookupFeedVersionDownload(ctx, graphqlHandler, chi.URLParam(r, "feed_version_key"))
	if err != nil {
		util.WriteJsonError(w, "server error", http.StatusInternalServerError)
		return
	}
	if d.FeedVersionSHA1 == "" || d.FeedOnestopID == "" {
		util.WriteJsonError(w, "not found", http.StatusNotFound)
		return
	}
	if !d.RedistributionAllowed {
		util.WriteJsonError(w, "not authorized", http.StatusUnauthorized)
		return
	}

	dims := downloadDimensions(d)
	if !checkDownloadQuota(ctx, dims) {
		util.WriteJsonError(w, "too many requests", http.StatusTooManyRequests)
		return
	}
	if err := ServeFeedVersion(ctx, w, model.ForContext(ctx).Storage, d); err != nil {
		// Do not meter
		log.For(ctx).Error().Err(err).Msg("feed version download failed")
		return
	}
	recordDownload(ctx, dims)
}

func serveFromStorage(ctx context.Context, w http.ResponseWriter, storage string, fvsha1 string, downloadKey string) error {
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
		defer rdr.Close()
		if _, err := io.Copy(w, rdr); err != nil {
			util.WriteJsonError(w, "failed access file", http.StatusInternalServerError)
			return fmt.Errorf("failed to access file; failed to copy to client: %w", err)
		}
	}
	return nil
}
