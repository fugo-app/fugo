package storage

import (
	"database/sql"
	"fmt"
	"strconv"
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
	// Comparison Operators
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
		ival, err := strconv.ParseInt(val, 0, 64)
		if err != nil {
			return fmt.Errorf("invalid int value: %s", val)
		}
		q.filters = append(q.filters, &QueryOperator{name: name, op: opType, ival: ival})
	} else {
		q.filters = append(q.filters, &QueryOperator{name: name, op: opType, sval: val})
	}

	return nil
}
