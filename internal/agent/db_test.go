package agent

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/fugo-app/fugo/internal/field"
)

func Test_createTable(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	name := "test_agent"
	fields := []*field.Field{
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
	}

	for _, f := range fields {
		require.NoError(t, f.Init(), "Failed to initialize field: %s", f.Name)
	}

	t.Run("create table", func(t *testing.T) {
		require.NoError(t, createTable(db, name, fields), "Failed to create table")
	})

	t.Run("check table exists", func(t *testing.T) {
		exists, err := checkTable(db, name)
		require.NoError(t, err, "Failed to check if table exists")
		require.True(t, exists, "Table should exist after creation")
	})

	t.Run("check table structure", func(t *testing.T) {
		columns, err := getColumns(db, name)
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
		rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(`%s`)", name))
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

func Test_migrateTable_AddColumn(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	// Create initial agent with some fields
	name := "test_migration"
	fields1 := []*field.Field{
		{
			Name: "timestamp",
			Timestamp: &field.TimestampFormat{
				Format: "2006-01-02 15:04:05",
			},
		},
		{Name: "level", Type: "string"},
		{Name: "message", Type: "string"},
	}

	for _, f := range fields1 {
		require.NoError(t, f.Init(), "Failed to initialize field: %s", f.Name)
	}

	t.Run("create table", func(t *testing.T) {
		require.NoError(t, createTable(db, name, fields1), "Failed to create table")
		verifyTableStructure(t, db, name, fields1)
	})

	// Create updated agent with additional fields
	fields2 := []*field.Field{
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
	}

	for _, f := range fields2 {
		require.NoError(t, f.Init(), "Failed to initialize field: %s", f.Name)
	}

	t.Run("migrate table", func(t *testing.T) {
		require.NoError(t, migrateTable(db, name, fields2), "Failed to migrate table")
		verifyTableStructure(t, db, name, fields2)
	})
}

func TestAgent_migrateTable_RemoveColumn(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	// Create initial agent with some fields
	name := "test_removal"
	fields1 := []*field.Field{
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
	}

	for _, f := range fields1 {
		require.NoError(t, f.Init(), "Failed to initialize field: %s", f.Name)
	}

	t.Run("create table", func(t *testing.T) {
		require.NoError(t, createTable(db, name, fields1), "Failed to create table")
		verifyTableStructure(t, db, name, fields1)
	})

	// Create updated agent with fewer fields
	fields2 := []*field.Field{
		{
			Name: "timestamp",
			Timestamp: &field.TimestampFormat{
				Format: "2006-01-02 15:04:05",
			},
		},
		{Name: "message", Type: "string"}, // Keep these columns
		{Name: "value", Type: "float"},    // Keep these columns
		// Removed "level" and "count" columns
	}

	for _, f := range fields2 {
		require.NoError(t, f.Init(), "Failed to initialize field: %s", f.Name)
	}

	t.Run("migrate table", func(t *testing.T) {
		require.NoError(t, migrateTable(db, name, fields2), "Failed to migrate table")
		verifyTableStructure(t, db, name, fields2)
	})
}

func TestAgent_MigrateChangeColumnType(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	// Create initial agent with some fields
	name := "test_type_change"
	fields1 := []*field.Field{
		{
			Name: "timestamp",
			Timestamp: &field.TimestampFormat{
				Format: "2006-01-02 15:04:05",
			},
		},
		{Name: "level", Type: "string"},
		{Name: "count", Type: "int"},     // This will be changed to float
		{Name: "status", Type: "string"}, // This will be changed to int
	}

	for _, f := range fields1 {
		require.NoError(t, f.Init(), "Failed to initialize field: %s", f.Name)
	}

	t.Run("create table", func(t *testing.T) {
		require.NoError(t, createTable(db, name, fields1), "Failed to create table")
		verifyTableStructure(t, db, name, fields1)
	})

	// Create updated agent with changed column types
	fields2 := []*field.Field{
		{
			Name: "timestamp",
			Timestamp: &field.TimestampFormat{
				Format: "2006-01-02 15:04:05",
			},
		},
		{Name: "level", Type: "string"},
		{Name: "count", Type: "float"}, // Changed from int to float
		{Name: "status", Type: "int"},  // Changed from string to int
	}

	for _, f := range fields2 {
		require.NoError(t, f.Init(), "Failed to initialize field: %s", f.Name)
	}

	t.Run("migrate table", func(t *testing.T) {
		require.NoError(t, migrateTable(db, name, fields2), "Failed to migrate table")
		verifyTableStructure(t, db, name, fields2)
	})
}

// verifyTableStructure checks that the table was created with the correct columns
func verifyTableStructure(t *testing.T, db *sql.DB, name string, fields []*field.Field) {
	tableExists, err := checkTable(db, name)
	require.NoError(t, err, "Failed to check if table exists")
	require.True(t, tableExists, "Table does not exist")

	currentColumns, err := getColumns(db, name)
	require.NoError(t, err, "Failed to get table columns")

	expectedColumns := make(map[string]string)
	for _, f := range fields {
		expectedColumns[f.Name] = getSqlType(f.Type)
	}

	require.Equal(
		t,
		expectedColumns,
		currentColumns,
		"Table columns do not match expected structure",
	)
}
