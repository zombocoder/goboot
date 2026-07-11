package compiler

import (
	"go/types"

	"github.com/zombocoder/goboot/annotation"
	"github.com/zombocoder/goboot/model"
	"github.com/zombocoder/goboot/sqlgen"
)

// Repository diagnostic codes (GOBREP family, §39.4).
const (
	// CodeInvalidQuerySignature is a query/exec method with an unsupported
	// signature (§27.6).
	CodeInvalidQuerySignature = "GOBREP001"
	// CodeUnknownQueryParam is a named SQL parameter that no method argument
	// satisfies (§27.10).
	CodeUnknownQueryParam = "GOBREP002"
	// CodeMissingQuery is a generated-repository method without @Query or @Exec.
	CodeMissingQuery = "GOBREP003"
)

// discoverRepositoryInterface creates a component for an
// @Repository(generate=true) interface. Its implementation is generated from
// the interface's @Query/@Exec methods and injected with a db.DBProvider.
func (a *analysis) discoverRepositoryInterface(decl *Declaration, app *model.Application) {
	tn := decl.TypeName
	if tn == nil {
		return
	}
	if _, ok := tn.Type().Underlying().(*types.Interface); !ok {
		return
	}
	name := tn.Name()
	if explicit, ok := stringArg(decl, "Repository", "name"); ok {
		name = explicit
	}
	app.Components = append(app.Components, &model.Component{
		ID:           model.NewComponentID(decl.PkgPath, tn.Name()),
		Name:         name,
		PackagePath:  decl.PkgPath,
		ProvidedType: tn.Type(),
		Named:        namedOf(tn.Type()),
		Kind:         model.ComponentRepository,
		Scope:        model.ScopeSingleton,
		Constructor: &model.Constructor{
			PackagePath:    decl.PkgPath,
			FuncName:       "New" + tn.Name() + "Impl",
			ReturnType:     tn.Type(),
			RepositoryImpl: true,
		},
		Repository: &model.RepositoryInfo{},
		Conditions: extractConditions(decl),
		Position:   decl.Pos,
	})
}

// generatedRepository reports whether a declaration is an
// @Repository(generate=true) interface.
func generatedRepository(decl *Declaration) bool {
	if decl.Target != annotation.TargetInterface || !decl.Has("Repository") {
		return false
	}
	ann, _ := decl.Find("Repository")
	v, ok := ann.Arg("generate")
	if !ok {
		return false
	}
	b, ok := v.(annotation.BoolValue)
	return ok && b.Val
}

// discoverRepositories attaches @Query/@Exec methods to their repository
// interfaces, validating each method's signature and SQL parameters (§27).
func (a *analysis) discoverRepositories(scan *ScanResult, app *model.Application) {
	repos := make(map[string]*model.Component)
	for _, c := range app.Components {
		if c.Kind == model.ComponentRepository && c.Repository != nil {
			repos[string(c.ID)] = c
		}
	}
	if len(repos) == 0 {
		return
	}

	for _, decl := range scan.Declarations {
		if decl.Target != annotation.TargetMethod || decl.Recv == nil || decl.Func == nil {
			continue
		}
		comp := repos[typeKey(decl.PkgPath, decl.Recv.Name())]
		if comp == nil {
			continue
		}
		method, ok := a.repositoryMethod(decl)
		if ok {
			comp.Repository.Methods = append(comp.Repository.Methods, method)
		}
	}
}

// repositoryMethod builds and validates a single repository method.
func (a *analysis) repositoryMethod(decl *Declaration) (model.RepositoryMethod, bool) {
	sql, annName, ok := querySQL(decl)
	if !ok {
		a.diags = append(a.diags, diagErr(CodeMissingQuery, decl.Pos,
			"repository method %s must have an @Query, @Exec, @Batch, or @Call annotation", decl.Name))
		return model.RepositoryMethod{}, false
	}
	sig, ok := decl.Func.Type().(*types.Signature)
	if !ok {
		return model.RepositoryMethod{}, false
	}
	if params := sig.Params(); params.Len() == 0 || !isContextType(params.At(0).Type()) {
		a.diags = append(a.diags, diagErr(CodeInvalidQuerySignature, decl.Pos,
			"repository method %s must take context.Context as its first parameter", decl.Name))
		return model.RepositoryMethod{}, false
	}

	kind := resolveQueryKind(annName, sig)

	var batch *model.BatchInfo
	if kind == model.QueryBatch {
		b, reason := batchInfo(sig)
		if reason != "" {
			a.diags = append(a.diags, diagErr(CodeInvalidQuerySignature, decl.Pos,
				"repository method %s: %s", decl.Name, reason))
			return model.RepositoryMethod{}, false
		}
		batch = b
	}

	shape, reason := classifyReturn(sig, kind)
	if reason != "" {
		a.diags = append(a.diags, diagErr(CodeInvalidQuerySignature, decl.Pos,
			"repository method %s: %s", decl.Name, reason))
		return model.RepositoryMethod{}, false
	}

	a.validateParams(decl, sig, sql)

	return model.RepositoryMethod{
		Name:      decl.Name,
		RawSQL:    sql,
		Kind:      kind,
		Return:    shape,
		Batch:     batch,
		Signature: sig,
	}, true
}

