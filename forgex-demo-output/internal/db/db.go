package db

import (
	"database/sql"
	"log"

	_ "github.com/mattn/go-sqlite3"
)

var DB *sql.DB

func Init(dsn string) {
	var err error
	DB, err = sql.Open("sqlite3", dsn)
	if err != nil {
		log.Fatalf("failed to open database: %v", err)
	}
	DB.SetMaxOpenConns(1)
	migrate()
}

func migrate() {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id    INTEGER PRIMARY KEY AUTOINCREMENT,
			email TEXT NOT NULL UNIQUE,
			hash  TEXT NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS todos (
			id      INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL REFERENCES users(id),
			title   TEXT    NOT NULL,
			done    INTEGER NOT NULL DEFAULT 0,
			note    TEXT    NOT NULL DEFAULT ''
		);`,
	}
	for _, q := range queries {
		if _, err := DB.Exec(q); err != nil {
			log.Fatalf("migration failed: %v", err)
		}
	}
}
