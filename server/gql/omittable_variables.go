package gql

import (
	"context"
	"reflect"
	"strings"

	"github.com/99designs/gqlgen/graphql"
	"github.com/vektah/gqlparser/v2/ast"
)

// unsetMissingVariables leaves out an input field written as a variable the
// request didn't provide. The GraphQL spec says such a field is not provided,
// but gqlparser builds the input object with it as null, which an Omittable
// field would take as a clear: set:{id: $id, traversal_time: $tt} without $tt
// would clear traversal_time. A variable's default counts as provided.
func unsetMissingVariables[T any](ctx context.Context, argName string, obj *T) {
	if !graphql.HasOperationContext(ctx) {
		return
	}
	fc := graphql.GetFieldContext(ctx)
	if fc == nil || fc.Field.Field == nil {
		return
	}
	arg := fc.Field.Arguments.ForName(argName)
	if arg == nil || arg.Value == nil || arg.Value.Kind != ast.ObjectValue {
		return
	}
	vars := graphql.GetOperationContext(ctx).Variables
	missing := map[string]bool{}
	for _, child := range arg.Value.Children {
		if child.Value != nil && child.Value.Kind == ast.Variable {
			if _, ok := vars[child.Value.Raw]; !ok {
				missing[child.Name] = true
			}
		}
	}
	if len(missing) == 0 {
		return
	}
	v := reflect.ValueOf(obj).Elem()
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		name, _, _ := strings.Cut(t.Field(i).Tag.Get("json"), ",")
		ft := t.Field(i).Type
		if missing[name] && ft.PkgPath() == "github.com/99designs/gqlgen/graphql" && strings.HasPrefix(ft.Name(), "Omittable[") {
			v.Field(i).Set(reflect.Zero(ft))
		}
	}
}
