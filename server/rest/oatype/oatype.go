// Package oatype builds OpenAPI schemas for GraphQL types.
package oatype

import (
	oa "github.com/getkin/kin-openapi/openapi3"
	"github.com/vektah/gqlparser/v2/ast"
)

// Scalars maps each GraphQL scalar to its JSON representation. A custom
// scalar does not declare one, so every scalar in the GraphQL schema must be
// listed here; TestScalarsMatchTheGraphQLSchema checks that.
//
// An entry with no Type is typeless: its values are not one JSON type.
var Scalars = map[string]oa.Schema{
	// Built in
	"Int":     {Type: &oa.Types{oa.TypeInteger}},
	"Float":   {Type: &oa.Types{oa.TypeNumber}},
	"String":  {Type: &oa.Types{oa.TypeString}},
	"Boolean": {Type: &oa.Types{oa.TypeBoolean}},
	"ID":      {Type: &oa.Types{oa.TypeInteger}, Format: "int64"},

	// Strings
	"Time":     {Type: &oa.Types{oa.TypeString}, Format: "datetime", Example: "2019-11-15T00:45:55.409906"},
	"Date":     {Type: &oa.Types{oa.TypeString}, Format: "date", Example: "2019-11-15"},
	"Seconds":  {Type: &oa.Types{oa.TypeString}, Format: "hms", Example: "15:21:04"},
	"Color":    {Type: &oa.Types{oa.TypeString}},
	"Language": {Type: &oa.Types{oa.TypeString}},
	"Url":      {Type: &oa.Types{oa.TypeString}},
	"Email":    {Type: &oa.Types{oa.TypeString}, Format: "email"},
	"Timezone": {Type: &oa.Types{oa.TypeString}},

	// Other JSON values
	"Bool":    {Type: &oa.Types{oa.TypeBoolean}},
	"Strings": {Type: &oa.Types{oa.TypeArray}, Items: oa.NewSchemaRef("", &oa.Schema{Type: &oa.Types{oa.TypeString}})},
	"Counts":  {Type: &oa.Types{oa.TypeObject}},
	"Tags":    {Type: &oa.Types{oa.TypeObject}},
	"Map":     {Type: &oa.Types{oa.TypeObject}},

	// GeoJSON geometries
	"Geometry":     {Type: &oa.Types{oa.TypeObject}},
	"Point":        {Type: &oa.Types{oa.TypeObject}},
	"LineString":   {Type: &oa.Types{oa.TypeObject}},
	"Polygon":      {Type: &oa.Types{oa.TypeObject}},
	"MultiPolygon": {Type: &oa.Types{oa.TypeObject}},

	// Typeless
	"Any":    {},
	"Upload": {},
}

// Schema returns the schema for a value of GraphQL type t: one array per list
// level, each carrying its own nullability, around the schema for the named
// type. props holds the selected fields of an object type.
//
// x-graphql-type names the GraphQL type where it identifies something the JSON
// type does not: objects, including object-valued scalars, and enums.
func Schema(gs *ast.Schema, t *ast.Type, props oa.Schemas) *oa.Schema {
	if t.Elem != nil {
		return &oa.Schema{
			Type:     &oa.Types{oa.TypeArray},
			Nullable: !t.NonNull,
			Items:    oa.NewSchemaRef("", Schema(gs, t.Elem, props)),
		}
	}
	s := &oa.Schema{Nullable: !t.NonNull, Extensions: map[string]any{}}
	name := t.NamedType
	def := gs.Types[name]
	if scalar, ok := Scalars[name]; ok {
		// Copied, so editing one field's schema cannot change the map.
		if scalar.Type != nil {
			s.Type = &oa.Types{}
			*s.Type = append(*s.Type, *scalar.Type...)
		}
		s.Format = scalar.Format
		s.Example = scalar.Example
		if scalar.Items != nil {
			items := *scalar.Items.Value
			s.Items = oa.NewSchemaRef("", &items)
		}
		if s.Type.Is(oa.TypeObject) {
			s.Extensions["x-graphql-type"] = name
		}
		return s
	}
	if def != nil && def.Kind == ast.Enum {
		s.Type = &oa.Types{oa.TypeString}
		for _, v := range def.EnumValues {
			s.Enum = append(s.Enum, v.Name)
		}
		s.Extensions["x-graphql-type"] = name
		return s
	}
	// Objects, interfaces, and unions. An unmapped scalar fails
	// TestScalarsMatchTheGraphQLSchema rather than reaching here.
	s.Type = &oa.Types{oa.TypeObject}
	s.Properties = props
	s.Extensions["x-graphql-type"] = name
	return s
}

// Element returns the schema of the innermost list element, or s itself if
// s is not a list.
func Element(s *oa.Schema) *oa.Schema {
	for s.Items != nil && s.Items.Value != nil && s.Type.Is(oa.TypeArray) {
		s = s.Items.Value
	}
	return s
}
