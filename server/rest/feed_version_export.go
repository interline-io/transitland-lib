package rest

// Feed Version Export Handler
//
// This handler provides synchronous export of feed versions with optional transformations.
// The export is streamed directly to the client as a ZIP file.

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"

	oa "github.com/getkin/kin-openapi/openapi3"
	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/internal/util"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/tlcsv"
	"github.com/tidwall/gjson"
)

// FeedVersionExportRequest defines the export request structure
type FeedVersionExportRequest struct {
	// Feed version IDs or SHA1 hashes to export
	FeedVersionKeys []string `json:"feed_version_keys"`
	FeedVersionIDs  []int    `json:"-"`
	// Output format (default: gtfs_zip)
	Format string `json:"format,omitempty"`
	// Transformation options
	Transforms *ExportTransforms `json:"transforms,omitempty"`
}

// ExportTransforms defines available transformations
type ExportTransforms struct {
	// ID prefix for namespacing
	Prefix string `json:"prefix,omitempty"`
	// Files to apply prefix to (default: all applicable files)
	PrefixFiles []string `json:"prefix_files,omitempty"`
	// Normalize timezones (e.g., US/Pacific -> America/Los_Angeles)
	NormalizeTimezones bool `json:"normalize_timezones,omitempty"`
	// Simplify shapes with this tolerance value
	SimplifyShapes *float64 `json:"simplify_shapes,omitempty"`
	// Use basic route types (convert extended to primitive)
	UseBasicRouteTypes bool `json:"use_basic_route_types,omitempty"`
	// Entity value overrides (filename.entity_id.field = value)
	SetValues map[string]string `json:"set_values,omitempty"`
}

// FeedVersionExportOpenAPIRequest defines OpenAPI schema for export endpoint
type FeedVersionExportOpenAPIRequest struct{}

