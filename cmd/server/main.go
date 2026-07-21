package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/alecthomas/kong"
	"github.com/sdroscher/job-search-pipeline/internal/api"
	"github.com/sdroscher/job-search-pipeline/internal/config"
	"github.com/sdroscher/job-search-pipeline/internal/db"
)

func main() {
	var cfg config.Config
	kong.Parse(&cfg)

	err := os.MkdirAll(cfg.DataDir, 0o750)
	if err != nil {
		log.Fatalf("create data dir: %v", err)
	}

	err = os.MkdirAll(cfg.OutputDir, 0o750)
	if err != nil {
		log.Fatalf("create output dir: %v", err)
	}

	store, err := db.NewStore(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("open store: %v", err)
	}

	srv := api.NewServer(store, api.Config{OutputDir: cfg.OutputDir})
	addr := fmt.Sprintf(":%d", cfg.Port)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)

	server := &http.Server{
		Addr:         addr,
		Handler:      srv.Router(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		<-ctx.Done()
		// Parent ctx is already cancelled; use a fresh background ctx for the shutdown grace period.
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx) //nolint:contextcheck
	}()

	log.Printf("listening on %s", addr)
	serveErr := server.ListenAndServe()
	_ = store.Close()
	stop()

	if serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
		log.Fatal(serveErr)
	}
}
