package di

import (
	"flag"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/zombocoder/goboot/compiler"
	"github.com/zombocoder/goboot/sqlgen"
)

var update = flag.Bool("update", false, "update golden files")

// analyzeDiapp loads and analyzes the multi-package example under the compiler
// package's testdata.
func analyzeDiapp(t *testing.T) *compiler.AnalysisResult {
	t.Helper()
	l := &compiler.Loader{Dir: filepath.Join("..", "..", "compiler")}
	scan, err := l.Load("./testdata/diapp/...")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	res := compiler.Analyze(scan)
	for _, d := range res.Diagnostics {
		if d.Severity == 2 { // SeverityError
			t.Fatalf("analysis error: %s", d.Error())
		}
	}
	return res
}

func generateDiapp(t *testing.T) string {
	t.Helper()
	res := analyzeDiapp(t)
	src, err := Generate(res.App, res.Graph, Options{Package: "wiring"})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	return src
}

func TestGenerateWiringGolden(t *testing.T) {
	src := generateDiapp(t)
	golden := filepath.Join("testdata", "golden", "diapp_wiring.gen.go")

	if *update {
		if err := os.MkdirAll(filepath.Dir(golden), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(golden, []byte(src), 0o644); err != nil {
			t.Fatal(err)
		}
		t.Logf("wrote golden %s", golden)
		return
	}

	want, err := os.ReadFile(golden)
	if err != nil {
		t.Fatalf("reading golden (run with -update to create): %v", err)
	}
	if src != string(want) {
		t.Errorf("generated output differs from golden.\n--- got ---\n%s", src)
	}
}

func TestGenerateWiringContent(t *testing.T) {
	src := generateDiapp(t)

	// Marker and package clause.
	if !strings.HasPrefix(src, GeneratedMarker) {
		t.Errorf("missing generated marker")
	}
	if !strings.Contains(src, "package wiring") {
		t.Errorf("missing package clause")
	}
	// Every component constructor should be called, and the HTTP handlers and
	// route registration should be emitted.
	for _, want := range []string{
		"repo.NewPostgresUserRepository()",
		"config.ProvideIDGenerator()",
		"service.NewUserService(",
		"controller.NewUserController(",
		"func buildComponents() (*Components, error)",
		"func RegisterRoutes(mux *http.ServeMux",
		`mux.HandleFunc("GET /api/v1/users/{id}"`,
		`mux.HandleFunc("POST /api/v1/users"`,
		"deps.Binder.Bind(ctx, r, &request)",
		"deps.ResponseWriter.Write(ctx, w, 200, response)",
	} {
		if !strings.Contains(src, want) {
			t.Errorf("generated output missing %q", want)
		}
	}
	// The repository must be constructed before the service that consumes it.
	if idx(src, "NewPostgresUserRepository") > idx(src, "NewUserService") {
		t.Errorf("repository should be constructed before service")
	}
}

func TestGenerateDeterministic(t *testing.T) {
	first := generateDiapp(t)
	for i := 0; i < 5; i++ {
		if got := generateDiapp(t); got != first {
			t.Fatalf("generation is not deterministic")
		}
	}
}

// TestGeneratedWiringCompiles writes the generated file into a temporary
// package inside the module and compiles it, satisfying the Milestone 3
// acceptance criterion that a multi-package example compiles (§48.3).
func TestGeneratedWiringCompiles(t *testing.T) {
	src := generateDiapp(t)

	moduleRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	dir, err := os.MkdirTemp(moduleRoot, "genwire")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	if err := os.WriteFile(filepath.Join(dir, "wiring.gen.go"), []byte(src), 0o644); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("go", "build", ".")
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("generated wiring did not compile: %v\n%s\n--- source ---\n%s", err, out, src)
	}
}

func TestGenerateRejectsCycle(t *testing.T) {
	l := &compiler.Loader{Dir: filepath.Join("..", "..", "compiler")}
	scan, err := l.Load("./testdata/cycle")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	res := compiler.Analyze(scan)
	if _, err := Generate(res.App, res.Graph, Options{Package: "wiring"}); err == nil {
		t.Fatal("expected an error generating wiring for a cyclic graph")
	}
}

func idx(s, sub string) int { return strings.Index(s, sub) }

