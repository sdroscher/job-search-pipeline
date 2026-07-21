package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
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

	server := &http.Server{
		Addr:         addr,
		Handler:      srv.Router(),
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("listening on %s", addr)

	err = server.ListenAndServe()

	_ = store.Close()

	if err != nil {
		log.Fatal(err)
	}
}
