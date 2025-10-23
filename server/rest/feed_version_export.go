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
	"strings"
	"time"

	oa "github.com/getkin/kin-openapi/openapi3"
	"github.com/interline-io/log"
	"github.com/interline-io/transitland-lib/adapters"
	"github.com/interline-io/transitland-lib/adapters/multireader"
	"github.com/interline-io/transitland-lib/copier"
	"github.com/interline-io/transitland-lib/extract"
	"github.com/interline-io/transitland-lib/filters"
	"github.com/interline-io/transitland-lib/internal/util"
	"github.com/interline-io/transitland-lib/server/model"
	"github.com/interline-io/transitland-lib/tlcsv"
	"github.com/interline-io/transitland-lib/tldb"
	"github.com/interline-io/transitland-lib/tldb/postgres"
	"github.com/tidwall/gjson"
)

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

	var err error
	req, err = CheckFeedVersionExportRequest(ctx, req, graphqlHandler)
	if err != nil {
		util.WriteJsonError(w, "invalid request: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Create CSV writer that writes to ZIP temporary file
	tmpFilename := ""
	if tmpfile, err := os.CreateTemp("", "*-export.zip"); err != nil {
		log.For(ctx).Error().Err(err).Msg("failed to create temp file for zip")
		util.WriteJsonError(w, "failed to create temp file for zip: "+err.Error(), http.StatusInternalServerError)
		return
	} else {
		defer os.Remove(tmpFilename)
		tmpFilename = tmpfile.Name()
		tmpfile.Close()
	}

	csvWriter, err := tlcsv.NewWriter(tmpFilename)
	if err != nil {
		log.For(ctx).Error().Err(err).Msg("failed to create CSV writer")
		util.WriteJsonError(w, "failed to create CSV writer: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// DO THE THING
	cpResult, err := ExportFeedVersions(ctx, req, csvWriter)
	if err != nil {
		util.WriteJsonError(w, "export failed: "+err.Error(), http.StatusInternalServerError)
		return
	}
	_ = cpResult

	if err := csvWriter.Close(); err != nil {
		log.For(ctx).Error().Err(err).Msg("failed to close CSV writer")
		util.WriteJsonError(w, "failed to close CSV writer: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Set response headers for ZIP download
	filename := "export.zip" // generateExportFilename(feedOnestopIds, fvSha1s)
	w.Header().Set("Content-Type", "application/zip")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", filename))
	w.WriteHeader(http.StatusOK)

	// Stream the ZIP file to response
	log.For(ctx).Trace().Msgf("Sending file %s", tmpFilename)
	zipData, err := os.Open(tmpFilename)
	if err != nil {
		log.For(ctx).Error().Err(err).Msg("failed to open zip file for streaming")
		util.WriteJsonError(w, "failed to open zip file for streaming: "+err.Error(), http.StatusInternalServerError)
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

// generateExportFilename creates a descriptive filename for the export
func generateExportFilename(feedOnestopIds []string, sha1s []string) string {
	timestamp := time.Now().Format("20060102-150405")

	if len(feedOnestopIds) == 1 && len(sha1s) == 1 {
		return fmt.Sprintf("%s-%s-%s.zip", feedOnestopIds[0], sha1s[0][:8], timestamp)
	} else if len(feedOnestopIds) > 0 {
		return fmt.Sprintf("export-%s-%d-feeds-%s.zip", feedOnestopIds[0], len(feedOnestopIds), timestamp)
	}

	return fmt.Sprintf("export-%s.zip", timestamp)
}

// sumEntityCounts sums all entity counts from the result
func sumEntityCounts(result *copier.Result) int {
	total := 0
	for _, count := range result.EntityCount {
		total += count
	}
	return total
}

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
}
`

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
}
`

func CheckFeedVersionExportRequest(ctx context.Context, req *FeedVersionExportRequest, graphqlHandler http.Handler) (*FeedVersionExportRequest, error) {
	// Validate request
	if len(req.FeedVersionKeys) == 0 {
		return nil, fmt.Errorf("feed_version_keys is required and must not be empty")
	}

	// Default format
	if req.Format == "" {
		req.Format = "gtfs_zip"
	}

	// Only support gtfs_zip for now
	if req.Format != "gtfs_zip" {
		return nil, fmt.Errorf("only 'gtfs_zip' format is currently supported")
	}

	// Separate IDs and SHA1s, then resolve SHA1s to IDs
	var allIds []int
	var sha1s []string
	for _, key := range req.FeedVersionKeys {
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
			return nil, fmt.Errorf("failed to query feed version %s: %s", sha1, err.Error())
		}

		sha1Json, _ := json.Marshal(sha1Response)
		fvs := gjson.Get(string(sha1Json), "feed_versions")
		if !fvs.Exists() || len(fvs.Array()) == 0 {
			return nil, fmt.Errorf("feed version not found: %s", sha1)
		}

		// Add the ID from this SHA1 lookup
		fvid := int(fvs.Array()[0].Get("id").Int())
		allIds = append(allIds, fvid)
	}

	// Query all feed versions by IDs
	fvResponse, err := makeGraphQLRequest(ctx, graphqlHandler, feedVersionExportQuery, hw{"ids": allIds})
	if err != nil {
		return nil, fmt.Errorf("failed to query feed versions: %s", err.Error())
	}

	// Parse response
	responseJson, err := json.Marshal(fvResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal feed version response: %s", err.Error())
	}

	feedVersionsJson := gjson.Get(string(responseJson), "feed_versions")
	if !feedVersionsJson.Exists() || len(feedVersionsJson.Array()) == 0 {
		return nil, fmt.Errorf("no feed versions found for the provided keys")
	}

	// Check import status, redistribution permissions, and collect feed version info
	// TODO: Consider adding entity-level OpenFGA permission checks here
	// Currently uses role-based access (tl_export_feed_versions) + redistribution license check
	// Future enhancement: Add can_export action to OpenFGA model and check:
	//   - Add can_export = 12 to Action enum in server/auth/authz/azpb.proto
	//   - Add "define can_export: editor" (or viewer/manager) to feed_version in testdata/server/authz/tls.model
	//   - Check permission: checker.checkActionOrError(ctx, CanExport, newEntityID(FeedVersionType, fvid), ctxTk)
	//   This would enable per-feed-version export control in multi-tenant deployments
	var fvids []int
	var fvSha1s []string
	var feedOnestopIds []string
	for _, fv := range feedVersionsJson.Array() {
		fvid := int(fv.Get("id").Int())
		sha1 := fv.Get("sha1").String()
		feedOnestopId := fv.Get("feed.onestop_id").String()

		// Check import status - export requires a successful, completed import
		importRecord := fv.Get("feed_version_gtfs_import")
		if !importRecord.Exists() {
			return nil, fmt.Errorf("feed version %s has not been imported (export requires a successful import)", sha1)
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
			return nil, fmt.Errorf("feed version %s cannot be exported: %s (export requires a successful import)", sha1, status)
		}

		if fv.Get("feed.license.redistribution_allowed").String() == "no" {
			return nil, fmt.Errorf("feed version %s does not allow redistribution", sha1)
		}

		fvids = append(fvids, fvid)
		fvSha1s = append(fvSha1s, sha1)
		feedOnestopIds = append(feedOnestopIds, feedOnestopId)
	}

	if len(fvids) == 0 {
		return nil, fmt.Errorf("no valid feed versions found")
	}
	return &FeedVersionExportRequest{
		FeedVersionIDs: fvids,
		Transforms:     req.Transforms,
		Format:         req.Format,
	}, nil
}

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

func ExportFeedVersions(ctx context.Context, req *FeedVersionExportRequest, writer adapters.Writer) (*copier.Result, error) {
	// Create database readers for each feed version
	fvids := req.FeedVersionIDs
	var readers []adapters.Reader
	cfg := model.ForContext(ctx)
	dbx := cfg.Finder.DBX()

	for _, fvid := range req.FeedVersionIDs {
		reader := &tldb.Reader{
			Adapter:        postgres.NewPostgresAdapterFromDBX(dbx),
			PageSize:       1_000,
			FeedVersionIDs: []int{fvid},
		}
		if err := reader.Open(); err != nil {
			log.For(ctx).Error().Err(err).Int("feed_version_id", fvid).Msg("failed to open feed version reader")
			return nil, fmt.Errorf("failed to open feed version reader for %d: %s", fvid, err.Error())
		}
		defer reader.Close()
		readers = append(readers, reader)
	}

	// Use multireader if multiple feed versions, otherwise use single reader
	var reader adapters.Reader
	if len(readers) == 1 {
		reader = readers[0]
	} else {
		reader = multireader.NewReader(readers...)
		if err := reader.Open(); err != nil {
			log.For(ctx).Error().Err(err).Msg("failed to open multireader")
			return nil, fmt.Errorf("failed to open multireader: %s", err.Error())
		}
		defer reader.Close()
	}

	// Configure copier options with transformations
	opts := copier.Options{
		AllowEntityErrors:    true,
		AllowReferenceErrors: false,
		ErrorLimit:           100,
		Quiet:                true,
	}

	// Apply transformations
	if req.Transforms != nil {
		if err := applyTransforms(&opts, req.Transforms, fvids); err != nil {
			log.For(ctx).Error().Err(err).Msg("failed to apply transforms")
			return nil, fmt.Errorf("failed to apply transforms: %s", err.Error())
		}
	}

	// Perform the copy operation (streaming to ZIP)
	result, err := copier.CopyWithOptions(ctx, reader, writer, opts)
	if err != nil {
		// Can't write error to response as headers are already sent
		log.For(ctx).Error().Err(err).Msg("export failed")
		return nil, fmt.Errorf("export failed: %s", err.Error())
	}

	return result, nil
}

// applyTransforms configures copier options based on transform request
func applyTransforms(opts *copier.Options, transforms *ExportTransforms, fvids []int) error {
	// ID prefix/namespacing
	if transforms.Prefix != "" {
		prefixFilter, err := filters.NewPrefixFilter()
		if err != nil {
			return fmt.Errorf("failed to create prefix filter: %w", err)
		}

		// Set prefix for each feed version
		for _, fvid := range fvids {
			prefixFilter.SetPrefix(fvid, transforms.Prefix)
		}

		// Configure which files to prefix
		if len(transforms.PrefixFiles) > 0 {
			for _, file := range transforms.PrefixFiles {
				prefixFilter.PrefixFile(file)
			}
		}

		opts.AddExtension(prefixFilter)
	}

	// Normalize timezones
	if transforms.NormalizeTimezones {
		opts.NormalizeTimezones = true
	}

	// Simplify shapes
	if transforms.SimplifyShapes != nil && *transforms.SimplifyShapes > 0 {
		opts.SimplifyShapes = *transforms.SimplifyShapes
	}

	// Use basic route types
	if transforms.UseBasicRouteTypes {
		opts.UseBasicRouteTypes = true
	}

	// Set specific values
	if len(transforms.SetValues) > 0 {
		setterFilter := extract.NewSetterFilter()
		for key, value := range transforms.SetValues {
			// Parse key format: "filename.entity_id.field"
			parts := strings.SplitN(key, ".", 3)
			if len(parts) != 3 {
				return fmt.Errorf("invalid set_values key format: %s (expected: filename.entity_id.field)", key)
			}
			setterFilter.AddValue(parts[0], parts[1], parts[2], value)
		}
		opts.AddExtension(setterFilter)
	}

	return nil
}
