package openapi

import (
	"encoding/json"
	"go/types"
	"net/http"
	"reflect"
	"sort"
	"strconv"
	"strings"

	"github.com/zombocoder/goboot/model"
)

// buildDocument assembles the OpenAPI 3.0 document. It is built from Go maps and
// marshaled with sorted keys, so the output is deterministic (§46.4).
func buildDocument(app *model.Application) ([]byte, error) {
	schemas := map[string]any{}
	paths := map[string]any{}

	for _, r := range app.Routes {
		item, ok := paths[r.Pattern].(map[string]any)
		if !ok {
			item = map[string]any{}
			paths[r.Pattern] = item
		}
		item[strings.ToLower(r.Method)] = operationFor(r, schemas)
	}

	doc := map[string]any{
		"openapi": "3.0.3",
		"info": map[string]any{
			"title":   titleOf(app),
			"version": "1.0.0",
		},
		"paths": paths,
	}
	if len(schemas) > 0 {
		doc["components"] = map[string]any{"schemas": schemas}
	}
	return json.MarshalIndent(doc, "", "  ")
}

// titleOf derives the document title from the application name.
func titleOf(app *model.Application) string {
	if app.Name != "" {
		return app.Name
	}
	return "goboot application"
}

// operationFor renders one path+method operation.
func operationFor(r *model.Route, schemas map[string]any) map[string]any {
	op := map[string]any{"operationId": r.HandlerName}
	if len(r.Authorize) > 0 {
		op["security"] = []any{map[string]any{"roles": toAny(r.Authorize)}}
	}
	if params := parametersFor(r, schemas); len(params) > 0 {
		op["parameters"] = params
	}
	if body := requestBodyFor(r, schemas); body != nil {
		op["requestBody"] = body
	}
	op["responses"] = responsesFor(r, schemas)
	return op
}

// parametersFor extracts path/query/header/cookie parameters from the request
// struct's binding tags, sorted by location then name for determinism.
func parametersFor(r *model.Route, schemas map[string]any) []any {
	st, ok := structOf(r.RequestType)
	if !ok {
		return nil
	}
	var params []map[string]any
	for i := 0; i < st.NumFields(); i++ {
		f := st.Field(i)
		if !f.Exported() {
			continue
		}
		in, name := parameterLocation(reflect.StructTag(st.Tag(i)))
		if in == "" {
			continue
		}
		params = append(params, map[string]any{
			"name":     name,
			"in":       in,
			"required": in == "path",
			"schema":   schemaFor(f.Type(), schemas),
		})
	}
	sort.Slice(params, func(i, j int) bool {
		if params[i]["in"] != params[j]["in"] {
			return params[i]["in"].(string) < params[j]["in"].(string)
		}
		return params[i]["name"].(string) < params[j]["name"].(string)
	})
	out := make([]any, len(params))
	for i, p := range params {
		out[i] = p
	}
	return out
}

// parameterLocation maps a field's binding tag to an OpenAPI parameter location
// and name, or ("", "") when the field is not a parameter.
func parameterLocation(tag reflect.StructTag) (in, name string) {
	for _, key := range []struct{ tag, in string }{
		{"path", "path"}, {"query", "query"}, {"header", "header"}, {"cookie", "cookie"},
	} {
		if v, ok := tag.Lookup(key.tag); ok {
			return key.in, strings.Split(v, ",")[0]
		}
	}
	return "", ""
}

// requestBodyFor builds a JSON request body from the request struct's json
// fields, for methods that carry a body. It returns nil when there is none.
func requestBodyFor(r *model.Route, schemas map[string]any) map[string]any {
	if !methodHasBody(r.Method) {
		return nil
	}
	st, ok := structOf(r.RequestType)
	if !ok {
		return nil
	}
	props := map[string]any{}
	for i := 0; i < st.NumFields(); i++ {
		f := st.Field(i)
		if !f.Exported() {
			continue
		}
		tag := reflect.StructTag(st.Tag(i))
		if in, _ := parameterLocation(tag); in != "" {
			continue // path/query/header/cookie handled as parameters
		}
		name := jsonName(tag, f.Name())
		if name == "-" {
			continue
		}
		props[name] = schemaFor(f.Type(), schemas)
	}
	if len(props) == 0 {
		return nil
	}
	schema := map[string]any{"type": "object", "properties": props}
	return map[string]any{
		"required": true,
		"content":  contentFor(r.Consumes, schema),
	}
}

