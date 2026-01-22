package main

import (
	"database/sql"
	"fmt"
	"github.com/guitarkeegan/diet-tracker/internal/db"
	"log"

	_ "modernc.org/sqlite"
)

func main() {
	sqlDB, err := dbConn("file:./app.db")
	if err != nil {
		log.Fatalf("db connection failed: %s", err)
	}
	defer sqlDB.Close()

	q := db.New(sqlDB)

}

func dbConn(dsn string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	pragmas := `
		PRAGMA journal_mode = WAL;
		PRAGMA busy_timeout = 5000;
		PRAGMA synchronous = NORMAL;
		PRAGMA cache_size = -64000;
		PRAGMA foreign_keys = ON;
	`

	if _, err := db.Exec(pragmas); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to set PRAGMAs: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	return db, nil
}