// querySQL extracts the SQL and the annotation name from an @Query, @Exec,
// @Batch, or @Call annotation.
func querySQL(decl *Declaration) (string, string, bool) {
	for _, name := range []string{"Query", "Exec", "Batch", "Call"} {
		if ann, ok := decl.Find(name); ok {
			if v, ok := ann.Positional(); ok {
				if s, ok := annotation.AsString(v); ok {
					return s, name, true
				}
			}
		}
	}
	return "", "", false
}

// resolveQueryKind maps an annotation name to a query kind. @Call resolves to a
// read when it returns a value and to an exec when it returns only error, so a
// procedure that yields a result set is scanned like a query.
func resolveQueryKind(annName string, sig *types.Signature) model.QueryKind {
	switch annName {
	case "Exec":
		return model.QueryExec
	case "Batch":
		return model.QueryBatch
	case "Call":
		if sig.Results().Len() >= 2 {
			return model.QueryRead
		}
		return model.QueryExec
	default: // Query
		return model.QueryRead
	}
}

// batchInfo finds the single slice parameter an @Batch method iterates, or a
// reason it is invalid (§27.3).
func batchInfo(sig *types.Signature) (*model.BatchInfo, string) {
	var found *model.BatchInfo
	for i := 1; i < sig.Params().Len(); i++ {
		p := sig.Params().At(i)
		if slice, ok := p.Type().(*types.Slice); ok {
			if found != nil {
				return nil, "an @Batch method must have exactly one slice parameter to iterate"
			}
			found = &model.BatchInfo{ParamIndex: i, ParamName: p.Name(), Elem: slice.Elem()}
		}
	}
	if found == nil {
		return nil, "an @Batch method must have a slice parameter to iterate"
	}
	return found, ""
}

// classifyReturn determines a method's return shape or a reason it is invalid
// (§27.6).
func classifyReturn(sig *types.Signature, kind model.QueryKind) (model.ReturnShape, string) {
	results := sig.Results()
	if results.Len() == 0 || !isErrorType(results.At(results.Len()-1).Type()) {
		return model.ReturnShape{}, "the last result must be error"
	}

	if kind == model.QueryExec || kind == model.QueryBatch {
		noun := "@Exec"
		if kind == model.QueryBatch {
			noun = "@Batch"
		}
		switch results.Len() {
		case 1:
			return model.ReturnShape{}, "" // error only
		case 2:
			if isInteger(results.At(0).Type()) {
				return model.ReturnShape{RowsAffected: true}, ""
			}
			return model.ReturnShape{}, "an " + noun + " may return only error or (int64, error)"
		default:
			return model.ReturnShape{}, "an " + noun + " may return only error or (int64, error)"
		}
	}

	// @Query must return (value, error).
	if results.Len() != 2 {
		return model.ReturnShape{}, "an @Query must return (T, error)"
	}
	ret := results.At(0).Type()
	shape := model.ReturnShape{}
	if slice, ok := ret.(*types.Slice); ok {
		shape.Multi = true
		ret = slice.Elem()
	}
	if ptr, ok := ret.(*types.Pointer); ok {
		shape.Pointer = true
		ret = ptr.Elem()
	}
	shape.Elem = ret
	if _, ok := ret.Underlying().(*types.Struct); ok {
		shape.Scalar = false
	} else if isScalar(ret) {
		shape.Scalar = true
	} else {
		return model.ReturnShape{}, "@Query result must be a struct or scalar type"
	}
	return shape, ""
}

// validateParams checks that every named SQL parameter maps to a method
// argument (§27.10).
func (a *analysis) validateParams(decl *Declaration, sig *types.Signature, sql string) {
	names := map[string]bool{}
	for i := 0; i < sig.Params().Len(); i++ {
		if n := sig.Params().At(i).Name(); n != "" {
			names[n] = true
		}
	}
	compiled := sqlgen.Compile(sql, sqlgen.Postgres)
	for _, ref := range compiled.Params {
		base := ref
		if i := indexByte(ref, '.'); i >= 0 {
			base = ref[:i]
		}
		if !names[base] {
			a.diags = append(a.diags, diagErr(CodeUnknownQueryParam, decl.Pos,
				"repository method %s references :%s, but there is no parameter %q", decl.Name, ref, base))
		}
	}
}

// isScalar reports whether t is a basic scalar suitable for a single-column
// scan.
func isScalar(t types.Type) bool {
	b, ok := t.Underlying().(*types.Basic)
	if !ok {
		return false
	}
	return b.Info()&(types.IsBoolean|types.IsInteger|types.IsFloat|types.IsString) != 0
}

// isInteger reports whether t is an integer type.
func isInteger(t types.Type) bool {
	b, ok := t.Underlying().(*types.Basic)
	return ok && b.Info()&types.IsInteger != 0
}

// indexByte returns the index of the first b in s, or -1.
func indexByte(s string, b byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == b {
			return i
		}
	}
	return -1
}
