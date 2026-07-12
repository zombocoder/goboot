package validate

import (
	"fmt"
	"go/types"
	"reflect"
	"sort"
	"strings"

	"github.com/zombocoder/goboot/annotation"
	"github.com/zombocoder/goboot/model"
)

// Diagnostic codes this plugin emits (§39.4, plugin-owned family).
const (
	codeTypeMismatch = "GOBVAL001" // constraint applied to an incompatible field type
	codeBadPattern   = "GOBVAL002" // @Pattern regex does not compile
	codeBadSize      = "GOBVAL003" // @Size bounds missing or min > max
	codeUnenforced   = "GOBVAL004" // constrained field is not on a request type
)

// fieldKind is the coarse category of a field's Go type that decides which
// constraints apply and how they are generated.
type fieldKind int

const (
	kindOther   fieldKind = iota // unsupported for constraints
	kindString                   // string: required/size(length)/pattern/email
	kindNumeric                  // int*/uint*/float*: min/max
	kindLenable                  // slice/array/map: required/size(len)
)

// fieldRule is the resolved set of constraints for one struct field.
type fieldRule struct {
	index      int    // position in the struct, for deterministic ordering
	goName     string // Go field name (v.<goName>)
	wire       string // error field name (json/path tag, or Go name)
	kind       fieldKind
	required   bool
	min, max   *int64 // @Min / @Max (numeric)
	sizeMin    *int64 // @Size(min=)
	sizeMax    *int64 // @Size(max=)
	pattern    string // @Pattern raw regex (empty if none)
	patternVar string // package-level var name holding the compiled pattern
	email      bool
}

// active reports whether the rule carries any enforceable constraint.
func (r fieldRule) active() bool {
	return r.required || r.email || r.pattern != "" ||
		r.min != nil || r.max != nil || r.sizeMin != nil || r.sizeMax != nil
}

// structRules groups a request struct's field rules.
type structRules struct {
	pkgPath string
	pkgName string
	name    string
	fields  []fieldRule
}

// structInfo describes a request struct discovered on the routes.
type structInfo struct {
	pkgPath string
	pkgName string
	name    string
	st      *types.Struct
}

// resolve turns the analyzed application into the validated constraint model
// plus any diagnostics (type mismatches, bad regexes, unenforced constraints).
// Only enforceable constraints on request-struct fields survive into the
// returned rules, so a caller can generate from them without re-checking.
func resolve(app *model.Application) ([]structRules, []*annotation.Diagnostic) {
	requests := requestStructs(app)
	var diags []*annotation.Diagnostic
	byKey := map[string]*structRules{}
	var order []string

	for _, d := range app.Declarations {
		if d.Target != annotation.TargetField || !hasOurAnnotation(d) {
			continue
		}
		key := d.Package + "." + d.Receiver
		info, ok := requests[key]
		if !ok {
			diags = append(diags, diag(annotation.SeverityWarning, codeUnenforced, d.Position,
				fmt.Sprintf("field %s.%s has validation constraints but %s is not an HTTP request type; they will not be enforced",
					d.Receiver, d.Name, d.Receiver)))
			continue
		}
		field, tag, idx, found := structField(info.st, d.Name)
		if !found {
			continue
		}
		rule := fieldRule{
			index:  idx,
			goName: d.Name,
			wire:   wireName(tag, d.Name),
			kind:   classify(field.Type()),
		}
		diags = append(diags, applyConstraints(&rule, info, d)...)
		if !rule.active() {
			continue
		}
		sr := byKey[key]
		if sr == nil {
			sr = &structRules{pkgPath: info.pkgPath, pkgName: info.pkgName, name: info.name}
			byKey[key] = sr
			order = append(order, key)
		}
		sr.fields = append(sr.fields, rule)
	}

	out := make([]structRules, 0, len(order))
	for _, key := range order {
		sr := byKey[key]
		sort.Slice(sr.fields, func(i, j int) bool { return sr.fields[i].index < sr.fields[j].index })
		out = append(out, *sr)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].pkgPath != out[j].pkgPath {
			return out[i].pkgPath < out[j].pkgPath
		}
		return out[i].name < out[j].name
	})
	return out, diags
}

