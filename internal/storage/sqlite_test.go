package storage

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/fugo-app/fugo/internal/field"
)

func TestSQLiteStorage_createTable(t *testing.T) {
	storage := &SQLiteStorage{Path: ":memory:"}
	require.NoError(t, storage.Open(), "Failed to open SQLite database")
	defer storage.Close()

	name := "test_agent"
	fields := testSqlite_InitFields(t, []*field.Field{
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
		testSqlite_VerifyDB(t, storage, name, fields)
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
		fields := testSqlite_InitFields(t, []*field.Field{
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
		testSqlite_VerifyDB(t, storage, name, fields)
	})

	// Create updated agent with additional fields
	t.Run("migrate table", func(t *testing.T) {
		fields := testSqlite_InitFields(t, []*field.Field{
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
		testSqlite_VerifyDB(t, storage, name, fields)
	})
}

func TestSQLiteStorage_migrateTable_RemoveColumn(t *testing.T) {
	storage := &SQLiteStorage{Path: ":memory:"}
	require.NoError(t, storage.Open(), "Failed to open SQLite database")
	defer storage.Close()

	// Create initial agent with some fields
	name := "test_removal"

	t.Run("create table", func(t *testing.T) {
		fields := testSqlite_InitFields(t, []*field.Field{
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
		testSqlite_VerifyDB(t, storage, name, fields)
	})

	// Create updated agent with fewer fields
	t.Run("migrate table", func(t *testing.T) {
		fields := testSqlite_InitFields(t, []*field.Field{
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
		testSqlite_VerifyDB(t, storage, name, fields)
	})
}

func TestSQLiteStorage_MigrateChangeColumnType(t *testing.T) {
	storage := &SQLiteStorage{Path: ":memory:"}
	require.NoError(t, storage.Open(), "Failed to open SQLite database")
	defer storage.Close()

	// Create initial agent with some fields
	name := "test_type_change"

	t.Run("create table", func(t *testing.T) {
		fields := testSqlite_InitFields(t, []*field.Field{
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
		testSqlite_VerifyDB(t, storage, name, fields)
	})

	// Create updated agent with changed column types
	t.Run("migrate table", func(t *testing.T) {
		fields := testSqlite_InitFields(t, []*field.Field{
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
		testSqlite_VerifyDB(t, storage, name, fields)
	})
}

func TestSQLiteStorage_Cleanup(t *testing.T) {
	storage := &SQLiteStorage{Path: ":memory:"}
	require.NoError(t, storage.Open(), "Failed to open SQLite database")
	defer storage.Close()

	tableName := "test_cleanup"
	fieldName := "time"

	fields := testSqlite_InitFields(t, []*field.Field{
		{
			Name: "time",
			Timestamp: &field.TimestampFormat{
				Format: "2006-01-02 15:04:05",
			},
		},
		{Name: "message", Type: "string"},
	})

	require.NoError(t, storage.createTable(tableName, fields), "Failed to create table")

	now := time.Now().Unix()

	// Insert test data with different timestamps
	testData := []map[string]any{
		{"time": now - (3600 * 24 * 7), "message": "one week old message"},
		{"time": now - (3600 * 24 * 3), "message": "three days old message"},
		{"time": now - (3600 * 24), "message": "one day old message"},
		{"time": now - 3600, "message": "one hour old message"},
		{"time": now, "message": "current message"},
	}

	// Insert all test records
	for _, data := range testData {
		require.NoError(t, storage.insertData(tableName, data), "Failed to insert test data")
	}

	tests := []struct {
		name      string
		tableName string
		fieldName string
		retention time.Duration
		wantCount int
		wantErr   bool
	}{
		{
			name:      "10 days retention",
			tableName: tableName,
			fieldName: fieldName,
			retention: 10 * 24 * time.Hour,
			wantCount: 5, // 10 days retention, should keep all records
		},
		{
			name:      "2 days retention",
			tableName: tableName,
			fieldName: fieldName,
			retention: 2 * 24 * time.Hour,
			wantCount: 3,
		},
		{
			name:      "6 hours retention",
			tableName: tableName,
			fieldName: fieldName,
			retention: 6 * time.Hour,
			wantCount: 2,
		},
		{
			name:      "non-existent field",
			tableName: tableName,
			fieldName: "non_existent_field",
			retention: 24 * time.Hour,
			wantErr:   true,
		},
		{
			name:      "non-existent table",
			tableName: "non_existent_table",
			fieldName: fieldName,
			retention: 24 * time.Hour,
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := storage.Cleanup(tt.tableName, tt.fieldName, tt.retention)
			if tt.wantErr {
				require.Error(t, err, "Expected an error but got none")
			} else {
				require.NoError(t, err, "Cleanup operation failed")

				var count int
				err := storage.db.
					QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM `%s`", tt.tableName)).
					Scan(&count)
				require.NoError(t, err, "Failed to count records")

				require.Equal(t, tt.wantCount, count, "Record count mismatch after cleanup")
			}
		})
	}
}

func TestSQLiteStorage_insertData(t *testing.T) {
	storage := &SQLiteStorage{Path: ":memory:"}
	require.NoError(t, storage.Open(), "Failed to open SQLite database")
	defer storage.Close()

	name := "test_insert"
	fields := testSqlite_InitFields(t, []*field.Field{
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

	t.Run("number", func(t *testing.T) {
		testQuery_Number(t, storage)
	})

	t.Run("string", func(t *testing.T) {
		testQuery_String(t, storage)
	})

	t.Run("time", func(t *testing.T) {
		testQuery_Time(t, storage)
	})
}

func testSqlite_InitFields(t *testing.T, fields []*field.Field) []*field.Field {
	for _, f := range fields {
		require.NoError(t, f.Init(), "Failed to initialize field: %s", f.Name)
	}

	return fields
}

// verifyTableStructure checks that the table was created with the correct columns
func testSqlite_VerifyDB(t *testing.T, storage *SQLiteStorage, name string, fields []*field.Field) {
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
