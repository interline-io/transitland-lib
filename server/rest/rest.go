package rest

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/internal/util"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/rs/zerolog"
)

// DEFAULTLIMIT is the default API limit
const DEFAULTLIMIT = 20

// MAXLIMIT is the API limit maximum
var MAXLIMIT = 1_000

// MAXRADIUS is the maximum point search radius
const MAXRADIUS = 100 * 1000.0

// Handler constructors, one per endpoint. Each binds the request type that
// serves it, so a caller mounting these chooses paths and middleware without
// also having to know which request type belongs to which route.
//
// Index handlers answer with an empty array when nothing matches; entity
// handlers answer 404. The comment on each names the chi URL parameters it
// reads: makeHandler copies them by name into the request struct, so a mount
// that spells one differently silently drops that filter.

// NewFeedIndexHandler serves the feed index. Reads {format}.
func NewFeedIndexHandler(graphqlHandler http.Handler) http.HandlerFunc {
	return makeIndexHandler[FeedRequest](graphqlHandler)
}

// NewFeedEntityHandler serves one feed. Reads {feed_key}, {format}.
func NewFeedEntityHandler(graphqlHandler http.Handler) http.HandlerFunc {
	return makeEntityHandler[FeedKeyRequest](graphqlHandler)
}

// NewFeedVersionIndexHandler serves the feed version index. Reads {feed_key}, {format}.
func NewFeedVersionIndexHandler(graphqlHandler http.Handler) http.HandlerFunc {
	return makeIndexHandler[FeedVersionRequest](graphqlHandler)
}

// NewFeedVersionEntityHandler serves one feed version. Reads {feed_version_key}, {format}.
func NewFeedVersionEntityHandler(graphqlHandler http.Handler) http.HandlerFunc {
	return makeEntityHandler[FeedVersionKeyRequest](graphqlHandler)
}

// NewAgencyIndexHandler serves the agency index. Reads {format}.
func NewAgencyIndexHandler(graphqlHandler http.Handler) http.HandlerFunc {
	return makeIndexHandler[AgencyRequest](graphqlHandler)
}

// NewAgencyEntityHandler serves one agency. Reads {agency_key}, {format}.
func NewAgencyEntityHandler(graphqlHandler http.Handler) http.HandlerFunc {
	return makeEntityHandler[AgencyKeyRequest](graphqlHandler)
}

// NewRouteIndexHandler serves the route index. Reads {agency_key}, {format}.
func NewRouteIndexHandler(graphqlHandler http.Handler) http.HandlerFunc {
	return makeIndexHandler[RouteRequest](graphqlHandler)
}

// NewRouteEntityHandler serves one route. Reads {route_key}, {format}.
func NewRouteEntityHandler(graphqlHandler http.Handler) http.HandlerFunc {
	return makeEntityHandler[RouteKeyRequest](graphqlHandler)
}

// NewTripIndexHandler serves the trip index. Reads {route_key}, {format}.
func NewTripIndexHandler(graphqlHandler http.Handler) http.HandlerFunc {
	return makeIndexHandler[TripRequest](graphqlHandler)
}

// NewTripEntityHandler serves one trip. Reads {route_key}, {id}, {format} — the
// trip parameter is {id}, not {trip_key}.
func NewTripEntityHandler(graphqlHandler http.Handler) http.HandlerFunc {
	return makeEntityHandler[TripEntityRequest](graphqlHandler)
}

// NewStopIndexHandler serves the stop index. Reads {format}.
func NewStopIndexHandler(graphqlHandler http.Handler) http.HandlerFunc {
	return makeIndexHandler[StopRequest](graphqlHandler)
}

// NewStopEntityHandler serves one stop. Reads {stop_key}, {format}.
func NewStopEntityHandler(graphqlHandler http.Handler) http.HandlerFunc {
	return makeEntityHandler[StopEntityRequest](graphqlHandler)
}

// NewStopDepartureHandler serves departures for one stop. Reads {stop_key}.
func NewStopDepartureHandler(graphqlHandler http.Handler) http.HandlerFunc {
	return makeIndexHandler[StopDepartureRequest](graphqlHandler)
}