// TestCfgE2EWiringUpToDate guards the committed config/lifecycle integration
// wiring against the generator, the same way TestE2EWiringUpToDate does for the
// HTTP example.
func TestCfgE2EWiringUpToDate(t *testing.T) {
	l := &compiler.Loader{Dir: filepath.Join("..", "..", "compiler")}
	scan, err := l.Load("./testdata/cfgapp")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	res := compiler.Analyze(scan)
	for _, d := range res.Diagnostics {
		if d.Severity == 2 {
			t.Fatalf("analysis error: %s", d.Error())
		}
	}
	src, err := Generate(res.App, res.Graph, Options{Package: "cfge2e"})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	// Assert the config and lifecycle sections are present.
	for _, want := range []string{
		"func buildComponents(configSource config.Source)",
		"func LoadServerProperties(source config.Source) (cfgapp.ServerProperties, error)",
		`config.Bind("server", source, &out)`,
		"func buildLifecycle(components *Components) *runtime.Lifecycle",
		"components.Engine.Start(ctx)",
		"components.Engine.Stop()",
		"func NewApplication(configSource config.Source)",
	} {
		if !strings.Contains(src, want) {
			t.Errorf("generated config/lifecycle wiring missing %q", want)
		}
	}
	path := filepath.Join("..", "..", "internal", "cfge2e", "wiring.gen.go")
	committed, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading committed wiring: %v", err)
	}
	if src != string(committed) {
		t.Errorf("internal/cfge2e/wiring.gen.go is stale; regenerate it from the cfgapp example")
	}
}

// TestRepoE2EWiringUpToDate guards the committed repository integration wiring
// against the generator and asserts the repository sections are present.
func TestRepoE2EWiringUpToDate(t *testing.T) {
	res := analyzeRepoApp(t)
	src, err := Generate(res.App, res.Graph, Options{Package: "repoe2e"})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	for _, want := range []string{
		"func buildComponents(dbProvider db.DBProvider)",
		"type UserRepositoryImpl struct {",
		"func NewUserRepositoryImpl(db db.DBProvider) *UserRepositoryImpl",
		"r.db.DB(a0).QueryRowContext(a0, `SELECT id, name, email FROM users WHERE id = $1`, a1)",
		"r.db.DB(a0).ExecContext(a0, `INSERT INTO users (id, name, email) VALUES ($1, $2, $3)`, a1, a2, a3)",
		"return res.RowsAffected()",
		"for rows.Next() {",
	} {
		if !strings.Contains(src, want) {
			t.Errorf("generated repository wiring missing %q", want)
		}
	}
	path := filepath.Join("..", "..", "internal", "repoe2e", "wiring.gen.go")
	committed, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading committed wiring: %v", err)
	}
	if src != string(committed) {
		t.Errorf("internal/repoe2e/wiring.gen.go is stale; regenerate it from the repoapp example")
	}
}

// TestRepositoryDialectSwap proves the driver seam: the same repository, with a
// different dialect, produces ?-style placeholders instead of $n — no other
// change to the generated code.
func TestRepositoryDialectSwap(t *testing.T) {
	res := analyzeRepoApp(t)
	src, err := Generate(res.App, res.Graph, Options{Package: "repoe2e", Dialect: sqlgen.Question})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if !strings.Contains(src, "WHERE id = ?`, a1)") {
		t.Errorf("question dialect should render ? placeholders")
	}
	if strings.Contains(src, "WHERE id = $1") {
		t.Errorf("question dialect should not render $n placeholders")
	}
}

func analyzeRepoApp(t *testing.T) *compiler.AnalysisResult {
	t.Helper()
	l := &compiler.Loader{Dir: filepath.Join("..", "..", "compiler")}
	scan, err := l.Load("./testdata/repoapp")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	res := compiler.Analyze(scan)
	for _, d := range res.Diagnostics {
		if d.Severity == 2 {
			t.Fatalf("analysis error: %s", d.Error())
		}
	}
	return res
}

// TestSchedE2EWiringUpToDate guards the committed scheduler integration wiring
// and asserts the scheduler sections are present.
func TestSchedE2EWiringUpToDate(t *testing.T) {
	l := &compiler.Loader{Dir: filepath.Join("..", "..", "compiler")}
	scan, err := l.Load("./testdata/schedtick")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	res := compiler.Analyze(scan)
	src, err := Generate(res.App, res.Graph, Options{Package: "schede2e"})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	for _, want := range []string{
		"func buildScheduler(components *Components) *runtime.Scheduler",
		"sched.Register(runtime.ScheduledTask{",
		"5000000,", // 5ms in nanoseconds
		"return components.Ticker.Tick(ctx)",
		"Scheduler: sched",
	} {
		if !strings.Contains(src, want) {
			t.Errorf("generated scheduler wiring missing %q", want)
		}
	}
	path := filepath.Join("..", "..", "internal", "schede2e", "wiring.gen.go")
	committed, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading committed wiring: %v", err)
	}
	if src != string(committed) {
		t.Errorf("internal/schede2e/wiring.gen.go is stale; regenerate it from the schedtick example")
	}
}

