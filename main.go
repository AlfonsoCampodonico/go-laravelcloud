// main.go
package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alfonso/go-laravelcloud/internal/handlers"
	"github.com/alfonso/go-laravelcloud/internal/middleware"
	"github.com/alfonso/go-laravelcloud/internal/store"
)

func main() {
	mux := http.NewServeMux()
	handlers.Register(mux, store.NewMemoryStore())

	chain := middleware.Chain(
		middleware.Recovery,
		middleware.CORS,
		middleware.Logging,
		middleware.RequestID,
	)

	addr := resolveAddr()
	srv := &http.Server{
		Addr:              addr,
		Handler:           chain(mux),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		log.Printf("listening on %s", addr)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server failed: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Println("shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
}

// resolveAddr prefers the platform-injected PORT, then ADDR, then a default.
func resolveAddr() string {
	if p := os.Getenv("PORT"); p != "" {
		return ":" + p
	}
	if a := os.Getenv("ADDR"); a != "" {
		return a
	}
	return ":9114"
}
