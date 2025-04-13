package storage

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/fugo-app/fugo/internal/field"
)

type privateStorageDriver interface {
	createTable(name string, fields []*field.Field) error
	insertData(name string, data map[string]any) error
}

func testStorage_InitDriver(
	t *testing.T,
	name string,
	storage StorageDriver,
	fields []*field.Field,
	data []map[string]any,
) {
	ps, ok := storage.(privateStorageDriver)
	require.True(t, ok, "Storage driver does not implement privateStorageDriver")

	for _, item := range fields {
		require.NoError(t, item.Init(), "Failed to initialize field: %s", item.Name)
	}

	// Create the table
	require.NoError(t, ps.createTable(name, fields), "Failed to create table")

	// Insert test data
	for _, item := range data {
		require.NoError(t, ps.insertData(name, item), "Failed to insert test data")
	}
}

func testStorage_Cleanup(t *testing.T, storage StorageDriver) {
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

	now := time.Now().UnixMilli()

	testData := []map[string]any{
		{"time": now - (3600 * 24 * 7 * 1000), "message": "one week old message"},
		{"time": now - (3600 * 24 * 3 * 1000), "message": "three days old message"},
		{"time": now - (3600 * 24 * 1 * 1000), "message": "one day old message"},
		{"time": now - (3600 * 1000), "message": "one hour old message"},
		{"time": now, "message": "current message"},
	}

	testStorage_InitDriver(t, tableName, storage, fields, testData)

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

	query := NewQuery(tableName)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := storage.Cleanup(tt.tableName, tt.fieldName, tt.retention)
			if tt.wantErr {
				require.Error(t, err, "Expected an error but got none")
			} else {
				require.NoError(t, err, "Cleanup operation failed")

				buf := new(bytes.Buffer)
				require.NoError(t, storage.Query(buf, query), "Failed to execute query")
				payload := bytes.TrimSpace(buf.Bytes())
				lines := [][]byte{}
				if len(payload) > 0 {
					lines = bytes.Split(payload, []byte{'\n'})
				}
				require.Len(t, lines, tt.wantCount, "Record count mismatch after cleanup")
			}
		})
	}
}
