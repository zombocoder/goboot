package compiler

import (
	"testing"
)

func analyzeConditions(t *testing.T, opts Options) *AnalysisResult {
	t.Helper()
	scan := loadPkg(t, "./testdata/conditions")
	return AnalyzeWith(scan, opts)
}

func present(res *AnalysisResult, name string) bool {
	return componentByName(res.App, name) != nil
}

func TestProfileInactiveByDefault(t *testing.T) {
	// With no active profiles, unconditional components remain but
	// profile-gated ones are excluded.
	res := analyzeConditions(t, Options{})
	if !present(res, "always") {
		t.Error("unconditional component should always be present")
	}
	if present(res, "prodOnly") {
		t.Error("prodOnly should be excluded with no active profile")
	}
	if present(res, "devOnly") {
		t.Error("devOnly should be excluded with no active profile")
	}
}

func TestProfileSelectsComponents(t *testing.T) {
	res := analyzeConditions(t, Options{Profiles: []string{"production"}})
	if !present(res, "prodOnly") {
		t.Error("prodOnly should be present under the production profile")
	}
	if present(res, "devOnly") {
		t.Error("devOnly should be excluded under the production profile")
	}
}

func TestConditionalOnProperty(t *testing.T) {
	// Absent property -> excluded.
	if present(analyzeConditions(t, Options{}), "cacheEnabled") {
		t.Error("cacheEnabled should be excluded when the property is unset")
	}
	// Matching value -> included.
	res := analyzeConditions(t, Options{Properties: map[string]string{"cache.enabled": "true"}})
	if !present(res, "cacheEnabled") {
		t.Error("cacheEnabled should be present when cache.enabled=true")
	}
	// Wrong value -> excluded.
	res = analyzeConditions(t, Options{Properties: map[string]string{"cache.enabled": "false"}})
	if present(res, "cacheEnabled") {
		t.Error("cacheEnabled should be excluded when cache.enabled=false")
	}
}

func TestConditionalOnNut(t *testing.T) {
	// "always" is present, so needsAlways is included.
	res := analyzeConditions(t, Options{})
	if !present(res, "needsAlways") {
		t.Error("needsAlways should be present because 'always' is present")
	}
}

func TestConditionalOnMissingNut(t *testing.T) {
	// No PrimaryClock exists, so the fallback is included.
	res := analyzeConditions(t, Options{})
	if !present(res, "fallbackClock") {
		t.Error("fallbackClock should be present because no PrimaryClock exists")
	}
}

func TestConditionCascadeFixpoint(t *testing.T) {
	// needsProd requires prodOnly, which is profile-gated. Without the
	// production profile, prodOnly is removed and needsProd must cascade out.
	res := analyzeConditions(t, Options{})
	if present(res, "needsProd") {
		t.Error("needsProd should cascade out when prodOnly is excluded")
	}
	// With production active, both are present.
	res = analyzeConditions(t, Options{Profiles: []string{"production"}})
	if !present(res, "prodOnly") || !present(res, "needsProd") {
		t.Error("prodOnly and needsProd should both be present under production")
	}
}

func TestNoErrorsForConditions(t *testing.T) {
	res := analyzeConditions(t, Options{Profiles: []string{"production"}})
	if errs := errorDiags(res.Diagnostics); len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
}
