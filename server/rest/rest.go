package rest

import (
	"bytes"
	"crypto/sha1"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strconv"

	"github.com/99designs/gqlgen/client"
	"github.com/gorilla/mux"
	"github.com/interline-io/transitland-lib/server/config"
)

// DEFAULTLIMIT is the default API limit
const DEFAULTLIMIT = 20

// MAXLIMIT is the API limit maximum
const MAXLIMIT = 1000

// MAXRADIUS is the maximum point search radius
const MAXRADIUS = 100 * 1000.0

// restConfig holds the base config and the graphql handler
type restConfig struct {
	config.Config
	srv http.Handler
}

// NewServer .
func NewServer(cfg config.Config, srv http.Handler) (http.Handler, error) {
	restcfg := restConfig{Config: cfg, srv: srv}
	r := mux.NewRouter()

	feedHandler := makeHandler(restcfg, func() apiHandler { return &FeedRequest{} })
	fvHandler := makeHandler(restcfg, func() apiHandler { return &FeedVersionRequest{} })
	agencyHandler := makeHandler(restcfg, func() apiHandler { return &AgencyRequest{} })
	routeHandler := makeHandler(restcfg, func() apiHandler { return &RouteRequest{} })
	tripHandler := makeHandler(restcfg, func() apiHandler { return &TripRequest{} })
	stopHandler := makeHandler(restcfg, func() apiHandler { return &StopRequest{} })
	operatorHandler := makeHandler(restcfg, func() apiHandler { return &OperatorRequest{} })

	r.HandleFunc("/feeds.{format}", feedHandler)
	r.HandleFunc("/feeds", feedHandler)
	r.HandleFunc("/feeds/{feed_key}.{format}", feedHandler)
	r.HandleFunc("/feeds/{feed_key}", feedHandler)
	r.HandleFunc("/feeds/{feed_key}/download_latest_feed_version", makeHandlerFunc(restcfg, feedDownloadLatestFeedVersionHandler))

	r.HandleFunc("/feed_versions.{format}", fvHandler)
	r.HandleFunc("/feed_versions", fvHandler)
	r.HandleFunc("/feed_versions/{feed_version_key}.{format}", fvHandler)
	r.HandleFunc("/feed_versions/{feed_version_key}", fvHandler)
	r.HandleFunc("/feeds/{feed_key}/feed_versions", fvHandler)
	r.HandleFunc("/feed_versions/{feed_version_key}/download", makeHandlerFunc(restcfg, fvDownloadHandler))

	r.HandleFunc("/agencies.{format}", agencyHandler)
	r.HandleFunc("/agencies", agencyHandler)
	r.HandleFunc("/agencies/{agency_key}.{format}", agencyHandler)
	r.HandleFunc("/agencies/{agency_key}", agencyHandler)

	r.HandleFunc("/routes.{format}", routeHandler)
	r.HandleFunc("/routes", routeHandler)
	r.HandleFunc("/routes/{route_key}.{format}", routeHandler)
	r.HandleFunc("/routes/{route_key}", routeHandler)
	r.HandleFunc("/agencies/{agency_key}/routes.{format}", routeHandler)
	r.HandleFunc("/agencies/{agency_key}/routes", routeHandler)

	r.HandleFunc("/routes/{route_key}/trips.{format}", tripHandler)
	r.HandleFunc("/routes/{route_key}/trips", tripHandler)
	r.HandleFunc("/routes/{route_key}/trips/{id}", tripHandler)
	r.HandleFunc("/routes/{route_key}/trips/{id}.{format}", tripHandler)

	r.HandleFunc("/stops.{format}", stopHandler)
	r.HandleFunc("/stops", stopHandler)
	r.HandleFunc("/stops/{stop_key}.{format}", stopHandler)
	r.HandleFunc("/stops/{stop_key}", stopHandler)

	r.HandleFunc("/operators.{format}", operatorHandler)
	r.HandleFunc("/operators", operatorHandler)
	r.HandleFunc("/operators/{key}.{format}", operatorHandler)
	r.HandleFunc("/operators/{key}", operatorHandler)
	// r.HandleFunc("/stops/{stop_id}/departures", stopTimeHandler)

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

// makeHandler wraps an apiHandler into an HandlerFunc and performs common checks.
func makeHandler(cfg restConfig, f func() apiHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ent := f()
		opts := queryToMap(r.URL.Query())
		for k, v := range mux.Vars(r) {
			opts[k] = v
		}
		format := opts["format"]
		if format == "png" && cfg.DisableImage {
			http.Error(w, "image generation disabled", http.StatusInternalServerError)
			return
		}

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
		if err != nil {
			fmt.Println("err:", err)
			http.Error(w, "parameter error", http.StatusInternalServerError)
			return
		}
		if err := json.Unmarshal(s, ent); err != nil {
			fmt.Println("err:", err)
			http.Error(w, "parameter error", http.StatusInternalServerError)
			return
		}

		// Make the request
		response, err := makeRequest(cfg, ent, format, r.URL)
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

// makeGraphQLRequest issues the graphql request and unpacks the response.
func makeGraphQLRequest(srv http.Handler, q string, vars map[string]interface{}) (map[string]interface{}, error) {
	d := hw{}
	c2 := client.New(srv)
	opts := []client.Option{}
	for k, v := range vars {
		opts = append(opts, client.Var(k, v))
	}
	err := c2.Post(q, &d, opts...)
	return d, err
}

// makeRequest prepares an apiHandler and makes the request.
func makeRequest(cfg restConfig, ent apiHandler, format string, u *url.URL) ([]byte, error) {
	query, vars := ent.Query()
	// fmt.Printf("debug query: %s\n vars:\n %s\n", query, vars)
	response, err := makeGraphQLRequest(cfg.srv, query, vars)
	x, _ := json.Marshal(vars)
	if err != nil {
		fmt.Printf("debug query: %s\n vars:\n %s\nresponse:\n%s\n", query, x, response)
		return nil, errors.New("request error")
	}

	// get highest ID
	if maxid, err := getMaxID(ent, response); err != nil {
		fmt.Println("getmaxid err:", err)
	} else if maxid > 0 && u != nil {
		rq := u.Query()
		rq.Set("after", strconv.Itoa(maxid))
		u.RawQuery = rq.Encode()
		nextUrl := cfg.RestPrefix + u.String()
		response["meta"] = hw{"after": maxid, "next": nextUrl}
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

func makeHandlerFunc(cfg restConfig, f func(restConfig, http.ResponseWriter, *http.Request)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		f(cfg, w, r)
	}
}