// TestScheduledTimeUnitCompilesToDuration checks the fixedRate+timeUnit form
// (2 MINUTES) renders as the correct nanosecond interval.
func TestScheduledTimeUnitCompilesToDuration(t *testing.T) {
	l := &compiler.Loader{Dir: filepath.Join("..", "..", "compiler")}
	scan, err := l.Load("./testdata/schedapp")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	res := compiler.Analyze(scan)
	src, err := Generate(res.App, res.Graph, Options{Package: "schedapp"})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	if !strings.Contains(src, "120000000000") {
		t.Errorf("fixedRate=2 MINUTES should render as 120000000000 ns")
	}
	if !strings.Contains(src, "InitialDelay:") || !strings.Contains(src, "5000000000") {
		t.Errorf("initialDelay=5s should render as 5000000000 ns")
	}
}

// TestAuthE2EWiringUpToDate guards the committed authorization wiring and
// asserts the @Authorize interceptor is rendered.
func TestAuthE2EWiringUpToDate(t *testing.T) {
	l := &compiler.Loader{Dir: filepath.Join("..", "..", "compiler")}
	scan, err := l.Load("./testdata/authapp")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	res := compiler.Analyze(scan)
	src, err := Generate(res.App, res.Graph, Options{Package: "authe2e"})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	for _, want := range []string{
		`p.authorizer.Authorize(a0, runtime.AuthorizationRequest{Roles: []string{"admin"}, Mode: runtime.AuthorizationModeAll})`,
		`Roles: []string{"reader"}`,
		"authorizer   runtime.Authorizer",
	} {
		if !strings.Contains(src, want) {
			t.Errorf("generated auth wiring missing %q", want)
		}
	}
	committed, err := os.ReadFile(filepath.Join("..", "..", "internal", "authe2e", "wiring.gen.go"))
	if err != nil {
		t.Fatalf("reading committed wiring: %v", err)
	}
	if src != string(committed) {
		t.Errorf("internal/authe2e/wiring.gen.go is stale; regenerate it from the authapp example")
	}
}

// TestResilienceE2EWiringUpToDate guards the committed resilience wiring and
// asserts the @Retry/@Timeout interceptors are rendered.
func TestResilienceE2EWiringUpToDate(t *testing.T) {
	l := &compiler.Loader{Dir: filepath.Join("..", "..", "compiler")}
	scan, err := l.Load("./testdata/resilienceapp")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	res := compiler.Analyze(scan)
	src, err := Generate(res.App, res.Graph, Options{Package: "resiliencee2e"})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	for _, want := range []string{
		"err = runtime.Retry(a0, runtime.RetryPolicy{MaxAttempts: 4",
		"context.WithTimeout(a0, 20000000)",
		"defer cancel()",
	} {
		if !strings.Contains(src, want) {
			t.Errorf("generated resilience wiring missing %q", want)
		}
	}
	path := filepath.Join("..", "..", "internal", "resiliencee2e", "wiring.gen.go")
	committed, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading committed wiring: %v", err)
	}
	if src != string(committed) {
		t.Errorf("internal/resiliencee2e/wiring.gen.go is stale; regenerate it from the resilienceapp example")
	}
}

// TestObsE2EWiringUpToDate guards the committed observability wiring and asserts
// the @Logged/@Audit interceptors are rendered.
func TestObsE2EWiringUpToDate(t *testing.T) {
	l := &compiler.Loader{Dir: filepath.Join("..", "..", "compiler")}
	scan, err := l.Load("./testdata/obsapp")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	res := compiler.Analyze(scan)
	src, err := Generate(res.App, res.Graph, Options{Package: "obse2e"})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	for _, want := range []string{
		`logDone := p.logger.Log(a0, "VaultService.Store", "debug")`,
		"defer func() { logDone(err) }()",
		`p.audit.Record(a0, runtime.AuditEvent{Method: "VaultService.Store", Action: "store", Resource: "secret"}, err)`,
		"logger       runtime.MethodLogger",
		"audit        runtime.AuditSink",
	} {
		if !strings.Contains(src, want) {
			t.Errorf("generated observability wiring missing %q", want)
		}
	}
	path := filepath.Join("..", "..", "internal", "obse2e", "wiring.gen.go")
	committed, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading committed wiring: %v", err)
	}
	if src != string(committed) {
		t.Errorf("internal/obse2e/wiring.gen.go is stale; regenerate it from the obsapp example")
	}
}