// NewOperatorIndexHandler serves the operator index. Reads {format}.
func NewOperatorIndexHandler(graphqlHandler http.Handler) http.HandlerFunc {
	return makeIndexHandler[OperatorRequest](graphqlHandler)
}

// NewOperatorEntityHandler serves one operator. Reads {operator_key}, {format}.
func NewOperatorEntityHandler(graphqlHandler http.Handler) http.HandlerFunc {
	return makeEntityHandler[OperatorKeyRequest](graphqlHandler)
}

// NewFeedVersionDownloadLatestHandler redirects to the latest feed version file
// for a feed, when its license allows redistribution. Reads {feed_key}.
func NewFeedVersionDownloadLatestHandler(graphqlHandler http.Handler) http.HandlerFunc {
	return makeHandlerFunc(graphqlHandler, feedVersionDownloadLatestHandler)
}

// NewFeedDownloadRtHandler serves the latest GTFS Realtime message for a feed.
// Reads {feed_key}, {rt_type}, {format}.
func NewFeedDownloadRtHandler(graphqlHandler http.Handler) http.HandlerFunc {
	return makeHandlerFunc(graphqlHandler, feedDownloadRtHelper)
}

// NewFeedVersionDownloadHandler serves one feed version file, when its license
// allows redistribution and the caller is within quota. Reads {feed_version_key}.
func NewFeedVersionDownloadHandler(graphqlHandler http.Handler) http.HandlerFunc {
	return makeHandlerFunc(graphqlHandler, feedVersionDownloadHandler)
}

// NewFeedVersionExportHandler builds a filtered feed version export. POST only.
func NewFeedVersionExportHandler(graphqlHandler http.Handler) http.HandlerFunc {
	return makeHandlerFunc(graphqlHandler, feedVersionExportHandler)
}