// applyConstraints reads a field's annotations into rule, validating each
// against the field's type and returning diagnostics for any that don't apply.
func applyConstraints(rule *fieldRule, info *structInfo, d model.AnnotatedDecl) []*annotation.Diagnostic {
	var diags []*annotation.Diagnostic
	for _, a := range d.Annotations {
		switch a.Name {
		case annRequired:
			if rule.kind == kindString || rule.kind == kindLenable {
				rule.required = true
			} else {
				diags = append(diags, typeErr(a, "@Required applies to string, slice, or map fields"))
			}
		case annEmail:
			if rule.kind == kindString {
				rule.email = true
			} else {
				diags = append(diags, typeErr(a, "@Email applies to string fields"))
			}
		case annMin, annMax:
			n, ok := intPositional(a)
			if !ok {
				continue
			}
			if rule.kind != kindNumeric {
				diags = append(diags, typeErr(a, "@"+a.Name+" applies to numeric fields"))
				continue
			}
			if a.Name == annMin {
				rule.min = &n
			} else {
				rule.max = &n
			}
		case annPattern:
			raw, ok := stringPositional(a)
			if !ok {
				continue
			}
			if rule.kind != kindString {
				diags = append(diags, typeErr(a, "@Pattern applies to string fields"))
				continue
			}
			if bad, ok := compilePattern(a, raw); !ok {
				diags = append(diags, bad)
				continue
			}
			rule.pattern = raw
			rule.patternVar = patternVarName(info, rule.goName)
		case annSize:
			min, hasMin := intArg(a, "min")
			max, hasMax := intArg(a, "max")
			if !hasMin && !hasMax {
				diags = append(diags, diag(annotation.SeverityError, codeBadSize, a.Position,
					"@Size requires a min and/or max argument"))
				continue
			}
			if rule.kind != kindString && rule.kind != kindLenable {
				diags = append(diags, typeErr(a, "@Size applies to string, slice, or map fields"))
				continue
			}
			if hasMin && hasMax && min > max {
				diags = append(diags, diag(annotation.SeverityError, codeBadSize, a.Position,
					fmt.Sprintf("@Size min (%d) is greater than max (%d)", min, max)))
				continue
			}
			if hasMin {
				m := min
				rule.sizeMin = &m
			}
			if hasMax {
				m := max
				rule.sizeMax = &m
			}
		}
	}
	return diags
}

// requestStructs collects the request struct types referenced by the routes,
// keyed by "<pkgpath>.<TypeName>".
func requestStructs(app *model.Application) map[string]*structInfo {
	out := map[string]*structInfo{}
	for _, r := range app.Routes {
		t := r.RequestType
		if t == nil {
			continue
		}
		if p, ok := t.(*types.Pointer); ok {
			t = p.Elem()
		}
		named, ok := t.(*types.Named)
		if !ok {
			continue
		}
		st, ok := named.Underlying().(*types.Struct)
		if !ok {
			continue
		}
		obj := named.Obj()
		if obj.Pkg() == nil {
			continue
		}
		out[obj.Pkg().Path()+"."+obj.Name()] = &structInfo{
			pkgPath: obj.Pkg().Path(),
			pkgName: obj.Pkg().Name(),
			name:    obj.Name(),
			st:      st,
		}
	}
	return out
}

// structField finds a field by name, returning it, its tag, its index, and ok.
func structField(st *types.Struct, name string) (*types.Var, reflect.StructTag, int, bool) {
	for i := 0; i < st.NumFields(); i++ {
		if st.Field(i).Name() == name {
			return st.Field(i), reflect.StructTag(st.Tag(i)), i, true
		}
	}
	return nil, "", 0, false
}

// classify maps a Go type to the constraint category it supports.
func classify(t types.Type) fieldKind {
	switch u := t.Underlying().(type) {
	case *types.Basic:
		info := u.Info()
		switch {
		case info&types.IsString != 0:
			return kindString
		case info&(types.IsInteger|types.IsFloat) != 0:
			return kindNumeric
		}
	case *types.Slice, *types.Array, *types.Map:
		return kindLenable
	}
	return kindOther
}

// wireName picks the error field name from the binding tags, matching the wire
// contract; it falls back to the Go field name.
func wireName(tag reflect.StructTag, goName string) string {
	for _, key := range []string{"json", "path", "query", "header", "cookie"} {
		if v, ok := tag.Lookup(key); ok {
			if name := strings.Split(v, ",")[0]; name != "" && name != "-" {
				return name
			}
		}
	}
	return goName
}

// patternVarName builds a deterministic package-level var name for a compiled
// @Pattern regex.
func patternVarName(info *structInfo, field string) string {
	return "validatePattern" + exported(info.pkgName) + info.name + exported(field)
}

func exported(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
