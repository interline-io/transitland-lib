package rest

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/internal/util"
	"github.com/interline-io/transitland-lib/server/auth/mw/usercheck"
	"github.com/interline-io/transitland-lib/server/meters"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/rs/zerolog"
)

// DEFAULTLIMIT is the default API limit
const DEFAULTLIMIT = 20

// MAXLIMIT is the API limit maximum
var MAXLIMIT = 1_000

// MAXRADIUS is the maximum point search radius
const MAXRADIUS = 100 * 1000.0

// NewServer .
func NewServer(graphqlHandler http.Handler) (http.Handler, error) {
	r := chi.NewRouter()

	feedIndexHandler := makeIndexHandler(graphqlHandler, "feeds", func() apiHandler { return &FeedRequest{} })
	feedEntityHandler := makeEntityHandler(graphqlHandler, "feeds", func() apiHandler { return &FeedRequest{} })
	feedVersionIndexHandler := makeIndexHandler(graphqlHandler, "feedVersions", func() apiHandler { return &FeedVersionRequest{} })
	feedVersionEntityHandler := makeEntityHandler(graphqlHandler, "feedVersions", func() apiHandler { return &FeedVersionRequest{} })
	agencyIndexHandler := makeIndexHandler(graphqlHandler, "agencies", func() apiHandler { return &AgencyRequest{} })
	agencyEntityHandler := makeEntityHandler(graphqlHandler, "agencies", func() apiHandler { return &AgencyRequest{} })
	routeIndexHandler := makeIndexHandler(graphqlHandler, "routes", func() apiHandler { return &RouteRequest{} })
	routeEntityHandler := makeEntityHandler(graphqlHandler, "routes", func() apiHandler { return &RouteRequest{} })
	tripIndexHandler := makeIndexHandler(graphqlHandler, "trips", func() apiHandler { return &TripRequest{} })
	tripEntityHandler := makeEntityHandler(graphqlHandler, "trips", func() apiHandler { return &TripRequest{} })
	stopIndexHandler := makeIndexHandler(graphqlHandler, "stops", func() apiHandler { return &StopRequest{} })
	stopEntityHandler := makeEntityHandler(graphqlHandler, "stops", func() apiHandler { return &StopRequest{} })
	stopDepartureHandler := makeIndexHandler(graphqlHandler, "stopDepartures", func() apiHandler { return &StopDepartureRequest{} })
	operatorIndexHandler := makeIndexHandler(graphqlHandler, "operators", func() apiHandler { return &OperatorRequest{} })
	operatorEntityHandler := makeEntityHandler(graphqlHandler, "operators", func() apiHandler { return &OperatorRequest{} })

	// Redirect root to OpenAPI documentation
	r.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// Get the base path from the request URL
		basePath := strings.TrimSuffix(r.URL.Path, "/")
		// When path is "/", basePath will be empty, which is correct for root
		redirectPath := basePath + "/openapi.json"
		http.Redirect(w, r, redirectPath, http.StatusMovedPermanently)
	})

	// OpenAPI Schema endpoint
	r.HandleFunc("/openapi.json", func(w http.ResponseWriter, r *http.Request) {
		cfg := model.ForContext(r.Context())
		schema, err := GenerateOpenAPI(cfg.RestPrefix)
		if err != nil {
			http.Error(w, "Failed to generate schema", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(schema); err != nil {
			http.Error(w, "Failed to encode schema", http.StatusInternalServerError)
			return
		}
	})

	r.HandleFunc("/feeds.{format}", feedIndexHandler)
	r.HandleFunc("/feeds", feedIndexHandler)
	r.HandleFunc("/feeds/{feed_key}.{format}", feedEntityHandler)
	r.HandleFunc("/feeds/{feed_key}", feedEntityHandler)
	r.Handle("/feeds/{feed_key}/download_latest_feed_version", usercheck.RoleRequired("tl_download_fv_current")(makeHandlerFunc(graphqlHandler, "feedVersionDownloadLatest", feedVersionDownloadLatestHandler)))

	r.Handle("/feeds/{feed_key}/download_latest_rt/{rt_type}.{format}", makeHandlerFunc(graphqlHandler, "feedDownloadRtHelper", feedDownloadRtHelper))

	r.HandleFunc("/feed_versions.{format}", feedVersionIndexHandler)
	r.HandleFunc("/feed_versions", feedVersionIndexHandler)
	r.HandleFunc("/feed_versions/{feed_version_key}.{format}", feedVersionEntityHandler)
	r.HandleFunc("/feed_versions/{feed_version_key}", feedVersionEntityHandler)
	r.HandleFunc("/feeds/{feed_key}/feed_versions", feedVersionIndexHandler)
	r.Handle("/feed_versions/{feed_version_key}/download", usercheck.RoleRequired("tl_download_fv_historic")(makeHandlerFunc(graphqlHandler, "feedVersionDownload", feedVersionDownloadHandler)))
	r.Method("POST", "/feed_versions/export", usercheck.RoleRequired("tl_export_feed_versions")(makeHandlerFunc(graphqlHandler, "feedVersionExport", feedVersionExportHandler)))

	r.HandleFunc("/agencies.{format}", agencyIndexHandler)
	r.HandleFunc("/agencies", agencyIndexHandler)
	r.HandleFunc("/agencies/{agency_key}.{format}", agencyEntityHandler)
	r.HandleFunc("/agencies/{agency_key}", agencyEntityHandler)

	r.HandleFunc("/routes.{format}", routeIndexHandler)
	r.HandleFunc("/routes", routeIndexHandler)
	r.HandleFunc("/routes/{route_key}.{format}", routeEntityHandler)
	r.HandleFunc("/routes/{route_key}", routeEntityHandler)
	r.HandleFunc("/agencies/{agency_key}/routes.{format}", routeIndexHandler)
	r.HandleFunc("/agencies/{agency_key}/routes", routeIndexHandler)

	r.HandleFunc("/routes/{route_key}/trips.{format}", tripIndexHandler)
	r.HandleFunc("/routes/{route_key}/trips", tripIndexHandler)
	r.HandleFunc("/routes/{route_key}/trips/{id}", tripEntityHandler)
	r.HandleFunc("/routes/{route_key}/trips/{id}.{format}", tripEntityHandler)

	r.HandleFunc("/stops.{format}", stopIndexHandler)
	r.HandleFunc("/stops", stopIndexHandler)
	r.HandleFunc("/stops/{stop_key}.{format}", stopEntityHandler)
	r.HandleFunc("/stops/{stop_key}", stopEntityHandler)

	r.HandleFunc("/stops/{stop_key}/departures", stopDepartureHandler)

	r.HandleFunc("/operators.{format}", operatorIndexHandler)
	r.HandleFunc("/operators", operatorIndexHandler)
	r.HandleFunc("/operators/{operator_key}.{format}", operatorEntityHandler)
	r.HandleFunc("/operators/{operator_key}", operatorEntityHandler)

	// OnestopID generic handler
	r.Handle("/onestop_id/{onestop_id}", &OnestopIdEntityRedirectRequest{})

	return r, nil
}

func getKey(value string) string {
	h := sha1.New()
	h.Write([]byte(value))
	bs := h.Sum(nil)
	return fmt.Sprintf("%x", bs)
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

// checkEmptyResponse returns true if the response is empty for the given format
func checkEmptyResponse(response []byte, format string, responseKey string) bool {
	// For PNG format, nil response indicates empty results (checked in makeRequest)
	if format == "png" {
		return response == nil
	}

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

func makeHandlerFunc(graphqlHandler http.Handler, handlerName string, f func(http.Handler, http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		if apiMeter := meters.ForContext(ctx); apiMeter != nil {
			apiMeter.ApplyDimension("handler", handlerName)
		}
		f(graphqlHandler, w, r.WithContext(ctx))
	}
}

// makeIndexHandler creates a handler for list/index endpoints that returns empty arrays normally.
func makeIndexHandler(graphqlHandler http.Handler, handlerName string, f func() apiHandler) http.HandlerFunc {
	return makeHandler(graphqlHandler, handlerName, f, false)
}

// makeEntityHandler creates a handler for single-entity endpoints that returns 404 on empty results.
func makeEntityHandler(graphqlHandler http.Handler, handlerName string, f func() apiHandler) http.HandlerFunc {
	return makeHandler(graphqlHandler, handlerName, f, true)
}

// makeHandler wraps an apiHandler into an HandlerFunc and performs common checks.
func makeHandler(graphqlHandler http.Handler, handlerName string, f func() apiHandler, notFoundOnEmpty bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()
		cfg := model.ForContext(ctx)
		handler := f()
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

		// Meters
		if apiMeter := meters.ForContext(ctx); apiMeter != nil {
			apiMeter.ApplyDimension("handler", handlerName)
		}

		// Handle format
		format := opts["format"]
		if format == "png" && cfg.DisableImage {
			util.WriteJsonError(w, "image generation disabled", http.StatusInternalServerError)
			return
		}

		// If this is a image request, check the local cache
		urlkey := getKey(r.URL.Path + "/" + r.URL.RawQuery)
		if format == "png" && localFileCache != nil {
			if ok, _ := localFileCache.Has(urlkey); ok {
				w.WriteHeader(http.StatusOK)
				err := localFileCache.Get(w, urlkey)
				if err != nil {
					log.For(ctx).Error().Err(err).Msg("file cache error")
				}
				return
			}
		}

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
		if format == "png" {
			w.Header().Add("Content-Type", "image/png")
		} else {
			w.Header().Add("Content-Type", "application/json")
		}
		w.WriteHeader(http.StatusOK)
		w.Write(response)

		// Cache image response
		if format == "png" && localFileCache != nil {
			if err := localFileCache.Put(urlkey, bytes.NewReader(response)); err != nil {
				log.For(ctx).Error().Err(err).Msgf("file cache error")
			}
		}
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
				meta["next"] = cfg.RestPrefix + newUrl.String()
			}
			response["meta"] = meta
		}
	}

	if format == "geojson" || format == "geojsonl" || format == "png" {
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
		switch format {
		case "geojsonl":
			return renderGeojsonl(response)
		case "png":
			// Return nil for empty results so caller can return 404
			if features, ok := response["features"].([]map[string]any); ok && len(features) == 0 {
				return nil, nil
			}
			b, err := json.Marshal(response)
			if err != nil {
				return nil, err
			}
			return renderMap(ctx, b, 800, 800)
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
