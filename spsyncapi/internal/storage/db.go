package storage

import (
	"fmt"
	"os"
	"path/filepath"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Open opens (or creates) the SQLite database at the given path, runs
// AutoMigrate for all auth models, and returns the *gorm.DB handle.
func Open(sqlitePath string) (*gorm.DB, error) {
	if err := ensureDir(sqlitePath); err != nil {
		return nil, fmt.Errorf("storage: create db directory: %w", err)
	}

	db, err := gorm.Open(sqlite.Open(sqlitePath), &gorm.Config{
		// Silence GORM's built-in logger; the application uses slog.
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("storage: open sqlite %q: %w", sqlitePath, err)
	}

	// Enable WAL mode for better concurrent read performance.
	if err := db.Exec("PRAGMA journal_mode=WAL").Error; err != nil {
		return nil, fmt.Errorf("storage: enable WAL mode: %w", err)
	}

	if err := db.AutoMigrate(
		&Member{},
		&Session{},
		&PasswordResetToken{},
		&Organization{},
	); err != nil {
		return nil, fmt.Errorf("storage: auto migrate: %w", err)
	}

	return db, nil
}

// ensureDir creates the parent directory of the given file path if it does not
// already exist.
func ensureDir(filePath string) error {
	dir := filepath.Dir(filePath)
	if dir == "." || dir == "" {
		return nil
	}
	return os.MkdirAll(dir, 0o755)
}
