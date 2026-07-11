package compiler

import (
	"testing"

	"github.com/zombocoder/goboot/model"
)

func TestProxyDiscovery(t *testing.T) {
	res := analyzeApp(t, "./testdata/proxyapp")
	if errs := errorDiags(res.Diagnostics); len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}

	// The service target is marked proxied with its intercepted method.
	target := componentByName(res.App, "orderService")
	if target == nil {
		t.Fatal("orderService not found")
	}
	if !target.Proxied {
		t.Error("orderService should be marked proxied")
	}
	if len(target.Intercepted) != 1 || target.Intercepted[0].Name != "CreateOrder" {
		t.Fatalf("intercepted methods = %+v", target.Intercepted)
	}
	m := target.Intercepted[0]
	if !m.Traced || !m.Timed || !m.Transactional {
		t.Errorf("CreateOrder should be traced, timed, and transactional: %+v", m)
	}
	if m.TraceName != "orders.create" {
		t.Errorf("trace name = %q", m.TraceName)
	}

	// A proxy component was synthesized providing the interface.
	proxy := componentByName(res.App, "OrderServiceProxy")
	if proxy == nil {
		t.Fatal("OrderServiceProxy component not synthesized")
	}
	if proxy.Kind != model.ComponentProxy {
		t.Errorf("proxy kind = %v", proxy.Kind)
	}
	if proxy.ProxyTarget != target.ID {
		t.Errorf("proxy target = %q, want %q", proxy.ProxyTarget, target.ID)
	}
	// The proxy depends (pre-resolved) on its target.
	if len(proxy.Dependencies) != 1 || proxy.Dependencies[0].ResolvedTo != target.ID {
		t.Errorf("proxy dependency = %v", proxy.DependsOn())
	}
}

func TestControllerResolvesToProxy(t *testing.T) {
	res := analyzeApp(t, "./testdata/proxyapp")
	ctrl := componentByName(res.App, "OrderController")
	proxy := componentByName(res.App, "OrderServiceProxy")
	if ctrl == nil || proxy == nil {
		t.Fatal("controller or proxy missing")
	}
	if len(ctrl.Dependencies) != 1 || ctrl.Dependencies[0].ResolvedTo != proxy.ID {
		t.Errorf("controller should depend on the proxy %q, got %v", proxy.ID, ctrl.DependsOn())
	}
}

func TestConcreteInjectionRejected(t *testing.T) {
	res := analyzeApp(t, "./testdata/badinject")
	found := false
	for _, d := range res.Diagnostics {
		if d.Code == CodeConcreteInjection {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected a concrete-injection diagnostic, got %v", res.Diagnostics)
	}
}

func TestConstructionOrderProxyAfterTarget(t *testing.T) {
	res := analyzeApp(t, "./testdata/proxyapp")
	order, cyc := res.Graph.ConstructionOrder()
	if cyc != nil {
		t.Fatalf("unexpected cycle: %v", cyc.Path)
	}
	pos := map[model.ComponentID]int{}
	for i, id := range order {
		pos[id] = i
	}
	target := componentByName(res.App, "orderService")
	proxy := componentByName(res.App, "OrderServiceProxy")
	ctrl := componentByName(res.App, "OrderController")
	if pos[target.ID] >= pos[proxy.ID] {
		t.Error("target must be constructed before its proxy")
	}
	if pos[proxy.ID] >= pos[ctrl.ID] {
		t.Error("proxy must be constructed before the controller that uses it")
	}
}
