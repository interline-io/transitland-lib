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
	// Built in. gqlgen serializes ID as a string.
	"Int":     oa.NewIntegerSchema,
	"Float":   oa.NewFloat64Schema,
	"String":  oa.NewStringSchema,
	"Boolean": oa.NewBoolSchema,
	"ID":      oa.NewStringSchema,

	// Strings
	"Time": func() *oa.Schema {
		return withExample(oa.NewStringSchema().WithFormat("datetime"), "2019-11-15T00:45:55.409906")
	},
	"Date":     func() *oa.Schema { return withExample(oa.NewStringSchema().WithFormat("date"), "2019-11-15") },
	"Seconds":  func() *oa.Schema { return withExample(oa.NewStringSchema().WithFormat("hms"), "15:21:04") },
	"Color":    oa.NewStringSchema,
	"Language": oa.NewStringSchema,
	"Url":      oa.NewStringSchema,
	"Email":    func() *oa.Schema { return oa.NewStringSchema().WithFormat("email") },
	"Timezone": oa.NewStringSchema,

	// Other JSON values
	"Bool":    oa.NewBoolSchema,
	"Strings": func() *oa.Schema { return oa.NewArraySchema().WithItems(oa.NewStringSchema()) },
	"Counts":  oa.NewObjectSchema,
	"Tags":    oa.NewObjectSchema,
	"Map":     oa.NewObjectSchema,

	// GeoJSON geometries
	"Geometry":     oa.NewObjectSchema,
	"Point":        oa.NewObjectSchema,
	"LineString":   oa.NewObjectSchema,
	"Polygon":      oa.NewObjectSchema,
	"MultiPolygon": oa.NewObjectSchema,

	// Typeless
	"Any":    func() *oa.Schema { return &oa.Schema{} },
	"Upload": func() *oa.Schema { return &oa.Schema{} },
}

func withExample(s *oa.Schema, example string) *oa.Schema {
	s.Example = example
	return s
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
	} else if def := gs.Types[name]; def != nil && def.Kind == ast.Enum {
		isEnum = true
		s = &oa.Schema{Type: &oa.Types{oa.TypeString}}
		for _, v := range def.EnumValues {
			s.Enum = append(s.Enum, v.Name)
		}
		// OpenAPI 3.0 checks enum before nullable, so null must be listed.
		if !t.NonNull {
			s.Enum = append(s.Enum, nil)
		}
	} else {
		// Objects, interfaces, and unions, and any name gs does not define. An
		// unmapped scalar fails TestScalarsMatchTheGraphQLSchema rather than
		// reaching here.
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
