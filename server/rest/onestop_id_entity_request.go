package rest

import (
	"fmt"
	"net/http"
	"strings"

	oa "github.com/getkin/kin-openapi/openapi3"
	"github.com/go-chi/chi/v5"
	"github.com/interline-io/transitland-lib/server/model"
)

type OnestopIdEntityRedirectRequest struct {
}

func (handler OnestopIdEntityRedirectRequest) RequestInfo() RequestInfo {
	return RequestInfo{
		Path: "/onestop_id/{onestop_id}",
		Get: &RequestOperation{
			Operation: &oa.Operation{
				Summary: "Onestop ID Entity Redirect",

				Parameters: oa.Parameters{
					&pref{Value: &param{
						Name:        "onestop_id",
						In:          "path",
						Required:    true,
						Description: `Onestop ID lookup key`,
						Schema:      newSRVal("string", "", nil),
					}},
				},

				Responses: oa.NewResponses(
					oa.WithStatus(302, &oa.ResponseRef{
						Value: &oa.Response{
							Description: toPtr("Redirect to entity by Onestop ID"),
							Content:     oa.Content{},
						},
					}),
					oa.WithStatus(404, &oa.ResponseRef{
						Value: &oa.Response{
							Description: toPtr("Onestop ID not found or invalid format"),
						},
					}),
				),
			},
		},
	}
}

// Query returns a GraphQL query string and variables.
func (handler *OnestopIdEntityRedirectRequest) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	cfg := model.ForContext(r.Context())
	onestop_id := chi.URLParam(r, "onestop_id")
	// Targets need the mount segment; cfg.RestPrefix alone sends clients to
	// /api/v2/feeds/... instead of /api/v2/rest/feeds/...
	prefix := mountPrefix(cfg.RestPrefix, r.URL.Path, "/onestop_id/")
	var redirectUrl string
	if strings.HasPrefix(onestop_id, "f-") {
		redirectUrl = fmt.Sprintf("%s/feeds/%s", prefix, onestop_id)
		// redirect to feeds/
	} else if strings.HasPrefix(onestop_id, "o-") {
		redirectUrl = fmt.Sprintf("%s/operators/%s", prefix, onestop_id)
	} else if strings.HasPrefix(onestop_id, "s-") {
		redirectUrl = fmt.Sprintf("%s/stops/%s", prefix, onestop_id)
	} else if strings.HasPrefix(onestop_id, "r-") {
		redirectUrl = fmt.Sprintf("%s/routes/%s", prefix, onestop_id)
	}
	if redirectUrl != "" {
		w.Header().Add("Location", cfg.Link(r.Context(), redirectUrl))
		w.WriteHeader(http.StatusFound)
	} else {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}
}