// NewServer mounts every REST endpoint on one router.
//
// This is the default mounting, for the library's own server command and its
// tests. It applies no authorization: the download and export endpoints are
// served to anyone who asks, so anything serving real traffic mounts the
// handler constructors itself and gates them.
func NewServer(graphqlHandler http.Handler) (http.Handler, error) {
	r := chi.NewRouter()

	// Redirect root to OpenAPI documentation
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		cfg := model.ForContext(r.Context())
		http.Redirect(w, r, mountPrefix(cfg.RestPrefix, r.URL.Path, "")+"/openapi.json", http.StatusMovedPermanently)
	})

	// OpenAPI Schema endpoint
	r.Handle("/openapi.json", NewOpenAPIHandler())

	r.HandleFunc("/feeds.{format}", NewFeedIndexHandler(graphqlHandler))
	r.HandleFunc("/feeds", NewFeedIndexHandler(graphqlHandler))
	r.HandleFunc("/feeds/{feed_key}.{format}", NewFeedEntityHandler(graphqlHandler))
	r.HandleFunc("/feeds/{feed_key}", NewFeedEntityHandler(graphqlHandler))
	r.HandleFunc("/feeds/{feed_key}/download_latest_feed_version", NewFeedVersionDownloadLatestHandler(graphqlHandler))

	r.HandleFunc("/feeds/{feed_key}/download_latest_rt/{rt_type}.{format}", NewFeedDownloadRtHandler(graphqlHandler))

	r.HandleFunc("/feed_versions.{format}", NewFeedVersionIndexHandler(graphqlHandler))
	r.HandleFunc("/feed_versions", NewFeedVersionIndexHandler(graphqlHandler))
	r.HandleFunc("/feed_versions/{feed_version_key}.{format}", NewFeedVersionEntityHandler(graphqlHandler))
	r.HandleFunc("/feed_versions/{feed_version_key}", NewFeedVersionEntityHandler(graphqlHandler))
	r.HandleFunc("/feeds/{feed_key}/feed_versions", NewFeedVersionIndexHandler(graphqlHandler))
	r.HandleFunc("/feed_versions/{feed_version_key}/download", NewFeedVersionDownloadHandler(graphqlHandler))
	r.Method("POST", "/feed_versions/export", NewFeedVersionExportHandler(graphqlHandler))

	r.HandleFunc("/agencies.{format}", NewAgencyIndexHandler(graphqlHandler))
	r.HandleFunc("/agencies", NewAgencyIndexHandler(graphqlHandler))
	r.HandleFunc("/agencies/{agency_key}.{format}", NewAgencyEntityHandler(graphqlHandler))
	r.HandleFunc("/agencies/{agency_key}", NewAgencyEntityHandler(graphqlHandler))

	r.HandleFunc("/routes.{format}", NewRouteIndexHandler(graphqlHandler))
	r.HandleFunc("/routes", NewRouteIndexHandler(graphqlHandler))
	r.HandleFunc("/routes/{route_key}.{format}", NewRouteEntityHandler(graphqlHandler))
	r.HandleFunc("/routes/{route_key}", NewRouteEntityHandler(graphqlHandler))
	r.HandleFunc("/agencies/{agency_key}/routes.{format}", NewRouteIndexHandler(graphqlHandler))
	r.HandleFunc("/agencies/{agency_key}/routes", NewRouteIndexHandler(graphqlHandler))

	r.HandleFunc("/routes/{route_key}/trips.{format}", NewTripIndexHandler(graphqlHandler))
	r.HandleFunc("/routes/{route_key}/trips", NewTripIndexHandler(graphqlHandler))
	r.HandleFunc("/routes/{route_key}/trips/{id}", NewTripEntityHandler(graphqlHandler))
	r.HandleFunc("/routes/{route_key}/trips/{id}.{format}", NewTripEntityHandler(graphqlHandler))

	r.HandleFunc("/stops.{format}", NewStopIndexHandler(graphqlHandler))
	r.HandleFunc("/stops", NewStopIndexHandler(graphqlHandler))
	r.HandleFunc("/stops/{stop_key}.{format}", NewStopEntityHandler(graphqlHandler))
	r.HandleFunc("/stops/{stop_key}", NewStopEntityHandler(graphqlHandler))

	r.HandleFunc("/stops/{stop_key}/departures", NewStopDepartureHandler(graphqlHandler))

	r.HandleFunc("/operators.{format}", NewOperatorIndexHandler(graphqlHandler))
	r.HandleFunc("/operators", NewOperatorIndexHandler(graphqlHandler))
	r.HandleFunc("/operators/{operator_key}.{format}", NewOperatorEntityHandler(graphqlHandler))
	r.HandleFunc("/operators/{operator_key}", NewOperatorEntityHandler(graphqlHandler))

	// OnestopID generic handler
	r.Handle("/onestop_id/{onestop_id}", &OnestopIdEntityRedirectRequest{})

	return r, nil
}

// A type that can generate a GraphQL query and variables.
type apiHandler interface {
	Query(context.Context) (string, map[string]interface{})
}

// A type that can generate a GeoJSON response.
type canProcessGeoJSON interface {
	ProcessGeoJSON(context.Context, map[string]interface{}) error
}

// A type that defines if meta should be included or not
type canIncludeNext interface {
	IncludeNext() bool
}

// A type that defines a per-page limit
type canLimit interface {
	CheckLimit() int
}

type WithCursor struct {
	Limit int `json:"limit,string"`
	After int `json:"after,string"`
}

func (w WithCursor) CheckLimit() int {
	limit := w.Limit
	if limit <= 0 {
		return DEFAULTLIMIT
	}
	if limit > MAXLIMIT {
		return MAXLIMIT
	}
	return limit
}

func (w WithCursor) CheckAfter() int {
	after := w.After
	if after < 0 {
		return 0
	}
	return after
}

// A type that specifies a JSON response key.
type hasResponseKey interface {
	ResponseKey() string
}

