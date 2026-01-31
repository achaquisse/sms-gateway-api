package db

import (
	"database/sql"
	"embed"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

//go:embed migrations
var migrationsFS embed.FS

func RunMigrations() error {
	if err := ensureMigrationsTable(); err != nil {
		return fmt.Errorf("failed to ensure migrations table: %w", err)
	}

	appliedVersions, err := getAppliedMigrations()
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	migrations, err := getMigrationFiles()
	if err != nil {
		return fmt.Errorf("failed to get migration files: %w", err)
	}

	for _, migration := range migrations {
		if _, applied := appliedVersions[migration.Version]; applied {
			continue
		}

		if err := applyMigration(migration); err != nil {
			return fmt.Errorf("failed to apply migration %d: %w", migration.Version, err)
		}
	}

	return nil
}

func InitSchema() error {
	schemaPath := filepath.Join("..", "db-schema.sql")
	
	content, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("failed to read schema file: %w", err)
	}

	if _, err := DB.Exec(string(content)); err != nil {
		return fmt.Errorf("failed to execute schema: %w", err)
	}

	return nil
}

type Migration struct {
	Version int
	Name    string
	SQL     string
}

func ensureMigrationsTable() error {
	query := `
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version INTEGER PRIMARY KEY,
			applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`
	_, err := DB.Exec(query)
	return err
}

func getAppliedMigrations() (map[int]bool, error) {
	rows, err := DB.Query("SELECT version FROM schema_migrations")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	applied := make(map[int]bool)
	for rows.Next() {
		var version int
		if err := rows.Scan(&version); err != nil {
			return nil, err
		}
		applied[version] = true
	}

	return applied, rows.Err()
}

func getMigrationFiles() ([]Migration, error) {
	var migrations []Migration

	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}

		parts := strings.SplitN(entry.Name(), "_", 2)
		if len(parts) != 2 {
			continue
		}

		version, err := strconv.Atoi(parts[0])
		if err != nil {
			continue
		}

		content, err := migrationsFS.ReadFile("migrations/" + entry.Name())
		if err != nil {
			return nil, err
		}

		migrations = append(migrations, Migration{
			Version: version,
			Name:    strings.TrimSuffix(parts[1], ".sql"),
			SQL:     string(content),
		})
	}

	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

func applyMigration(migration Migration) error {
	tx, err := DB.Begin()
	if err != nil {
		return err
	}
	defer func() {
		if err != nil {
			tx.Rollback()
		}
	}()

	if _, err = tx.Exec(migration.SQL); err != nil {
		return err
	}

	if _, err = tx.Exec(
		"INSERT INTO schema_migrations (version) VALUES ($1)",
		migration.Version,
	); err != nil {
		return err
	}

	return tx.Commit()
}

func GetCurrentVersion() (int, error) {
	var version sql.NullInt64
	err := DB.QueryRow("SELECT MAX(version) FROM schema_migrations").Scan(&version)
	if err != nil {
		return 0, err
	}
	if !version.Valid {
		return 0, nil
	}
	return int(version.Int64), nil
}
