package storage

import (
	"bytes"
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/fugo-app/fugo/internal/field"
)

type queryTest struct {
	name     string
	modifier func(q *Query)
	want     []map[string]any
}

func testQuery_CheckResult(t *testing.T, name string, storage StorageDriver, tests []*queryTest) {
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

			for i, line := range lines {
				var record map[string]any
				require.NoError(t, json.Unmarshal(line, &record), "Failed to unmarshal JSON")

				// convert float64 to int64
				for k, v := range record {
					if _, ok := v.(float64); ok {
						record[k] = int64(v.(float64))
					}
				}

				require.Equal(t, tt.want[i], record, "Record mismatch")
			}
		})
	}
}

func testQuery_Number(t *testing.T, storage StorageDriver) {
	name := "test_query_number"

	fields := []*field.Field{
		{Name: "status", Type: "int"},
	}

	// Insert test data
	testData := []map[string]any{
		{"status": int64(200)},
		{"status": int64(404)},
		{"status": int64(403)},
		{"status": int64(500)},
		{"status": int64(400)},
	}

	testStorage_InitDriver(t, name, storage, fields, testData)

	tests := []*queryTest{
		{
			name:     "query all records",
			modifier: func(q *Query) {},
			want: []map[string]any{
				{"_cursor": "0000000000000001", "status": int64(200)},
				{"_cursor": "0000000000000002", "status": int64(404)},
				{"_cursor": "0000000000000003", "status": int64(403)},
				{"_cursor": "0000000000000004", "status": int64(500)},
				{"_cursor": "0000000000000005", "status": int64(400)},
			},
		},
		{
			name: "query with limit",
			modifier: func(q *Query) {
				q.SetLimit(3)
			},
			want: []map[string]any{
				{"_cursor": "0000000000000003", "status": int64(403)},
				{"_cursor": "0000000000000004", "status": int64(500)},
				{"_cursor": "0000000000000005", "status": int64(400)},
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
			want: []map[string]any{
				{"_cursor": "0000000000000003", "status": int64(403)},
				{"_cursor": "0000000000000004", "status": int64(500)},
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
			want: []map[string]any{
				{"_cursor": "0000000000000002", "status": int64(404)},
				{"_cursor": "0000000000000003", "status": int64(403)},
			},
		},
		{
			name: "eq filter",
			modifier: func(q *Query) {
				q.SetFilter("status", "eq", "403")
			},
			want: []map[string]any{
				{"_cursor": "0000000000000003", "status": int64(403)},
			},
		},
		{
			name: "ne filter",
			modifier: func(q *Query) {
				q.SetFilter("status", "ne", "403")
			},
			want: []map[string]any{
				{"_cursor": "0000000000000001", "status": int64(200)},
				{"_cursor": "0000000000000002", "status": int64(404)},
				{"_cursor": "0000000000000004", "status": int64(500)},
				{"_cursor": "0000000000000005", "status": int64(400)},
			},
		},
		{
			name: "lt filter",
			modifier: func(q *Query) {
				q.SetFilter("status", "lt", "403")
			},
			want: []map[string]any{
				{"_cursor": "0000000000000001", "status": int64(200)},
				{"_cursor": "0000000000000005", "status": int64(400)},
			},
		},
		{
			name: "lte filter",
			modifier: func(q *Query) {
				q.SetFilter("status", "lte", "403")
			},
			want: []map[string]any{
				{"_cursor": "0000000000000001", "status": int64(200)},
				{"_cursor": "0000000000000003", "status": int64(403)},
				{"_cursor": "0000000000000005", "status": int64(400)},
			},
		},
		{
			name: "gt filter",
			modifier: func(q *Query) {
				q.SetFilter("status", "gt", "403")
			},
			want: []map[string]any{
				{"_cursor": "0000000000000002", "status": int64(404)},
				{"_cursor": "0000000000000004", "status": int64(500)},
			},
		},
		{
			name: "gte filter",
			modifier: func(q *Query) {
				q.SetFilter("status", "gte", "403")
			},
			want: []map[string]any{
				{"_cursor": "0000000000000002", "status": int64(404)},
				{"_cursor": "0000000000000003", "status": int64(403)},
				{"_cursor": "0000000000000004", "status": int64(500)},
			},
		},
	}

	testQuery_CheckResult(t, name, storage, tests)
}

func testQuery_String(t *testing.T, storage StorageDriver) {
	name := "test_query_string"

	fields := []*field.Field{
		{Name: "message", Type: "string"},
	}

	// Insert test data
	testData := []map[string]any{
		{"message": "apple pie"},
		{"message": "pineapple juice"},
		{"message": "grapefruit"},
		{"message": "apple"},
		{"message": "green apple"},
	}

	testStorage_InitDriver(t, name, storage, fields, testData)

	tests := []*queryTest{
		{
			name:     "query all records",
			modifier: func(q *Query) {},
			want: []map[string]any{
				{"_cursor": "0000000000000001", "message": "apple pie"},
				{"_cursor": "0000000000000002", "message": "pineapple juice"},
				{"_cursor": "0000000000000003", "message": "grapefruit"},
				{"_cursor": "0000000000000004", "message": "apple"},
				{"_cursor": "0000000000000005", "message": "green apple"},
			},
		},
		{
			name: "exact filter",
			modifier: func(q *Query) {
				q.SetFilter("message", "exact", "apple")
			},
			want: []map[string]any{
				{"_cursor": "0000000000000004", "message": "apple"},
			},
		},
		{
			name: "like filter",
			modifier: func(q *Query) {
				q.SetFilter("message", "like", "apple")
			},
			want: []map[string]any{
				{"_cursor": "0000000000000001", "message": "apple pie"},
				{"_cursor": "0000000000000002", "message": "pineapple juice"},
				{"_cursor": "0000000000000004", "message": "apple"},
				{"_cursor": "0000000000000005", "message": "green apple"},
			},
		},
		{
			name: "prefix filter",
			modifier: func(q *Query) {
				q.SetFilter("message", "prefix", "apple")
			},
			want: []map[string]any{
				{"_cursor": "0000000000000001", "message": "apple pie"},
				{"_cursor": "0000000000000004", "message": "apple"},
			},
		},
		{
			name: "suffix filter",
			modifier: func(q *Query) {
				q.SetFilter("message", "suffix", "apple")
			},
			want: []map[string]any{
				{"_cursor": "0000000000000004", "message": "apple"},
				{"_cursor": "0000000000000005", "message": "green apple"},
			},
		},
	}

	testQuery_CheckResult(t, name, storage, tests)
}

func testQuery_Time(t *testing.T, storage StorageDriver) {
	name := "test_query_time"

	fields := []*field.Field{
		{
			Name: "time",
			Timestamp: &field.TimestampFormat{
				Format: "stamp",
			},
		},
	}

	// Insert test data
	testData := []map[string]any{
		{"time": int64(1735812000000)}, // 2025-01-02 10:00:00
		{"time": int64(1735817400000)}, // 2025-01-02 11:30:00
		{"time": int64(1735823700000)}, // 2025-01-02 13:15:00
		{"time": int64(1735829100000)}, // 2025-01-02 14:45:00
		{"time": int64(1735833600000)}, // 2025-01-02 16:00:00
	}

	testStorage_InitDriver(t, name, storage, fields, testData)

	tests := []*queryTest{
		{
			name:     "query all records",
			modifier: func(q *Query) {},
			want: []map[string]any{
				{"_cursor": "0000000000000001", "time": int64(1735812000000)},
				{"_cursor": "0000000000000002", "time": int64(1735817400000)},
				{"_cursor": "0000000000000003", "time": int64(1735823700000)},
				{"_cursor": "0000000000000004", "time": int64(1735829100000)},
				{"_cursor": "0000000000000005", "time": int64(1735833600000)},
			},
		},
		{
			name: "since filter",
			modifier: func(q *Query) {
				q.SetFilter("time", "since", "2025-01-02 13:00:00")
			},
			want: []map[string]any{
				{"_cursor": "0000000000000003", "time": int64(1735823700000)},
				{"_cursor": "0000000000000004", "time": int64(1735829100000)},
				{"_cursor": "0000000000000005", "time": int64(1735833600000)},
			},
		},
		{
			name: "until filter",
			modifier: func(q *Query) {
				q.SetFilter("time", "until", "2025-01-02 13:00:00")
			},
			want: []map[string]any{
				{"_cursor": "0000000000000001", "time": int64(1735812000000)},
				{"_cursor": "0000000000000002", "time": int64(1735817400000)},
			},
		},
		{
			name: "since filter with limit",
			modifier: func(q *Query) {
				q.SetFilter("time", "since", "2025-01-02 13:00:00")
				q.SetLimit(2)
			},
			want: []map[string]any{
				{"_cursor": "0000000000000003", "time": int64(1735823700000)},
				{"_cursor": "0000000000000004", "time": int64(1735829100000)},
			},
		},
		{
			name: "until filter with limit",
			modifier: func(q *Query) {
				q.SetFilter("time", "until", "2025-01-02 14:00:00")
				q.SetLimit(2)
			},
			want: []map[string]any{
				{"_cursor": "0000000000000002", "time": int64(1735817400000)},
				{"_cursor": "0000000000000003", "time": int64(1735823700000)},
			},
		},
		{
			name: "since filter with after cursor",
			modifier: func(q *Query) {
				q.SetFilter("time", "since", "2025-01-02 13:00:00")
				q.SetAfter(2)
			},
			want: []map[string]any{}, // No records should be returned
		},
		{
			name: "until filter with before cursor",
			modifier: func(q *Query) {
				q.SetFilter("time", "until", "2025-01-02 14:00:00")
				q.SetBefore(2)
			},
			want: []map[string]any{}, // No records should be returned
		},
		{
			name: "since filter with before cursor",
			modifier: func(q *Query) {
				q.SetFilter("time", "since", "2025-01-02 13:00:00")
				q.SetBefore(5)
			},
			want: []map[string]any{
				{"_cursor": "0000000000000003", "time": int64(1735823700000)},
				{"_cursor": "0000000000000004", "time": int64(1735829100000)},
			},
		},
		{
			name: "until filter with after cursor",
			modifier: func(q *Query) {
				q.SetFilter("time", "until", "2025-01-02 14:00:00")
				q.SetAfter(1)
			},
			want: []map[string]any{
				{"_cursor": "0000000000000002", "time": int64(1735817400000)},
				{"_cursor": "0000000000000003", "time": int64(1735823700000)},
			},
		},
		{
			// time.Now() returns 2025-01-02 13:00:00 UTC
			name: "since filter relative",
			modifier: func(q *Query) {
				q.SetFilter("time", "since", "1h")
			},
			want: []map[string]any{
				{"_cursor": "0000000000000003", "time": int64(1735823700000)},
				{"_cursor": "0000000000000004", "time": int64(1735829100000)},
				{"_cursor": "0000000000000005", "time": int64(1735833600000)},
			},
		},
		{
			// time.Now() returns 2025-01-02 13:00:00 UTC
			name: "until filter relative",
			modifier: func(q *Query) {
				q.SetFilter("time", "until", "1h")
			},
			want: []map[string]any{
				{"_cursor": "0000000000000001", "time": int64(1735812000000)},
				{"_cursor": "0000000000000002", "time": int64(1735817400000)},
			},
		},
	}

	defaultTimeNow := stdTimeNow
	stdTimeNow = func() time.Time {
		return time.Date(2025, 1, 2, 13, 0, 0, 0, time.UTC)
	}
	defer func() {
		stdTimeNow = defaultTimeNow
	}()

	testQuery_CheckResult(t, name, storage, tests)
}
