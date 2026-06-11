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
	created_at      TEXT NOT NULL,
	role            TEXT NOT NULL DEFAULT 'player',
	password_hash   TEXT NOT NULL DEFAULT ''
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

CREATE TABLE IF NOT EXISTS invites (
	token      TEXT PRIMARY KEY,
	created_by TEXT NOT NULL DEFAULT '',
	created_at TEXT NOT NULL,
	used_by    TEXT,
	used_at    TEXT,
	revoked    INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS challenge_scores (
	period     TEXT NOT NULL,
	period_key TEXT NOT NULL,
	player_id  TEXT NOT NULL,
	username   TEXT NOT NULL,
	score      INTEGER NOT NULL,
	correct    INTEGER NOT NULL,
	total      INTEGER NOT NULL,
	played_at  TEXT NOT NULL,
	PRIMARY KEY (period, period_key, player_id)
);

CREATE TABLE IF NOT EXISTS curated_quizzes (
	id         TEXT PRIMARY KEY,
	name       TEXT NOT NULL,
	species    TEXT NOT NULL DEFAULT '[]',
	created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS challenge_schedule (
	period     TEXT NOT NULL,
	period_key TEXT NOT NULL,
	quiz_id    TEXT NOT NULL,
	PRIMARY KEY (period, period_key)
);

CREATE INDEX IF NOT EXISTS idx_challenge ON challenge_scores(period, period_key, score DESC);
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

	if err := migrate(db); err != nil {
		_ = db.Close()
		return nil, err
	}

	return db, nil
}

// migrate brings older databases up to date by adding columns that were
// introduced after the initial schema. SQLite lacks ADD COLUMN IF NOT
// EXISTS, so each column is checked before being added.
func migrate(db *sql.DB) error {
	for _, c := range []struct{ table, column, def string }{
		{"players", "role", "TEXT NOT NULL DEFAULT 'player'"},
		{"players", "password_hash", "TEXT NOT NULL DEFAULT ''"},
		{"players", "category_correct", "TEXT NOT NULL DEFAULT '{}'"},
	} {
		has, err := hasColumn(db, c.table, c.column)
		if err != nil {
			return err
		}
		if has {
			continue
		}
		if _, err := db.Exec("ALTER TABLE " + c.table + " ADD COLUMN " + c.column + " " + c.def); err != nil {
			return fmt.Errorf("migrating %s.%s: %w", c.table, c.column, err)
		}
	}
	return nil
}

// hasColumn reports whether a table already has a column.
func hasColumn(db *sql.DB, table, column string) (bool, error) {
	rows, err := db.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		return false, fmt.Errorf("inspecting %s: %w", table, err)
	}
	defer func() { _ = rows.Close() }()

	for rows.Next() {
		var (
			cid, notnull, pk int
			name, ctype      string
			dfltValue        sql.NullString
		)
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dfltValue, &pk); err != nil {
			return false, err
		}
		if name == column {
			return true, nil
		}
	}
	return false, rows.Err()
}
