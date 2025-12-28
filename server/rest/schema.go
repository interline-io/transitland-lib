package rest

import (
	"fmt"
	"regexp"
	"strings"

	oa "github.com/getkin/kin-openapi/openapi3"
	"github.com/interline-io/transitland-lib/internal/generated/gqlout"
	"github.com/interline-io/transitland-lib/server/gql"
	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
)

type RestHandlers interface {
	RequestInfo() RequestInfo
}

// RestHandlersList contains all REST API handlers in logical order
var RestHandlersList = []RestHandlers{
	// Core entity collection endpoints (for searching/filtering)
	&FeedRequest{},          // /feeds
	&FeedVersionRequest{},   // /feed_versions
	&OperatorRequest{},      // /operators
	&AgencyRequest{},        // /agencies
	&RouteRequest{},         // /routes
	&TripRequest{},          // /routes/{route_key}/trips
	&StopRequest{},          // /stops
	&StopDepartureRequest{}, // /stops/{stop_key}/departures

	// Individual resource endpoints (for direct lookups)
	&FeedKeyRequest{},        // /feeds/{feed_key}
	&FeedVersionKeyRequest{}, // /feed_versions/{feed_version_key}
	&OperatorKeyRequest{},    // /operators/{operator_key}
	&AgencyKeyRequest{},      // /agencies/{agency_key}
	&RouteKeyRequest{},       // /routes/{route_key}
	&TripEntityRequest{},     // /routes/{route_key}/trips/{id}
	&StopEntityRequest{},     // /stops/{stop_key}

	// Download/special endpoints
	&FeedDownloadLatestFeedVersionRequest{}, // /feeds/{feed_key}/download_latest_feed_version
	&FeedVersionDownloadRequest{},           // /feed_versions/{feed_version_key}/download
	&FeedVersionExportOpenAPIRequest{},      // /feed_versions/export
	&FeedDownloadRtRequest{},                // /feeds/{feed_key}/download_latest_rt/{rt_type}.{format}
	&OnestopIdEntityRedirectRequest{},       // /onestop_id/{onestop_id} - redirect to entity by Onestop ID
}

// entityKeyParams lists URL parameters that indicate a single-entity lookup request.
// When a request includes one of these parameters and returns no results, the API
// returns 404 Not Found instead of 200 with an empty array.
var entityKeyParams = []string{
	"feed_key",         // /feeds/{feed_key}
	"feed_version_key", // /feed_versions/{feed_version_key}
	"agency_key",       // /agencies/{agency_key}
	"route_key",        // /routes/{route_key}
	"stop_key",         // /stops/{stop_key}
	"operator_key",     // /operators/{operator_key}
}

// hasEntityKey returns true if any single-entity key parameter is present in the request options.
func hasEntityKey(opts map[string]string) bool {
	for _, keyParam := range entityKeyParams {
		if v, exists := opts[keyParam]; exists && v != "" {
			return true
		}
	}
	return false
}

