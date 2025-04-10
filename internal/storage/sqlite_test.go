package storage

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/fugo-app/fugo/internal/field"
)

func TestSQLiteStorage_createTable(t *testing.T) {
	storage := &SQLiteStorage{Path: ":memory:"}
	require.NoError(t, storage.Open(), "Failed to open SQLite database")
	defer storage.Close()

	name := "test_agent"
	fields := initFields(t, []*field.Field{
		{
			Name: "timestamp",
			Timestamp: &field.TimestampFormat{
				Format: "2006-01-02 15:04:05",
			},
		},
		{Name: "level", Type: "string"},
		{Name: "message", Type: "string"},
		{Name: "count", Type: "int"},
		{Name: "value", Type: "float"},
	})

	t.Run("create table", func(t *testing.T) {
		require.NoError(t, storage.createTable(name, fields), "Failed to create table")
		verifySqliteDB(t, storage, name, fields)
	})

	t.Run("check table exists", func(t *testing.T) {
		exists, err := storage.checkTable(name)
		require.NoError(t, err, "Failed to check if table exists")
		require.True(t, exists, "Table should exist after creation")
	})

	t.Run("check table structure", func(t *testing.T) {
		columns, err := storage.getColumns(name)
		require.NoError(t, err, "Failed to get table columns")

		expectedColumns := map[string]string{
			"timestamp": "INTEGER",
			"level":     "TEXT",
			"message":   "TEXT",
			"count":     "INTEGER",
			"value":     "REAL",
		}

		require.Equal(t, expectedColumns, columns, "Table columns do not match expected structure")
	})

	t.Run("check _cursor column", func(t *testing.T) {
		rows, err := storage.db.Query(fmt.Sprintf("PRAGMA table_info(`%s`)", name))
		require.NoError(t, err, "Failed to query table info")
		defer rows.Close()

		hasCursor := false
		for rows.Next() {
			var (
				cid     int
				name    string
				ctype   string
				notnull int
				dflt    sql.NullString
				pk      int
			)

			err := rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk)
			require.NoError(t, err, "Failed to scan column info")

			if name == "_cursor" {
				hasCursor = true

				require.Equal(t, 1, pk, "Cursor column should be primary key")
			}
		}

		require.True(t, hasCursor, "Table should have _cursor column")
	})
}

func TestSQLiteStorage_migrateTable_AddColumn(t *testing.T) {
	storage := &SQLiteStorage{Path: ":memory:"}
	require.NoError(t, storage.Open(), "Failed to open SQLite database")
	defer storage.Close()

	// Create initial agent with some fields
	name := "test_migration"

	t.Run("create table", func(t *testing.T) {
		fields := initFields(t, []*field.Field{
			{
				Name: "timestamp",
				Timestamp: &field.TimestampFormat{
					Format: "2006-01-02 15:04:05",
				},
			},
			{Name: "level", Type: "string"},
			{Name: "message", Type: "string"},
		})

		require.NoError(t, storage.createTable(name, fields), "Failed to create table")
		verifySqliteDB(t, storage, name, fields)
	})

	// Create updated agent with additional fields
	t.Run("migrate table", func(t *testing.T) {
		fields := initFields(t, []*field.Field{
			{
				Name: "timestamp",
				Timestamp: &field.TimestampFormat{
					Format: "2006-01-02 15:04:05",
				},
			},
			{Name: "level", Type: "string"},
			{Name: "message", Type: "string"},
			{Name: "count", Type: "int"},    // New column
			{Name: "severity", Type: "int"}, // New column
		})

		require.NoError(t, storage.migrateTable(name, fields), "Failed to migrate table")
		verifySqliteDB(t, storage, name, fields)
	})
}

func TestSQLiteStorage_migrateTable_RemoveColumn(t *testing.T) {
	storage := &SQLiteStorage{Path: ":memory:"}
	require.NoError(t, storage.Open(), "Failed to open SQLite database")
	defer storage.Close()

	// Create initial agent with some fields
	name := "test_removal"

	t.Run("create table", func(t *testing.T) {
		fields := initFields(t, []*field.Field{
			{
				Name: "timestamp",
				Timestamp: &field.TimestampFormat{
					Format: "2006-01-02 15:04:05",
				},
			},
			{Name: "level", Type: "string"},
			{Name: "message", Type: "string"},
			{Name: "count", Type: "int"},
			{Name: "value", Type: "float"},
		})

		require.NoError(t, storage.createTable(name, fields), "Failed to create table")
		verifySqliteDB(t, storage, name, fields)
	})

	// Create updated agent with fewer fields
	t.Run("migrate table", func(t *testing.T) {
		fields := initFields(t, []*field.Field{
			{
				Name: "timestamp",
				Timestamp: &field.TimestampFormat{
					Format: "2006-01-02 15:04:05",
				},
			},
			{Name: "message", Type: "string"}, // Keep these columns
			{Name: "value", Type: "float"},    // Keep these columns
			// Removed "level" and "count" columns
		})

		require.NoError(t, storage.migrateTable(name, fields), "Failed to migrate table")
		verifySqliteDB(t, storage, name, fields)
	})
}