// responsesFor renders the success response (and a generic default error).
func responsesFor(r *model.Route, schemas map[string]any) map[string]any {
	status := strconv.Itoa(r.SuccessStatus)
	success := map[string]any{"description": statusText(r.SuccessStatus)}
	if r.ResponseType != nil {
		success["content"] = contentFor(r.Produces, schemaFor(r.ResponseType, schemas))
	}
	registerProblem(schemas)
	return map[string]any{
		status: success,
		"default": map[string]any{
			"description": "Error",
			"content": map[string]any{
				"application/problem+json": map[string]any{
					"schema": ref("Problem"),
				},
			},
		},
	}
}

// contentFor wraps a schema in a content map keyed by media type. It defaults to
// application/json when no media types are declared.
func contentFor(mediaTypes []string, schema map[string]any) map[string]any {
	if len(mediaTypes) == 0 {
		mediaTypes = []string{"application/json"}
	}
	content := map[string]any{}
	for _, mt := range mediaTypes {
		content[mt] = map[string]any{"schema": schema}
	}
	return content
}

// schemaFor maps a Go type to an OpenAPI schema, registering named struct types
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
			return ref(name)
		}
		return schemaFor(u.Underlying(), schemas)
	case *types.Basic:
		return basicSchema(u)
	default:
		return map[string]any{}
	}
}

// registerSchema populates components/schemas for a named struct type once,
// using a placeholder first to tolerate recursive types.
func registerSchema(name string, n *types.Named, schemas map[string]any) {
	if _, ok := schemas[name]; ok {
		return
	}
	schemas[name] = map[string]any{} // placeholder breaks reference cycles
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

// registerProblem adds the RFC-7807 Problem schema used by error responses.
func registerProblem(schemas map[string]any) {
	if _, ok := schemas["Problem"]; ok {
		return
	}
	schemas["Problem"] = map[string]any{
		"type": "object",
		"properties": map[string]any{
			"type":   map[string]any{"type": "string"},
			"title":  map[string]any{"type": "string"},
			"status": map[string]any{"type": "integer", "format": "int32"},
			"detail": map[string]any{"type": "string"},
			"code":   map[string]any{"type": "string"},
		},
	}
}

// basicSchema maps a Go basic type to an OpenAPI type/format.
func basicSchema(b *types.Basic) map[string]any {
	info := b.Info()
	switch {
	case info&types.IsBoolean != 0:
		return map[string]any{"type": "boolean"}
	case info&types.IsInteger != 0:
		format := "int32"
		switch b.Kind() {
		case types.Int, types.Int64, types.Uint, types.Uint64, types.Uintptr:
			format = "int64"
		}
		return map[string]any{"type": "integer", "format": format}
	case info&types.IsFloat != 0:
		format := "double"
		if b.Kind() == types.Float32 {
			format = "float"
		}
		return map[string]any{"type": "number", "format": format}
	case info&types.IsString != 0:
		return map[string]any{"type": "string"}
	default:
		return map[string]any{}
	}
}

// ---- small helpers -------------------------------------------------------

func structOf(t types.Type) (*types.Struct, bool) {
	if t == nil {
		return nil, false
	}
	st, ok := t.Underlying().(*types.Struct)
	return st, ok
}

func methodHasBody(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch:
		return true
	default:
		return false
	}
}

func jsonName(tag reflect.StructTag, field string) string {
	if v, ok := tag.Lookup("json"); ok {
		if name := strings.Split(v, ",")[0]; name != "" {
			return name
		}
	}
	return field
}

func isTime(n *types.Named) bool {
	obj := n.Obj()
	return obj.Pkg() != nil && obj.Pkg().Path() == "time" && obj.Name() == "Time"
}

func ref(name string) map[string]any {
	return map[string]any{"$ref": "#/components/schemas/" + name}
}

func statusText(code int) string {
	if t := http.StatusText(code); t != "" {
		return t
	}
	return "Response"
}

func toAny(ss []string) []any {
	out := make([]any, len(ss))
	for i, s := range ss {
		out[i] = s
	}
	return out
}