func GenerateOpenAPI(restPrefix string, opts ...SchemaOption) (*oa.T, error) {
	// Apply options
	config := &SchemaConfig{}
	for _, opt := range opts {
		opt(config)
	}

	// Determine server URL based on RestPrefix
	serverURL := ""
	if restPrefix != "" {
		serverURL = restPrefix
	}

	outdoc := &oa.T{
		OpenAPI: "3.0.0",
		Info: &oa.Info{
			Title:       "Transitland REST API",
			Description: "Transitland REST API - Access transit data including feeds, agencies, routes, stops, operators, and real-time departures",
			Version:     "2.0.0",
			Contact: &oa.Contact{
				Email: "info@interline.io",
			},
		},
	}

	// Add server configuration only if URL is provided
	if serverURL != "" {
		outdoc.Servers = []*oa.Server{
			{
				URL:         serverURL,
				Description: "Transitland REST API",
			},
		}
	}

	// Add parameter components
	outdoc.Components = &oa.Components{
		Parameters: oa.ParametersMap{},
	}
	for paramName, paramRef := range ParameterComponents {
		outdoc.Components.Parameters[paramName] = paramRef
	}

	// Apply custom components if provided
	if config.Components != nil {
		if config.Components.SecuritySchemes != nil {
			outdoc.Components.SecuritySchemes = config.Components.SecuritySchemes
		}
		// Could add other component types here (schemas, responses, etc.)
	}

	// Create PathItem for each handler
	var pathOpts []oa.NewPathsOption
	var handlers = RestHandlersList
	for _, handler := range handlers {
		requestInfo := handler.RequestInfo()
		pathItem := &oa.PathItem{}

		// Helper function to process operation (GET or POST)
		processOperation := func(reqOp *RequestOperation) (*oa.Operation, error) {
			if reqOp == nil {
				return nil, nil
			}
			if reqOp.Operation == nil {
				return nil, nil
			}

			op := reqOp.Operation
			op.Description = requestInfo.Description

			// Set responses if not already defined
			if reqOp.Operation.Responses.Len() > 0 {
				op.Responses = reqOp.Operation.Responses
			} else if reqOp.Query != "" {
				oaResponse, err := queryToOAResponses(reqOp.Query)
				if err != nil {
					return nil, err
				}
				op.Responses = oaResponse
			}

			// Apply custom security if provided
			if config.GlobalSecurity != nil {
				op.Security = config.GlobalSecurity
			}

			return op, nil
		}

		// Handle GET operation
		if getOp, err := processOperation(requestInfo.Get); err != nil {
			return outdoc, err
		} else if getOp != nil {
			pathItem.Get = getOp
		}

		// Handle POST operation
		if postOp, err := processOperation(requestInfo.Post); err != nil {
			return outdoc, err
		} else if postOp != nil {
			pathItem.Post = postOp
		}

		pathOpts = append(pathOpts, oa.WithPath(requestInfo.Path, pathItem))
	}
	outdoc.Paths = oa.NewPaths(pathOpts...)
	return outdoc, nil
}

// SchemaConfig holds configuration options for schema generation
type SchemaConfig struct {
	Components     *oa.Components
	GlobalSecurity *oa.SecurityRequirements
}

// SchemaOption is a function that modifies schema configuration
type SchemaOption func(*SchemaConfig)

// WithComponents adds custom components to the schema
func WithComponents(components *oa.Components) SchemaOption {
	return func(config *SchemaConfig) {
		config.Components = components
	}
}

// WithSecurity adds global security requirements to all operations
func WithSecurity(security *oa.SecurityRequirements) SchemaOption {
	return func(config *SchemaConfig) {
		config.GlobalSecurity = security
	}
}

func queryToOAResponses(queryString string) (*oa.Responses, error) {
	// Load schema
	schema := gqlout.NewExecutableSchema(gqlout.Config{Resolvers: &gql.Resolver{}})
	gs := schema.Schema()

	// Prepare document
	query, err := gqlparser.LoadQuery(gs, queryString)
	if err != nil {
		return nil, err
	}

	///////////
	responseObj := oa.SchemaRef{Value: &oa.Schema{
		Title:      "data",
		Properties: oa.Schemas{},
	}}
	for _, op := range query.Operations {
		for selOrder, sel := range op.SelectionSet {
			queryRecurse(gs, sel, responseObj.Value.Properties, 0, selOrder)
		}
	}
	desc := "ok"
	res := oa.WithStatus(200, &oa.ResponseRef{Value: &oa.Response{
		Description: &desc,
		Content:     oa.NewContentWithSchemaRef(&responseObj, []string{"application/json"}),
	}})
	ret := oa.NewResponses(res)

	// Add common error responses
	badRequestDesc := "Bad request - invalid parameters"
	ret.Set("400", &oa.ResponseRef{
		Value: &oa.Response{
			Description: &badRequestDesc,
			Content: oa.NewContentWithJSONSchema(&oa.Schema{
				Type: &oa.Types{"object"},
			}),
		},
	})

	serverErrorDesc := "Internal server error"
	ret.Set("500", &oa.ResponseRef{
		Value: &oa.Response{
			Description: &serverErrorDesc,
			Content: oa.NewContentWithJSONSchema(&oa.Schema{
				Type: &oa.Types{"object"},
			}),
		},
	})

	// Add explicit default response to avoid empty description
	defaultDesc := "Unexpected error"
	ret.Set("default", &oa.ResponseRef{
		Value: &oa.Response{
			Description: &defaultDesc,
			Content: oa.NewContentWithJSONSchema(&oa.Schema{
				Type: &oa.Types{"object"},
			}),
		},
	})
	return ret, nil
}