func TestSQLiteStorage_MigrateChangeColumnType(t *testing.T) {
	storage := &SQLiteStorage{Path: ":memory:"}
	require.NoError(t, storage.Open(), "Failed to open SQLite database")
	defer storage.Close()

	// Create initial agent with some fields
	name := "test_type_change"

	t.Run("create table", func(t *testing.T) {
		fields := initFields(t, []*field.Field{
			{
				Name: "timestamp",
				Timestamp: &field.TimestampFormat{
					Format: "2006-01-02 15:04:05",
				},
			},
			{Name: "level", Type: "string"},
			{Name: "count", Type: "int"},     // This will be changed to float
			{Name: "status", Type: "string"}, // This will be changed to int
		})

		require.NoError(t, storage.createTable(name, fields), "Failed to create table")
		verifySqliteDB(t, storage, name, fields)
	})

	// Create updated agent with changed column types
	t.Run("migrate table", func(t *testing.T) {
		fields := initFields(t, []*field.Field{
			{
				Name: "timestamp",
				Timestamp: &field.TimestampFormat{
					Format: "2006-01-02 15:04:05",
				},
			},
			{Name: "level", Type: "string"},
			{Name: "count", Type: "float"}, // Changed from int to float
			{Name: "status", Type: "int"},  // Changed from string to int
		})

		require.NoError(t, storage.migrateTable(name, fields), "Failed to migrate table")
		verifySqliteDB(t, storage, name, fields)
	})
}

func TestSQLiteStorage_insertData(t *testing.T) {
	storage := &SQLiteStorage{Path: ":memory:"}
	require.NoError(t, storage.Open(), "Failed to open SQLite database")
	defer storage.Close()

	name := "test_insert"
	fields := initFields(t, []*field.Field{
		{
			Name: "timestamp",
			Timestamp: &field.TimestampFormat{
				Format: "2006-01-02 15:04:05",
			},
		},
		{Name: "level", Type: "string"},
		{Name: "message", Type: "string"},
		{Name: "count", Type: "int"},
		{Name: "value", Type: "float"},
	})

	// Create the table
	require.NoError(t, storage.createTable(name, fields), "Failed to create table")

	// Test inserting data
	testData := map[string]any{
		"timestamp": int64(1648735200000),
		"level":     "info",
		"message":   "test message",
		"count":     42,
		"value":     3.14,
	}

	// Insert the data
	err := storage.insertData(name, testData)
	require.NoError(t, err, "Failed to insert data")

	// Verify the data was inserted correctly
	row := storage.db.QueryRow(fmt.Sprintf("SELECT timestamp, level, message, count, value FROM `%s` LIMIT 1", name))

	var (
		timestamp int64
		level     string
		message   string
		count     int
		value     float64
	)

	err = row.Scan(&timestamp, &level, &message, &count, &value)
	require.NoError(t, err, "Failed to query inserted data")

	require.Equal(t, testData["timestamp"], timestamp, "Timestamp value mismatch")
	require.Equal(t, testData["level"], level, "Level value mismatch")
	require.Equal(t, testData["message"], message, "Message value mismatch")
	require.Equal(t, testData["count"], count, "Count value mismatch")
	require.Equal(t, testData["value"], value, "Value value mismatch")
}

