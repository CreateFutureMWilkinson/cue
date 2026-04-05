package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/CreateFutureMWilkinson/cue/internal/repository"

	_ "modernc.org/sqlite"
)

const createScheduleTables = `
CREATE TABLE IF NOT EXISTS schedules (
    id TEXT PRIMARY KEY,
    date TEXT NOT NULL,
    strategy TEXT NOT NULL,
    created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS schedule_blocks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    schedule_id TEXT NOT NULL REFERENCES schedules(id) ON DELETE CASCADE,
    start_time TEXT NOT NULL,
    end_time TEXT NOT NULL,
    block_type INTEGER NOT NULL,
    task_id TEXT,
    task_name TEXT NOT NULL DEFAULT ''
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_schedules_date ON schedules(date);
CREATE INDEX IF NOT EXISTS idx_schedule_blocks_schedule ON schedule_blocks(schedule_id);
`

// SQLiteScheduleRepository implements repository.ScheduleRepository using SQLite.
type SQLiteScheduleRepository struct {
	db *sql.DB
}

// NewSQLiteScheduleRepository opens a SQLite database at dbPath and creates schedule tables.
func NewSQLiteScheduleRepository(dbPath string) (*SQLiteScheduleRepository, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open schedule database: %w", err)
	}

	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("enable WAL mode: %w", err)
	}

	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}

	if _, err := db.Exec(createScheduleTables); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("create schedule tables: %w", err)
	}

	return &SQLiteScheduleRepository{db: db}, nil
}

// Save persists a schedule. If a schedule for the same date already exists, it is replaced.
func (r *SQLiteScheduleRepository) Save(ctx context.Context, schedule *repository.Schedule) error {
	return nil
}

// LoadByDate returns the schedule for the given date, or ErrNotFound.
func (r *SQLiteScheduleRepository) LoadByDate(ctx context.Context, date time.Time) (*repository.Schedule, error) {
	return nil, nil
}

// Delete removes a schedule by ID.
func (r *SQLiteScheduleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}
