// internal/handlers/docs.go
package handlers

import (
	"io/fs"
	"net/http"

	"github.com/alfonso/go-laravelcloud/api"
)

func DocsSpec(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	_, _ = w.Write(api.OpenAPISpec)
}

func DocsUI() http.Handler {
	sub, err := fs.Sub(api.SwaggerUI, "swagger-ui")
	if err != nil {
		panic(err) // embedded FS is known at build time; a failure here is a build bug
	}
	return http.StripPrefix("/docs/", http.FileServer(http.FS(sub)))
}
