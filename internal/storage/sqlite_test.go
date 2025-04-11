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
		Cursor  string `json:"_cursor"`
		Message string `json:"message"`
		Status  int    `json:"status"`
	}

	// Insert test data
	testData := []map[string]any{
		{"message": "apple pie", "status": 200},
		{"message": "pineapple juice", "status": 404},
		{"message": "grapefruit", "status": 403},
		{"message": "apple", "status": 500},
		{"message": "green apple", "status": 400},
	}

	for _, data := range testData {
		require.NoError(t, storage.insertData(name, data), "Failed to insert test data")
	}

	tests := []struct {
		name     string
		modifier func(q *Query)
		want     []logRecord
	}{
		{
			name:     "query all records",
			modifier: func(q *Query) {},
			want: []logRecord{
				{Cursor: "0000000000000001", Message: "apple pie", Status: 200},
				{Cursor: "0000000000000002", Message: "pineapple juice", Status: 404},
				{Cursor: "0000000000000003", Message: "grapefruit", Status: 403},
				{Cursor: "0000000000000004", Message: "apple", Status: 500},
				{Cursor: "0000000000000005", Message: "green apple", Status: 400},
			},
		},
		{
			name: "query with limit",
			modifier: func(q *Query) {
				q.SetLimit(3)
			},
			want: []logRecord{
				{Cursor: "0000000000000003", Message: "grapefruit", Status: 403},
				{Cursor: "0000000000000004", Message: "apple", Status: 500},
				{Cursor: "0000000000000005", Message: "green apple", Status: 400},
			},
		},
		{
			// Input: 1, 2, 3, 4, 5
			// Output without limit: 3, 4, 5
			// Output with limit: 3, 4
			name: "query with after cursor",
			modifier: func(q *Query) {
				q.SetLimit(2)
				q.SetAfter(2) // After second record
			},
			want: []logRecord{
				{Cursor: "0000000000000003", Message: "grapefruit", Status: 403},
				{Cursor: "0000000000000004", Message: "apple", Status: 500},
			},
		},
		{
			// Input: 1, 2, 3, 4, 5
			// Output without limit: 1, 2, 3
			// Output with limit: 2, 3
			name: "query with before cursor",
			modifier: func(q *Query) {
				q.SetLimit(2)
				q.SetBefore(4) // Before fourth record
			},
			want: []logRecord{
				{Cursor: "0000000000000002", Message: "pineapple juice", Status: 404},
				{Cursor: "0000000000000003", Message: "grapefruit", Status: 403},
			},
		},
		{
			name: "eq filter",
			modifier: func(q *Query) {
				q.SetFilter("status", "eq", "403")
			},
			want: []logRecord{
				{Cursor: "0000000000000003", Message: "grapefruit", Status: 403},
			},
		},
		{
			name: "ne filter",
			modifier: func(q *Query) {
				q.SetFilter("status", "ne", "403")
			},
			want: []logRecord{
				{Cursor: "0000000000000001", Message: "apple pie", Status: 200},
				{Cursor: "0000000000000002", Message: "pineapple juice", Status: 404},
				{Cursor: "0000000000000004", Message: "apple", Status: 500},
				{Cursor: "0000000000000005", Message: "green apple", Status: 400},
			},
		},
		{
			name: "lt filter",
			modifier: func(q *Query) {
				q.SetFilter("status", "lt", "403")
			},
			want: []logRecord{
				{Cursor: "0000000000000001", Message: "apple pie", Status: 200},
				{Cursor: "0000000000000005", Message: "green apple", Status: 400},
			},
		},
		{
			name: "lte filter",
			modifier: func(q *Query) {
				q.SetFilter("status", "lte", "403")
			},
			want: []logRecord{
				{Cursor: "0000000000000001", Message: "apple pie", Status: 200},
				{Cursor: "0000000000000003", Message: "grapefruit", Status: 403},
				{Cursor: "0000000000000005", Message: "green apple", Status: 400},
			},
		},
		{
			name: "gt filter",
			modifier: func(q *Query) {
				q.SetFilter("status", "gt", "403")
			},
			want: []logRecord{
				{Cursor: "0000000000000002", Message: "pineapple juice", Status: 404},
				{Cursor: "0000000000000004", Message: "apple", Status: 500},
			},
		},
		{
			name: "gte filter",
			modifier: func(q *Query) {
				q.SetFilter("status", "gte", "403")
			},
			want: []logRecord{
				{Cursor: "0000000000000002", Message: "pineapple juice", Status: 404},
				{Cursor: "0000000000000003", Message: "grapefruit", Status: 403},
				{Cursor: "0000000000000004", Message: "apple", Status: 500},
			},
		},
		{
			name: "exact filter",
			modifier: func(q *Query) {
				q.SetFilter("message", "exact", "apple")
			},
			want: []logRecord{
				{Cursor: "0000000000000004", Message: "apple", Status: 500},
			},
		},
		{
			name: "like filter",
			modifier: func(q *Query) {
				q.SetFilter("message", "like", "apple")
			},
			want: []logRecord{
				{Cursor: "0000000000000001", Message: "apple pie", Status: 200},
				{Cursor: "0000000000000002", Message: "pineapple juice", Status: 404},
				{Cursor: "0000000000000004", Message: "apple", Status: 500},
				{Cursor: "0000000000000005", Message: "green apple", Status: 400},
			},
		},
		{
			name: "prefix filter",
			modifier: func(q *Query) {
				q.SetFilter("message", "prefix", "apple")
			},
			want: []logRecord{
				{Cursor: "0000000000000001", Message: "apple pie", Status: 200},
				{Cursor: "0000000000000004", Message: "apple", Status: 500},
			},
		},
		{
			name: "suffix filter",
			modifier: func(q *Query) {
				q.SetFilter("message", "suffix", "apple")
			},
			want: []logRecord{
				{Cursor: "0000000000000004", Message: "apple", Status: 500},
				{Cursor: "0000000000000005", Message: "green apple", Status: 400},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := NewQuery(name)
			tt.modifier(query)
			buf := new(bytes.Buffer)

			require.NoError(t, storage.Query(buf, query), "Failed to execute query")
			payload := bytes.TrimSpace(buf.Bytes())
			lines := [][]byte{}
			if len(payload) > 0 {
				lines = bytes.Split(payload, []byte{'\n'})
			}
			require.Len(t, lines, len(tt.want), "Expected %d records after query", len(tt.want))

			var record logRecord
			for i, line := range lines {
				require.NoError(t, json.Unmarshal(line, &record), "Failed to unmarshal JSON")
				require.Equal(t, tt.want[i], record, "Record mismatch")
			}
		})
	}
}

