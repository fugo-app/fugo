package storage

import (
	"bufio"
	"database/sql"
	"fmt"
	"io"
	"log"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/fugo-app/fugo/internal/field"
)

type SQLiteStorage struct {
	Path string `yaml:"path"`

	// Default: "wal"
	JournalMode string `yaml:"journal_mode,omitempty"`

	// Default: "normal"
	Synchronous string `yaml:"synchronous,omitempty"`

	// Default: 10000
	CacheSize int `yaml:"cache_size,omitempty"`

	db          *sql.DB
	insertQueue chan *insertQueueItem
	stop        chan struct{}
}

type insertQueueItem struct {
	name string
	data map[string]any
}

func (ss *SQLiteStorage) Open() error {
	sourceName := ss.Path

	// Create parent directory if it doesn't exist
	if !strings.HasPrefix(sourceName, ":") {
		// Remove SQLite query parameters
		dir := filepath.Dir(sourceName)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("create directory for sqlite database: %w", err)
		}

		params := url.Values{}
		params.Set("_foreign_keys", "off")

		journalMode := ss.JournalMode
		if journalMode == "" {
			journalMode = "wal"
		}
		params.Set("_journal_mode", journalMode)

		synchronous := ss.Synchronous
		if synchronous == "" {
			synchronous = "normal"
		}
		params.Set("_synchronous", synchronous)

		cacheSize := ss.CacheSize
		if cacheSize == 0 {
			cacheSize = 10000
		}
		params.Set("_cache_size", fmt.Sprintf("%d", cacheSize))

		sourceName = fmt.Sprintf("%s?%s", sourceName, params.Encode())
	}

	db, err := sql.Open("sqlite3", sourceName)
	if err != nil {
		return fmt.Errorf("open sqlite database: %w", err)
	}
	ss.db = db

	ss.insertQueue = make(chan *insertQueueItem, 256)
	ss.stop = make(chan struct{})
	go ss.watch()

	return nil
}

func (ss *SQLiteStorage) Close() error {
	close(ss.stop)

	if err := ss.db.Close(); err != nil {
		return fmt.Errorf("close sqlite database: %w", err)
	}

	return nil
}

func (ss *SQLiteStorage) Migrate(name string, fields []*field.Field) error {
	exists, err := ss.checkTable(name)
	if err != nil {
		return fmt.Errorf("check table: %w", err)
	}

	if !exists {
		if err := ss.createTable(name, fields); err != nil {
			return fmt.Errorf("create table: %w", err)
		}
	} else {
		if err := ss.migrateTable(name, fields); err != nil {
			return fmt.Errorf("migrate table: %w", err)
		}
	}

	return nil
}

func (ss *SQLiteStorage) Cleanup(name string, field string, retention time.Duration) error {
	query := fmt.Sprintf("DELETE FROM `%s` WHERE `%s` < ?", name, field)
	value := time.Now().Add(-retention).Unix()

	_, err := ss.db.Exec(query, value)
	if err != nil {
		return fmt.Errorf("cleanup table: %w", err)
	}

	return nil
}

func (ss *SQLiteStorage) Write(name string, data map[string]any) {
	ss.insertQueue <- &insertQueueItem{name, data}
}

