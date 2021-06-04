package rest

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/gorilla/mux"
	"github.com/interline-io/transitland-lib/server/config"
	"github.com/machinebox/graphql"
)

// DEFAULTLIMIT is the default API limit
const DEFAULTLIMIT = 20

// MAXLIMIT is the API limit maximum
const MAXLIMIT = 1000

// MAXRADIUS is the maximum point search radius
const MAXRADIUS = 100 * 1000.0

// MakeHandlers .
func MakeHandlers(cfg config.Config) *mux.Router {
	r := mux.NewRouter()

	feedHandler := makeHandler(cfg, func() apiHandler { return &FeedRequest{} })
	fvHandler := makeHandler(cfg, func() apiHandler { return &FeedVersionRequest{} })
	agencyHandler := makeHandler(cfg, func() apiHandler { return &AgencyRequest{} })
	routeHandler := makeHandler(cfg, func() apiHandler { return &RouteRequest{} })
	tripHandler := makeHandler(cfg, func() apiHandler { return &TripRequest{} })
	stopHandler := makeHandler(cfg, func() apiHandler { return &StopRequest{} })
	stopTimeHandler := makeHandler(cfg, func() apiHandler { return &StopTimeRequest{} })

	r.HandleFunc("/feeds.{format}", feedHandler)
	r.HandleFunc("/feeds", feedHandler)
	r.HandleFunc("/feeds/{key}.{format}", feedHandler)
	r.HandleFunc("/feeds/{key}", feedHandler)
	r.HandleFunc("/feeds/{key}/download_latest_feed_version", makeHandlerFunc(cfg, feedDownloadLatestFeedVersionHandler))

	r.HandleFunc("/feed_versions.{format}", fvHandler)
	r.HandleFunc("/feed_versions", fvHandler)
	r.HandleFunc("/feed_versions/{key}.{format}", fvHandler)
	r.HandleFunc("/feed_versions/{key}", fvHandler)
	r.HandleFunc("/feed_versions/{key}/download", makeHandlerFunc(cfg, fvDownloadHandler))

	r.HandleFunc("/agencies.{format}", agencyHandler)
	r.HandleFunc("/agencies", agencyHandler)
	r.HandleFunc("/agencies/{key}.{format}", agencyHandler)
	r.HandleFunc("/agencies/{key}", agencyHandler)

	r.HandleFunc("/agencies/{agency_id}/routes.{format}", routeHandler)
	r.HandleFunc("/agencies/{agency_id}/routes", routeHandler)

	r.HandleFunc("/routes.{format}", routeHandler)
	r.HandleFunc("/routes", routeHandler)
	r.HandleFunc("/routes/{key}.{format}", routeHandler)
	r.HandleFunc("/routes/{key}", routeHandler)

	r.HandleFunc("/routes/{route_id}/trips.{format}", tripHandler)
	r.HandleFunc("/routes/{route_id}/trips", tripHandler)

	r.HandleFunc("/routes/{route_id}/trips/{id}", tripHandler)
	r.HandleFunc("/routes/{route_id}/trips/{id}.{format}", tripHandler)

	r.HandleFunc("/stops.{format}", stopHandler)
	r.HandleFunc("/stops", stopHandler)
	r.HandleFunc("/stops/{key}.{format}", stopHandler)
	r.HandleFunc("/stops/{key}", stopHandler)

	r.HandleFunc("/stops/{stop_id}/departures", stopTimeHandler)
	return r
}

func getKey(value string) string {
	h := sha1.New()
	h.Write([]byte(value))
	bs := h.Sum(nil)
	return fmt.Sprintf("%x", bs)
}

// A type that can generate a GraphQL query and variables.
type apiHandler interface {
	Query() (string, map[string]interface{})
}

// A type that can generate a GeoJSON response.
type canProcessGeoJSON interface {
	ProcessGeoJSON(map[string]interface{}) error
}

// A type that specifies a JSON response key.
type hasResponseKey interface {
	ResponseKey() string
}

// Alias for map string interface
type hw = map[string]interface{}

// checkIds returns a id as a []int{id} slice if >0, otherwise nil.
func checkIds(id int) []int {
	if id > 0 {
		return []int{id}
	}
	return nil
}

// checkAfter checks the value is positive.
func checkAfter(after int) int {
	if after < 0 {
		return 0
	}
	return after
}

// checkLimit checks the limit is positive and below the maximum limit.
func checkLimit(limit int) int {
	if limit <= 0 {
		return DEFAULTLIMIT
	}
	if limit > MAXLIMIT {
		return MAXLIMIT
	}
	return limit
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

// makeGraphQLRequest issues the graphql request and unpacks the response.
func makeGraphQLRequest(endpoint string, q string, vars map[string]interface{}) (map[string]interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 100000*time.Millisecond)
	defer cancel()
	req := graphql.NewRequest(q)
	for k, v := range vars {
		req.Var(k, v)
	}
	client := graphql.NewClient(endpoint)
	d := hw{}
	if err := client.Run(ctx, req, &d); err != nil {
		return d, err
	}
	return d, nil
}

// makeRequest prepares an apiHandler and makes the request.
func makeRequest(endpoint string, ent apiHandler, format string) ([]byte, error) {
	query, vars := ent.Query()
	// fmt.Printf("debug query: %s\n vars:\n %s\n", query, vars)
	response, err := makeGraphQLRequest(endpoint, query, vars)
	x, _ := json.Marshal(vars)
	if err != nil {
		fmt.Printf("debug query: %s\n vars:\n %s\nresponse:\n%s\n", query, x, response)
		return nil, errors.New("request error")
	}
	if format == "geojson" || format == "png" {
		// TODO: Don't process response in-place.
		if v, ok := ent.(canProcessGeoJSON); ok {
			if err := v.ProcessGeoJSON(response); err != nil {
				return nil, err
			}
		} else {
			if err := processGeoJSON(ent, response); err != nil {
				return nil, err
			}
		}
		if format == "png" {
			b, err := json.Marshal(response)
			if err != nil {
				return nil, err
			}
			return renderMap(b, 800, 800)
		}
	}
	return json.Marshal(response)
}

func makeHandlerFunc(cfg config.Config, f func(config.Config, http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f(cfg, w, r)
	}
}

// makeHandler wraps an apiHandler into an HandlerFunc and performs common checks.
func makeHandler(cfg config.Config, f func() apiHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ent := f()
		opts := queryToMap(r.URL.Query())
		for k, v := range mux.Vars(r) {
			opts[k] = v
		}
		format := opts["format"]

		// If this is a image request, check the local cache
		urlkey := getKey(r.URL.Path + "/" + r.URL.RawQuery)
		if format == "png" && localFileCache != nil {
			if ok, _ := localFileCache.Has(urlkey); ok {
				w.WriteHeader(http.StatusOK)
				err := localFileCache.Get(w, urlkey)
				if err != nil {
					fmt.Println("file cache error:", err)
				}
				return
			}
		}

		// Use json marshal/unmarshal to convert string params to correct types
		s, err := json.Marshal(opts)
		if err := json.Unmarshal(s, ent); err != nil {
			http.Error(w, "parameter error", http.StatusInternalServerError)
			return
		}

		// Make the request
		response, err := makeRequest(cfg.Endpoint, ent, format)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
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
				fmt.Println("file cache error:", err)
			}
		}
	}
}
