// Package oatype builds OpenAPI schemas for GraphQL types.
package oatype

import (
	oa "github.com/getkin/kin-openapi/openapi3"
	"github.com/vektah/gqlparser/v2/ast"
)

// Scalars maps each GraphQL scalar to a constructor for its JSON
// representation. A custom scalar does not declare one, so every scalar in the
// GraphQL schema must be listed here; TestScalarsMatchTheGraphQLSchema checks
// that. Each call returns a new schema, so fields never share one.
//
// A schema with no Type is typeless: its values are not one JSON type.
var Scalars = map[string]func() *oa.Schema{
	// Built in
	"Int":     typed(oa.TypeInteger, ""),
	"Float":   typed(oa.TypeNumber, ""),
	"String":  typed(oa.TypeString, ""),
	"Boolean": typed(oa.TypeBoolean, ""),
	"ID":      typed(oa.TypeInteger, "int64"),

	// Strings
	"Time":     example(oa.TypeString, "datetime", "2019-11-15T00:45:55.409906"),
	"Date":     example(oa.TypeString, "date", "2019-11-15"),
	"Seconds":  example(oa.TypeString, "hms", "15:21:04"),
	"Color":    typed(oa.TypeString, ""),
	"Language": typed(oa.TypeString, ""),
	"Url":      typed(oa.TypeString, ""),
	"Email":    typed(oa.TypeString, "email"),
	"Timezone": typed(oa.TypeString, ""),

	// Other JSON values
	"Bool": typed(oa.TypeBoolean, ""),
	"Strings": func() *oa.Schema {
		return &oa.Schema{Type: &oa.Types{oa.TypeArray}, Items: oa.NewSchemaRef("", typed(oa.TypeString, "")())}
	},
	"Counts": typed(oa.TypeObject, ""),
	"Tags":   typed(oa.TypeObject, ""),
	"Map":    typed(oa.TypeObject, ""),

	// GeoJSON geometries
	"Geometry":     typed(oa.TypeObject, ""),
	"Point":        typed(oa.TypeObject, ""),
	"LineString":   typed(oa.TypeObject, ""),
	"Polygon":      typed(oa.TypeObject, ""),
	"MultiPolygon": typed(oa.TypeObject, ""),

	// Typeless
	"Any":    func() *oa.Schema { return &oa.Schema{} },
	"Upload": func() *oa.Schema { return &oa.Schema{} },
}

func typed(jsonType string, format string) func() *oa.Schema {
	return func() *oa.Schema { return &oa.Schema{Type: &oa.Types{jsonType}, Format: format} }
}

func example(jsonType string, format string, value string) func() *oa.Schema {
	return func() *oa.Schema { return &oa.Schema{Type: &oa.Types{jsonType}, Format: format, Example: value} }
}

// Schema returns the schema for a value of GraphQL type t: one array per list
// level, each carrying its own nullability, around the schema for the named
// type. It maps the type only; a field's selected properties and docstring
// belong on Element of the result.
//
// x-graphql-type names the GraphQL type where it identifies something the JSON
// type does not: objects, including object-valued scalars, and enums. It is set
// on the element only.
func Schema(gs *ast.Schema, t *ast.Type) *oa.Schema {
	if t.Elem != nil {
		return &oa.Schema{
			Type:       &oa.Types{oa.TypeArray},
			Nullable:   !t.NonNull,
			Items:      oa.NewSchemaRef("", Schema(gs, t.Elem)),
			Extensions: map[string]any{},
		}
	}
	name := t.NamedType
	var s *oa.Schema
	isEnum := false
	if scalar, ok := Scalars[name]; ok {
		s = scalar()
	} else if def := gs.Types[name]; def.Kind == ast.Enum {
		isEnum = true
		s = &oa.Schema{Type: &oa.Types{oa.TypeString}}
		for _, v := range def.EnumValues {
			s.Enum = append(s.Enum, v.Name)
		}
	} else {
		// Objects, interfaces, and unions. An unmapped scalar fails
		// TestScalarsMatchTheGraphQLSchema rather than reaching here.
		s = &oa.Schema{Type: &oa.Types{oa.TypeObject}}
	}
	s.Nullable = !t.NonNull
	s.Extensions = map[string]any{}
	if isEnum || s.Type.Is(oa.TypeObject) {
		s.Extensions["x-graphql-type"] = name
	}
	return s
}

// Element returns the schema of the innermost array element, or s itself if
// s is not an array.
func Element(s *oa.Schema) *oa.Schema {
	for s.Items != nil {
		s = s.Items.Value
	}
	return s
}
