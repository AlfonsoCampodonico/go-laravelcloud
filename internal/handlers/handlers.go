package handlers

import (
	"net/http"

	"github.com/alfonso/go-laravelcloud/internal/store"
)

type Route struct {
	Method       string
	Pattern      string
	Handler      http.HandlerFunc
	Undocumented bool // excluded from the OpenAPI contract test
}

// Routes is the single source of truth for the API surface. The contract test
// (contract_test.go) asserts every documented route appears in api/openapi.json.
func Routes(st store.Store) []Route {
	u := &UserHandler{Store: st}
	return []Route{
		{Method: "GET", Pattern: "/", Handler: Root, Undocumented: true},
		{Method: "GET", Pattern: "/hello", Handler: Hello},
		{Method: "GET", Pattern: "/health", Handler: Health},
		{Method: "GET", Pattern: "/version", Handler: Version},
		{Method: "GET", Pattern: "/time", Handler: Time},
		{Method: "GET", Pattern: "/random", Handler: Random},
		{Method: "GET", Pattern: "/quote", Handler: Quote},
		{Method: "POST", Pattern: "/echo", Handler: Echo},

		{Method: "GET", Pattern: "/users", Handler: u.List},
		{Method: "GET", Pattern: "/users/{id}", Handler: u.Get},
		{Method: "POST", Pattern: "/users", Handler: u.Create},
		{Method: "PUT", Pattern: "/users/{id}", Handler: u.Update},
		{Method: "DELETE", Pattern: "/users/{id}", Handler: u.Delete},

		{Method: "GET", Pattern: "/panic", Handler: Panic},
		{Method: "GET", Pattern: "/slow", Handler: Slow},

		{Method: "GET", Pattern: "/openapi.json", Handler: DocsSpec, Undocumented: true},
	}
}

func Register(mux *http.ServeMux, st store.Store) {
	for _, rt := range Routes(st) {
		mux.HandleFunc(rt.Method+" "+rt.Pattern, rt.Handler)
	}
	// Swagger UI is a static-file subtree served under /docs/. It is registered
	// directly (not via Routes) because it is not a single method+path handler,
	// so it is intentionally NOT enumerated by Routes() or checked by the
	// contract test. ServeMux auto-redirects GET /docs -> GET /docs/.
	mux.Handle("GET /docs/", DocsUI())
}
