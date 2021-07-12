package postgres

import (
	"bytes"
	"context"
	"database/sql"

	"github.com/GGP1/groove/internal/bufferpool"

	"github.com/pkg/errors"
)

const (
	eventDefaultFields   = "id, name, public, virtual, start_time, end_time, ticket_cost, min_age, slots"
	mediaDefaultFields   = "id, event_id, url"
	productDefaultFields = "id, event_id, stock, brand, type, subtotal, total"
	userDefaultFields    = "id, name, username, email, created_at, updated_at"

	// Events table
	Events table = "events"
	// Media table
	Media table = "events_media"
	// Products table
	Products table = "events_products"
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

// QueryBool returns a boolean scanned from a single row.
func QueryBool(ctx context.Context, tx *sql.Tx, query string, args ...interface{}) (bool, error) {
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

// QueryString returns a string scanned from a single row.
func QueryString(ctx context.Context, tx *sql.Tx, query string, args ...interface{}) (string, error) {
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

// ScanStringSlice returns a slice of strings scanned from sql rows.
func ScanStringSlice(rows *sql.Rows) ([]string, error) {
	var (
		// Reuse string, no need to reset as it will be overwritten every iteration
		str   string
		slice []string
	)

	for rows.Next() {
		if err := rows.Scan(&str); err != nil {
			return nil, err
		}
		slice = append(slice, str)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return slice, nil
}

// SelectWhereID builds a postgres select from statement.
//
// 	SELECT fields FROM table WHERE idfield='id'
func SelectWhereID(table table, fields []string, idField, id string) string {
	buf := bufferpool.Get()

	buf.WriteString("SELECT ")
	writeFields(buf, table, fields)
	buf.WriteString(" FROM ")
	buf.WriteString(string(table))
	buf.WriteString(" WHERE ")
	buf.WriteString(idField)
	buf.WriteString("='")
	buf.WriteString(id)
	buf.WriteByte('\'')

	query := buf.String()
	bufferpool.Put(buf)

	return query
}

// SelectInID builds a postgres select from in statement.
//
// 	SELECT fields FROM table WHERE id IN ('id1', 'id2', ...)
func SelectInID(table table, ids, fields []string) string {
	buf := bufferpool.Get()

	buf.WriteString("SELECT ")
	writeFields(buf, table, fields)
	buf.WriteString(" FROM ")
	buf.WriteString(string(table))
	buf.WriteString(" WHERE id IN (")
	for j, id := range ids {
		if j != 0 {
			buf.WriteByte(',')
			buf.WriteByte(' ')
		}
		buf.WriteByte('\'')
		buf.WriteString(id)
		buf.WriteByte('\'')
	}
	buf.WriteByte(')')

	query := buf.String()
	bufferpool.Put(buf)

	return query
}

func writeFields(buf *bytes.Buffer, table table, fields []string) {
	if fields == nil {
		// Write default fields
		switch table {
		case Events:
			buf.WriteString(eventDefaultFields)
		case Media:
			buf.WriteString(mediaDefaultFields)
		case Products:
			buf.WriteString(productDefaultFields)
		case Users:
			buf.WriteString(userDefaultFields)
		}
	} else {
		for i, f := range fields {
			if i != 0 {
				buf.WriteString(", ")
			}
			buf.WriteString(f)
		}
	}
}