func TestSQLiteStorage_Query_time(t *testing.T) {
	storage := &SQLiteStorage{Path: ":memory:"}
	require.NoError(t, storage.Open(), "Failed to open SQLite database")
	defer storage.Close()

	name := "test_query"
	fields := initFields(t, []*field.Field{
		{
			Name: "time",
			Timestamp: &field.TimestampFormat{
				Format: "stamp",
			},
		},
	})

	// Create the table
	require.NoError(t, storage.createTable(name, fields), "Failed to create table")

	type logRecord struct {
		Cursor string `json:"_cursor"`
		Time   int64  `json:"time"`
	}

	// Insert test data
	testData := []map[string]any{
		{"time": 1735812000000}, // 2025-01-02 10:00:00
		{"time": 1735817400000}, // 2025-01-02 11:30:00
		{"time": 1735823700000}, // 2025-01-02 13:15:00
		{"time": 1735829100000}, // 2025-01-02 14:45:00
		{"time": 1735833600000}, // 2025-01-02 16:00:00
	}

	for _, data := range testData {
		require.NoError(t, storage.insertData(name, data), "Failed to insert test data")
	}

	tests := []struct {
		name     string
		modifier func(q *Query)
		want     []logRecord
	}{
		{
			name:     "query all records",
			modifier: func(q *Query) {},
			want: []logRecord{
				{Cursor: "0000000000000001", Time: 1735812000000},
				{Cursor: "0000000000000002", Time: 1735817400000},
				{Cursor: "0000000000000003", Time: 1735823700000},
				{Cursor: "0000000000000004", Time: 1735829100000},
				{Cursor: "0000000000000005", Time: 1735833600000},
			},
		},
		{
			name: "since filter",
			modifier: func(q *Query) {
				q.SetFilter("time", "since", "2025-01-02 13:00:00")
			},
			want: []logRecord{
				{Cursor: "0000000000000003", Time: 1735823700000},
				{Cursor: "0000000000000004", Time: 1735829100000},
				{Cursor: "0000000000000005", Time: 1735833600000},
			},
		},
		{
			name: "until filter",
			modifier: func(q *Query) {
				q.SetFilter("time", "until", "2025-01-02 13:00:00")
			},
			want: []logRecord{
				{Cursor: "0000000000000001", Time: 1735812000000},
				{Cursor: "0000000000000002", Time: 1735817400000},
			},
		},
		{
			name: "since filter with limit",
			modifier: func(q *Query) {
				q.SetFilter("time", "since", "2025-01-02 13:00:00")
				q.SetLimit(2)
			},
			want: []logRecord{
				{Cursor: "0000000000000003", Time: 1735823700000},
				{Cursor: "0000000000000004", Time: 1735829100000},
			},
		},
		{
			name: "until filter with limit",
			modifier: func(q *Query) {
				q.SetFilter("time", "until", "2025-01-02 14:00:00")
				q.SetLimit(2)
			},
			want: []logRecord{
				{Cursor: "0000000000000002", Time: 1735817400000},
				{Cursor: "0000000000000003", Time: 1735823700000},
			},
		},
		{
			name: "since filter with after cursor",
			modifier: func(q *Query) {
				q.SetFilter("time", "since", "2025-01-02 13:00:00")
				q.SetAfter(2)
			},
			want: []logRecord{}, // No records should be returned
		},
		{
			name: "until filter with before cursor",
			modifier: func(q *Query) {
				q.SetFilter("time", "until", "2025-01-02 14:00:00")
				q.SetBefore(2)
			},
			want: []logRecord{}, // No records should be returned
		},
		{
			name: "since filter with before cursor",
			modifier: func(q *Query) {
				q.SetFilter("time", "since", "2025-01-02 13:00:00")
				q.SetBefore(5)
			},
			want: []logRecord{
				{Cursor: "0000000000000003", Time: 1735823700000},
				{Cursor: "0000000000000004", Time: 1735829100000},
			},
		},
		{
			name: "until filter with after cursor",
			modifier: func(q *Query) {
				q.SetFilter("time", "until", "2025-01-02 14:00:00")
				q.SetAfter(1)
			},
			want: []logRecord{
				{Cursor: "0000000000000002", Time: 1735817400000},
				{Cursor: "0000000000000003", Time: 1735823700000},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			query := NewQuery(name)
			tt.modifier(query)
			buf := new(bytes.Buffer)

			require.NoError(t, storage.Query(buf, query), "Failed to execute query")
			payload := bytes.TrimSpace(buf.Bytes())
			lines := [][]byte{}
			if len(payload) > 0 {
				lines = bytes.Split(payload, []byte{'\n'})
			}
			require.Len(t, lines, len(tt.want), "Expected %d records after query", len(tt.want))

			var record logRecord
			for i, line := range lines {
				require.NoError(t, json.Unmarshal(line, &record), "Failed to unmarshal JSON")
				require.Equal(t, tt.want[i], record, "Record mismatch")
			}
		})
	}
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
