// Package fact provides a SQLite-backed fact store with FTS5 full-text search.
// Facts are verified pieces of knowledge (file contents, test results, API schemas)
// that agents can query to avoid redundant LLM calls.
package fact

import (
	"crypto/sha256"
	"database/sql"
	"fmt"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/awch-D/ForgeX/forgex-core/logger"
)

// Fact represents a single verified piece of knowledge.
type Fact struct {
	ID        int64     `json:"id"`
	Tag       string    `json:"tag"`       // e.g. "file", "test_result", "api_schema"
	Content   string    `json:"content"`
	Hash      string    `json:"hash"`
	CreatedAt time.Time `json:"created_at"`
}

// Store manages the fact database.
type Store struct {
	db *sql.DB
	mu sync.Mutex
}

// NewStore creates and initializes a fact store at the given path.
func NewStore(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL")
	if err != nil {
		return nil, fmt.Errorf("open fact db: %w", err)
	}
	db.SetMaxOpenConns(1)

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("migrate fact db: %w", err)
	}

	logger.L().Infow("🧠 Fact store initialized", "path", dbPath)
	return s, nil
}

func (s *Store) migrate() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS facts (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			tag        TEXT NOT NULL,
			content    TEXT NOT NULL,
			hash       TEXT NOT NULL UNIQUE,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE VIRTUAL TABLE IF NOT EXISTS facts_fts USING fts5(
			tag, content, content='facts', content_rowid='id'
		);`,
		// Triggers to keep FTS in sync
		`CREATE TRIGGER IF NOT EXISTS facts_ai AFTER INSERT ON facts BEGIN
			INSERT INTO facts_fts(rowid, tag, content) VALUES (new.id, new.tag, new.content);
		END;`,
		`CREATE TRIGGER IF NOT EXISTS facts_ad AFTER DELETE ON facts BEGIN
			INSERT INTO facts_fts(facts_fts, rowid, tag, content) VALUES('delete', old.id, old.tag, old.content);
		END;`,
	}
	for _, q := range queries {
		if _, err := s.db.Exec(q); err != nil {
			return fmt.Errorf("exec %q: %w", q[:40], err)
		}
	}
	return nil
}

// Put inserts a fact. Deduplicates by content hash.
func (s *Store) Put(tag, content string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	hash := fmt.Sprintf("%x", sha256.Sum256([]byte(content)))

	_, err := s.db.Exec(
		`INSERT OR IGNORE INTO facts (tag, content, hash) VALUES (?, ?, ?)`,
		tag, content, hash,
	)
	return err
}

// Search performs full-text search and returns matching facts.
func (s *Store) Search(query string, limit int) ([]Fact, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if limit <= 0 {
		limit = 10
	}

	rows, err := s.db.Query(`
		SELECT f.id, f.tag, f.content, f.hash, f.created_at
		FROM facts f
		JOIN facts_fts fts ON f.id = fts.rowid
		WHERE facts_fts MATCH ?
		ORDER BY rank
		LIMIT ?
	`, query, limit)
	if err != nil {
		return nil, fmt.Errorf("fts search: %w", err)
	}
	defer rows.Close()

	var results []Fact
	for rows.Next() {
		var f Fact
		if err := rows.Scan(&f.ID, &f.Tag, &f.Content, &f.Hash, &f.CreatedAt); err != nil {
			return nil, err
		}
		results = append(results, f)
	}
	return results, rows.Err()
}

// Count returns the total number of facts.
func (s *Store) Count() (int, error) {
	var count int
	err := s.db.QueryRow(`SELECT COUNT(*) FROM facts`).Scan(&count)
	return count, err
}

// Close closes the database.
func (s *Store) Close() error {
	return s.db.Close()
}
