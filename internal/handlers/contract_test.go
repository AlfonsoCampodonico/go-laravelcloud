// internal/handlers/contract_test.go
package handlers

import (
	"encoding/json"
	"sort"
	"strings"
	"testing"

	"github.com/alfonso/go-laravelcloud/api"
	"github.com/alfonso/go-laravelcloud/internal/store"
)

func TestOpenAPISpecMatchesRoutes(t *testing.T) {
	// Documented routes registered by the app.
	want := map[string]bool{}
	for _, rt := range Routes(store.NewMemoryStore()) {
		if rt.Undocumented {
			continue
		}
		want[strings.ToUpper(rt.Method)+" "+rt.Pattern] = true
	}

	// Operations declared in the spec.
	var spec struct {
		Paths map[string]map[string]json.RawMessage `json:"paths"`
	}
	if err := json.Unmarshal(api.OpenAPISpec, &spec); err != nil {
		t.Fatalf("parse openapi.json: %v", err)
	}
	got := map[string]bool{}
	for path, ops := range spec.Paths {
		for method := range ops {
			got[strings.ToUpper(method)+" "+path] = true
		}
	}

	if missing := diff(want, got); len(missing) > 0 {
		t.Errorf("routes registered but MISSING from openapi.json: %v", missing)
	}
	if extra := diff(got, want); len(extra) > 0 {
		t.Errorf("operations in openapi.json with NO registered route: %v", extra)
	}
}

func diff(a, b map[string]bool) []string {
	var out []string
	for k := range a {
		if !b[k] {
			out = append(out, k)
		}
	}
	sort.Strings(out)
	return out
}