func TestSQLiteStorage_Query(t *testing.T) {
	storage := &SQLiteStorage{Path: ":memory:"}
	require.NoError(t, storage.Open(), "Failed to open SQLite database")
	defer storage.Close()

	name := "test_query"
	fields := initFields(t, []*field.Field{
		{Name: "message", Type: "string"},
		{Name: "status", Type: "int"},
	})

	// Create the table
	require.NoError(t, storage.createTable(name, fields), "Failed to create table")

	type logRecord struct {
		Cursor  int64  `json:"_cursor"`
		Message string `json:"message"`
		Status  int    `json:"status"`
	}

	// Insert test data
	testData := []map[string]any{
		{"message": "item1", "status": 200},
		{"message": "item2", "status": 404},
		{"message": "item3", "status": 403},
		{"message": "item4", "status": 500},
		{"message": "item5", "status": 400},
	}

	for _, data := range testData {
		require.NoError(t, storage.insertData(name, data), "Failed to insert test data")
	}

	t.Run("query all records", func(t *testing.T) {
		query := NewQuery(name)
		buf := new(bytes.Buffer)

		require.NoError(t, storage.Query(buf, query), "Failed to execute query")

		lines := bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte{'\n'})
		require.Len(t, lines, 5, "Expected 5 records in output") // 5 records

		var record logRecord
		for i, line := range lines {
			offset := i
			require.NoError(t, json.Unmarshal(line, &record), "Failed to unmarshal JSON line %d", i)
			require.Equal(t, int64(offset+1), record.Cursor, "Cursor value mismatch for record %d", i)
			require.Equal(t, testData[offset]["message"], record.Message, "Message value mismatch for record %d", i)
			require.Equal(t, testData[offset]["status"], record.Status, "Status value mismatch for record %d", i)
		}
	})

	t.Run("query with limit", func(t *testing.T) {
		// Input: 1, 2, 3, 4, 5
		// Output with limit: 3, 4, 5
		query := NewQuery(name)
		query.SetLimit(3)
		buf := new(bytes.Buffer)

		require.NoError(t, storage.Query(buf, query), "Failed to execute query")

		lines := bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte{'\n'})
		require.Len(t, lines, 3, "Expected 3 records in output")

		var record logRecord
		for i, line := range lines {
			require.NoError(t, json.Unmarshal(line, &record), "Failed to unmarshal JSON line %d", i)
			require.Equal(t, int64(i+3), record.Cursor, "Cursor value mismatch for record %d", i)
			require.Equal(t, testData[i+2]["message"], record.Message, "Message value mismatch for record %d", i)
			require.Equal(t, testData[i+2]["status"], record.Status, "Status value mismatch for record %d", i)
		}
	})

	t.Run("query with after cursor", func(t *testing.T) {
		// Input: 1, 2, 3, 4, 5
		// Output without limit: 3, 4, 5
		// Output with limit: 3, 4
		query := NewQuery(name)
		query.SetLimit(2)
		query.SetAfter(2) // After second record
		buf := new(bytes.Buffer)

		require.NoError(t, storage.Query(buf, query), "Failed to execute query with after cursor")

		lines := bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte{'\n'})
		require.Len(t, lines, 2, "Expected 2 records after cursor")

		var record logRecord
		for i, line := range lines {
			require.NoError(t, json.Unmarshal(line, &record), "Failed to unmarshal JSON line %d", i)
			require.Equal(t, int64(i+3), record.Cursor, "Cursor value mismatch for record %d", i)
		}
	})

	t.Run("query with before cursor", func(t *testing.T) {
		// Input: 1, 2, 3, 4, 5
		// Output without limit: 1, 2, 3
		// Output with limit: 2, 3
		query := NewQuery(name)
		query.SetLimit(2)
		query.SetBefore(4) // Before fourth record
		buf := new(bytes.Buffer)

		require.NoError(t, storage.Query(buf, query), "Failed to execute query with before cursor")

		lines := bytes.Split(bytes.TrimSpace(buf.Bytes()), []byte{'\n'})
		require.Len(t, lines, 2, "Expected 2 records after cursor")

		var record logRecord
		for i, line := range lines {
			require.NoError(t, json.Unmarshal(line, &record), "Failed to unmarshal JSON line %d", i)
			require.Equal(t, int64(i+2), record.Cursor, "Cursor value mismatch for record %d", i)
		}
	})
}

func initFields(t *testing.T, fields []*field.Field) []*field.Field {
	for _, f := range fields {
		require.NoError(t, f.Init(), "Failed to initialize field: %s", f.Name)
	}

	return fields
}

// verifyTableStructure checks that the table was created with the correct columns
func verifySqliteDB(t *testing.T, storage *SQLiteStorage, name string, fields []*field.Field) {
	tableExists, err := storage.checkTable(name)
	require.NoError(t, err, "Failed to check if table exists")
	require.True(t, tableExists, "Table does not exist")

	currentColumns, err := storage.getColumns(name)
	require.NoError(t, err, "Failed to get table columns")

	expectedColumns := make(map[string]string)
	for _, f := range fields {
		expectedColumns[f.Name] = storage.getSqlType(f)
	}

	require.Equal(
		t,
		expectedColumns,
		currentColumns,
		"Table columns do not match expected structure",
	)
}
