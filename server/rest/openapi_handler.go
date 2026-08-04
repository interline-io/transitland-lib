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

// openAPICacheControl is sent with every schema response; the document changes
// only when the binary does.
const openAPICacheControl = "public, max-age=300"

// openAPIDocument is a generated schema, marshaled once and reused.
type openAPIDocument struct {
	body []byte
	etag string
}

// NewOpenAPIHandler returns a handler serving the REST OpenAPI document, with
// servers[0].url built from mountPrefix so documented paths resolve against it.
// Exported for callers mounting the REST server outside this package.
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

// etagMatch reports whether an If-None-Match field value matches etag, using
// the weak comparison RFC 9110 13.1.2 requires: "*", lists, and W/ prefixes all
// match. Splitting on commas is safe only because our tags are hex.
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

// openAPICache memoizes generated documents by server URL. The map is unbounded
// but keys derive from the registered mount path, not request input — which
// holds unless a caller registers this handler under a wildcard route.
type openAPICache struct {
	opts []SchemaOption
	mu   sync.Mutex
	docs map[string]*openAPIDocument
}

// get returns the document for serverURL, generating it on first use.
// Generation costs ~100ms, so the lock is held across it: concurrent cold
// requests wait rather than duplicating the work.
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
	// Marshal eagerly: GenerateOpenAPI hands back a shared mutable *oa.T.
	body, err := json.Marshal(schema)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256(body)
	doc := &openAPIDocument{body: body, etag: `"` + hex.EncodeToString(sum[:16]) + `"`}
	c.docs[serverURL] = doc
	return doc, nil
}