var gqlScalarToOASchema = map[string]oa.Schema{
	"Time": {
		Type:    oa.NewStringSchema().Type,
		Format:  "datetime",
		Example: "2019-11-15T00:45:55.409906",
	},
	"Int": {
		Type: oa.NewIntegerSchema().Type,
	},
	"Float": {
		Type: oa.NewFloat64Schema().Type,
	},
	"String": {
		Type: oa.NewStringSchema().Type,
	},
	"Boolean": {
		Type: oa.NewBoolSchema().Type,
	},
	"ID": {
		Type: oa.NewInt64Schema().Type,
	},
	"Counts": {
		Type: oa.NewObjectSchema().Type,
	},
	"Tags": {
		Type: oa.NewObjectSchema().Type,
	},
	"Date": {
		Type:    oa.NewStringSchema().Type,
		Format:  "date",
		Example: "2019-11-15",
	},
	"Seconds": {
		Type:    oa.NewStringSchema().Type,
		Format:  "hms",
		Example: "15:21:04",
	},
	"Map": {
		Type: oa.NewObjectSchema().Type,
	},
	"Bool": {
		Type: oa.NewBoolSchema().Type,
	},
	"Strings": {
		Type: oa.NewArraySchema().Type,
	},
	"Color": {
		Type: oa.NewStringSchema().Type,
	},
	"Language": {
		Type: oa.NewStringSchema().Type,
	},
	"Url": {
		Type: oa.NewStringSchema().Type,
	},
	"Email": {
		Type:   oa.NewStringSchema().Type,
		Format: "email",
	},
	"Timezone": {
		Type: oa.NewStringSchema().Type,
	},
	"Any":        {},
	"Upload":     {},
	"Key":        {},
	"Polygon":    {},
	"Geometry":   {},
	"Point":      {},
	"LineString": {},
}

type ParsedUrl struct {
	Text string
	URL  string
}

type ParsedDocstring struct {
	Text         string
	Type         string
	ExternalDocs []ParsedUrl
	Examples     []string
	Enum         []string
	Hide         bool
}

var reLinks = regexp.MustCompile(`(\[(?P<text>.+)\]\((?P<url>.+)\))`)
var reAnno = regexp.MustCompile(`(\[(?P<annotype>.+):(?P<value>.+)\])`)

func ParseDocstring(v string) ParsedDocstring {
	ret := ParsedDocstring{}
	for _, matchGroup := range parseGroups(reLinks, v) {
		text := matchGroup["text"]
		url := matchGroup["url"]
		ret.ExternalDocs = append(ret.ExternalDocs, ParsedUrl{URL: url, Text: text})
	}
	for _, matchGroup := range parseGroups(reAnno, v) {
		annotype := matchGroup["annotype"]
		value := strings.TrimSpace(matchGroup["value"])
		switch annotype {
		case "example":
			ret.Examples = append(ret.Examples, value)
		case "see":
			ret.ExternalDocs = append(ret.ExternalDocs, ParsedUrl{URL: value})
		case "enum":
			for _, e := range strings.Split(value, ",") {
				ret.Enum = append(ret.Enum, strings.TrimSpace(e))
			}
		case "hide":
			ret.Hide = true
		}
	}
	ret.Text = strings.TrimSpace(reAnno.ReplaceAllString(v, ""))
	return ret
}

