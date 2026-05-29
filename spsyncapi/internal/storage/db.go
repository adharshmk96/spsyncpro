package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"spsyncapi/internal/config"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// OpenSQLite opens SQLite at sqlitePath (primarily for tests).
func OpenSQLite(sqlitePath string) (*gorm.DB, error) {
	return openSQLite(sqlitePath)
}

// Open opens the application database from configuration, runs AutoMigrate, and returns a handle.
func Open(cfg config.DBConfig) (*gorm.DB, error) {
	switch strings.ToLower(strings.TrimSpace(cfg.Driver)) {
	case "", "sqlite":
		return openSQLite(cfg.SQLitePath)
	case "postgres", "postgresql":
		return openPostgres(cfg.PostgresDSN)
	default:
		return nil, fmt.Errorf("storage: unsupported db driver %q", cfg.Driver)
	}
}

func openSQLite(sqlitePath string) (*gorm.DB, error) {
	if err := ensureDir(sqlitePath); err != nil {
		return nil, fmt.Errorf("storage: create db directory: %w", err)
	}

	db, err := gorm.Open(sqlite.Open(sqlitePath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("storage: open sqlite %q: %w", sqlitePath, err)
	}

	if err := db.Exec("PRAGMA journal_mode=WAL").Error; err != nil {
		return nil, fmt.Errorf("storage: enable WAL mode: %w", err)
	}

	return migrate(db)
}

func openPostgres(dsn string) (*gorm.DB, error) {
	dsn = strings.TrimSpace(dsn)
	if dsn == "" {
		return nil, fmt.Errorf("storage: postgres dsn must not be empty when driver is postgres")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	})
	if err != nil {
		return nil, fmt.Errorf("storage: open postgres: %w", err)
	}

	return migrate(db)
}

func migrate(db *gorm.DB) (*gorm.DB, error) {
	if err := db.AutoMigrate(
		&Member{},
		&Session{},
		&PasswordResetToken{},
		&Organization{},
		&BucketStore{},
		&BackupJob{},
		&BackupRun{},
		&BackupRunFileTransfer{},
		&RestoreJob{},
		&RestoreRun{},
		&RestoreRunFileTransfer{},
	); err != nil {
		return nil, fmt.Errorf("storage: auto migrate: %w", err)
	}
	return db, nil
}

// ensureDir creates the parent directory of the given file path if it does not already exist.
func ensureDir(filePath string) error {
	dir := filepath.Dir(filePath)
	if dir == "." || dir == "" {
		return nil
	}
	return os.MkdirAll(dir, 0o755)
}
