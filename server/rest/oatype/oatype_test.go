package oatype_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/interline-io/transitland-lib/server/gql"
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
		{"mode", `{"type":"string","nullable":true,"enum":["BUS","RAIL"],"x-graphql-type":"Mode"}`},
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
