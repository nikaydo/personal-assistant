package localcombineddb

import (
	"database/sql"
	"embed"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

const migrationTableName = "schema_migrations"

//go:embed migrations/*.sql
var migrationFiles embed.FS

func (s *PostgresStore) runMigrations() error {
	if err := s.ensureMigrationTable(); err != nil {
		return err
	}

	entries, err := migrationFiles.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	migrationNames := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if filepath.Ext(name) != ".sql" {
			continue
		}
		migrationNames = append(migrationNames, name)
	}
	sort.Strings(migrationNames)

	for _, name := range migrationNames {
		applied, err := s.isMigrationApplied(name)
		if err != nil {
			return err
		}
		if applied {
			continue
		}
		if err := s.applyMigration(name); err != nil {
			return err
		}
	}
	return nil
}

func (s *PostgresStore) ensureMigrationTable() error {
	q := fmt.Sprintf(`
CREATE TABLE IF NOT EXISTS %s (
	version TEXT PRIMARY KEY,
	applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
)`, migrationTableName)

	if _, err := s.db.Exec(q); err != nil {
		return fmt.Errorf("ensure migration table: %w", err)
	}
	return nil
}

func (s *PostgresStore) isMigrationApplied(name string) (bool, error) {
	q := fmt.Sprintf(`SELECT 1 FROM %s WHERE version = $1 LIMIT 1`, migrationTableName)
	var one int
	err := s.db.QueryRow(q, name).Scan(&one)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return false, fmt.Errorf("check migration %s: %w", name, err)
}

func (s *PostgresStore) applyMigration(name string) error {
	raw, err := migrationFiles.ReadFile("migrations/" + name)
	if err != nil {
		return fmt.Errorf("read migration %s: %w", name, err)
	}

	sqlText := strings.ReplaceAll(string(raw), "{{table}}", s.table)
	sqlText = strings.ReplaceAll(sqlText, "{{dimension}}", fmt.Sprintf("%d", s.dimension))
	statements := splitSQLStatements(sqlText)
	if len(statements) == 0 {
		return nil
	}

	tx, err := s.db.Begin()
	if err != nil {
		return fmt.Errorf("begin migration %s: %w", name, err)
	}
	defer tx.Rollback()

	for _, stmt := range statements {
		if _, err := tx.Exec(stmt); err != nil {
			if !isIgnorableMigrationError(err) {
				return fmt.Errorf("execute migration %s: %w", name, err)
			}
		}
	}

	q := fmt.Sprintf(`INSERT INTO %s (version) VALUES ($1)`, migrationTableName)
	if _, err := tx.Exec(q, name); err != nil {
		return fmt.Errorf("mark migration %s as applied: %w", name, err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit migration %s: %w", name, err)
	}
	return nil
}

func splitSQLStatements(sqlText string) []string {
	parts := strings.Split(sqlText, ";")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		stmt := strings.TrimSpace(part)
		if stmt == "" {
			continue
		}
		out = append(out, stmt)
	}
	return out
}

func isIgnorableMigrationError(err error) bool {
	if err == nil {
		return false
	}
	e := strings.ToLower(err.Error())
	return strings.Contains(e, "already exists") ||
		strings.Contains(e, "duplicate")
}