// mountPrefix returns the mount's absolute public base URL: restPrefix (the
// mount's parent) plus the mount segment, recovered by cutting the route-owned
// tail off urlPath. routeMarker is the matched route's literal head ("" for a
// route at "/"), and must come from the route pattern — chi.URLParam values
// stay percent-encoded while urlPath is decoded.
func mountPrefix(restPrefix string, urlPath string, routeMarker string) string {
	p := strings.TrimRight(urlPath, "/")
	mount := p
	if routeMarker != "" {
		// First occurrence, not last: urlPath is decoded, so a path parameter can
		// carry a second literal marker and re-anchor the mount inside it.
		i := strings.Index(p, routeMarker)
		if i < 0 {
			// Only reachable if something upstream rewrote the path. Fall back to
			// the bare prefix rather than splice the request path into a URL.
			return restPrefix
		}
		mount = p[:i]
	}
	// A protocol-relative mount would redirect off-site from a relative Location.
	// A non-empty restPrefix already fixes the authority, so only guard here.
	if restPrefix == "" && isProtocolRelative(mount) {
		return ""
	}
	return restPrefix + mount
}

// isProtocolRelative reports whether s reads as //host/... rather than a path.
// Backslashes count: browsers treat \ as / when parsing a special scheme.
func isProtocolRelative(s string) bool {
	isSlash := func(c byte) bool { return c == '/' || c == '\\' }
	return len(s) > 1 && isSlash(s[0]) && isSlash(s[1])
}

// checkEmptyResponse returns true if the response is empty for the given format
func checkEmptyResponse(response []byte, format string, responseKey string) bool {
	// For geojsonl format, check if response bytes are empty
	if format == "geojsonl" {
		return len(response) == 0 || len(strings.TrimSpace(string(response))) == 0
	}

	// For other formats, parse JSON and check structure
	var responseData map[string]interface{}
	if err := json.Unmarshal(response, &responseData); err != nil {
		return false
	}

	// Check for GeoJSON format (has "features" array)
	if format == "geojson" {
		if features, ok := responseData["features"].([]interface{}); ok {
			return len(features) == 0
		}
		return false
	}

	// Check for regular JSON format (has response key array)
	if entities, ok := responseData[responseKey].([]interface{}); ok {
		return len(entities) == 0
	}
	return false
}

// Alias for map string interface
type hw = map[string]interface{}

func commaSplit(v string) []string {
	var ret []string
	for _, i := range strings.Split(v, ",") {
		b := strings.TrimSpace(i)
		if b != "" {
			ret = append(ret, b)
		}
	}
	return ret
}

// checkIds returns a id as a []int{id} slice if >0, otherwise nil.
func checkIds(id int) []int {
	if id > 0 {
		return []int{id}
	}
	return nil
}

// queryToMap converts url.Values to map[string]string
func queryToMap(vars url.Values) map[string]string {
	m := map[string]string{}
	for k := range vars {
		if b := vars.Get(k); b != "" {
			m[k] = vars.Get(k)
		}
	}
	return m
}

func makeHandlerFunc(graphqlHandler http.Handler, f func(http.Handler, http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f(graphqlHandler, w, r)
	}
}

// apiHandlerPtr constrains PT to *T where the pointer is an apiHandler. Request
// params are unmarshaled into the value, so it has to be addressable; stating
// that in the constraint means the compiler enforces it.
type apiHandlerPtr[T any] interface {
	*T
	apiHandler
}

// makeIndexHandler creates a handler for list/index endpoints that returns empty arrays normally.
func makeIndexHandler[T any, PT apiHandlerPtr[T]](graphqlHandler http.Handler) http.HandlerFunc {
	return makeHandler[T, PT](graphqlHandler, false)
}

// makeEntityHandler creates a handler for single-entity endpoints that returns 404 on empty results.
func makeEntityHandler[T any, PT apiHandlerPtr[T]](graphqlHandler http.Handler) http.HandlerFunc {
	return makeHandler[T, PT](graphqlHandler, true)
}

