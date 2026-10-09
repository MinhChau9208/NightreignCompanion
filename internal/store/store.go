// Package store is the local SQLite database (pure Go driver, no CGO).
//
// The runs table uses the same shape as the future community-data payload
// (M6) so enabling sharing later needs no migration of existing rows.
package store

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// migrations are applied in order; never edit one that has shipped,
// append a new entry instead.
var migrations = []string{
	`CREATE TABLE runs (
		id             TEXT PRIMARY KEY,
		created_at     TEXT NOT NULL,
		mode           TEXT NOT NULL,
		depth          INTEGER NOT NULL DEFAULT 0,
		character      TEXT NOT NULL DEFAULT '',
		night1_boss    TEXT NOT NULL DEFAULT '',
		night2_boss    TEXT NOT NULL DEFAULT '',
		shifting_earth TEXT NOT NULL DEFAULT '',
		nightlord      TEXT NOT NULL DEFAULT '',
		result         TEXT NOT NULL DEFAULT '',
		clear_seconds  INTEGER NOT NULL DEFAULT 0,
		build_json     TEXT NOT NULL DEFAULT '',
		schema_version INTEGER NOT NULL DEFAULT 1,
		shared_at      TEXT
	);
	CREATE INDEX runs_created_at ON runs(created_at);
	CREATE TABLE builds (
		id         TEXT PRIMARY KEY,
		name       TEXT NOT NULL,
		character  TEXT NOT NULL,
		origin     TEXT NOT NULL,
		data_json  TEXT NOT NULL,
		created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL
	);`,
}

// timeFormat is fixed-width so stored timestamps sort correctly as text.
const timeFormat = "2006-01-02T15:04:05.000000000Z"

type DB struct{ sql *sql.DB }

// Open opens (or creates) the database at path and applies migrations.
func Open(ctx context.Context, path string) (*DB, error) {
	dsn := "file:" + path + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)"
	sqldb, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, err
	}
	sqldb.SetMaxOpenConns(1)
	db := &DB{sql: sqldb}
	if err := db.migrate(ctx); err != nil {
		sqldb.Close()
		return nil, err
	}
	return db, nil
}

func (db *DB) Close() error { return db.sql.Close() }

func (db *DB) migrate(ctx context.Context) error {
	if _, err := db.sql.ExecContext(ctx,
		`CREATE TABLE IF NOT EXISTS schema_migrations (version INTEGER PRIMARY KEY)`); err != nil {
		return err
	}
	var current int
	if err := db.sql.QueryRowContext(ctx,
		`SELECT COALESCE(MAX(version), 0) FROM schema_migrations`).Scan(&current); err != nil {
		return err
	}
	for v := current + 1; v <= len(migrations); v++ {
		tx, err := db.sql.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		if _, err := tx.ExecContext(ctx, migrations[v-1]); err != nil {
			tx.Rollback()
			return fmt.Errorf("migration %d: %w", v, err)
		}
		if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations(version) VALUES (?)`, v); err != nil {
			tx.Rollback()
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
	}
	return nil
}

// Run is one finished (or abandoned) expedition.
type Run struct {
	ID            string    `json:"id"`
	CreatedAt     time.Time `json:"createdAt"`
	Mode          string    `json:"mode"` // normal | deep
	Depth         int       `json:"depth"`
	Character     string    `json:"character"`
	Night1Boss    string    `json:"night1Boss"`
	Night2Boss    string    `json:"night2Boss"`
	ShiftingEarth string    `json:"shiftingEarth"`
	Nightlord     string    `json:"nightlord"`
	Result        string    `json:"result"` // win | loss | abandoned
	ClearSeconds  int       `json:"clearSeconds"`
	BuildJSON     string    `json:"buildJson"`
}

// InsertRun stores r, filling ID and CreatedAt when empty.
func (db *DB) InsertRun(ctx context.Context, r *Run) error {
	if r.ID == "" {
		r.ID = newID()
	}
	if r.CreatedAt.IsZero() {
		r.CreatedAt = time.Now().UTC()
	}
	_, err := db.sql.ExecContext(ctx, `INSERT INTO runs
		(id, created_at, mode, depth, character, night1_boss, night2_boss,
		 shifting_earth, nightlord, result, clear_seconds, build_json)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		r.ID, r.CreatedAt.UTC().Format(timeFormat), r.Mode, r.Depth, r.Character,
		r.Night1Boss, r.Night2Boss, r.ShiftingEarth, r.Nightlord, r.Result,
		r.ClearSeconds, r.BuildJSON)
	return err
}

// ListRuns returns the most recent runs first.
func (db *DB) ListRuns(ctx context.Context, limit int) ([]Run, error) {
	rows, err := db.sql.QueryContext(ctx, `SELECT id, created_at, mode, depth, character,
		night1_boss, night2_boss, shifting_earth, nightlord, result, clear_seconds, build_json
		FROM runs ORDER BY created_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Run
	for rows.Next() {
		var r Run
		var created string
		if err := rows.Scan(&r.ID, &created, &r.Mode, &r.Depth, &r.Character,
			&r.Night1Boss, &r.Night2Boss, &r.ShiftingEarth, &r.Nightlord, &r.Result,
			&r.ClearSeconds, &r.BuildJSON); err != nil {
			return nil, err
		}
		if r.CreatedAt, err = time.Parse(timeFormat, created); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func newID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err) // crypto/rand never fails on supported platforms
	}
	return hex.EncodeToString(b[:])
}
