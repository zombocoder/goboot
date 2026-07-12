package di

import (
	"fmt"
	"go/types"
	"strconv"
	"strings"

	"github.com/zombocoder/goboot/model"
)

// renderProxies emits the service proxy types, their constructors, and their
// interface method implementations (§24.4). Intercepted methods are wrapped with
// tracing, metrics, and transaction interceptors in a fixed order (§25);
// non-intercepted interface methods delegate straight to the target.
func renderProxies(app *model.Application, byID map[model.ComponentID]*binding, im *imports) string {
	rt := func(sym string) string { return im.qualify(runtimePath, "runtime", sym) }

	var b strings.Builder
	for _, c := range app.Components {
		if c.Kind != model.ComponentProxy {
			continue
		}
		target := app.ComponentByID(c.ProxyTarget)
		if target == nil {
			continue
		}
		iface, ok := c.Interface.Underlying().(*types.Interface)
		if !ok {
			continue
		}
		b.WriteString(renderProxyType(c, target, iface, im, rt))
		b.WriteString("\n")
	}
	return b.String()
}

// renderProxyType emits one proxy: its struct, constructor, and methods.
func renderProxyType(proxy, target *model.Component, iface *types.Interface, im *imports, rt func(string) string) string {
	proxyName := proxy.Name
	targetType := renderType(target.ProvidedType, im)
	targetTypeName := target.Named.Obj().Name()

	intercepted := make(map[string]model.InterceptedMethod, len(proxy.Intercepted))
	usesCache := false
	for _, m := range proxy.Intercepted {
		intercepted[m.Name] = m
		if m.Cacheable != nil || m.CacheEvict != nil {
			usesCache = true
		}
	}

	var b strings.Builder
	// Struct.
	fmt.Fprintf(&b, "// %s is the generated interception proxy for %s.\n", proxyName, targetTypeName)
	fmt.Fprintf(&b, "type %s struct {\n", proxyName)
	fmt.Fprintf(&b, "\ttarget      %s\n", targetType)
	fmt.Fprintf(&b, "\ttransaction %s\n", rt("TransactionManager"))
	fmt.Fprintf(&b, "\ttracer      %s\n", rt("Tracer"))
	fmt.Fprintf(&b, "\tmetrics     %s\n", rt("MethodMetrics"))
	fmt.Fprintf(&b, "\tauthorizer  %s\n", rt("Authorizer"))
	fmt.Fprintf(&b, "\tlogger      %s\n", rt("MethodLogger"))
	fmt.Fprintf(&b, "\taudit       %s\n", rt("AuditSink"))
	fmt.Fprintf(&b, "\tbreakers     %s\n", rt("CircuitBreakerProvider"))
	fmt.Fprintf(&b, "\trateLimiters %s\n", rt("RateLimiterProvider"))
	fmt.Fprintf(&b, "\tbulkheads    %s\n", rt("BulkheadProvider"))
	if usesCache {
		fmt.Fprintf(&b, "\tcache        %s\n", rt("Cache"))
	}
	b.WriteString("}\n\n")

	// Constructor.
	ctorFields := "target: target, transaction: deps.Transactions, tracer: deps.Tracer, metrics: deps.Metrics, authorizer: deps.Authorizer, logger: deps.Logger, audit: deps.Audit, breakers: deps.Breakers, rateLimiters: deps.RateLimiters, bulkheads: deps.Bulkheads"
	if usesCache {
		ctorFields += ", cache: deps.Cache"
	}
	fmt.Fprintf(&b, "// New%s builds the %s.\n", proxyName, proxyName)
	fmt.Fprintf(&b, "func New%s(target %s, deps %s) *%s {\n", proxyName, targetType, rt("ProxyDependencies"), proxyName)
	fmt.Fprintf(&b, "\treturn &%s{%s}\n", proxyName, ctorFields)
	b.WriteString("}\n\n")

	// Methods, in the interface's (name-sorted) order for deterministic output.
	for i := 0; i < iface.NumMethods(); i++ {
		method := iface.Method(i)
		sig := method.Type().(*types.Signature)
		if m, ok := intercepted[method.Name()]; ok {
			b.WriteString(renderInterceptedMethod(proxyName, targetTypeName, method.Name(), sig, m, im, rt))
		} else {
			b.WriteString(renderDelegateMethod(proxyName, method.Name(), sig, im))
		}
		b.WriteString("\n")
	}
	return b.String()
}

