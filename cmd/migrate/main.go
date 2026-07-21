package main

import (
	"database/sql"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3" // register sqlite3 driver
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

	err = migrate.Run(db)
	if err != nil {
		_ = db.Close()
		log.Fatal(err)
	}

	_ = db.Close()
	log.Println("migrations applied")
}
