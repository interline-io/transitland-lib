package rest

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
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
		if r.Header.Get("If-None-Match") == doc.etag {
			w.WriteHeader(http.StatusNotModified)
			return
		}
		w.Write(doc.body)
	})
}

// openAPICache memoizes generated documents by server URL.
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
