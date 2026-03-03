package db

import (
	"database/sql"
	"fmt"

	_ "github.com/mattn/go-sqlite3"
)

const createUsersTable = `
CREATE TABLE IF NOT EXISTS users (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	username TEXT UNIQUE NOT NULL,
	password_hash TEXT NOT NULL,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP
);`

const createTodosTable = `
CREATE TABLE IF NOT EXISTS todos (
	id INTEGER PRIMARY KEY AUTOINCREMENT,
	user_id INTEGER NOT NULL,
	title TEXT NOT NULL,
	completed BOOLEAN DEFAULT FALSE,
	created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
	FOREIGN KEY (user_id) REFERENCES users(id)
);`

// InitDB opens a SQLite database at dbPath, creates the required tables,
// and returns the *sql.DB connection.
func InitDB(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Enable foreign key support.
	if _, err := db.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	if _, err := db.Exec(createUsersTable); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create users table: %w", err)
	}

	if _, err := db.Exec(createTodosTable); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create todos table: %w", err)
	}

	return db, nil
}
