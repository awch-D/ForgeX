// Package audit provides operation logging and traceability.
// It records every tool invocation with timestamps, safety levels, and results.
package audit

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/awch-D/ForgeX/forgex-core/logger"
	"github.com/awch-D/ForgeX/forgex-governance/safety"
)

// Entry represents a single audit log entry.
type Entry struct {
	ID          int64
	Timestamp   time.Time
	ToolName    string
	Args        string
	SafetyLevel safety.Level
	Approved    bool
	Success     bool
	Error       string
	OutputLen   int
}

// Logger records tool invocations to a SQLite database.
type Logger struct {
	db *sql.DB
}

// NewLogger creates a new audit logger backed by a SQLite file.
func NewLogger(dbPath string) (*Logger, error) {
	db, err := sql.Open("sqlite3", dbPath+"?_journal_mode=WAL")
	if err != nil {
		return nil, fmt.Errorf("open audit db: %w", err)
	}

	if err := migrate(db); err != nil {
		db.Close()
		return nil, err
	}

	logger.L().Infow("📋 Audit logger initialized", "path", dbPath)
	return &Logger{db: db}, nil
}

func migrate(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS audit_log (
			id           INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp    DATETIME DEFAULT CURRENT_TIMESTAMP,
			tool_name    TEXT NOT NULL,
			args         TEXT,
			safety_level INTEGER NOT NULL,
			approved     BOOLEAN NOT NULL,
			success      BOOLEAN,
			error        TEXT,
			output_len   INTEGER DEFAULT 0
		)
	`)
	if err != nil {
		return fmt.Errorf("migrate audit db: %w", err)
	}
	return nil
}

// Record logs a tool invocation.
func (l *Logger) Record(entry Entry) error {
	_, err := l.db.Exec(
		`INSERT INTO audit_log (tool_name, args, safety_level, approved, success, error, output_len)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		entry.ToolName, entry.Args, int(entry.SafetyLevel),
		entry.Approved, entry.Success, entry.Error, entry.OutputLen,
	)
	if err != nil {
		return fmt.Errorf("record audit entry: %w", err)
	}
	return nil
}

// Recent returns the most recent N audit entries.
func (l *Logger) Recent(limit int) ([]Entry, error) {
	rows, err := l.db.Query(
		`SELECT id, timestamp, tool_name, args, safety_level, approved, success, error, output_len
		 FROM audit_log ORDER BY id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, fmt.Errorf("query audit log: %w", err)
	}
	defer rows.Close()

	var entries []Entry
	for rows.Next() {
		var e Entry
		var ts string
		var lvl int
		if err := rows.Scan(&e.ID, &ts, &e.ToolName, &e.Args, &lvl,
			&e.Approved, &e.Success, &e.Error, &e.OutputLen); err != nil {
			return nil, fmt.Errorf("scan audit entry: %w", err)
		}
		e.Timestamp, _ = time.Parse("2006-01-02 15:04:05", ts)
		e.SafetyLevel = safety.Level(lvl)
		entries = append(entries, e)
	}
	return entries, nil
}

// Count returns the total number of audit entries.
func (l *Logger) Count() (int, error) {
	var count int
	err := l.db.QueryRow("SELECT COUNT(*) FROM audit_log").Scan(&count)
	return count, err
}

// Stats returns summary statistics of the audit log.
type Stats struct {
	Total    int
	Approved int
	Blocked  int
	Failed   int
	ByLevel  map[safety.Level]int
}

func (l *Logger) Stats() (*Stats, error) {
	s := &Stats{ByLevel: make(map[safety.Level]int)}

	err := l.db.QueryRow("SELECT COUNT(*) FROM audit_log").Scan(&s.Total)
	if err != nil {
		return nil, err
	}
	l.db.QueryRow("SELECT COUNT(*) FROM audit_log WHERE approved = 1").Scan(&s.Approved)
	l.db.QueryRow("SELECT COUNT(*) FROM audit_log WHERE approved = 0").Scan(&s.Blocked)
	l.db.QueryRow("SELECT COUNT(*) FROM audit_log WHERE success = 0 AND approved = 1").Scan(&s.Failed)

	rows, err := l.db.Query("SELECT safety_level, COUNT(*) FROM audit_log GROUP BY safety_level")
	if err != nil {
		return s, nil
	}
	defer rows.Close()
	for rows.Next() {
		var lvl, cnt int
		rows.Scan(&lvl, &cnt)
		s.ByLevel[safety.Level(lvl)] = cnt
	}
	return s, nil
}

// Close closes the audit database.
func (l *Logger) Close() error {
	return l.db.Close()
}
