package agent

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	_ "github.com/mattn/go-sqlite3"

	"github.com/fugo-app/fugo/internal/field"
	"github.com/fugo-app/fugo/internal/source/file"
)

type Agent struct {
	name string

	// Fields to include in the final log record.
	Fields []field.Field `yaml:"fields"`

	// File-based log source.
	File *file.FileWatcher `yaml:"file,omitempty"`
}

func (a *Agent) Init(name string) error {
	if name == "" {
		return fmt.Errorf("name is required")
	}
	a.name = name

	if len(a.Fields) == 0 {
		return fmt.Errorf("fields are required")
	}

	for i := range a.Fields {
		field := &a.Fields[i]
		if err := field.Init(); err != nil {
			return fmt.Errorf("field %s init: %w", field.Name, err)
		}
	}

	if a.File != nil {
		if err := a.File.Init(a); err != nil {
			return fmt.Errorf("file agent init: %w", err)
		}
	}

	return nil
}

func (a *Agent) Start() {
	if a.File != nil {
		a.File.Start()
	}
}

func (a *Agent) Stop() {
	if a.File != nil {
		a.File.Stop()
	}
}

// Process receives raw data from source and converts to the logs record.
func (a *Agent) Process(data map[string]string) {
	if len(data) == 0 {
		return
	}

	result := make(map[string]any)

	for i := range a.Fields {
		field := &a.Fields[i]
		if val, err := field.Convert(data); err == nil && val != nil {
			result[field.Name] = val
		} else {
			result[field.Name] = field.Default()
		}
	}

	line, _ := json.Marshal(result)
	fmt.Println(a.name, string(line))

	// TODO: send data to sink
}

// Migrate checks if the agent's table exists, and creates it if it doesn't.
func (a *Agent) Migrate(db *sql.DB) error {
	if db == nil {
		return fmt.Errorf("database connection is required")
	}

	tableExists, err := a.checkTable(db)
	if err != nil {
		return fmt.Errorf("check table: %w", err)
	}

	if !tableExists {
		if err := a.createTable(db); err != nil {
			return fmt.Errorf("create table: %w", err)
		}

		return nil
	}

	currentColumns, err := a.getColumns(db)
	if err != nil {
		return fmt.Errorf("get columns: %w", err)
	}

	desiredColumns := make(map[string]string)
	for _, f := range a.Fields {
		desiredColumns[f.Name] = getSqlType(f.Type)
	}

	for name, currentType := range currentColumns {
		desiredType, exists := desiredColumns[name]
		if exists && currentType != desiredType {
			delete(currentColumns, name)
			exists = false
		}

		if !exists {
			alterQuery := fmt.Sprintf("ALTER TABLE `%s` DROP COLUMN `%s`", a.name, name)
			if _, err := db.Exec(alterQuery); err != nil {
				return fmt.Errorf("drop column %s: %w", name, err)
			}
		}
	}

	for name, desiredType := range desiredColumns {
		if _, exists := currentColumns[name]; !exists {
			alterQuery := fmt.Sprintf("ALTER TABLE `%s` ADD COLUMN `%s` %s", a.name, name, desiredType)
			if _, err := db.Exec(alterQuery); err != nil {
				return fmt.Errorf("add column %s: %w", name, err)
			}
		}
	}

	return nil
}

func (a *Agent) checkTable(db *sql.DB) (bool, error) {
	var tableExists bool
	const checkQuery = `SELECT COUNT(*) > 0 FROM sqlite_master WHERE type = 'table' AND name = ?`
	err := db.QueryRow(checkQuery, a.name).Scan(&tableExists)
	if err != nil && err != sql.ErrNoRows {
		return false, err
	}

	return tableExists, nil
}

// createTable creates a new table in the SQLite database with the agent's name.
func (a *Agent) createTable(db *sql.DB) error {
	var columns []string
	for _, f := range a.Fields {
		fieldType := getSqlType(f.Type)
		columns = append(columns, fmt.Sprintf("`%s` %s", f.Name, fieldType))
	}

	createTableSQL := fmt.Sprintf("CREATE TABLE `%s` (%s)", a.name, strings.Join(columns, ", "))

	_, err := db.Exec(createTableSQL)
	return err
}

func (a *Agent) getColumns(db *sql.DB) (map[string]string, error) {
	query := fmt.Sprintf("PRAGMA table_info(`%s`)", a.name)
	rows, err := db.Query(query)
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
		columns[name] = ctype
	}

	return columns, nil
}

func getSqlType(fieldType string) string {
	switch fieldType {
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
