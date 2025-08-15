package database

import (
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jmoiron/sqlx"

	"github.com/kaanevranportfolio/Commercium/pkg/logger"
)

// Migrator handles database migrations
type Migrator struct {
	migrate *migrate.Migrate
	logger  *logger.Logger
}

// NewMigrator creates a new database migrator
func NewMigrator(db *sqlx.DB, migrationsPath string, log *logger.Logger) (*Migrator, error) {
	driver, err := postgres.WithInstance(db.DB, &postgres.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to create postgres driver: %w", err)
	}

	m, err := migrate.NewWithDatabaseInstance(
		fmt.Sprintf("file://%s", migrationsPath),
		"postgres",
		driver,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create migrate instance: %w", err)
	}

	return &Migrator{
		migrate: m,
		logger:  log,
	}, nil
}

// Up runs all pending migrations
func (m *Migrator) Up() error {
	m.logger.Info("Running database migrations up")
	
	err := m.migrate.Up()
	if err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			m.logger.Info("No pending migrations")
			return nil
		}
		return fmt.Errorf("failed to run migrations up: %w", err)
	}
	
	m.logger.Info("Database migrations completed successfully")
	return nil
}

// Down runs all down migrations
func (m *Migrator) Down() error {
	m.logger.Info("Running database migrations down")
	
	err := m.migrate.Down()
	if err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			m.logger.Info("No migrations to rollback")
			return nil
		}
		return fmt.Errorf("failed to run migrations down: %w", err)
	}
	
	m.logger.Info("Database migrations rollback completed")
	return nil
}

// Steps runs n migration steps
func (m *Migrator) Steps(n int) error {
	m.logger.Info("Running migration steps", "steps", n)
	
	err := m.migrate.Steps(n)
	if err != nil {
		if errors.Is(err, migrate.ErrNoChange) {
			m.logger.Info("No migrations to run")
			return nil
		}
		return fmt.Errorf("failed to run migration steps: %w", err)
	}
	
	m.logger.Info("Migration steps completed", "steps", n)
	return nil
}

// Version returns the current migration version
func (m *Migrator) Version() (uint, bool, error) {
	version, dirty, err := m.migrate.Version()
	if err != nil {
		if errors.Is(err, migrate.ErrNilVersion) {
			return 0, false, nil
		}
		return 0, false, fmt.Errorf("failed to get migration version: %w", err)
	}
	
	return version, dirty, nil
}

// Force sets the migration version without running migrations
func (m *Migrator) Force(version int) error {
	m.logger.Warn("Forcing migration version", "version", version)
	
	err := m.migrate.Force(version)
	if err != nil {
		return fmt.Errorf("failed to force migration version: %w", err)
	}
	
	m.logger.Info("Migration version forced", "version", version)
	return nil
}

// Close closes the migrator
func (m *Migrator) Close() error {
	sourceErr, databaseErr := m.migrate.Close()
	if sourceErr != nil {
		m.logger.Error("Failed to close migration source", "error", sourceErr)
	}
	if databaseErr != nil {
		m.logger.Error("Failed to close migration database", "error", databaseErr)
	}
	
	if sourceErr != nil || databaseErr != nil {
		return fmt.Errorf("failed to close migrator: source_err=%v, database_err=%v", sourceErr, databaseErr)
	}
	
	return nil
}