// renderDelegateMethod emits a method that forwards directly to the target.
func renderDelegateMethod(proxyName, name string, sig *types.Signature, im *imports) string {
	params, argNames := renderParamList(sig, im)
	results := renderResultList(sig, im, false)
	call := fmt.Sprintf("p.target.%s(%s)", name, callArgs(argNames, sig.Variadic()))
	return fmt.Sprintf("func (p *%s) %s(%s) %s {\n\treturn %s\n}\n", proxyName, name, params, results, call)
}

// renderInterceptedMethod emits a method wrapped with the requested
// interceptors. It uses named results so the deferred trace span can observe the
// returned error (§35.1).
func renderInterceptedMethod(proxyName, targetTypeName, name string, sig *types.Signature, m model.InterceptedMethod, im *imports, rt func(string) string) string {
	params, argNames := renderParamList(sig, im)
	results, valueNames := renderNamedResults(sig, im)
	opName := targetTypeName + "." + name
	ctxType := im.qualify("context", "context", "Context")

	ctxVar := "ctx0"
	if len(argNames) > 0 {
		ctxVar = argNames[0]
	}
	restArgs := callArgs(argNames[minInt(1, len(argNames)):], sig.Variadic())

	var b strings.Builder
	fmt.Fprintf(&b, "func (p *%s) %s(%s) %s {\n", proxyName, name, params, results)

	// Caching short-circuits before every other interceptor: on a hit the cached
	// value is returned without invoking the target (§32).
	if m.Cacheable != nil {
		fmt.Fprintf(&b, "\tcacheKey := %s\n", renderCacheKeyExpr(m.Cacheable.Parts, argNames, im))
		fmt.Fprintf(&b, "\tif data, ok, _ := p.cache.Get(%s, cacheKey); ok {\n", ctxVar)
		fmt.Fprintf(&b, "\t\tif %s(data, &%s) == nil {\n", im.qualify("encoding/json", "json", "Unmarshal"), valueNames[0])
		b.WriteString("\t\t\treturn\n\t\t}\n\t}\n")
	}
	if m.CacheEvict != nil {
		fmt.Fprintf(&b, "\tcacheEvictKey := %s\n", renderCacheKeyExpr(m.CacheEvict.Parts, argNames, im))
	}

	// Timeout is the outermost interceptor: it bounds the whole call (§25).
	if m.Timeout > 0 {
		fmt.Fprintf(&b, "\t%s, cancel := %s(%s, %d)\n", ctxVar, im.qualify("context", "context", "WithTimeout"), ctxVar, int64(m.Timeout))
		b.WriteString("\tdefer cancel()\n")
	}

	// Tracing spans the whole call and observes the returned error (§35.1).
	if m.Traced {
		traceName := orDefault(m.TraceName, opName)
		fmt.Fprintf(&b, "\t%s, span := p.tracer.Begin(%s, %s)\n", ctxVar, ctxVar, strconv.Quote(traceName))
		b.WriteString("\tdefer func() { span.End(err) }()\n")
	}

	// Logging brackets the call, observing the returned error (§35.3).
	if m.Logged {
		fmt.Fprintf(&b, "\tlogDone := p.logger.Log(%s, %s, %s)\n", ctxVar, strconv.Quote(opName), strconv.Quote(orDefault(m.LogLevel, "info")))
		b.WriteString("\tdefer func() { logDone(err) }()\n")
	}

	// Audit records the outcome of the action after it completes (§35.4).
	if m.Audit != nil {
		fmt.Fprintf(&b, "\tdefer func() { p.audit.Record(%s, %s, err) }()\n", ctxVar, renderAuditEvent(opName, m.Audit, rt))
	}

	// Resilience gates (bulkhead → circuit breaker → rate limit) wrap the whole
	// guarded unit; when present, the authorize check and core run inside a
	// closure so a rejected gate short-circuits before the target (§36.3–§36.5).
	if m.CircuitBreaker != nil || m.RateLimit != nil || m.Bulkhead != nil {
		fmt.Fprintf(&b, "\terr = %s\n", renderGatedCore(name, ctxVar, ctxType, restArgs, valueNames, m, rt))
	} else if m.Authorize != nil {
		// Authorization gates the call; a rejection skips the core and is recorded
		// by metrics like any other failure (§25, §34).
		core := renderCore(name, ctxVar, ctxType, restArgs, valueNames, m, rt)
		fmt.Fprintf(&b, "\tif err = p.authorizer.Authorize(%s, %s); err == nil {\n", ctxVar, renderAuthRequest(m.Authorize, rt))
		b.WriteString(core)
		b.WriteString("\t}\n")
	} else {
		b.WriteString(renderCore(name, ctxVar, ctxType, restArgs, valueNames, m, rt))
	}

	// Metrics record the final outcome (§35.2).
	if m.Timed {
		metricName := orDefault(m.MetricName, opName)
		fmt.Fprintf(&b, "\tif err != nil {\n\t\tp.metrics.RecordFailure(%s)\n\t\treturn\n\t}\n", strconv.Quote(metricName))
		fmt.Fprintf(&b, "\tp.metrics.RecordSuccess(%s)\n", strconv.Quote(metricName))
	}

	// On success, store the result (@Cacheable) and invalidate (@CacheEvict).
	if m.Cacheable != nil {
		b.WriteString("\tif err == nil {\n")
		fmt.Fprintf(&b, "\t\tif data, mErr := %s(%s); mErr == nil {\n", im.qualify("encoding/json", "json", "Marshal"), valueNames[0])
		fmt.Fprintf(&b, "\t\t\t_ = p.cache.Set(%s, cacheKey, data, %d)\n", ctxVar, int64(m.Cacheable.TTL))
		b.WriteString("\t\t}\n\t}\n")
	}
	if m.CacheEvict != nil {
		fmt.Fprintf(&b, "\tif err == nil {\n\t\t_ = p.cache.Delete(%s, cacheEvictKey)\n\t}\n", ctxVar)
	}

	b.WriteString("\treturn\n}\n")
	return b.String()
}

