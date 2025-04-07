package agent

import (
	"database/sql"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/fugo-app/fugo/internal/field"
)

func TestAgent_CreateTable(t *testing.T) {
	// Create an in-memory SQLite database for testing
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	agent := &Agent{
		name: "test_agent",
		Fields: []*field.Field{
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
		},
	}

	require.NoError(t, agent.Init(agent.name), "Failed to initialize agent")

	require.NoError(t, agent.Migrate(db), "Failed to migrate agent")
	verifyTableStructure(t, db, agent)
}

func TestAgent_MigrateAddColumn(t *testing.T) {
	// Create an in-memory SQLite database for testing
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	// Create initial agent with some fields
	initialAgent := &Agent{
		name: "test_migration",
		Fields: []*field.Field{
			{
				Name: "timestamp",
				Timestamp: &field.TimestampFormat{
					Format: "2006-01-02 15:04:05",
				},
			},
			{Name: "level", Type: "string"},
			{Name: "message", Type: "string"},
		},
	}
	require.NoError(t, initialAgent.Init(initialAgent.name))
	require.NoError(t, initialAgent.Migrate(db))

	// Verify initial table structure
	verifyTableStructure(t, db, initialAgent)

	// Create updated agent with additional fields
	updatedAgent := &Agent{
		name: "test_migration",
		Fields: []*field.Field{
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
		},
	}
	require.NoError(t, updatedAgent.Init(updatedAgent.name))
	require.NoError(t, updatedAgent.Migrate(db))

	// Verify updated table structure
	verifyTableStructure(t, db, updatedAgent)
}

func TestAgent_MigrateRemoveColumn(t *testing.T) {
	// Create an in-memory SQLite database for testing
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	// Create initial agent with some fields
	initialAgent := &Agent{
		name: "test_removal",
		Fields: []*field.Field{
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
		},
	}
	require.NoError(t, initialAgent.Init(initialAgent.name))
	require.NoError(t, initialAgent.Migrate(db))

	// Verify initial table structure
	verifyTableStructure(t, db, initialAgent)

	// Create updated agent with fewer fields
	updatedAgent := &Agent{
		name: "test_removal",
		Fields: []*field.Field{
			{
				Name: "timestamp",
				Timestamp: &field.TimestampFormat{
					Format: "2006-01-02 15:04:05",
				},
			},
			{Name: "message", Type: "string"}, // Keep these columns
			{Name: "value", Type: "float"},    // Keep these columns
			// Removed "level" and "count" columns
		},
	}
	require.NoError(t, updatedAgent.Init(updatedAgent.name))
	require.NoError(t, updatedAgent.Migrate(db))

	// Verify updated table structure
	verifyTableStructure(t, db, updatedAgent)
}

func TestAgent_MigrateChangeColumnType(t *testing.T) {
	// Create an in-memory SQLite database for testing
	db, err := sql.Open("sqlite3", ":memory:")
	require.NoError(t, err)
	defer db.Close()

	// Create initial agent with some fields
	initialAgent := &Agent{
		name: "test_type_change",
		Fields: []*field.Field{
			{
				Name: "timestamp",
				Timestamp: &field.TimestampFormat{
					Format: "2006-01-02 15:04:05",
				},
			},
			{Name: "level", Type: "string"},
			{Name: "count", Type: "int"},     // This will be changed to float
			{Name: "status", Type: "string"}, // This will be changed to int
		},
	}
	require.NoError(t, initialAgent.Init(initialAgent.name))
	require.NoError(t, initialAgent.Migrate(db))

	// Verify initial table structure
	verifyTableStructure(t, db, initialAgent)

	// Create updated agent with changed column types
	updatedAgent := &Agent{
		name: "test_type_change",
		Fields: []*field.Field{
			{
				Name: "timestamp",
				Timestamp: &field.TimestampFormat{
					Format: "2006-01-02 15:04:05",
				},
			},
			{Name: "level", Type: "string"},
			{Name: "count", Type: "float"}, // Changed from int to float
			{Name: "status", Type: "int"},  // Changed from string to int
		},
	}
	require.NoError(t, updatedAgent.Init(updatedAgent.name))
	require.NoError(t, updatedAgent.Migrate(db))

	// Verify updated table structure
	verifyTableStructure(t, db, updatedAgent)
}

// verifyTableStructure checks that the table was created with the correct columns
func verifyTableStructure(t *testing.T, db *sql.DB, agent *Agent) {
	var tableExists bool
	err := db.
		QueryRow(`SELECT COUNT(*) > 0 FROM sqlite_master WHERE type = 'table' AND name = ?`, agent.name).
		Scan(&tableExists)
	require.NoError(t, err)
	require.True(t, tableExists, "Table does not exist")

	// Get table info
	rows, err := db.Query(fmt.Sprintf("PRAGMA table_info(`%s`)", agent.name))
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
		currentColumns[name] = ctype
	}

	expectedColumns := make(map[string]string)
	for _, f := range agent.Fields {
		expectedColumns[f.Name] = getSqlType(f.Type)
	}

	require.Equal(t, expectedColumns, currentColumns, "Table columns do not match expected structure")
}
