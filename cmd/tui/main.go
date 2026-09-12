package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/dukedelaet/diet-tracker/migrations"
	"github.com/pressly/goose/v3"

	_ "modernc.org/sqlite"
)

func main() {
	dbPath := flag.String("db", "", "override db path")
	flag.Parse()
	if err := run(*dbPath); err != nil {
		log.Fatal(err)
	}
}

func run(override string) error {
	dsn := dbDSN(override)
	db, err := sqliteOpen(dsn)
	if err != nil {
		return err
	}
	defer db.Close()
	if err := applyMigrations(db); err != nil {
		return err
	}
	m := NewModel(NewModelOpts{
		DB:   db,
		Out:  os.Stdout,
		Ctx:  context.Background(),
		Width: 80,
		Height: 24,
	})
	m.Run()
	return nil
}

func dbDSN(override string) string {
	if override != "" {
		if err := os.MkdirAll(filepath.Dir(override), 0o755); err != nil {
			return ""
		}
		return "file:" + override
	}
	if p := strings.TrimSpace(os.Getenv("DIET_DB_PATH")); p != "" {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			return ""
		}
		return "file:" + p
	}
	home, _ := os.UserHomeDir()
	p := filepath.Join(home, ".local", "share", "diet-tracker", "app.db")
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		return ""
	}
	return "file:" + p
}

func sqliteOpen(dsn string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}
	pragmas := `PRAGMA journal_mode = WAL;
PRAGMA busy_timeout = 5000;
PRAGMA foreign_keys = ON;`
	if _, err := db.Exec(pragmas); err != nil {
		db.Close()
		return nil, fmt.Errorf("set pragmas: %w", err)
	}
	return db, nil
}

func applyMigrations(db *sql.DB) error {
	goose.SetBaseFS(migrations.FS)
	goose.SetLogger(goose.NopLogger())
	if err := goose.SetDialect("sqlite3"); err != nil {
		return fmt.Errorf("goose dialect: %w", err)
	}
	if err := goose.Up(db, "."); err != nil {
		return fmt.Errorf("goose up: %w", err)
	}
	return nil
}