func (ss *SQLiteStorage) Query(w io.Writer, q *Query) error {
	query := fmt.Sprintf("SELECT * FROM `%s`", q.name)

	var (
		args       []any
		conditions []string
		reverse    bool
	)

	if q.after.Valid {
		reverse = false
		conditions = append(conditions, "_cursor > ?")
		args = append(args, q.after.Int64)
	} else if q.before.Valid {
		reverse = true
		conditions = append(conditions, "_cursor < ?")
		args = append(args, q.before.Int64)
	} else {
		reverse = true
	}

	for _, filter := range q.filters {
		switch filter.op {
		case Eq:
			conditions = append(conditions, fmt.Sprintf("`%s` = ?", filter.name))
			args = append(args, filter.ival)
		case Ne:
			conditions = append(conditions, fmt.Sprintf("`%s` != ?", filter.name))
			args = append(args, filter.ival)
		case Lt:
			conditions = append(conditions, fmt.Sprintf("`%s` < ?", filter.name))
			args = append(args, filter.ival)
		case Lte:
			conditions = append(conditions, fmt.Sprintf("`%s` <= ?", filter.name))
			args = append(args, filter.ival)
		case Gt:
			conditions = append(conditions, fmt.Sprintf("`%s` > ?", filter.name))
			args = append(args, filter.ival)
		case Gte:
			conditions = append(conditions, fmt.Sprintf("`%s` >= ?", filter.name))
			args = append(args, filter.ival)
		case Exact:
			conditions = append(conditions, fmt.Sprintf("`%s` = ?", filter.name))
			args = append(args, filter.sval)
		case Like:
			conditions = append(conditions, fmt.Sprintf("`%s` LIKE ?", filter.name))
			value := "%" + filter.sval + "%"
			args = append(args, value)
		case Prefix:
			conditions = append(conditions, fmt.Sprintf("`%s` LIKE ?", filter.name))
			value := filter.sval + "%"
			args = append(args, value)
		case Suffix:
			conditions = append(conditions, fmt.Sprintf("`%s` LIKE ?", filter.name))
			value := "%" + filter.sval
			args = append(args, value)
		case Since:
			if q.after.Valid {
				// Could be used only with before-cursor. For example:
				// Get all records since defined date but before cursor.
				return nil
			}
			reverse = false
			conditions = append(conditions, fmt.Sprintf("`%s` > ?", filter.name))
			args = append(args, filter.ival)
		case Until:
			if q.before.Valid {
				// Could be used only with after-cursor. For example:
				// Get all records after defined cursor but until date.
				return nil
			}
			reverse = true
			conditions = append(conditions, fmt.Sprintf("`%s` < ?", filter.name))
			args = append(args, filter.ival)
		}
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	if reverse {
		query += " ORDER BY _cursor DESC"
	} else {
		query += " ORDER BY _cursor ASC"
	}

	if q.limit.Valid {
		query += " LIMIT ?"
		args = append(args, q.limit.Int64)
	}

	query = "SELECT * FROM ( " + query + " ) temp ORDER BY _cursor ASC"

	rows, err := ss.db.Query(query, args...)
	if err != nil {
		return err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return err
	}

	out := bufio.NewWriter(w)
	defer out.Flush()

	values := make([]any, len(columns))
	pointers := make([]any, len(columns))
	for i := range values {
		pointers[i] = &values[i]
	}

	for rows.Next() {
		if err := rows.Scan(pointers...); err != nil {
			return err
		}

		writeJsonLine(out, columns, values)
	}

	return rows.Err()
}

func (ss *SQLiteStorage) getSqlType(f *field.Field) string {
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

func (s *SQLiteStorage) checkTable(name string) (bool, error) {
	var tableExists bool
	const checkQuery = `SELECT COUNT(*) > 0 FROM sqlite_master WHERE type = 'table' AND name = ?`
	err := s.db.QueryRow(checkQuery, name).Scan(&tableExists)
	if err != nil && err != sql.ErrNoRows {
		return false, err
	}

	return tableExists, nil
}

func (ss *SQLiteStorage) getColumns(name string) (map[string]string, error) {
	query := fmt.Sprintf("PRAGMA table_info(`%s`)", name)
	rows, err := ss.db.Query(query)
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

func (ss *SQLiteStorage) createTable(name string, fields []*field.Field) error {
	var columns []string

	columns = append(columns, "`_cursor` INTEGER PRIMARY KEY AUTOINCREMENT")

	for _, f := range fields {
		fieldType := ss.getSqlType(f)
		columns = append(columns, fmt.Sprintf("`%s` %s", f.Name, fieldType))
	}

	createTableSQL := fmt.Sprintf("CREATE TABLE `%s` (%s)", name, strings.Join(columns, ", "))

	_, err := ss.db.Exec(createTableSQL)
	return err
}

func (ss *SQLiteStorage) migrateTable(name string, fields []*field.Field) error {
	currentColumns, err := ss.getColumns(name)
	if err != nil {
		return fmt.Errorf("get columns: %w", err)
	}

	desiredColumns := make(map[string]string)
	for _, f := range fields {
		desiredColumns[f.Name] = ss.getSqlType(f)
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
			if _, err := ss.db.Exec(alterQuery); err != nil {
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
			if _, err := ss.db.Exec(alterQuery); err != nil {
				return fmt.Errorf("add column %s: %w", desiredName, err)
			}
		}
	}

	return nil
}

func (ss *SQLiteStorage) insertData(name string, data map[string]any) error {
	columns := []string{}
	placeholders := []string{}
	values := []any{}

	for col, val := range data {
		columns = append(columns, fmt.Sprintf("`%s`", col)) // экранируем имя столбца
		placeholders = append(placeholders, "?")
		values = append(values, val)
	}

	query := fmt.Sprintf(
		"INSERT INTO `%s` (%s) VALUES (%s)",
		name,
		strings.Join(columns, ", "),
		strings.Join(placeholders, ", "),
	)

	_, err := ss.db.Exec(query, values...)
	return err
}

func (ss *SQLiteStorage) watch() {
	for {
		select {
		case <-ss.stop:
			return
		case item := <-ss.insertQueue:
			if err := ss.insertData(item.name, item.data); err != nil {
				log.Printf("failed to insert log record into sqlite storage: %v", err)
			}
		}
	}
}
