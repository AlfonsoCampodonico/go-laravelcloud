package handlers

import (
	"net/http"

	"github.com/alfonso/go-laravelcloud/internal/respond"
)

func Root(w http.ResponseWriter, r *http.Request) {
	respond.JSON(w, http.StatusOK, map[string]any{
		"name":    "go-laravelcloud",
		"message": "A Go API showcasing Go on Laravel Cloud.",
		"docs":    "/docs",
		"openapi": "/openapi.json",
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
