package storage

import (
	"database/sql"
	"fmt"
	"strconv"
	"time"

	"github.com/fugo-app/fugo/pkg/duration"
)

type Query struct {
	name  string
	limit sql.NullInt64

	after  sql.NullInt64 // After Cursor
	before sql.NullInt64 // Before Cursor

	filters []*QueryOperator
}

type QueryOperator struct {
	name string
	op   QueryOperatorType
	ival int64
	sval string
}

type QueryOperatorType int

const (
	// Integer Operators
	Eq QueryOperatorType = iota
	Ne
	Lt
	Lte
	Gt
	Gte

	// String Operators
	Exact
	Like
	Prefix
	Suffix

	// Time Operators
	Since
	Until
)

var opmap = map[string]QueryOperatorType{
	"eq":     Eq,
	"ne":     Ne,
	"lt":     Lt,
	"lte":    Lte,
	"gt":     Gt,
	"gte":    Gte,
	"exact":  Exact,
	"like":   Like,
	"prefix": Prefix,
	"suffix": Suffix,
	"since":  Since,
	"until":  Until,
}

func NewQuery(name string) *Query {
	return &Query{
		name: name,
	}
}

func (q *Query) SetLimit(limit int64) {
	q.limit.Int64 = limit
	q.limit.Valid = true
}

// SetAfter sets the after cursor for pagination
// Move to the next page, or refresh tail
func (q *Query) SetAfter(after int64) {
	q.after.Int64 = after
	q.after.Valid = true
}

// SetBefore sets the before cursor for pagination
// Move to the previous page
func (q *Query) SetBefore(before int64) {
	q.before.Int64 = before
	q.before.Valid = true
}

func (q *Query) SetFilter(name string, op string, val string) error {
	opType, ok := opmap[op]
	if !ok {
		return fmt.Errorf("invalid operator: %s", op)
	}

	if opType >= Eq && opType <= Gte {
		// Integer operators
		ival, err := strconv.ParseInt(val, 0, 64)
		if err != nil {
			return fmt.Errorf("invalid int value: %s", val)
		}
		q.filters = append(q.filters, &QueryOperator{name: name, op: opType, ival: ival})
	} else if opType >= Exact && opType <= Suffix {
		// String operators
		q.filters = append(q.filters, &QueryOperator{name: name, op: opType, sval: val})
	} else if opType >= Since && opType <= Until {
		// Time operators
		timestamp, err := parseTimestamp(val)
		if err != nil {
			return fmt.Errorf("invalid timestamp value: %s, error: %w", val, err)
		}
		q.filters = append(q.filters, &QueryOperator{name: name, op: opType, ival: timestamp})
	}

	return nil
}

// parseTimestamp converts timestamp string to unix milliseconds.
// Supported formats:
// - "2006-01-02 15:04:05" - date and time format
// - "2006-01-02" - date only format
// - "5d" - relative time (now minus 5 days)
// - "2h" - relative time (now minus 2 hours)
func parseTimestamp(val string) (int64, error) {
	if duration.Match(val) {
		d, err := duration.Parse(val)
		if err != nil {
			return 0, err
		}
		return time.Now().Add(-d).UnixMilli(), nil
	}

	layouts := []string{
		"2006-01-02 15:04:05",
		"2006-01-02",
	}

	for _, layout := range layouts {
		t, err := time.Parse(layout, val)
		if err == nil {
			return t.UnixMilli(), nil
		}
	}

	return 0, fmt.Errorf("invalid format")
}