// makeHandler wraps an apiHandler into an HandlerFunc and performs common checks.
func makeHandler[T any, PT apiHandlerPtr[T]](graphqlHandler http.Handler, notFoundOnEmpty bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		var req T
		// Bound to apiHandler so the optional-capability assertions below are
		// legal: a type-parameter value cannot be type-asserted directly.
		var handler apiHandler = PT(&req)
		opts := queryToMap(r.URL.Query())

		// Add endpoint info to context for logging
		if info, ok := handler.(interface{ RequestInfo() RequestInfo }); ok {
			endpointPath := info.RequestInfo().Path
			zerolog.Ctx(ctx).UpdateContext(func(c zerolog.Context) zerolog.Context {
				return c.Str("endpoint_path", endpointPath)
			})
		}

		// Extract URL params from request
		if rctx := chi.RouteContext(ctx); rctx != nil {
			for _, k := range rctx.URLParams.Keys {
				if k == "*" {
					continue
				}
				opts[k] = rctx.URLParam(k)
			}
		}

		// Handle format
		format := opts["format"]

		// Use json marshal/unmarshal to convert string params to correct types
		s, err := json.Marshal(opts)
		if err != nil {
			log.For(ctx).Error().Err(err).Msg("failed to marshal request params")
			util.WriteJsonError(w, "parameter error", http.StatusInternalServerError)
			return
		}
		if err := json.Unmarshal(s, handler); err != nil {
			log.For(ctx).Error().Err(err).Msg("failed to unmarshal request params")
			util.WriteJsonError(w, "parameter error", http.StatusInternalServerError)
			return
		}

		// Make the request
		response, err := makeRequest(ctx, graphqlHandler, handler, format, r.URL)
		if err != nil {
			util.WriteJsonError(w, err.Error(), http.StatusInternalServerError)
			return
		}

		// Return 404 for single-entity requests that returned empty results
		if notFoundOnEmpty {
			if h, ok := handler.(hasResponseKey); ok {
				if checkEmptyResponse(response, format, h.ResponseKey()) {
					util.WriteJsonError(w, "not found", http.StatusNotFound)
					return
				}
			}
		}

		// Write the output data
		w.Header().Add("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write(response)
	}
}

// makeRequest prepares an apiHandler and makes the request.
func makeRequest(ctx context.Context, graphqlHandler http.Handler, ent apiHandler, format string, u *url.URL) ([]byte, error) {
	cfg := model.ForContext(ctx)
	query, vars := ent.Query(ctx)
	response, err := makeGraphQLRequest(ctx, graphqlHandler, query, vars)
	if err != nil {
		vjson, _ := json.Marshal(vars)
		log.For(ctx).Error().Err(err).Str("query", query).Str("vars", string(vjson)).Msg("graphql request failed")
		return nil, err
	}

	// Add meta
	addMeta := true
	if v, ok := ent.(canIncludeNext); ok {
		addMeta = v.IncludeNext()
	}
	if addMeta {
		if lastId, nextPage, err := getAfterID(ent, response); err != nil {
			log.For(ctx).Error().Err(err).Msg("pagination failed to get max entity id")
		} else if nextPage && lastId > 0 {
			meta := hw{"after": lastId}
			if u != nil {
				newUrl, err := url.Parse(u.String())
				if err != nil {
					panic(err)
				}
				rq := newUrl.Query()
				rq.Set("after", strconv.Itoa(lastId))
				newUrl.RawQuery = rq.Encode()
				meta["next"] = cfg.Link(ctx, cfg.RestPrefix+newUrl.String())
			}
			response["meta"] = meta
		}
	}

	if format == "geojson" || format == "geojsonl" {
		// TODO: Don't process response in-place.
		if v, ok := ent.(canProcessGeoJSON); ok {
			if err := v.ProcessGeoJSON(ctx, response); err != nil {
				return nil, err
			}
		} else {
			if err := processGeoJSON(ctx, ent, response); err != nil {
				return nil, err
			}
		}
		if format == "geojsonl" {
			return renderGeojsonl(response)
		}
	}
	return json.Marshal(response)
}