func (r FeedVersionExportOpenAPIRequest) RequestInfo() RequestInfo {
	return RequestInfo{
		Path:        "/feed_versions/export",
		Description: `Export one or more feed versions as a GTFS zip file with optional transformations (ID prefixing, timezone normalization, shape simplification, etc.). Feed versions must be successfully imported before they can be exported. Available only using Transitland professional or enterprise plan API keys.`,
		Post: &RequestOperation{
			Operation: &oa.Operation{
				Summary: "Export feed versions with transformations",
				Extensions: map[string]any{
					"x-required-role": "tl_export_feed_versions",
				},
				RequestBody: &oa.RequestBodyRef{
					Value: &oa.RequestBody{
						Description: "Export request with feed version keys and optional transformations",
						Required:    true,
						Content: oa.Content{
							"application/json": &oa.MediaType{
								Schema: &oa.SchemaRef{
									Value: &oa.Schema{
										Type: &oa.Types{"object"},
										Properties: oa.Schemas{
											"feed_version_keys": &oa.SchemaRef{
												Value: &oa.Schema{
													Type:        &oa.Types{"array"},
													Description: "Array of feed version IDs or SHA1 hashes to export",
													Items: &oa.SchemaRef{
														Value: &oa.Schema{
															Type: &oa.Types{"string"},
														},
													},
												},
											},
											"format": &oa.SchemaRef{
												Value: &oa.Schema{
													Type:        &oa.Types{"string"},
													Description: "Output format (default: gtfs_zip)",
													Default:     "gtfs_zip",
													Enum:        []any{"gtfs_zip"},
												},
											},
											"transforms": &oa.SchemaRef{
												Value: &oa.Schema{
													Type:        &oa.Types{"object"},
													Description: "Optional transformations to apply",
													Properties: oa.Schemas{
														"prefix": &oa.SchemaRef{
															Value: &oa.Schema{
																Type:        &oa.Types{"string"},
																Description: "Prefix to add to entity IDs for namespacing (e.g., 'bart_' prefixes all IDs)",
															},
														},
														"prefix_files": &oa.SchemaRef{
															Value: &oa.Schema{
																Type:        &oa.Types{"array"},
																Description: "Specific files to apply prefix to (e.g., ['routes.txt', 'trips.txt']). If omitted, prefix is applied to all entity types. Useful when merging feeds with shared stops or zones.",
																Items: &oa.SchemaRef{
																	Value: &oa.Schema{
																		Type: &oa.Types{"string"},
																		Enum: []any{"agency.txt", "routes.txt", "trips.txt", "stops.txt", "shapes.txt", "calendar.txt", "fare_attributes.txt", "fare_rules.txt", "levels.txt", "pathways.txt"},
																	},
																},
															},
														},
														"normalize_timezones": &oa.SchemaRef{
															Value: &oa.Schema{
																Type:        &oa.Types{"boolean"},
																Description: "Normalize timezone names (e.g., US/Pacific -> America/Los_Angeles)",
															},
														},
														"simplify_shapes": &oa.SchemaRef{
															Value: &oa.Schema{
																Type:        &oa.Types{"number"},
																Description: "Tolerance value for shape simplification (in meters)",
															},
														},
														"use_basic_route_types": &oa.SchemaRef{
															Value: &oa.Schema{
																Type:        &oa.Types{"boolean"},
																Description: "Convert extended route types to basic GTFS route types",
															},
														},
														"set_values": &oa.SchemaRef{
															Value: &oa.Schema{
																Type:        &oa.Types{"object"},
																Description: "Override specific entity field values. Format: 'filename.entity_id.field' = 'value' (e.g., 'agency.txt.BART.agency_url' = 'https://new-url.com'). Use '*' as entity_id to apply to all entities in a file.",
																AdditionalProperties: oa.AdditionalProperties{
																	Schema: &oa.SchemaRef{
																		Value: &oa.Schema{
																			Type: &oa.Types{"string"},
																		},
																	},
																},
															},
														},
													},
												},
											},
										},
										Required: []string{"feed_version_keys"},
									},
								},
								Example: map[string]any{
									"feed_version_keys": []string{
										"dd7aca4a8e4c90908fd3603c097fabee75fea907",
										"d2813c293bcfd7a97dde599527ae6c62c98e66c6",
									},
									"format": "gtfs_zip",
									"transforms": map[string]any{
										"prefix":                "agency1_",
										"prefix_files":          []string{"routes.txt", "trips.txt"},
										"normalize_timezones":   true,
										"simplify_shapes":       10.0,
										"use_basic_route_types": true,
										"set_values": map[string]string{
											"agency.txt.*.agency_url": "https://example.com",
										},
									},
								},
							},
						},
					},
				},
				Responses: oa.NewResponses(
					oa.WithStatus(200, &oa.ResponseRef{
						Value: &oa.Response{
							Description: toPtr("Successful export - returns GTFS zip file"),
							Content: oa.Content{
								"application/zip": &oa.MediaType{
									Schema: &oa.SchemaRef{
										Value: &oa.Schema{
											Type:   &oa.Types{"string"},
											Format: "binary",
										},
									},
								},
							},
						},
					}),
					oa.WithStatus(400, &oa.ResponseRef{
						Value: &oa.Response{
							Description: toPtr("Bad request - invalid parameters or feed version not imported"),
						},
					}),
					oa.WithStatus(403, &oa.ResponseRef{
						Value: &oa.Response{
							Description: toPtr("Forbidden - feed version does not allow redistribution"),
						},
					}),
					oa.WithStatus(404, &oa.ResponseRef{
						Value: &oa.Response{
							Description: toPtr("Not found - no feed versions found"),
						},
					}),
				),
			},
		},
	}
}

