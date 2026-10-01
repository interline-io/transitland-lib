package oatype_test

import (
	"context"
	"encoding/json"
	"testing"

	oa "github.com/getkin/kin-openapi/openapi3"
	"github.com/interline-io/transitland-lib/server/gql"
	"github.com/interline-io/transitland-lib/server/rest"
	"github.com/interline-io/transitland-lib/server/rest/oatype"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vektah/gqlparser/v2"
	"github.com/vektah/gqlparser/v2/ast"
)

const testSDL = `
scalar Date
scalar Point
scalar Any
enum Mode { BUS RAIL }
type Stop { id: Int! }
type Query {
	nullableInt: Int
	intList: [Int!]!
	nullableIntList: [Int]
	floatMatrix: [[Float!]!]
	mode: Mode
	modeList: [Mode!]
	modeMatrix: [[Mode!]]
	stop: Stop
	stopList: [Stop]!
	point: Point
	pointList: [Point!]
	any: Any
	date: Date!
	dateList: [Date!]
}
`

// Each GraphQL type maps to one schema: one array per list level with its own
// nullability, and the named type's JSON type at the bottom.
func TestSchema(t *testing.T) {
	gs, err := gqlparser.LoadSchema(&ast.Source{Input: testSDL})
	require.NoError(t, err)
	query := gs.Types["Query"]

	tcs := []struct {
		field  string
		expect string
	}{
		{"nullableInt", `{"type":"integer","nullable":true}`},
		{"intList", `{"type":"array","items":{"type":"integer"}}`},
		{"nullableIntList", `{"type":"array","nullable":true,"items":{"type":"integer","nullable":true}}`},
		{"floatMatrix", `{"type":"array","nullable":true,"items":{"type":"array","items":{"type":"number"}}}`},
		{"mode", `{"type":"string","nullable":true,"enum":["BUS","RAIL",null],"x-graphql-type":"Mode"}`},
		{"modeList", `{"type":"array","nullable":true,"items":{"type":"string","enum":["BUS","RAIL"],"x-graphql-type":"Mode"}}`},
		{"modeMatrix", `{"type":"array","nullable":true,"items":{"type":"array","nullable":true,"items":{"type":"string","enum":["BUS","RAIL"],"x-graphql-type":"Mode"}}}`},
		{"stop", `{"type":"object","nullable":true,"x-graphql-type":"Stop"}`},
		{"stopList", `{"type":"array","items":{"type":"object","nullable":true,"x-graphql-type":"Stop"}}`},
		{"point", `{"type":"object","nullable":true,"x-graphql-type":"Point"}`},
		{"pointList", `{"type":"array","nullable":true,"items":{"type":"object","x-graphql-type":"Point"}}`},
		{"any", `{"nullable":true}`},
		{"date", `{"type":"string","format":"date","example":"2019-11-15"}`},
		{"dateList", `{"type":"array","nullable":true,"items":{"type":"string","format":"date","example":"2019-11-15"}}`},
	}
	for _, tc := range tcs {
		t.Run(tc.field, func(t *testing.T) {
			def := query.Fields.ForName(tc.field)
			require.NotNil(t, def)
			s := oatype.Schema(gs, def.Type)
			got, err := json.Marshal(s)
			require.NoError(t, err)
			assert.JSONEq(t, tc.expect, string(got))
			assert.NoError(t, s.Validate(context.Background()))
		})
	}
}

// Every scalar the GraphQL schema declares has an entry, and every entry is
// declared, so a new scalar cannot silently fall through to an object.
func TestScalarsMatchTheGraphQLSchema(t *testing.T) {
	gs := gql.NewExecutableSchema().Schema()
	for name, def := range gs.Types {
		if def.Kind != ast.Scalar {
			continue
		}
		_, ok := oatype.Scalars[name]
		assert.True(t, ok, "scalar %s is declared in the GraphQL schema but missing from oatype.Scalars", name)
	}
	for name := range oatype.Scalars {
		def, ok := gs.Types[name]
		assert.True(t, ok && def.Kind == ast.Scalar, "oatype.Scalars has %s, which the GraphQL schema does not declare as a scalar", name)
	}
}

// The generated REST document applies Schema to every response field. This
// runs without a database, unlike the rest package's tests. Object-valued
// scalars are checked across the whole document, since they carry
// x-graphql-type; primitive scalars and enums are spot-checked.
func TestRESTDocumentIsTypedByGraphQLType(t *testing.T) {
	doc, err := rest.GenerateOpenAPI("/rest")
	require.NoError(t, err)

	require.NoError(t, doc.WalkSchemas(func(ptr string, ref *oa.SchemaRef) error {
		s := ref.Value
		name, named := s.Extensions["x-graphql-type"].(string)
		if scalar, ok := oatype.Scalars[name]; named && ok {
			assert.Equal(t, scalar().Type, s.Type, "%s: scalar %s", ptr, name)
		}
		if s.Type.Is(oa.TypeArray) {
			assert.False(t, named, "%s: array repeats its items' x-graphql-type", ptr)
		}
		return nil
	}))

	routeTypes := responseProperty(t, doc, "/agencies", "agencies", "route_types")
	require.NotNil(t, routeTypes.Items)
	assert.True(t, routeTypes.Items.Value.Type.Is(oa.TypeInteger))

	languages := responseProperty(t, doc, "/feeds", "feeds", "languages")
	require.NotNil(t, languages.Items)
	assert.True(t, languages.Items.Value.Type.Is(oa.TypeString))

	addedDates := responseProperty(t, doc, "/routes/{route_key}/trips", "trips", "calendar", "added_dates")
	require.NotNil(t, addedDates.Items)
	assert.Equal(t, "date", addedDates.Items.Value.Format)

	spec := responseProperty(t, doc, "/feeds", "feeds", "spec")
	assert.True(t, spec.Type.Is(oa.TypeString))
	assert.Contains(t, spec.Enum, "GTFS")
	assert.Equal(t, "FeedSpecTypes", spec.Extensions["x-graphql-type"])

	geometry := responseProperty(t, doc, "/stops", "stops", "geometry")
	assert.True(t, geometry.Type.Is(oa.TypeObject))
	assert.Equal(t, "Point", geometry.Extensions["x-graphql-type"])
}

// responseProperty follows a list response's items down to one property,
// failing rather than panicking if any step is missing.
func responseProperty(t *testing.T, doc *oa.T, path string, names ...string) *oa.Schema {
	t.Helper()
	item := doc.Paths.Value(path)
	require.NotNil(t, item, path)
	require.NotNil(t, item.Get, path)
	require.NotNil(t, item.Get.Responses, path)
	resp := item.Get.Responses.Value("200")
	require.NotNil(t, resp, path)
	require.NotNil(t, resp.Value, path)
	media := resp.Value.Content.Get("application/json")
	require.NotNil(t, media, path)
	require.NotNil(t, media.Schema, path)
	require.NotNil(t, media.Schema.Value, path)
	s := media.Schema.Value
	for _, name := range names {
		if s.Items != nil {
			require.NotNil(t, s.Items.Value, "%s: %s", path, name)
			s = s.Items.Value
		}
		prop, ok := s.Properties[name]
		require.True(t, ok, "%s: no property %s", path, name)
		require.NotNil(t, prop.Value, "%s: %s", path, name)
		s = prop.Value
	}
	return s
}