// makeGraphQLRequest issues the graphql request and unpacks the response.
func makeGraphQLRequest(ctx context.Context, srv http.Handler, query string, vars map[string]interface{}) (map[string]interface{}, error) {
	gqlData := map[string]any{
		"query":     query,
		"variables": vars,
	}
	gqlBody, err := json.Marshal(gqlData)
	if err != nil {
		return nil, err
	}
	gqlRequest, err := http.NewRequestWithContext(ctx, "POST", "/", bytes.NewReader(gqlBody))
	gqlRequest.Header.Set("Content-Type", "application/json")
	if err != nil {
		return nil, err
	}
	wr := httptest.NewRecorder()
	srv.ServeHTTP(wr, gqlRequest)
	response := map[string]any{}
	if err := json.Unmarshal(wr.Body.Bytes(), &response); err != nil {
		return nil, err
	}
	if e, ok := response["errors"].([]interface{}); ok && len(e) > 0 {
		if emsg, ok := e[0].(map[string]interface{}); ok && emsg["message"] != nil {
			return nil, errors.New(emsg["message"].(string))
		}
	}

	data, ok := response["data"].(map[string]interface{})
	if !ok {
		return nil, err
	}
	return data, nil
}

func renderGeojsonl(response map[string]any) ([]byte, error) {
	var ret []byte
	feats, ok := response["features"].([]map[string]any)
	if !ok {
		return nil, errors.New("not features")
	}
	for i, feat := range feats {
		j, err := json.Marshal(feat)
		if err != nil {
			return nil, err
		}
		ret = append(ret, j...)
		if i < len(feats)-1 {
			ret = append(ret, byte('\n'))
		}
	}

	return ret, nil
}

func getAfterID(ent apiHandler, response map[string]interface{}) (int, bool, error) {
	maxid := 0
	fkey := ""

	// Get request limit
	limit := MAXLIMIT
	if v, ok := ent.(canLimit); ok {
		limit = v.CheckLimit()
	}

	// Get response key
	if v, ok := ent.(hasResponseKey); ok {
		fkey = v.ResponseKey()
	} else {
		return 0, false, errors.New("pagination: response key missing")
	}

	// Get entities
	entities, ok := response[fkey].([]interface{})
	if !ok {
		return 0, false, errors.New("pagination: unknown response key value")
	}

	// No next page if there are no entities, or if less entities than the limit
	if len(entities) == 0 {
		return 0, false, nil
	}
	if len(entities) < limit {
		return 0, false, nil
	}

	// Get last entity ID
	lastEnt, ok := entities[len(entities)-1].(map[string]interface{})
	if !ok {
		return 0, false, errors.New("pagination: last entity not map[string]interface{}")
	}
	switch id := lastEnt["id"].(type) {
	case int:
		maxid = id
	case float64:
		maxid = int(id)
	case int64:
		maxid = int(id)
	default:
		return 0, false, errors.New("pagination: last entity id not numeric")
	}
	return maxid, true, nil
}

//

type restBbox struct {
	model.BoundingBox
}

func (bbox *restBbox) UnmarshalText(v []byte) error {
	s := strings.Split(string(v), ",")
	if len(s) != 4 {
		return errors.New("4 values needed")
	}
	if a, err := strconv.ParseFloat(s[0], 64); err != nil {
		return err
	} else {
		bbox.MinLon = a
	}
	if a, err := strconv.ParseFloat(s[1], 64); err != nil {
		return err
	} else {
		bbox.MinLat = a
	}
	if a, err := strconv.ParseFloat(s[2], 64); err != nil {
		return err
	} else {
		bbox.MaxLon = a
	}
	if a, err := strconv.ParseFloat(s[3], 64); err != nil {
		return err
	} else {
		bbox.MaxLat = a
	}
	return nil
}

func (bbox *restBbox) AsJson() map[string]any {
	return map[string]any{
		"min_lon": bbox.MinLon,
		"min_lat": bbox.MinLat,
		"max_lon": bbox.MaxLon,
		"max_lat": bbox.MaxLat,
	}
}

func toPtr[T any, P *T](v T) P {
	vcopy := v
	return &vcopy
}
