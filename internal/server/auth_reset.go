package server

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/CreateFutureMWilkinson/cue/internal/repository/implementation/sqlite"
	_ "modernc.org/sqlite"
)

// ResetAuth opens the database at dbPath, deletes all auth tokens, and closes the connection.
func ResetAuth(dbPath string) error {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return fmt.Errorf("opening database: %w", err)
	}
	defer db.Close()

	repo, err := sqlite.NewSQLiteAuthTokenRepository(db)
	if err != nil {
		return fmt.Errorf("creating auth token repository: %w", err)
	}

	if err := repo.DeleteAll(context.Background()); err != nil {
		return fmt.Errorf("deleting auth tokens: %w", err)
	}

	return nil
}