func queryRecurse(gs *ast.Schema, recurseValue any, parentSchema oa.Schemas, level int, order int) int {
	schema := &oa.Schema{
		Properties: oa.Schemas{},
		Extensions: map[string]any{},
	}
	gqlType := ""
	namedType := ""
	isArray := false
	if field, ok := recurseValue.(*ast.Field); ok {
		if field.Comment != nil {
			for _, c := range field.Comment.List {
				pd := ParseDocstring(c.Value)
				if pd.Hide {
					return order
				}
			}
		}
		schema.Title = field.Name
		schema.Description = field.Definition.Description
		schema.Nullable = !field.Definition.Type.NonNull
		namedType = field.Definition.Type.NamedType
		gqlType = field.Definition.Type.NamedType
		if field.Definition.Type.Elem != nil {
			gqlType = field.Definition.Type.Elem.Name()

		}
		if gst, ok := gs.Types[field.Definition.Type.String()]; ok {
			for _, ev := range gst.EnumValues {
				schema.Enum = append(schema.Enum, ev.Name)
			}
		}
		if strings.HasPrefix(field.Definition.Type.String(), "[") {
			isArray = true
		}
		for _, sel := range field.SelectionSet {
			order = queryRecurse(gs, sel, schema.Properties, level+1, order+1)
		}
	} else if frag, ok := recurseValue.(*ast.FragmentSpread); ok {
		for _, sel := range frag.Definition.SelectionSet {
			// Ugly hack to put fragments at the end of the selection set
			order = queryRecurse(gs, sel, parentSchema, level, order+1)
		}
		return order
	} else {
		return order
	}

	fmt.Printf("%s %s (%s : %s : order %d)\n", strings.Repeat(" ", level*4), schema.Title, namedType, gqlType, order)
	order += 1
	schema.Extensions["x-order"] = order

	// Scalar types
	if scalarType, ok := gqlScalarToOASchema[namedType]; ok {
		schema.Type = scalarType.Type
		schema.Format = scalarType.Format
		schema.Example = scalarType.Example
	} else {
		schema.Type = oa.NewObjectSchema().Type
		if gqlType != "" {
			schema.Extensions["x-graphql-type"] = gqlType
		}
	}

	// Parse docstring
	parsed := ParseDocstring(schema.Description)
	if parsed.Text != "" {
		schema.Description = parsed.Text
	}
	for _, example := range parsed.Examples {
		schema.Example = example
	}
	for _, doc := range parsed.ExternalDocs {
		schema.ExternalDocs = &oa.ExternalDocs{URL: doc.URL, Description: doc.Text}
	}
	for _, e := range parsed.Enum {
		schema.Enum = append(schema.Enum, e)
	}

	if isArray {
		innerSchema := &oa.Schema{
			Properties: schema.Properties,
			Type:       schema.Type,
			Extensions: schema.Extensions,
		}
		outerSchema := &oa.Schema{
			Title:        schema.Title,
			Description:  schema.Description,
			Nullable:     schema.Nullable,
			Type:         oa.NewArraySchema().Type,
			ExternalDocs: schema.ExternalDocs,
			Enum:         schema.Enum,
			Items:        oa.NewSchemaRef("", innerSchema),
			Extensions:   schema.Extensions,
		}
		schema = outerSchema
	}

	// Add to parent
	parentSchema[schema.Title] = oa.NewSchemaRef("", schema)
	return order
}

func parseGroups(re *regexp.Regexp, v string) []map[string]string {
	var ret []map[string]string
	for _, match := range re.FindAllStringSubmatch(v, -1) {
		group := map[string]string{}
		for i, name := range re.SubexpNames() {
			if i != 0 && name != "" {
				group[name] = match[i]
			}
		}
		ret = append(ret, group)
	}
	return ret
}