// renderCacheKeyExpr builds the Go expression for a resolved cache key: literals
// and fmt.Sprint(arg) segments concatenated. Argument references use the proxy
// method's own parameter names, so it is independent of the annotated method's.
func renderCacheKeyExpr(parts []model.CacheKeyPart, argNames []string, im *imports) string {
	segs := make([]string, 0, len(parts))
	for _, p := range parts {
		if p.IsArg {
			if p.ArgIndex >= 0 && p.ArgIndex < len(argNames) {
				segs = append(segs, fmt.Sprintf("%s(%s)", im.qualify("fmt", "fmt", "Sprint"), argNames[p.ArgIndex]))
			}
			continue
		}
		segs = append(segs, strconv.Quote(p.Literal))
	}
	if len(segs) == 0 {
		return `""`
	}
	return strings.Join(segs, " + ")
}

// renderCore emits the statement that assigns err from the (optionally retried,
// optionally transactional) target invocation.
func renderCore(method, ctxVar, ctxType, restArgs string, valueNames []string, m model.InterceptedMethod, rt func(string) string) string {
	// The innermost work: a closure that invokes the target with its context,
	// assigning the named results and returning the error.
	targetBody := func(ctxParam string) string {
		call := fmt.Sprintf("p.target.%s(%s)", method, joinCtxArgs(ctxParam, restArgs))
		if len(valueNames) == 0 {
			return fmt.Sprintf("return %s", call)
		}
		return fmt.Sprintf("var e error\n\t\t%s, e = %s\n\t\treturn e", strings.Join(valueNames, ", "), call)
	}

	// transactional produces an error-valued expression that runs the target
	// within a transaction; otherwise the target closure is used directly.
	txExpr := func(ctxArg string) string {
		opts := renderTxOptions(m.Tx, rt)
		return fmt.Sprintf("p.transaction.WithinTransaction(%s, %s, func(ctx %s) error {\n\t\t%s\n\t})",
			ctxArg, opts, ctxType, indentClosureBody(targetBody("ctx")))
	}

	switch {
	case m.Retry != nil:
		// Retry wraps the transaction (or the target) per attempt.
		inner := func(ctxParam string) string {
			if m.Transactional {
				return "return " + txExpr(ctxParam)
			}
			return targetBody(ctxParam)
		}
		return fmt.Sprintf("\terr = %s(%s, %s, func(ctx %s) error {\n\t\t%s\n\t})\n",
			rt("Retry"), ctxVar, renderRetryPolicy(m.Retry, rt), ctxType, indentClosureBody(inner("ctx")))
	case m.Transactional:
		return fmt.Sprintf("\terr = %s\n", txExpr(ctxVar))
	default:
		call := fmt.Sprintf("p.target.%s(%s)", method, joinCtxArgs(ctxVar, restArgs))
		if len(valueNames) == 0 {
			return fmt.Sprintf("\terr = %s\n", call)
		}
		return fmt.Sprintf("\t%s, err = %s\n", strings.Join(valueNames, ", "), call)
	}
}

