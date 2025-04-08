package storage

import "database/sql"

type Query struct {
	Name  string
	Limit sql.NullInt64

	After  sql.NullInt64 // After Cursor
	Before sql.NullInt64 // Before Cursor

	Filter []*QueryFilter
}

type QueryFilter struct {
	Name  string
	Op    string
	Value string
}

var QueryOps = []string{
	"eq",
	"ne",
	"lt",
	"le",
	"gt",
	"ge",
	"like",
}
