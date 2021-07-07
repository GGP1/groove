package postgres

import (
	"context"
	"database/sql"

	"github.com/GGP1/groove/internal/bufferpool"

	"github.com/pkg/errors"
)

const (
	eventDefaultFields = "id, name, public, virtual, start_time, end_time, ticket_cost, min_age, slots"
	userDefaultFields  = "id, name, username, email, created_at, updated_at"

	// Events table
	Events table = "events"
	// Users table
	Users table = "users"
)

type table string

// IterRows iterates over the rows passed executing f() on each iteration.
func IterRows(rows *sql.Rows, f func(r *sql.Rows) error) error {
	for rows.Next() {
		if err := f(rows); err != nil {
			return err
		}
	}

	return rows.Err()
}

// ScanBool returns a boolean scanned from a single row.
func ScanBool(ctx context.Context, tx *sql.Tx, query string, args ...interface{}) (bool, error) {
	row := tx.QueryRowContext(ctx, query, args...)
	var b bool
	if err := row.Scan(&b); err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, err
	}

	return b, nil
}

// ScanString returns a string scanned from a single row.
func ScanString(ctx context.Context, tx *sql.Tx, query string, args ...interface{}) (string, error) {
	row := tx.QueryRowContext(ctx, query, args...)
	var str string
	if err := row.Scan(&str); err != nil {
		if err == sql.ErrNoRows {
			return "", errors.Wrap(err, "value not found")
		}
		return "", errors.Wrap(err, "scanning value")
	}

	return str, nil
}

// SelectInID builds a postgres select statement.
func SelectInID(table table, ids, fields []string) string {
	buf := bufferpool.Get()
	defer bufferpool.Put(buf)

	// SELECT fields FROM table WHERE id IN ('id1', 'id2', ...)
	buf.WriteString("SELECT ")
	if fields == nil {
		switch table {
		case Users:
			buf.WriteString(userDefaultFields)
		case Events:
			buf.WriteString(eventDefaultFields)
		}
	} else {
		for i, f := range fields {
			if i != 0 {
				buf.WriteString(", ")
			}
			buf.WriteString(f)
		}
	}
	buf.WriteString(" FROM ")
	buf.WriteString(string(table))
	buf.WriteString(" WHERE id IN (")
	for j, id := range ids {
		if j != 0 {
			buf.WriteString(", ")
		}
		buf.WriteString("'")
		buf.WriteString(id)
		buf.WriteString("'")
	}
	buf.WriteString(")")

	return buf.String()
}
