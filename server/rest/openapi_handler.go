package rest

import (
	"encoding/json"
	"net/http"

	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/server/model"
)

// NewOpenAPIHandler returns a handler serving the REST OpenAPI document, with
// servers[0].url built from mountPrefix so documented paths resolve against it.
// Exported for callers mounting the REST server outside this package.
//
// The document is regenerated per request, which is fine for the example server
// this package wires up. Callers serving it under load should generate once at
// startup with GenerateOpenAPI and serve the bytes themselves.
func NewOpenAPIHandler(opts ...SchemaOption) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cfg := model.ForContext(r.Context())
		schema, err := GenerateOpenAPI(mountPrefix(cfg.RestPrefix, r.URL.Path, "/openapi.json"), opts...)
		if err != nil {
			log.For(r.Context()).Error().Err(err).Msg("rest: failed to generate openapi schema")
			http.Error(w, "Failed to generate schema", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(schema); err != nil {
			log.For(r.Context()).Error().Err(err).Msg("rest: failed to encode openapi schema")
		}
	})
}
