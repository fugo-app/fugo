package agent

import (
	"database/sql"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/fugo-app/fugo/internal/field"
)

func Test_createTable(t *testing.T) {
	// Create an in-memory SQLite database for testing
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
	require.NoError(t, createTable(db, name, fields), "Failed to create table")

	verifyTableStructure(t, db, name, fields)

	exists, err := checkTable(db, name)
	require.NoError(t, err, "Failed to check if table exists")
	require.True(t, exists, "Table should exist after creation")
}

func Test_migrateTable_AddColumn(t *testing.T) {
	// Create an in-memory SQLite database for testing
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
	require.NoError(t, createTable(db, name, fields1), "Failed to create table")

	// Verify initial table structure
	verifyTableStructure(t, db, name, fields1)

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

	for _, f := range fields1 {
		require.NoError(t, f.Init(), "Failed to initialize field: %s", f.Name)
	}
	require.NoError(t, migrateTable(db, name, fields2), "Failed to migrate table")

	// Verify updated table structure
	verifyTableStructure(t, db, name, fields2)
}

func TestAgent_migrateTable_RemoveColumn(t *testing.T) {
	// Create an in-memory SQLite database for testing
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
	require.NoError(t, createTable(db, name, fields1), "Failed to create table")

	// Verify initial table structure
	verifyTableStructure(t, db, name, fields1)

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
	require.NoError(t, migrateTable(db, name, fields2), "Failed to migrate table")

	// Verify updated table structure
	verifyTableStructure(t, db, name, fields2)
}

func TestAgent_MigrateChangeColumnType(t *testing.T) {
	// Create an in-memory SQLite database for testing
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
	require.NoError(t, createTable(db, name, fields1), "Failed to create table")

	// Verify initial table structure
	verifyTableStructure(t, db, name, fields1)

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
	require.NoError(t, migrateTable(db, name, fields2), "Failed to migrate table")

	// Verify updated table structure
	verifyTableStructure(t, db, name, fields2)
}

// verifyTableStructure checks that the table was created with the correct columns
func verifyTableStructure(t *testing.T, db *sql.DB, name string, fields []*field.Field) {
	var tableExists bool
	err := db.
		QueryRow(`SELECT COUNT(*) > 0 FROM sqlite_master WHERE type = 'table' AND name = ?`, name).
		Scan(&tableExists)
	require.NoError(t, err)
	require.True(t, tableExists, "Table does not exist")

	// Get table info
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(`%s`)", name))
	require.NoError(t, err)
	defer rows.Close()

	currentColumns := make(map[string]string)
	for rows.Next() {
		var (
			cid     int
			name    string
			ctype   string
			notnull int
			dflt    sql.NullString
			pk      int
		)
		err = rows.Scan(&cid, &name, &ctype, &notnull, &dflt, &pk)
		require.NoError(t, err)

		// Skip internal columns
		if strings.HasPrefix(name, "_") {
			continue
		}

		currentColumns[name] = ctype
	}

	expectedColumns := make(map[string]string)
	for _, f := range fields {
		expectedColumns[f.Name] = getSqlType(f.Type)
	}

	require.Equal(t, expectedColumns, currentColumns, "Table columns do not match expected structure")
}
