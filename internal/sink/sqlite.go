package sink

import (
	"database/sql"
	"fmt"
	"log"
	"strings"

	_ "github.com/mattn/go-sqlite3"

	"github.com/fugo-app/fugo/internal/field"
)

type SQLiteSink struct {
	Path string `yaml:"path"`

	db *sql.DB
}

func (s *SQLiteSink) Open() error {
	db, err := sql.Open("sqlite3", s.Path)
	if err != nil {
		return fmt.Errorf("failed to open sqlite sink: %w", err)
	}
	s.db = db
	return nil
}

func (s *SQLiteSink) Close() {
	if err := s.db.Close(); err != nil {
		log.Printf("failed to close sqlite sink: %v\n", err)
	}
}

func (s *SQLiteSink) Migrate(name string, fields []*field.Field) error {
	exists, err := s.checkTable(name)
	if err != nil {
		return fmt.Errorf("check table: %w", err)
	}

	if !exists {
		if err := s.createTable(name, fields); err != nil {
			return fmt.Errorf("create table: %w", err)
		}
	} else {
		if err := s.migrateTable(name, fields); err != nil {
			return fmt.Errorf("migrate table: %w", err)
		}
	}

	return nil
}

func (s *SQLiteSink) getSqlType(f *field.Field) string {
	switch f.Type {
	case "string":
		return "TEXT"
	case "int", "time":
		return "INTEGER"
	case "float":
		return "REAL"
	default:
		return "TEXT"
	}
}

func (s *SQLiteSink) checkTable(name string) (bool, error) {
	var tableExists bool
	const checkQuery = `SELECT COUNT(*) > 0 FROM sqlite_master WHERE type = 'table' AND name = ?`
	err := s.db.QueryRow(checkQuery, name).Scan(&tableExists)
	if err != nil && err != sql.ErrNoRows {
		return false, err
	}

	return tableExists, nil
}

func (s *SQLiteSink) getColumns(name string) (map[string]string, error) {
	query := fmt.Sprintf("PRAGMA table_info(`%s`)", name)
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("query table info: %w", err)
	}
	defer rows.Close()

	columns := make(map[string]string)
	for rows.Next() {
		var (
			cid     int
			name    string
			ctype   string
			notnull int
			dflt    sql.NullString
			pk      int
		)
		if err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk); err != nil {
			return nil, fmt.Errorf("scan column info: %w", err)
		}

		// Ignore internal columns
		if strings.HasPrefix(name, "_") {
			continue
		}

		columns[name] = ctype
	}

	return columns, nil
}

func (s *SQLiteSink) createTable(name string, fields []*field.Field) error {
	var columns []string

	columns = append(columns, "`_cursor` INTEGER PRIMARY KEY AUTOINCREMENT")

	for _, f := range fields {
		fieldType := s.getSqlType(f)
		columns = append(columns, fmt.Sprintf("`%s` %s", f.Name, fieldType))
	}

	createTableSQL := fmt.Sprintf("CREATE TABLE `%s` (%s)", name, strings.Join(columns, ", "))

	_, err := s.db.Exec(createTableSQL)
	return err
}

func (s *SQLiteSink) migrateTable(name string, fields []*field.Field) error {
	currentColumns, err := s.getColumns(name)
	if err != nil {
		return fmt.Errorf("get columns: %w", err)
	}

	desiredColumns := make(map[string]string)
	for _, f := range fields {
		desiredColumns[f.Name] = s.getSqlType(f)
	}

	for currentName, currentType := range currentColumns {
		desiredType, exists := desiredColumns[currentName]
		if exists && currentType != desiredType {
			delete(currentColumns, currentName)
			exists = false
		}

		if !exists {
			alterQuery := fmt.Sprintf(
				"ALTER TABLE `%s` DROP COLUMN `%s`",
				name,
				currentName,
			)
			if _, err := s.db.Exec(alterQuery); err != nil {
				return fmt.Errorf("drop column %s: %w", currentName, err)
			}
		}
	}

	for desiredName, desiredType := range desiredColumns {
		if _, exists := currentColumns[desiredName]; !exists {
			alterQuery := fmt.Sprintf(
				"ALTER TABLE `%s` ADD COLUMN `%s` %s",
				name,
				desiredName,
				desiredType,
			)
			if _, err := s.db.Exec(alterQuery); err != nil {
				return fmt.Errorf("add column %s: %w", desiredName, err)
			}
		}
	}

	return nil
}
