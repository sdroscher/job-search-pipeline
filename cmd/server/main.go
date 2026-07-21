package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/alecthomas/kong"
	"github.com/sdroscher/job-search-pipeline/internal/api"
	"github.com/sdroscher/job-search-pipeline/internal/config"
	"github.com/sdroscher/job-search-pipeline/internal/db"
)

func main() {
	var cfg config.Config
	kong.Parse(&cfg)

	if err := os.MkdirAll(cfg.DataDir, 0o750); err != nil {
		log.Fatalf("create data dir: %v", err)
	}
	if err := os.MkdirAll(cfg.OutputDir, 0o750); err != nil {
		log.Fatalf("create output dir: %v", err)
	}

	store, err := db.NewStore(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	defer store.Close()

	srv := api.NewServer(store, api.Config{OutputDir: cfg.OutputDir})
	addr := fmt.Sprintf(":%d", cfg.Port)
	log.Printf("listening on %s", addr)
	if err := http.ListenAndServe(addr, srv.Router()); err != nil {
		log.Fatal(err)
	}
}
