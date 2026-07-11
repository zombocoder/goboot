package compiler

import "testing"

func TestAuthorizeDiscovery(t *testing.T) {
	res := analyzeApp(t, "./testdata/authapp")
	if errs := errorDiags(res.Diagnostics); len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	admin := componentByName(res.App, "admin")
	if admin == nil || !admin.Proxied {
		t.Fatal("admin service should be proxied")
	}
	byName := map[string]int{}
	for i, m := range admin.Intercepted {
		byName[m.Name] = i
	}
	del := admin.Intercepted[byName["DeleteAll"]]
	if del.Authorize == nil || len(del.Authorize.Roles) != 1 || del.Authorize.Roles[0] != "admin" || del.Authorize.Mode != "all" {
		t.Errorf("DeleteAll authorize = %+v", del.Authorize)
	}
	read := admin.Intercepted[byName["Read"]]
	if read.Authorize == nil || len(read.Authorize.Roles) != 1 || read.Authorize.Roles[0] != "reader" {
		t.Errorf("Read (RolesAllowed) authorize = %+v", read.Authorize)
	}
}
