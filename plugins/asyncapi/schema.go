package asyncapi

import (
	"go/types"
	"reflect"
	"strings"
)

// schemaFor maps a Go type to a JSON schema, registering named struct types
// under components/schemas and returning a $ref to them.
func schemaFor(t types.Type, schemas map[string]any) map[string]any {
	switch u := t.(type) {
	case *types.Pointer:
		return schemaFor(u.Elem(), schemas)
	case *types.Slice:
		return map[string]any{"type": "array", "items": schemaFor(u.Elem(), schemas)}
	case *types.Array:
		return map[string]any{"type": "array", "items": schemaFor(u.Elem(), schemas)}
	case *types.Map:
		return map[string]any{"type": "object", "additionalProperties": schemaFor(u.Elem(), schemas)}
	case *types.Named:
		if isTime(u) {
			return map[string]any{"type": "string", "format": "date-time"}
		}
		if _, ok := u.Underlying().(*types.Struct); ok {
			name := u.Obj().Name()
			registerSchema(name, u, schemas)
			return map[string]any{"$ref": "#/components/schemas/" + name}
		}
		return schemaFor(u.Underlying(), schemas)
	case *types.Basic:
		return basicSchema(u)
	default:
		return map[string]any{}
	}
}

// registerSchema populates components/schemas for a named struct once, using a
// placeholder first to tolerate recursive types.
func registerSchema(name string, n *types.Named, schemas map[string]any) {
	if _, ok := schemas[name]; ok {
		return
	}
	schemas[name] = map[string]any{}
	st, ok := n.Underlying().(*types.Struct)
	if !ok {
		return
	}
	props := map[string]any{}
	for i := 0; i < st.NumFields(); i++ {
		f := st.Field(i)
		if !f.Exported() {
			continue
		}
		propName := jsonName(reflect.StructTag(st.Tag(i)), f.Name())
		if propName == "-" {
			continue
		}
		props[propName] = schemaFor(f.Type(), schemas)
	}
	schemas[name] = map[string]any{"type": "object", "properties": props}
}

func basicSchema(b *types.Basic) map[string]any {
	info := b.Info()
	switch {
	case info&types.IsBoolean != 0:
		return map[string]any{"type": "boolean"}
	case info&types.IsInteger != 0:
		return map[string]any{"type": "integer"}
	case info&types.IsFloat != 0:
		return map[string]any{"type": "number"}
	case info&types.IsString != 0:
		return map[string]any{"type": "string"}
	default:
		return map[string]any{}
	}
}

func isTime(n *types.Named) bool {
	o := n.Obj()
	return o.Pkg() != nil && o.Pkg().Path() == "time" && o.Name() == "Time"
}

func jsonName(tag reflect.StructTag, field string) string {
	if v, ok := tag.Lookup("json"); ok {
		name, _, _ := strings.Cut(v, ",")
		if name != "" {
			return name
		}
	}
	return field
}