// renderGatedCore renders the resilience-gated invocation as a single
// error-valued expression: bulkhead(circuitBreaker(rateLimit(guarded))), where
// guarded is a closure running the authorize check, retry, transaction, and
// target (§36.3–§36.5). Formatting is normalized by go/format afterward.
func renderGatedCore(method, ctxVar, ctxType, restArgs string, valueNames []string, m model.InterceptedMethod, rt func(string) string) string {
	guarded := "func(ctx " + ctxType + ") error {\n" + renderGuardedBody("ctx", method, ctxType, restArgs, valueNames, m, rt) + "\n}"

	// Gate wrappers, inner (nearest the target) to outer.
	type gate struct{ expr string }
	var gates []gate
	if m.RateLimit != nil {
		gates = append(gates, gate{"p.rateLimiters.RateLimiter(" + renderRateLimitSpec(m.RateLimit, rt) + ")"})
	}
	if m.CircuitBreaker != nil {
		gates = append(gates, gate{"p.breakers.CircuitBreaker(" + renderCircuitBreakerSpec(m.CircuitBreaker, rt) + ")"})
	}
	if m.Bulkhead != nil {
		gates = append(gates, gate{"p.bulkheads.Bulkhead(" + renderBulkheadSpec(m.Bulkhead, rt) + ")"})
	}

	expr := ""
	for i, g := range gates {
		ctxArg := "ctx"
		if i == len(gates)-1 {
			ctxArg = ctxVar
		}
		var inner string
		if i == 0 {
			inner = guarded
		} else {
			inner = "func(ctx " + ctxType + ") error { return " + expr + " }"
		}
		expr = g.expr + ".Execute(" + ctxArg + ", " + inner + ")"
	}
	return expr
}

// renderGuardedBody renders the closure body (returning error) that runs the
// authorize check and the target invocation for the gated path.
func renderGuardedBody(ctxParam, method, ctxType, restArgs string, valueNames []string, m model.InterceptedMethod, rt func(string) string) string {
	targetReturn := func(ctxP string) string {
		call := fmt.Sprintf("p.target.%s(%s)", method, joinCtxArgs(ctxP, restArgs))
		if len(valueNames) == 0 {
			return "return " + call
		}
		return fmt.Sprintf("var e error\n%s, e = %s\nreturn e", strings.Join(valueNames, ", "), call)
	}
	txReturn := func(ctxArg string) string {
		return fmt.Sprintf("return p.transaction.WithinTransaction(%s, %s, func(ctx %s) error {\n%s\n})",
			ctxArg, renderTxOptions(m.Tx, rt), ctxType, targetReturn("ctx"))
	}

	var work string
	switch {
	case m.Retry != nil:
		inner := targetReturn("ctx")
		if m.Transactional {
			inner = txReturn("ctx")
		}
		work = fmt.Sprintf("return %s(%s, %s, func(ctx %s) error {\n%s\n})",
			rt("Retry"), ctxParam, renderRetryPolicy(m.Retry, rt), ctxType, inner)
	case m.Transactional:
		work = txReturn(ctxParam)
	default:
		work = targetReturn(ctxParam)
	}

	if m.Authorize != nil {
		return fmt.Sprintf("if err := p.authorizer.Authorize(%s, %s); err != nil {\nreturn err\n}\n%s",
			ctxParam, renderAuthRequest(m.Authorize, rt), work)
	}
	return work
}

