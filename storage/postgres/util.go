package postgres

import (
	"bytes"
	"context"
	"database/sql"
	"strings"
	"unicode/utf8"

	"github.com/GGP1/groove/internal/bufferpool"
	"github.com/GGP1/groove/internal/params"

	"github.com/pkg/errors"
)

const (
	// Events table
	Events table = "events"
	// Media table
	Media table = "events_media"
	// Products table
	Products table = "events_products"
	// Users table
	Users table = "users"

	eventDefaultFields   = "id, name, description, type, public, start_time, end_time, ticket_cost, min_age, slots"
	mediaDefaultFields   = "id, event_id, url"
	productDefaultFields = "id, event_id, stock, brand, type, subtotal, total"
	userDefaultFields    = "id, name, username, email, created_at, updated_at"
)

type table string

// AddPagination takes a query and adds pagination/lookup conditions to it.
func AddPagination(query, paginationField string, params params.Query) string {
	buf := bufferpool.Get()
	buf.WriteString(query)
	addPagination(buf, paginationField, params)
	q := buf.String()
	bufferpool.Put(buf)

	return q
}

// BulkInsertRoles adds the values to roles' insert query.
func BulkInsertRoles(query, eventID, roleName string, userIDs []string) string {
	buf := bufferpool.Get()

	buf.WriteString(query)
	for i, userID := range userIDs {
		// query ... ('eventID','userID','roleName'), (...)
		buf.WriteString(" ('")
		buf.WriteString(eventID)
		buf.WriteString("','")
		buf.WriteString(userID)
		buf.WriteString("','")
		buf.WriteString(roleName)
		buf.WriteString("')")
		if i != len(userIDs)-1 {
			buf.WriteByte(',')
		}
	}

	q := buf.String()
	bufferpool.Put(buf)
	return q
}

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

// FullTextSearch returns an SQL query implementing FTS.
func FullTextSearch(table table, query string, params params.Query) string {
	buf := bufferpool.Get()

	buf.WriteString("SELECT ")
	writeFields(buf, table, params.Fields)
	buf.WriteString(" FROM ")
	buf.WriteString(string(table))
	buf.WriteString(" WHERE search @@ to_tsquery('")
	// FTS operators: "&" (AND), "<->" (FOLLOWED BY)
	// See https://www.postgresql.org/docs/13/textsearch-controls.html
	replaceAll(buf, strings.TrimSpace(query), " ", "&")
	buf.WriteString(":*')")
	addPagination(buf, "id", params)

	q := buf.String()
	bufferpool.Put(buf)

	return q
}

// SelectInID builds a postgres select from in statement.
//
// 	SELECT fields FROM table WHERE id IN ('id1','id2',...) ORDER BY id DESC
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
		}
		buf.WriteByte('\'')
		buf.WriteString(id)
		buf.WriteByte('\'')
	}
	// Order like pagination does just in case it was used in a query prior to this one, so the client
	// receives the results in the order expected
	buf.WriteString(") ORDER BY id DESC")

	query := buf.String()
	bufferpool.Put(buf)

	return query
}

// SelectWhereID builds a postgres select from statement.
//
// 	Standard: "SELECT params.Fields FROM table WHERE idField='id' ORDER BY paginationField DESC LIMIT params.Limit"
//	LookupID: "SELECT params.Fields FROM table WHERE idField='id' AND paginationField='params.LookupID'"
//	Cursor: "SELECT params.Fields FROM table WHERE idField='id' AND paginationField < 'params.Cursor' ORDER BY paginationField DESC LIMIT params.Limit"
func SelectWhereID(table table, idField, id, paginationField string, params params.Query) string {
	buf := bufferpool.Get()

	buf.WriteString("SELECT ")
	writeFields(buf, table, params.Fields)
	buf.WriteString(" FROM ")
	buf.WriteString(string(table))
	buf.WriteString(" WHERE ")
	buf.WriteString(idField)
	buf.WriteString("='")
	buf.WriteString(id)
	buf.WriteByte('\'')
	addPagination(buf, paginationField, params)

	query := buf.String()
	bufferpool.Put(buf)

	return query
}

func addPagination(buf *bytes.Buffer, paginationField string, p params.Query) {
	if p.LookupID != "" {
		buf.WriteString(" AND ")
		buf.WriteString(paginationField)
		buf.WriteString("='")
		buf.WriteString(p.LookupID)
		buf.WriteByte('\'')
		return
	}
	if p.Cursor != params.DefaultCursor {
		buf.WriteString(" AND ")
		buf.WriteString(paginationField)
		buf.WriteString(" < '")
		buf.WriteString(p.Cursor)
		buf.WriteByte('\'')
	}
	buf.WriteString(" ORDER BY ")
	buf.WriteString(paginationField)
	buf.WriteString(" DESC LIMIT ")
	buf.WriteString(p.Limit)
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
				buf.WriteByte(',')
			}
			buf.WriteString(f)
		}
	}
}

// replaceAll is like strings.ReplaceAll but it writes the new string
// directly to the buffer passed to save an allocation.
func replaceAll(b *bytes.Buffer, s, old, new string) {
	if old == new {
		b.WriteString(s)
		return
	}

	// Compute number of replacements.
	m := strings.Count(s, old)
	if m == 0 {
		b.WriteString(s)
		return
	}

	if len(new) > len(old) {
		b.Grow(m * (len(new) - len(old)))
	}
	start := 0
	for i := 0; i < m; i++ {
		j := start
		if len(old) == 0 {
			if i > 0 {
				_, wid := utf8.DecodeRuneInString(s[start:])
				j += wid
			}
		} else {
			j += strings.Index(s[start:], old)
		}
		b.WriteString(s[start:j])
		b.WriteString(new)
		start = j + len(old)
	}
	b.WriteString(s[start:])
}
