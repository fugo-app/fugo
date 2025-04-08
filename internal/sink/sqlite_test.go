package sink

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/fugo-app/fugo/internal/field"
)

func TestSQLiteSink_createTable(t *testing.T) {
	sink := &SQLiteSink{Path: ":memory:"}
	require.NoError(t, sink.Open(), "Failed to open SQLite database")
	defer sink.Close()

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
		require.NoError(t, sink.createTable(name, fields), "Failed to create table")
	})

	t.Run("check table exists", func(t *testing.T) {
		exists, err := sink.checkTable(name)
		require.NoError(t, err, "Failed to check if table exists")
		require.True(t, exists, "Table should exist after creation")
	})

	t.Run("check table structure", func(t *testing.T) {
		columns, err := sink.getColumns(name)
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
		rows, err := sink.db.Query(fmt.Sprintf("PRAGMA table_info(`%s`)", name))
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
			}
		}

		require.True(t, hasCursor, "Table should have _cursor column")
	})
}

func TestSQLiteSink_migrateTable_AddColumn(t *testing.T) {
	sink := &SQLiteSink{Path: ":memory:"}
	require.NoError(t, sink.Open(), "Failed to open SQLite database")
	defer sink.Close()

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

		require.NoError(t, sink.createTable(name, fields), "Failed to create table")
		verifySqliteDB(t, sink, name, fields)
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

		require.NoError(t, sink.migrateTable(name, fields), "Failed to migrate table")
		verifySqliteDB(t, sink, name, fields)
	})
}

func TestSQLiteSink_migrateTable_RemoveColumn(t *testing.T) {
	sink := &SQLiteSink{Path: ":memory:"}
	require.NoError(t, sink.Open(), "Failed to open SQLite database")
	defer sink.Close()

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

		require.NoError(t, sink.createTable(name, fields), "Failed to create table")
		verifySqliteDB(t, sink, name, fields)
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

		require.NoError(t, sink.migrateTable(name, fields), "Failed to migrate table")
		verifySqliteDB(t, sink, name, fields)
	})
}

func TestSQLiteSink_MigrateChangeColumnType(t *testing.T) {
	sink := &SQLiteSink{Path: ":memory:"}
	require.NoError(t, sink.Open(), "Failed to open SQLite database")
	defer sink.Close()

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

		require.NoError(t, sink.createTable(name, fields), "Failed to create table")
		verifySqliteDB(t, sink, name, fields)
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

		require.NoError(t, sink.migrateTable(name, fields), "Failed to migrate table")
		verifySqliteDB(t, sink, name, fields)
	})
}

func TestSQLiteSink_insertData(t *testing.T) {
	sink := &SQLiteSink{Path: ":memory:"}
	require.NoError(t, sink.Open(), "Failed to open SQLite database")
	defer sink.Close()

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
	require.NoError(t, sink.createTable(name, fields), "Failed to create table")

	// Test inserting data
	testData := map[string]any{
		"timestamp": int64(1648735200000),
		"level":     "info",
		"message":   "test message",
		"count":     42,
		"value":     3.14,
	}

	// Insert the data
	err := sink.insertData(name, testData)
	require.NoError(t, err, "Failed to insert data")

	// Verify the data was inserted correctly
	row := sink.db.QueryRow(fmt.Sprintf("SELECT timestamp, level, message, count, value FROM `%s` LIMIT 1", name))

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

func initFields(t *testing.T, fields []*field.Field) []*field.Field {
	for _, f := range fields {
		require.NoError(t, f.Init(), "Failed to initialize field: %s", f.Name)
	}

	return fields
}

// verifyTableStructure checks that the table was created with the correct columns
func verifySqliteDB(t *testing.T, sink *SQLiteSink, name string, fields []*field.Field) {
	tableExists, err := sink.checkTable(name)
	require.NoError(t, err, "Failed to check if table exists")
	require.True(t, tableExists, "Table does not exist")

	currentColumns, err := sink.getColumns(name)
	require.NoError(t, err, "Failed to get table columns")

	expectedColumns := make(map[string]string)
	for _, f := range fields {
		expectedColumns[f.Name] = sink.getSqlType(f)
	}

	require.Equal(
		t,
		expectedColumns,
		currentColumns,
		"Table columns do not match expected structure",
	)
}