// renderCircuitBreakerSpec renders a runtime.CircuitBreakerSpec literal (§36.3).
func renderCircuitBreakerSpec(s *model.CircuitBreakerSpec, rt func(string) string) string {
	fields := []string{"Name: " + strconv.Quote(s.Name)}
	if s.FailureThreshold != 0 {
		fields = append(fields, fmt.Sprintf("FailureThreshold: %d", s.FailureThreshold))
	}
	if s.ResetTimeout != 0 {
		fields = append(fields, fmt.Sprintf("ResetTimeout: %d", int64(s.ResetTimeout)))
	}
	if s.HalfOpenMax != 0 {
		fields = append(fields, fmt.Sprintf("HalfOpenMax: %d", s.HalfOpenMax))
	}
	return rt("CircuitBreakerSpec") + "{" + strings.Join(fields, ", ") + "}"
}

// renderRateLimitSpec renders a runtime.RateLimitSpec literal (§36.4).
func renderRateLimitSpec(s *model.RateLimitSpec, rt func(string) string) string {
	fields := []string{"Name: " + strconv.Quote(s.Name)}
	if s.Limit != 0 {
		fields = append(fields, fmt.Sprintf("Limit: %d", s.Limit))
	}
	if s.Period != 0 {
		fields = append(fields, fmt.Sprintf("Period: %d", int64(s.Period)))
	}
	if s.Burst != 0 {
		fields = append(fields, fmt.Sprintf("Burst: %d", s.Burst))
	}
	return rt("RateLimitSpec") + "{" + strings.Join(fields, ", ") + "}"
}

// renderBulkheadSpec renders a runtime.BulkheadSpec literal (§36.5).
func renderBulkheadSpec(s *model.BulkheadSpec, rt func(string) string) string {
	fields := []string{"Name: " + strconv.Quote(s.Name)}
	if s.MaxConcurrent != 0 {
		fields = append(fields, fmt.Sprintf("MaxConcurrent: %d", s.MaxConcurrent))
	}
	if s.MaxWait != 0 {
		fields = append(fields, fmt.Sprintf("MaxWait: %d", int64(s.MaxWait)))
	}
	return rt("BulkheadSpec") + "{" + strings.Join(fields, ", ") + "}"
}

// renderAuthRequest renders a runtime.AuthorizationRequest literal (§34.2).
func renderAuthRequest(a *model.AuthorizeSpec, rt func(string) string) string {
	var fields []string
	if len(a.Roles) > 0 {
		fields = append(fields, "Roles: "+stringSliceLit(a.Roles))
	}
	if len(a.Permissions) > 0 {
		fields = append(fields, "Permissions: "+stringSliceLit(a.Permissions))
	}
	if a.Mode == "all" {
		fields = append(fields, "Mode: "+rt("AuthorizationModeAll"))
	}
	return rt("AuthorizationRequest") + "{" + strings.Join(fields, ", ") + "}"
}

// renderAuditEvent renders a runtime.AuditEvent literal for @Audit (§35.4).
func renderAuditEvent(method string, a *model.AuditSpec, rt func(string) string) string {
	fields := []string{"Method: " + strconv.Quote(method)}
	if a.Action != "" {
		fields = append(fields, "Action: "+strconv.Quote(a.Action))
	}
	if a.Resource != "" {
		fields = append(fields, "Resource: "+strconv.Quote(a.Resource))
	}
	return rt("AuditEvent") + "{" + strings.Join(fields, ", ") + "}"
}

// renderRetryPolicy renders a runtime.RetryPolicy literal, omitting zero fields.
func renderRetryPolicy(p *model.RetryPolicy, rt func(string) string) string {
	var fields []string
	if p.MaxAttempts != 0 {
		fields = append(fields, fmt.Sprintf("MaxAttempts: %d", p.MaxAttempts))
	}
	if p.Delay != 0 {
		fields = append(fields, fmt.Sprintf("Delay: %d", int64(p.Delay)))
	}
	if p.Multiplier != 0 {
		fields = append(fields, "Multiplier: "+strconv.FormatFloat(p.Multiplier, 'g', -1, 64))
	}
	if p.MaxDelay != 0 {
		fields = append(fields, fmt.Sprintf("MaxDelay: %d", int64(p.MaxDelay)))
	}
	return rt("RetryPolicy") + "{" + strings.Join(fields, ", ") + "}"
}

