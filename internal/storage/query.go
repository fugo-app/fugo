package storage

import (
	"database/sql"
	"fmt"
	"strconv"
)

type Query struct {
	Name  string
	Limit sql.NullInt64

	After  sql.NullInt64 // After Cursor
	Before sql.NullInt64 // Before Cursor

	Filter []*QueryOperator
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
)

var opmap = map[string]QueryOperatorType{
	"eq":    Eq,
	"ne":    Ne,
	"lt":    Lt,
	"lte":   Lte,
	"gt":    Gt,
	"gte":   Gte,
	"exact": Exact,
	"like":  Like,
}

func (q *Query) NewQueryOperator(name string, op string, val string) error {
	opType, ok := opmap[op]
	if !ok {
		return fmt.Errorf("invalid operator: %s", op)
	}

	if opType >= Eq && opType <= Gte {
		ival, err := strconv.ParseInt(val, 0, 64)
		if err != nil {
			return fmt.Errorf("invalid int value: %s", val)
		}
		q.Filter = append(q.Filter, &QueryOperator{name: name, op: opType, ival: ival})
	} else {
		q.Filter = append(q.Filter, &QueryOperator{name: name, op: opType, sval: val})
	}

	return nil
}
