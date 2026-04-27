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
    task_name TEXT NOT NULL DEFAULT '',
    sort_order INTEGER NOT NULL DEFAULT 0
);
CREATE UNIQUE INDEX IF NOT EXISTS idx_schedules_date ON schedules(date);
CREATE INDEX IF NOT EXISTS idx_schedule_blocks_schedule ON schedule_blocks(schedule_id);
`

const (
	queryInsertSchedule = "INSERT INTO schedules (id, date, strategy, created_at) VALUES (?, ?, ?, ?)"
	queryInsertBlock    = "INSERT INTO schedule_blocks (schedule_id, start_time, end_time, block_type, task_id, task_name, sort_order) VALUES (?, ?, ?, ?, ?, ?, ?)"
	querySelectByDate   = "SELECT id, date, strategy, created_at FROM schedules WHERE date = ?"
	querySelectBlocks   = "SELECT start_time, end_time, block_type, task_id, task_name FROM schedule_blocks WHERE schedule_id = ? ORDER BY sort_order ASC"
	queryDeleteSchedule = "DELETE FROM schedules WHERE id = ?"
	queryDeleteByDate   = "DELETE FROM schedules WHERE date = ?"
)

const dateFormat = "2006-01-02"

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
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Delete existing schedule for this date (cascade deletes blocks)
	dateStr := schedule.Date.UTC().Format(dateFormat)
	if _, err := tx.ExecContext(ctx, queryDeleteByDate, dateStr); err != nil {
		return fmt.Errorf("delete existing schedule: %w", err)
	}

	_, err = tx.ExecContext(ctx, queryInsertSchedule,
		schedule.ID.String(),
		dateStr,
		schedule.Strategy,
		schedule.CreatedAt.UTC().Format(time.RFC3339),
	)
	if err != nil {
		return fmt.Errorf("insert schedule: %w", err)
	}

	for i, block := range schedule.Blocks {
		var taskIDStr *string
		if block.TaskID != nil {
			s := block.TaskID.String()
			taskIDStr = &s
		}
		_, err = tx.ExecContext(ctx, queryInsertBlock,
			schedule.ID.String(),
			block.Start.UTC().Format(time.RFC3339),
			block.End.UTC().Format(time.RFC3339),
			int(block.Type),
			taskIDStr,
			block.TaskName,
			i,
		)
		if err != nil {
			return fmt.Errorf("insert schedule block: %w", err)
		}
	}

	return tx.Commit()
}

// LoadByDate returns the schedule for the given date, or ErrNotFound.
func (r *SQLiteScheduleRepository) LoadByDate(ctx context.Context, date time.Time) (*repository.Schedule, error) {
	dateStr := date.UTC().Format(dateFormat)

	var sched repository.Schedule
	var idStr, createdAtStr string

	err := r.db.QueryRowContext(ctx, querySelectByDate, dateStr).Scan(
		&idStr, &dateStr, &sched.Strategy, &createdAtStr,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("schedule not found for date %s: %w", dateStr, repository.ErrNotFound)
	}
	if err != nil {
		return nil, fmt.Errorf("query schedule for date %s: %w", dateStr, err)
	}

	sched.ID, err = uuid.Parse(idStr)
	if err != nil {
		return nil, fmt.Errorf("parse schedule ID: %w", err)
	}
	sched.Date, err = time.Parse(dateFormat, dateStr)
	if err != nil {
		return nil, fmt.Errorf("parse schedule date: %w", err)
	}
	sched.CreatedAt, err = time.Parse(time.RFC3339, createdAtStr)
	if err != nil {
		return nil, fmt.Errorf("parse schedule created_at: %w", err)
	}

	rows, err := r.db.QueryContext(ctx, querySelectBlocks, sched.ID.String())
	if err != nil {
		return nil, fmt.Errorf("load schedule blocks: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var block repository.ScheduleBlock
		var startStr, endStr string
		var blockType int
		var taskIDStr sql.NullString

		err := rows.Scan(&startStr, &endStr, &blockType, &taskIDStr, &block.TaskName)
		if err != nil {
			return nil, fmt.Errorf("scan schedule block: %w", err)
		}

		block.Start, err = time.Parse(time.RFC3339, startStr)
		if err != nil {
			return nil, fmt.Errorf("parse block start: %w", err)
		}
		block.End, err = time.Parse(time.RFC3339, endStr)
		if err != nil {
			return nil, fmt.Errorf("parse block end: %w", err)
		}
		block.Type = repository.ScheduleBlockType(blockType)

		if taskIDStr.Valid {
			id, err := uuid.Parse(taskIDStr.String)
			if err != nil {
				return nil, fmt.Errorf("parse block task ID: %w", err)
			}
			block.TaskID = &id
		}

		sched.Blocks = append(sched.Blocks, block)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate schedule blocks: %w", err)
	}

	return &sched, nil
}

// Delete removes a schedule by date.
func (r *SQLiteScheduleRepository) Delete(ctx context.Context, date time.Time) error {
	dateStr := date.UTC().Format(dateFormat)
	_, err := r.db.ExecContext(ctx, queryDeleteByDate, dateStr)
	if err != nil {
		return fmt.Errorf("delete schedule: %w", err)
	}
	return nil
}