// joinCtxArgs joins the context argument with the remaining arguments.
func joinCtxArgs(ctxArg, restArgs string) string {
	if restArgs == "" {
		return ctxArg
	}
	return ctxArg + ", " + restArgs
}

// indentClosureBody re-indents a multi-line closure body by one extra tab so the
// generated source is legible before gofmt normalizes it.
func indentClosureBody(body string) string {
	return strings.ReplaceAll(body, "\n\t\t", "\n\t\t\t")
}

// orDefault returns s, or def when s is empty.
func orDefault(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

// minInt returns the smaller of a and b.
func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// renderTxOptions renders a runtime.TransactionOptions literal, omitting default
// fields for readability.
func renderTxOptions(tx model.TxOptions, rt func(string) string) string {
	var fields []string
	if tx.ReadOnly {
		fields = append(fields, "ReadOnly: true")
	}
	if iso := isolationConst(tx.Isolation); iso != "" {
		fields = append(fields, "Isolation: "+rt(iso))
	}
	if prop := propagationConst(tx.Propagation); prop != "" {
		fields = append(fields, "Propagation: "+rt(prop))
	}
	if tx.Timeout != 0 {
		fields = append(fields, fmt.Sprintf("Timeout: %d", int64(tx.Timeout)))
	}
	return rt("TransactionOptions") + "{" + strings.Join(fields, ", ") + "}"
}

func isolationConst(s string) string {
	switch s {
	case "read_committed":
		return "IsolationReadCommitted"
	case "repeatable_read":
		return "IsolationRepeatableRead"
	case "serializable":
		return "IsolationSerializable"
	default:
		return ""
	}
}

func propagationConst(s string) string {
	switch s {
	case "requires_new":
		return "PropagationRequiresNew"
	case "supports":
		return "PropagationSupports"
	case "not_supported":
		return "PropagationNotSupported"
	default:
		return ""
	}
}

// renderParamList renders a signature's parameters as "a0 T0, a1 T1" and returns
// the argument names.
func renderParamList(sig *types.Signature, im *imports) (string, []string) {
	n := sig.Params().Len()
	parts := make([]string, n)
	names := make([]string, n)
	for i := 0; i < n; i++ {
		p := sig.Params().At(i)
		name := "a" + strconv.Itoa(i)
		names[i] = name
		if sig.Variadic() && i == n-1 {
			elem := p.Type().(*types.Slice).Elem()
			parts[i] = name + " ..." + renderType(elem, im)
		} else {
			parts[i] = name + " " + renderType(p.Type(), im)
		}
	}
	return strings.Join(parts, ", "), names
}

// callArgs renders an argument list, expanding a variadic final argument.
func callArgs(names []string, variadic bool) string {
	args := append([]string(nil), names...)
	if variadic && len(args) > 0 {
		args[len(args)-1] += "..."
	}
	return strings.Join(args, ", ")
}

// renderResultList renders a signature's results as an unnamed list, e.g.
// "(string, error)".
func renderResultList(sig *types.Signature, im *imports, _ bool) string {
	n := sig.Results().Len()
	parts := make([]string, n)
	for i := 0; i < n; i++ {
		parts[i] = renderType(sig.Results().At(i).Type(), im)
	}
	return "(" + strings.Join(parts, ", ") + ")"
}

// renderNamedResults renders results with names — value results r0, r1, ... and
// the final error as "err" — returning the declaration and the value names.
func renderNamedResults(sig *types.Signature, im *imports) (string, []string) {
	n := sig.Results().Len()
	parts := make([]string, n)
	var valueNames []string
	for i := 0; i < n; i++ {
		typ := renderType(sig.Results().At(i).Type(), im)
		if i == n-1 {
			parts[i] = "err " + typ // the last result is the error
		} else {
			name := "r" + strconv.Itoa(i)
			valueNames = append(valueNames, name)
			parts[i] = name + " " + typ
		}
	}
	return "(" + strings.Join(parts, ", ") + ")", valueNames
}
