package main

import (
	"context"
	"errors"
	"log"
	"net"
	"net/http"
	"service-share-link/internal/platform/config"
	"service-share-link/internal/platform/deps"
	"time"
)

func NewHttpServer(ctx context.Context, cfg *config.ConfigEnv, deps *deps.ConfigDeps) {
	srv := &http.Server{
		Addr:              net.JoinHostPort(cfg.HttpAddr, cfg.HttpPort),
		Handler:           NewSeverRouter(deps),
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Printf("HTTP server listening on %s", srv.Addr)
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if !errors.Is(err, http.ErrServerClosed) {
			log.Printf("HTTP server error: %v", err)
		}

	case <-ctx.Done():
		log.Println("shutting down...")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("HTTP shutdown error: %v", err)
	}
}
