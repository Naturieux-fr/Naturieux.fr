// Package sqlite provides SQLite-backed implementations of the repository
// ports, using a pure Go driver (no CGO required).
package sqlite

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite" // registers the "sqlite" driver
)

const schema = `
CREATE TABLE IF NOT EXISTS players (
	id              TEXT PRIMARY KEY,
	username        TEXT NOT NULL UNIQUE,
	total_xp        INTEGER NOT NULL DEFAULT 0,
	level           INTEGER NOT NULL DEFAULT 1,
	total_games     INTEGER NOT NULL DEFAULT 0,
	total_correct   INTEGER NOT NULL DEFAULT 0,
	total_questions INTEGER NOT NULL DEFAULT 0,
	best_streak     INTEGER NOT NULL DEFAULT 0,
	daily_streak    INTEGER NOT NULL DEFAULT 0,
	achievements    TEXT NOT NULL DEFAULT '[]',
	last_played_at  TEXT,
	created_at      TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS quiz_sessions (
	id              TEXT PRIMARY KEY,
	user_id         TEXT NOT NULL,
	status          TEXT NOT NULL,
	difficulty      TEXT NOT NULL,
	taxon_filter    TEXT NOT NULL DEFAULT '',
	total_score     INTEGER NOT NULL DEFAULT 0,
	max_streak      INTEGER NOT NULL DEFAULT 0,
	questions_count INTEGER NOT NULL DEFAULT 0,
	correct_count   INTEGER NOT NULL DEFAULT 0,
	started_at      TEXT NOT NULL,
	completed_at    TEXT,
	data            TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_sessions_user ON quiz_sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_players_xp ON players(total_xp DESC);
`

// Open opens (or creates) the SQLite database at path and applies the schema.
func Open(path string) (*sql.DB, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("opening database: %w", err)
	}

	// The pure Go driver serializes writes; a single connection avoids
	// SQLITE_BUSY errors under concurrent access.
	db.SetMaxOpenConns(1)

	if _, err := db.Exec("PRAGMA journal_mode = WAL; PRAGMA foreign_keys = ON;"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("configuring database: %w", err)
	}

	if _, err := db.Exec(schema); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("applying schema: %w", err)
	}

	return db, nil
}
