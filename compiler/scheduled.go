package compiler

import (
	"go/types"
	"strings"
	"time"

	"github.com/zombocoder/goboot/annotation"
	"github.com/zombocoder/goboot/model"
)

// Scheduling diagnostic codes (GOBSCH family, §39.4).
const (
	// CodeInvalidScheduled is a @Scheduled method with an unsupported signature.
	CodeInvalidScheduled = "GOBSCH001"
	// CodeInvalidSchedule is a @Scheduled annotation without a valid rate.
	CodeInvalidSchedule = "GOBSCH002"
)

// discoverScheduled attaches @Scheduled methods to their owning components
// (§4.2 background workers). Each method's rate is resolved from fixedRate and
// timeUnit (or a duration string) and its signature is validated.
func (a *analysis) discoverScheduled(scan *ScanResult, app *model.Application) {
	byType := make(map[string]*model.Component, len(app.Components))
	for _, c := range app.Components {
		byType[string(c.ID)] = c
	}

	for _, decl := range scan.Declarations {
		if decl.Target != annotation.TargetMethod || decl.Recv == nil || decl.Func == nil {
			continue
		}
		ann, ok := decl.Find("Scheduled")
		if !ok {
			continue
		}
		comp := byType[typeKey(decl.PkgPath, decl.Recv.Name())]
		if comp == nil {
			continue
		}
		if method, ok := a.scheduledMethod(decl, ann); ok {
			comp.Scheduled = append(comp.Scheduled, method)
		}
	}
}

// scheduledMethod validates a @Scheduled method and resolves its schedule.
func (a *analysis) scheduledMethod(decl *Declaration, ann annotation.Annotation) (model.ScheduledMethod, bool) {
	sig, ok := decl.Func.Type().(*types.Signature)
	if !ok {
		return model.ScheduledMethod{}, false
	}
	takesContext, returnsError, reason := validateHookSignature(sig)
	if reason != "" {
		a.diags = append(a.diags, diagErr(CodeInvalidScheduled, decl.Pos,
			"scheduled method %s: %s", decl.Name, reason))
		return model.ScheduledMethod{}, false
	}

	interval, initialDelay, reason := resolveSchedule(ann)
	if reason != "" {
		a.diags = append(a.diags, diagErr(CodeInvalidSchedule, decl.Pos,
			"scheduled method %s: %s", decl.Name, reason))
		return model.ScheduledMethod{}, false
	}

	return model.ScheduledMethod{
		MethodName:   decl.Name,
		Interval:     interval,
		InitialDelay: initialDelay,
		TakesContext: takesContext,
		ReturnsError: returnsError,
	}, true
}

// validateHookSignature checks the func()/func() error/func(ctx)/func(ctx) error
// forms shared by lifecycle and scheduled methods (§30.2).
func validateHookSignature(sig *types.Signature) (takesContext, returnsError bool, reason string) {
	params := sig.Params()
	switch {
	case params.Len() == 0:
	case params.Len() == 1 && isContextType(params.At(0).Type()):
		takesContext = true
	default:
		return false, false, "must take no parameters or a single context.Context"
	}
	results := sig.Results()
	switch {
	case results.Len() == 0:
	case results.Len() == 1 && isErrorType(results.At(0).Type()):
		returnsError = true
	default:
		return false, false, "must return nothing or a single error"
	}
	return takesContext, returnsError, ""
}

// resolveSchedule computes the interval and initial delay from a @Scheduled
// annotation (§4.2). fixedRate (or fixedDelay) may be an integer scaled by
// timeUnit, or a Go duration string such as "2m". It returns a reason when no
// valid positive rate is given.
func resolveSchedule(ann annotation.Annotation) (interval, initialDelay time.Duration, reason string) {
	unit := timeUnitDuration(unitName(ann))

	rate, ok := durationArg(ann, "fixedRate", unit)
	if !ok {
		rate, ok = durationArg(ann, "fixedDelay", unit)
	}
	if !ok {
		return 0, 0, "requires fixedRate or fixedDelay"
	}
	if rate <= 0 {
		return 0, 0, "rate must be positive"
	}
	initialDelay, _ = durationArg(ann, "initialDelay", unit)
	return rate, initialDelay, ""
}

// unitName returns the timeUnit argument's name without a "TimeUnit." prefix.
func unitName(ann annotation.Annotation) string {
	s, ok := stringArgValue(ann, "timeUnit")
	if !ok {
		return ""
	}
	return strings.TrimPrefix(s, "TimeUnit.")
}

// timeUnitDuration maps a time-unit name to its duration, defaulting to
// milliseconds (matching common @Scheduled fixedRate semantics).
func timeUnitDuration(unit string) time.Duration {
	switch strings.ToUpper(unit) {
	case "NANOSECONDS":
		return time.Nanosecond
	case "MICROSECONDS":
		return time.Microsecond
	case "MILLISECONDS", "":
		return time.Millisecond
	case "SECONDS":
		return time.Second
	case "MINUTES":
		return time.Minute
	case "HOURS":
		return time.Hour
	case "DAYS":
		return 24 * time.Hour
	default:
		return time.Millisecond
	}
}

// durationArg reads a duration argument: an integer scaled by unit, or a Go
// duration string.
func durationArg(ann annotation.Annotation, name string, unit time.Duration) (time.Duration, bool) {
	v, ok := ann.Arg(name)
	if !ok {
		return 0, false
	}
	switch t := v.(type) {
	case annotation.IntValue:
		return time.Duration(t.Val) * unit, true
	case annotation.StringValue:
		if d, err := time.ParseDuration(t.Val); err == nil {
			return d, true
		}
	}
	return 0, false
}
