package compiler

import (
	"go/types"

	"github.com/zombocoder/goboot/annotation"
	"github.com/zombocoder/goboot/model"
)

// Controller-advice diagnostic codes (GOBHTTP family, §39.4).
const (
	// CodeInvalidExceptionHandler is an @ExceptionHandler with an unsupported
	// signature.
	CodeInvalidExceptionHandler = "GOBHTTP005"
	// CodeOrphanExceptionHandler is an @ExceptionHandler on a type that is not a
	// @ControllerAdvice.
	CodeOrphanExceptionHandler = "GOBHTTP006"
)

// discoverExceptionHandlers attaches @ExceptionHandler methods to their
// @ControllerAdvice components, validating each signature (§20).
func (a *analysis) discoverExceptionHandlers(scan *ScanResult, app *model.Application) {
	advice := map[string]*model.Component{}
	for _, c := range app.Components {
		if c.Kind == model.ComponentAdvice && c.Named != nil {
			advice[typeKey(c.PackagePath, c.Named.Obj().Name())] = c
		}
	}

	for _, decl := range scan.Declarations {
		if decl.Target != annotation.TargetMethod || decl.Recv == nil || decl.Func == nil {
			continue
		}
		if !decl.Has("ExceptionHandler") {
			continue
		}
		comp := advice[typeKey(decl.PkgPath, decl.Recv.Name())]
		if comp == nil {
			a.diags = append(a.diags, diagErr(CodeOrphanExceptionHandler, decl.Pos,
				"@ExceptionHandler method %s is not on a @ControllerAdvice type", decl.Name))
			continue
		}
		if h, ok := a.exceptionHandler(decl); ok {
			comp.ExceptionHandlers = append(comp.ExceptionHandlers, h)
		}
	}
}

// exceptionHandler validates an @ExceptionHandler method and builds its model
// (§20.2). Supported forms mirror HTTP handlers: the method takes
// (context.Context, err) and returns (response, error) or (error).
func (a *analysis) exceptionHandler(decl *Declaration) (model.ExceptionHandler, bool) {
	sig, ok := decl.Func.Type().(*types.Signature)
	if !ok {
		return model.ExceptionHandler{}, false
	}
	params := sig.Params()
	if params.Len() != 2 || !isContextType(params.At(0).Type()) {
		a.diags = append(a.diags, diagErr(CodeInvalidExceptionHandler, decl.Pos,
			"@ExceptionHandler %s must take (context.Context, err) where err is the caught error type", decl.Name))
		return model.ExceptionHandler{}, false
	}
	errType := params.At(1).Type()
	if !implementsError(errType) {
		a.diags = append(a.diags, diagErr(CodeInvalidExceptionHandler, decl.Pos,
			"@ExceptionHandler %s second parameter %s does not implement error", decl.Name, typeString(errType)))
		return model.ExceptionHandler{}, false
	}

	results := sig.Results()
	if results.Len() == 0 || !isErrorType(results.At(results.Len()-1).Type()) {
		a.diags = append(a.diags, diagErr(CodeInvalidExceptionHandler, decl.Pos,
			"@ExceptionHandler %s must return error as its last result", decl.Name))
		return model.ExceptionHandler{}, false
	}
	var respType types.Type
	switch results.Len() {
	case 1: // (error) — transform form; the delegate renders the returned error.
	case 2: // (response, error) — response form.
		respType = results.At(0).Type()
	default:
		a.diags = append(a.diags, diagErr(CodeInvalidExceptionHandler, decl.Pos,
			"@ExceptionHandler %s must return (response, error) or (error)", decl.Name))
		return model.ExceptionHandler{}, false
	}

	return model.ExceptionHandler{
		MethodName:    decl.Name,
		ErrType:       errType,
		CatchAll:      isErrorType(errType),
		ResponseType:  respType,
		SuccessStatus: exceptionStatus(decl),
		Position:      decl.Pos,
	}, true
}

// exceptionStatus reads @ResponseStatus on a handler, defaulting to 500.
func exceptionStatus(decl *Declaration) int {
	if ann, ok := decl.Find("ResponseStatus"); ok {
		if pv, ok := ann.Positional(); ok {
			if iv, ok := pv.(annotation.IntValue); ok {
				return int(iv.Val)
			}
		}
	}
	return 500
}

// implementsError reports whether t implements the error interface (or is the
// error interface itself), decided by go/types rather than by name.
func implementsError(t types.Type) bool {
	errIface, _ := types.Universe.Lookup("error").Type().Underlying().(*types.Interface)
	return errIface != nil && types.Implements(t, errIface)
}
