package gql

import (
	"context"
	"reflect"
	"strings"

	"github.com/99designs/gqlgen/graphql"
	"github.com/vektah/gqlparser/v2/ast"
)

// unsetMissingVariables is a field middleware that leaves out an input field
// written as a variable the request didn't provide, at any depth. The
// GraphQL spec says such a field is not provided, but gqlparser builds the
// input object with it as null, which an Omittable field takes as a clear:
// set:{id: $id, traversal_time: $tt} without $tt would clear traversal_time.
// An input object whose fields are all such variables is left out too, so
// parent:{id: $p} without $p doesn't clear the parent. A variable's default
// counts as provided.
func unsetMissingVariables(ctx context.Context, next graphql.Resolver) (any, error) {
	fc := graphql.GetFieldContext(ctx)
	if fc == nil || fc.Field.Field == nil || len(fc.Field.Arguments) == 0 || !graphql.HasOperationContext(ctx) {
		return next(ctx)
	}
	vars := graphql.GetOperationContext(ctx).Variables
	for _, arg := range fc.Field.Arguments {
		v, ok := fc.Args[arg.Name]
		if !ok || v == nil || arg.Value == nil || arg.Value.Kind != ast.ObjectValue || !hasMissingVariable(arg.Value, vars) {
			continue
		}
		cp := reflect.New(reflect.TypeOf(v)).Elem()
		cp.Set(reflect.ValueOf(v))
		dropMissing(cp, arg.Value, vars)
		fc.Args[arg.Name] = cp.Interface()
	}
	return next(ctx)
}

func hasMissingVariable(v *ast.Value, vars map[string]any) bool {
	if v == nil {
		return false
	}
	if v.Kind == ast.Variable {
		_, provided := vars[v.Raw]
		return !provided
	}
	for _, c := range v.Children {
		if hasMissingVariable(c.Value, vars) {
			return true
		}
	}
	return false
}

var omittablePkg = reflect.TypeOf(graphql.Omittable[*int]{}).PkgPath()

func isOmittable(t reflect.Type) bool {
	return t.PkgPath() == omittablePkg && strings.HasPrefix(t.Name(), "Omittable[")
}

// dropMissing resets the fields of the input struct v (or pointer to one)
// that obj, its literal, gives as missing variables, and says whether every
// field obj gives was missing.
func dropMissing(v reflect.Value, obj *ast.Value, vars map[string]any) bool {
	if v.Kind() == reflect.Pointer {
		if v.IsNil() {
			return false
		}
		v = v.Elem()
	}
	if v.Kind() != reflect.Struct || len(obj.Children) == 0 {
		return false
	}
	fields := map[string]int{}
	for i := 0; i < v.NumField(); i++ {
		name, _, _ := strings.Cut(v.Type().Field(i).Tag.Get("json"), ",")
		fields[name] = i
	}
	all := true
	for _, child := range obj.Children {
		i, ok := fields[child.Name]
		if !ok || child.Value == nil {
			all = false
			continue
		}
		f := v.Field(i)
		missing := false
		switch child.Value.Kind {
		case ast.Variable:
			_, provided := vars[child.Value.Raw]
			missing = !provided
		case ast.ObjectValue:
			inner := f
			if isOmittable(f.Type()) {
				inner = f.MethodByName("Value").Call(nil)[0]
			}
			missing = dropMissing(inner, child.Value, vars)
		}
		if missing && f.CanSet() {
			f.Set(reflect.Zero(f.Type()))
		}
		all = all && missing
	}
	return all
}
