package handlers

import "net/http"

func Register(mux *http.ServeMux) {
	mux.HandleFunc("GET /", Root)
	mux.HandleFunc("GET /hello", Hello)
	mux.HandleFunc("GET /health", Health)
	mux.HandleFunc("GET /version", Version)
	mux.HandleFunc("GET /time", Time)
	mux.HandleFunc("GET /random", Random)
	mux.HandleFunc("GET /quote", Quote)
	mux.HandleFunc("POST /echo", Echo)

	mux.HandleFunc("GET /users", ListUsers)
	mux.HandleFunc("GET /users/{id}", GetUser)
	mux.HandleFunc("POST /users", CreateUser)
	mux.HandleFunc("DELETE /users/{id}", DeleteUser)

	mux.HandleFunc("GET /panic", Panic)
	mux.HandleFunc("GET /slow", Slow)
}
