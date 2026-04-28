package handlers

import (
	"net/http"

	"github.com/alfonso/go-laravelcloud/internal/respond"
)

func Root(w http.ResponseWriter, r *http.Request) {
	respond.JSON(w, http.StatusOK, map[string]any{
		"name":    "go-laravelcloud",
		"message": "welcome to the garbage API",
		"routes": []string{
			"GET /hello", "GET /health", "GET /version", "GET /time",
			"GET /random", "GET /quote", "POST /echo",
			"GET /users", "GET /users/{id}", "POST /users", "DELETE /users/{id}",
			"GET /panic", "GET /slow",
		},
	})
}

func Hello(w http.ResponseWriter, r *http.Request) {
	name := r.URL.Query().Get("name")
	if name == "" {
		name = "world"
	}
	respond.JSON(w, http.StatusOK, map[string]string{
		"message": "hello " + name,
	})
}
