package compiler

import "testing"

func TestContentNegotiationDiscovery(t *testing.T) {
	res := analyzeApp(t, "./testdata/negapp")
	if errs := errorDiags(res.Diagnostics); len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}

	byHandler := map[string]*routeView{}
	for _, r := range res.App.Routes {
		byHandler[r.HandlerName] = &routeView{consumes: r.Consumes, produces: r.Produces}
	}

	// Create declares consumes/produces via mapping arguments.
	create := byHandler["Create"]
	if create == nil || len(create.consumes) != 1 || create.consumes[0] != "application/json" {
		t.Errorf("Create consumes = %+v", create)
	}
	if len(create.produces) != 1 || create.produces[0] != "application/json" {
		t.Errorf("Create produces = %+v", create)
	}

	// Render declares produces via the standalone @Produces annotation only.
	render := byHandler["Render"]
	if render == nil || len(render.consumes) != 0 {
		t.Errorf("Render should have no consumes, got %+v", render)
	}
	if len(render.produces) != 1 || render.produces[0] != "application/json" {
		t.Errorf("Render produces = %+v", render)
	}
}

type routeView struct {
	consumes []string
	produces []string
}
