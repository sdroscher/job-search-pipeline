package main

import (
	"database/sql"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
	"github.com/sdroscher/job-search-pipeline/internal/migrate"
)

func main() {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "./data/pipeline.db"
	}
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	if err := migrate.Run(db); err != nil {
		log.Fatal(err)
	}
	log.Println("migrations applied")
}