// TestGateE2EWiringUpToDate guards the committed resilience-gate wiring and
// asserts the @CircuitBreaker/@RateLimit/@Bulkhead interceptors are rendered.
func TestGateE2EWiringUpToDate(t *testing.T) {
	l := &compiler.Loader{Dir: filepath.Join("..", "..", "compiler")}
	scan, err := l.Load("./testdata/gateapp")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	res := compiler.Analyze(scan)
	src, err := Generate(res.App, res.Graph, Options{Package: "gatee2e"})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	for _, want := range []string{
		`p.breakers.CircuitBreaker(runtime.CircuitBreakerSpec{Name: "downstream", FailureThreshold: 2, ResetTimeout: 50000000})`,
		`p.rateLimiters.RateLimiter(runtime.RateLimitSpec{Name: "DownstreamService.Fetch", Limit: 2, Period: 1000000000})`,
		`p.bulkheads.Bulkhead(runtime.BulkheadSpec{Name: "DownstreamService.Bounded", MaxConcurrent: 1})`,
		"breakers     runtime.CircuitBreakerProvider",
		"rateLimiters runtime.RateLimiterProvider",
		"bulkheads    runtime.BulkheadProvider",
	} {
		if !strings.Contains(src, want) {
			t.Errorf("generated gate wiring missing %q", want)
		}
	}
	path := filepath.Join("..", "..", "internal", "gatee2e", "wiring.gen.go")
	committed, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading committed wiring: %v", err)
	}
	if src != string(committed) {
		t.Errorf("internal/gatee2e/wiring.gen.go is stale; regenerate it from the gateapp example")
	}
}

// TestVerbE2EWiringUpToDate guards the committed HTTP-verb wiring and asserts
// each verb registers with its method prefix and default status.
func TestVerbE2EWiringUpToDate(t *testing.T) {
	l := &compiler.Loader{Dir: filepath.Join("..", "..", "compiler")}
	scan, err := l.Load("./testdata/verbapp")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	res := compiler.Analyze(scan)
	src, err := Generate(res.App, res.Graph, Options{Package: "verbe2e"})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	for _, want := range []string{
		`mux.HandleFunc("PUT /widgets/{id}",`,
		`mux.HandleFunc("PATCH /widgets/{id}",`,
		`mux.HandleFunc("DELETE /widgets/{id}",`,
		"deps.ResponseWriter.Write(ctx, w, 200, response)", // PUT/PATCH default
		"deps.ResponseWriter.Write(ctx, w, 204, nil)",      // DELETE default, no body
	} {
		if !strings.Contains(src, want) {
			t.Errorf("generated verb wiring missing %q", want)
		}
	}
	path := filepath.Join("..", "..", "internal", "verbe2e", "wiring.gen.go")
	committed, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading committed wiring: %v", err)
	}
	if src != string(committed) {
		t.Errorf("internal/verbe2e/wiring.gen.go is stale; regenerate it from the verbapp example")
	}
}

// TestProxyE2EWiringUpToDate guards the committed service-proxy integration
// wiring against the generator and asserts the proxy sections are present.
func TestProxyE2EWiringUpToDate(t *testing.T) {
	l := &compiler.Loader{Dir: filepath.Join("..", "..", "compiler")}
	scan, err := l.Load("./testdata/proxyapp")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	res := compiler.Analyze(scan)
	for _, d := range res.Diagnostics {
		if d.Severity == 2 {
			t.Fatalf("analysis error: %s", d.Error())
		}
	}
	src, err := Generate(res.App, res.Graph, Options{Package: "proxye2e"})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	for _, want := range []string{
		"func buildComponents(proxyDeps runtime.ProxyDependencies)",
		"type OrderServiceProxy struct {",
		"func NewOrderServiceProxy(target *proxyapp.OrderService, deps runtime.ProxyDependencies) *OrderServiceProxy",
		"p.transaction.WithinTransaction(",
		"p.tracer.Begin(",
		"p.metrics.RecordFailure(",
		"return p.target.GetOrder(", // delegated method
		"orderServiceProxy := NewOrderServiceProxy(orderService, proxyDeps)",
	} {
		if !strings.Contains(src, want) {
			t.Errorf("generated proxy wiring missing %q", want)
		}
	}
	path := filepath.Join("..", "..", "internal", "proxye2e", "wiring.gen.go")
	committed, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading committed wiring: %v", err)
	}
	if src != string(committed) {
		t.Errorf("internal/proxye2e/wiring.gen.go is stale; regenerate it from the proxyapp example")
	}
}

// TestE2EWiringUpToDate guards that the committed integration wiring in
// internal/e2e/wiring.gen.go matches what the generator produces. If this fails,
// regenerate it (the test message explains how) so the end-to-end tests exercise
// current output.
func TestE2EWiringUpToDate(t *testing.T) {
	res := analyzeDiapp(t)
	src, err := Generate(res.App, res.Graph, Options{Package: "e2e"})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	path := filepath.Join("..", "..", "internal", "e2e", "wiring.gen.go")
	committed, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading committed wiring: %v", err)
	}
	if src != string(committed) {
		t.Errorf("internal/e2e/wiring.gen.go is stale; regenerate it from the diapp example")
	}
}
