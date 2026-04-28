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
)

func main() {
	addr := os.Getenv("ADDR")
	if addr == "" {
		addr = ":9114"
	}

	mux := http.NewServeMux()
	handlers.Register(mux)

	chain := middleware.Chain(
		middleware.Recovery,
		middleware.Logging,
		middleware.RequestID,
	)

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
