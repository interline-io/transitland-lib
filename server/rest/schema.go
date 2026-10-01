package rest

import (
	"regexp"
	"strings"

	oa "github.com/getkin/kin-openapi/openapi3"
	"github.com/interline-io/transitland-lib/server/gql"
	"github.com/interline-io/transitland-lib/server/rest/oatype"
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
	gs := gql.NewExecutableSchema().Schema()

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
			queryRecurse(gs, sel, responseObj.Value.Properties, selOrder)
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

	notFoundDesc := "Not found"
	ret.Set("404", &oa.ResponseRef{
		Value: &oa.Response{
			Description: &notFoundDesc,
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

func queryRecurse(gs *ast.Schema, recurseValue any, parentSchema oa.Schemas, order int) int {
	if frag, ok := recurseValue.(*ast.FragmentSpread); ok {
		for _, sel := range frag.Definition.SelectionSet {
			// Ugly hack to put fragments at the end of the selection set
			order = queryRecurse(gs, sel, parentSchema, order+1)
		}
		return order
	}
	field, ok := recurseValue.(*ast.Field)
	if !ok {
		return order
	}
	if field.Comment != nil {
		for _, c := range field.Comment.List {
			if ParseDocstring(c.Value).Hide {
				return order
			}
		}
	}

	props := oa.Schemas{}
	for _, sel := range field.SelectionSet {
		order = queryRecurse(gs, sel, props, order+1)
	}
	schema := oatype.Schema(gs, field.Definition.Type)
	schema.Title = field.Name
	order += 1
	schema.Extensions["x-order"] = order

	// Text and links describe the field. Selected properties and docstring
	// enum values and examples belong to the innermost element.
	parsed := ParseDocstring(field.Definition.Description)
	schema.Description = field.Definition.Description
	if parsed.Text != "" {
		schema.Description = parsed.Text
	}
	if n := len(parsed.ExternalDocs); n > 0 {
		doc := parsed.ExternalDocs[n-1]
		schema.ExternalDocs = &oa.ExternalDocs{URL: doc.URL, Description: doc.Text}
	}
	element := oatype.Element(schema)
	if len(props) > 0 {
		element.Properties = props
	}
	for _, e := range parsed.Enum {
		element.Enum = append(element.Enum, e)
	}
	if n := len(parsed.Examples); n > 0 {
		element.Example = parsed.Examples[n-1]
	}

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
