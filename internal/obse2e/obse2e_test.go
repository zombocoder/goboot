// Package obse2e drives generated @Logged / @Audit proxies to confirm the
// logging and audit interceptors bracket the target call and observe its error.
// wiring.gen.go is produced by the goboot generator from the obsapp example.
package obse2e

import (
	"context"
	"testing"

	goruntime "github.com/zombocoder/goboot/runtime"
)

// recordingLogger captures each Log call and the error it completed with.
type recordingLogger struct {
	calls []logCall
}

type logCall struct {
	method string
	level  string
	err    error
	done   bool
}

func (l *recordingLogger) Log(_ context.Context, method, level string) func(error) {
	i := len(l.calls)
	l.calls = append(l.calls, logCall{method: method, level: level})
	return func(err error) {
		l.calls[i].err = err
		l.calls[i].done = true
	}
}

// recordingAudit captures each audit event.
type recordingAudit struct {
	events []goruntime.AuditEvent
	errs   []error
}

func (a *recordingAudit) Record(_ context.Context, ev goruntime.AuditEvent, err error) {
	a.events = append(a.events, ev)
	a.errs = append(a.errs, err)
}

func newComps(t *testing.T, log goruntime.MethodLogger, audit goruntime.AuditSink) *Components {
	t.Helper()
	deps := goruntime.DefaultProxyDependencies()
	if log != nil {
		deps.Logger = log
	}
	if audit != nil {
		deps.Audit = audit
	}
	comps, err := buildComponents(deps)
	if err != nil {
		t.Fatalf("buildComponents: %v", err)
	}
	return comps
}

func TestLoggedBracketsCall(t *testing.T) {
	log := &recordingLogger{}
	comps := newComps(t, log, nil)

	if err := comps.VaultServiceProxy.Store(context.Background(), "k"); err != nil {
		t.Fatalf("Store: %v", err)
	}
	if !comps.Vault.Stored() {
		t.Error("target Store should have run")
	}
	if len(log.calls) != 1 {
		t.Fatalf("expected 1 log call, got %d", len(log.calls))
	}
	c := log.calls[0]
	if c.method != "VaultService.Store" || c.level != "debug" {
		t.Errorf("log call = %+v, want method=VaultService.Store level=debug", c)
	}
	if !c.done {
		t.Error("log completion function should have been invoked")
	}
	if c.err != nil {
		t.Errorf("successful call logged err = %v, want nil", c.err)
	}
}

func TestLoggedDefaultLevel(t *testing.T) {
	log := &recordingLogger{}
	comps := newComps(t, log, nil)

	got, err := comps.VaultServiceProxy.Rotate(context.Background())
	if err != nil || got != "rotated" {
		t.Fatalf("Rotate = %q, %v", got, err)
	}
	if len(log.calls) != 1 || log.calls[0].level != "info" {
		t.Errorf("@Logged with no level should default to info, got %+v", log.calls)
	}
}

func TestLoggedObservesError(t *testing.T) {
	log := &recordingLogger{}
	comps := newComps(t, log, nil)
	comps.Vault.SetFail(true)

	if err := comps.VaultServiceProxy.Store(context.Background(), "k"); err == nil {
		t.Fatal("Store should fail")
	}
	if len(log.calls) != 1 || log.calls[0].err == nil {
		t.Errorf("failed call should log a non-nil error, got %+v", log.calls)
	}
}

func TestAuditRecordsOutcome(t *testing.T) {
	audit := &recordingAudit{}
	comps := newComps(t, nil, audit)

	if err := comps.VaultServiceProxy.Store(context.Background(), "k"); err != nil {
		t.Fatalf("Store: %v", err)
	}
	if len(audit.events) != 1 {
		t.Fatalf("expected 1 audit event, got %d", len(audit.events))
	}
	ev := audit.events[0]
	if ev.Method != "VaultService.Store" || ev.Action != "store" || ev.Resource != "secret" {
		t.Errorf("audit event = %+v", ev)
	}
	if audit.errs[0] != nil {
		t.Errorf("successful action audited with err = %v, want nil", audit.errs[0])
	}
}

func TestAuditRecordsFailure(t *testing.T) {
	audit := &recordingAudit{}
	comps := newComps(t, nil, audit)
	comps.Vault.SetFail(true)

	if err := comps.VaultServiceProxy.Store(context.Background(), "k"); err == nil {
		t.Fatal("Store should fail")
	}
	if len(audit.events) != 1 || audit.errs[0] == nil {
		t.Errorf("failed action should audit a non-nil error, got events=%+v errs=%+v", audit.events, audit.errs)
	}
	if comps.Vault.Stored() {
		t.Error("target must not have completed the store")
	}
}

func TestDefaultsAreNoop(t *testing.T) {
	// Default proxy dependencies use no-op logger/audit, so calls just succeed.
	comps, err := buildComponents(goruntime.DefaultProxyDependencies())
	if err != nil {
		t.Fatal(err)
	}
	if err := comps.VaultServiceProxy.Store(context.Background(), "k"); err != nil {
		t.Errorf("no-op deps should permit the call: %v", err)
	}
}
