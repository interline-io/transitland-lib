package rest

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"sync"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/server/model"
)

// openAPICacheControl is sent with every schema response. The document changes
// only when the binary does, so a few minutes of client-side caching is safe
// and keeps repeat fetches of a large document cheap.
const openAPICacheControl = "public, max-age=300"

// openAPIDocument is a generated schema, marshaled once and reused.
type openAPIDocument struct {
	body []byte
	etag string
}

// NewOpenAPIHandler returns a handler that serves the REST OpenAPI document.
//
// The document's servers[0].url is built with mountPrefix, so it includes the
// mount segment and a client resolving a documented path against it reaches the
// same URL the root redirect and pagination links produce.
//
// Callers outside this package need this constructor rather than the route
// NewServer registers: mounting the REST server elsewhere just to reach its
// schema would hand chi a path it does not own.
func NewOpenAPIHandler(opts ...SchemaOption) http.Handler {
	docs := &openAPICache{opts: opts, docs: map[string]*openAPIDocument{}}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg := model.ForContext(r.Context())
		doc, err := docs.get(mountPrefix(cfg.RestPrefix, r.URL.Path, "/openapi.json"))
		if err != nil {
			log.For(r.Context()).Error().Err(err).Msg("rest: failed to generate openapi schema")
			http.Error(w, "Failed to generate schema", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Cache-Control", openAPICacheControl)
		w.Header().Set("ETag", doc.etag)
		if etagMatch(r.Header.Get("If-None-Match"), doc.etag) {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Write(doc.body)
	})
}

// etagMatch reports whether an If-None-Match field value matches etag.
//
// RFC 9110 section 13.1.2 specifies the weak comparison function for
// If-None-Match, so W/"x" matches "x", and the field may be "*" or a
// comma-separated list. Comparing the raw header against our tag missed all
// three, which costs a full re-download of a large document whenever an
// intermediary weakens the validator — something CDNs do routinely when they
// re-encode a response.
//
// Splitting on commas is not strictly correct, since an entity-tag may itself
// contain a comma, but our tags are hex and cannot, so no real tag can be split
// into something that matches ours.
func etagMatch(ifNoneMatch string, etag string) bool {
	ifNoneMatch = strings.TrimSpace(ifNoneMatch)
	if ifNoneMatch == "" {
		return false
	}
	if ifNoneMatch == "*" {
		return true
	}
	for _, candidate := range strings.Split(ifNoneMatch, ",") {
		if trimWeakETag(candidate) == trimWeakETag(etag) {
			return true
		}
	}
	return false
}

// trimWeakETag drops surrounding space and the weak-validator prefix, leaving
// the opaque tag with its quotes.
func trimWeakETag(s string) string {
	return strings.TrimPrefix(strings.TrimSpace(s), "W/")
}

// openAPICache memoizes generated documents by server URL.
//
// The map is unbounded, which is safe because the number of distinct keys is
// fixed by how the handler is registered, not by request input: a key derives
// from the mount path, and chi only routes the exact registered path (verified:
// /rest//openapi.json, /rest/./openapi.json, /rest/%6fpenapi.json and friends
// all 404). A caller registering this handler under a wildcard route would
// break that assumption.
type openAPICache struct {
	opts []SchemaOption
	mu   sync.Mutex
	docs map[string]*openAPIDocument
}

// get returns the document for serverURL, generating it on first use.
//
// Generating rebuilds the gqlgen executable schema and walks every REST query's
// response tree, which costs on the order of 100ms and produces hundreds of
// kilobytes of JSON. The lock is deliberately held across generation so that
// cost is paid once per server URL rather than once per request; a burst of
// concurrent cold requests waits rather than duplicating the work.
//
// This serializes lookups for every key, not just the one being generated. That
// is fine at the one or two keys a deployment actually has, and only worth
// revisiting if a caller registers many.
func (c *openAPICache) get(serverURL string) (*openAPIDocument, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if doc, ok := c.docs[serverURL]; ok {
		return doc, nil
	}
	schema, err := GenerateOpenAPI(serverURL, c.opts...)
	if err != nil {
		return nil, err
	}
	// Marshal eagerly: GenerateOpenAPI hands back a shared mutable *oa.T, and
	// the bytes are what both the response body and the ETag derive from.
	body, err := json.Marshal(schema)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(body)
	doc := &openAPIDocument{body: body, etag: `"` + hex.EncodeToString(sum[:16]) + `"`}
	c.docs[serverURL] = doc
	return doc, nil
}