// feedVersionExportHandler handles the export endpoint
func feedVersionExportHandler(graphqlHandler http.Handler, w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	// Parse request body
	var req *FeedVersionExportRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		util.WriteJsonError(w, "invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Validate request
	var err error
	req, err = CheckFeedVersionExportRequest(ctx, req, graphqlHandler)
	if err != nil {
		// Check if it's an HTTP status error
		if httpErr, ok := err.(util.HTTPStatusError); ok {
			util.WriteStatusError(w, httpErr)
		} else {
			// Fallback for unexpected error types
			util.WriteJsonError(w, "invalid feed version export request", http.StatusBadRequest)
		}
		return
	}

	// Create CSV writer that writes to ZIP temporary file
	tmpFilename := ""
	if tmpfile, err := os.CreateTemp("", "*-export.zip"); err != nil {
		log.For(ctx).Error().Err(err).Msg("failed to create temp file for zip")
		util.WriteJsonError(w, "failed to create temp file for zip", http.StatusInternalServerError)
		return
	} else {
		defer os.Remove(tmpFilename)
		tmpFilename = tmpfile.Name()
		tmpfile.Close()
	}

	// Create CSV writer for ZIP output
	csvWriter, err := tlcsv.NewWriter(tmpFilename)
	if err != nil {
		log.For(ctx).Error().Err(err).Msg("failed to create CSV writer")
		util.WriteJsonError(w, "failed to create CSV writer", http.StatusInternalServerError)
		return
	}

	// DO THE THING
	cfg := model.ForContext(ctx)
	exporter := NewFeedVersionExporter(&cfg)
	cpResult, err := exporter.Export(ctx, req.FeedVersionIDs, req.Transforms, csvWriter)
	if err != nil {
		util.WriteJsonError(w, "export operation failed", http.StatusInternalServerError)
		return
	}
	_ = cpResult

	if err := csvWriter.Close(); err != nil {
		log.For(ctx).Error().Err(err).Msg("failed to close CSV writer")
		util.WriteJsonError(w, "failed to close CSV writer", http.StatusInternalServerError)
		return
	}

	// Set response headers for ZIP download
	filename := "export.zip"
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	w.WriteHeader(http.StatusOK)

	// Stream the ZIP file to response
	log.For(ctx).Trace().Msgf("Sending file %s", tmpFilename)
	zipData, err := os.Open(tmpFilename)
	if err != nil {
		log.For(ctx).Error().Err(err).Msg("failed to open zip file for streaming")
		util.WriteJsonError(w, "failed to open zip file for streaming", http.StatusInternalServerError)
		return
	}
	defer zipData.Close()

	if _, err := io.Copy(w, zipData); err != nil {
		// Can't write error to response as headers are already sent
		log.For(ctx).Error().Err(err).Msg("failed to stream zip file")
		return
	}

	// Log export metrics
	// if apiMeter := meters.ForContext(ctx); apiMeter != nil {
	// 	for _, fvid := range req.Feed {
	// 		apiMeter.Meter(ctx, meters.MeterEvent{
	// 			Name:  "feed-version-exports",
	// 			Value: 1.0,
	// 			Dimensions: []meters.Dimension{
	// 				{Key: "fvid", Value: fmt.Sprintf("%d", fvid)},
	// 				{Key: "format", Value: req.Format},
	// 				{Key: "entity_count", Value: fmt.Sprintf("%d", sumEntityCounts(result))},
	// 			},
	// 		})
	// 	}
	// }

	log.For(ctx).Info().
		Int("feed_versions", len(req.FeedVersionIDs)).
		Str("format", req.Format).
		Msg("export completed successfully")
}

func CheckFeedVersionExportRequest(ctx context.Context, req *FeedVersionExportRequest, graphqlHandler http.Handler) (*FeedVersionExportRequest, error) {
	// Basic request validation
	if err := validateExportRequest(req); err != nil {
		return nil, err
	}

	// Resolve feed version keys (IDs and SHA1s) to IDs
	allIds, err := resolveFeedVersionKeys(ctx, req.FeedVersionKeys, graphqlHandler)
	if err != nil {
		return nil, err
	}

	// Validate feed versions (import status, permissions, etc.)
	fvids, err := validateFeedVersionsForExport(ctx, allIds, graphqlHandler)
	if err != nil {
		return nil, err
	}

	return &FeedVersionExportRequest{
		FeedVersionIDs: fvids,
		Transforms:     req.Transforms,
		Format:         req.Format,
	}, nil
}

// validateExportRequest performs basic request validation
func validateExportRequest(req *FeedVersionExportRequest) error {
	if len(req.FeedVersionKeys) == 0 {
		return util.NewBadRequestError("feed_version_keys is required and must not be empty", nil)
	}

	// Default format
	if req.Format == "" {
		req.Format = "gtfs_zip"
	}

	// Only support gtfs_zip for now
	if req.Format != "gtfs_zip" {
		return util.NewBadRequestError("only 'gtfs_zip' format is currently supported", nil)
	}

	return nil
}

// resolveFeedVersionKeys converts feed version keys (IDs or SHA1s) to integer IDs
func resolveFeedVersionKeys(ctx context.Context, keys []string, graphqlHandler http.Handler) ([]int, error) {
	const feedVersionBySha1Query = `
	query($sha1: String!) {
		feed_versions(where: {sha1: $sha1}) {
			id
			sha1
			feed {
				onestop_id
				license {
					redistribution_allowed
				}
			}
			feed_version_gtfs_import {
				success
				in_progress
			}
		}
	}`
	// Separate IDs and SHA1s
	var allIds []int
	var sha1s []string
	for _, key := range keys {
		if id, err := strconv.Atoi(key); err == nil {
			allIds = append(allIds, id)
		} else {
			sha1s = append(sha1s, key)
		}
	}

	// Resolve SHA1s to IDs (GraphQL where.sha1 only supports single value, not array)
	for _, sha1 := range sha1s {
		sha1Response, err := makeGraphQLRequest(ctx, graphqlHandler, feedVersionBySha1Query, hw{"sha1": sha1})
		if err != nil {
			return nil, util.NewInternalServerError("failed to query feed version", err)
		}

		sha1Json, _ := json.Marshal(sha1Response)
		fvs := gjson.Get(string(sha1Json), "feed_versions")
		if !fvs.Exists() || len(fvs.Array()) == 0 {
			return nil, util.NewNotFoundError(fmt.Sprintf("feed version not found: %s", sha1), nil)
		}

		// Add the ID from this SHA1 lookup
		fvid := int(fvs.Array()[0].Get("id").Int())
		allIds = append(allIds, fvid)
	}

	return allIds, nil
}

// validateFeedVersionsForExport checks import status and redistribution permissions
func validateFeedVersionsForExport(ctx context.Context, ids []int, graphqlHandler http.Handler) ([]int, error) {
	// Query all feed versions by IDs
	const feedVersionExportQuery = `
	query($ids: [Int!]) {
		feed_versions(ids: $ids) {
			id
			sha1
			feed {
				onestop_id
				license {
					redistribution_allowed
				}
			}
			feed_version_gtfs_import {
				success
				in_progress
			}
		}
	}`
	fvResponse, err := makeGraphQLRequest(ctx, graphqlHandler, feedVersionExportQuery, hw{"ids": ids})
	if err != nil {
		return nil, util.NewInternalServerError("failed to query feed versions", err)
	}

	// Parse response
	responseJson, err := json.Marshal(fvResponse)
	if err != nil {
		return nil, util.NewInternalServerError("failed to marshal feed version response", err)
	}

	feedVersionsJson := gjson.Get(string(responseJson), "feed_versions")
	if !feedVersionsJson.Exists() || len(feedVersionsJson.Array()) == 0 {
		return nil, util.NewNotFoundError("no feed versions found for the provided keys", nil)
	}

	// Check import status, redistribution permissions, and collect feed version info
	// TODO: Consider adding entity-level OpenFGA permission checks here
	// Currently uses role-based access (tl_export_feed_versions) + redistribution license check
	// Future enhancement: Add can_export action to OpenFGA model and check:
	//   - Add can_export = 12 to Action enum in server/auth/authz/azpb.proto
	//   - Add "define can_export: editor" (or viewer/manager) to feed_version in testdata/server/authz/tls.model
	//   - Check permission: checker.checkActionOrError(ctx, CanExport, newEntityID(FeedVersionType, fvid), ctxTk)
	//   This would enable per-feed-version export control in multi-tenant deployments
	var validFvids []int
	for _, fv := range feedVersionsJson.Array() {
		fvid := int(fv.Get("id").Int())
		sha1 := fv.Get("sha1").String()

		// Check import status - export requires a successful, completed import
		importRecord := fv.Get("feed_version_gtfs_import")
		if !importRecord.Exists() {
			return nil, util.NewBadRequestError(fmt.Sprintf("feed version %s has not been imported (export requires a successful import)", sha1), nil)
		}

		importSuccess := importRecord.Get("success").Bool()
		importInProgress := importRecord.Get("in_progress").Bool()

		if !importSuccess || importInProgress {
			var status string
			if importInProgress {
				status = "import in progress"
			} else if !importSuccess {
				status = "import failed"
			}
			return nil, util.NewBadRequestError(fmt.Sprintf("feed version %s cannot be exported: %s (export requires a successful import)", sha1, status), nil)
		}

		if fv.Get("feed.license.redistribution_allowed").String() == "no" {
			return nil, util.NewForbiddenError(fmt.Sprintf("feed version %s does not allow redistribution", sha1), nil)
		}

		validFvids = append(validFvids, fvid)
	}

	if len(validFvids) == 0 {
		return nil, util.NewNotFoundError("no valid feed versions found", nil)
	}

	return validFvids, nil
}
