package postgres

import (
	"bytes"
	"context"
	"database/sql"
	"strings"

	"github.com/GGP1/groove/internal/bufferpool"
	"github.com/GGP1/groove/internal/params"
	"github.com/GGP1/groove/internal/sqltx"
	"github.com/GGP1/groove/model"

	"github.com/pkg/errors"
)

// AddPagination takes a query and adds pagination/lookup conditions to it.
func AddPagination(query, paginationField string, params params.Query) string {
	buf := bufferpool.Get()
	buf.WriteString(query)
	addPagination(buf, paginationField, params)
	q := buf.String()
	bufferpool.Put(buf)

	return q
}

// AppendInIDs appends an "AND IN (ids...)" string to the query.
func AppendInIDs(query string, ids []string, not bool) string {
	buf := bufferpool.Get()
	buf.WriteString(query)
	buf.WriteString(" AND id ")
	if not {
		buf.WriteString("NOT ")
	}
	buf.WriteString("IN (")
	for i, id := range ids {
		if i != 0 {
			buf.WriteByte(',')
		}
		buf.WriteByte('\'')
		buf.WriteString(id)
		buf.WriteByte('\'')
	}
	buf.WriteByte(')')
	q := buf.String()
	bufferpool.Put(buf)

	return q
}

// BeginTx returns a new sql transaction and a context with it stored.
func BeginTx(ctx context.Context, db *sql.DB, readOnly bool) (*sql.Tx, context.Context) {
	tx, err := db.BeginTx(ctx, &sql.TxOptions{ReadOnly: readOnly})
	if err != nil {
		panic(err)
	}

	return tx, sqltx.NewContext(ctx, tx)
}

// BulkInsert returns a statement to be executed multiple times to copy data into the target table.
//
// After all data has been processed Exec() should be called once with no arguments to flush all buffered data.
// Any call to Exec() might return an error which should be handled appropriately, but because of the internal
// buffering an error returned by Exec() might not be related to the data passed in the call that failed.
//
// BulkInsert uses COPY FROM internally. It is not possible to COPY outside of an explicit transaction in pq.
func BulkInsert(ctx context.Context, tx *sql.Tx, table string, fields ...string) (*sql.Stmt, error) {
	// Table and fields may be required to be enclosed by doublequotes
	buf := bufferpool.Get()
	buf.WriteString("COPY ")
	buf.WriteString(table)
	buf.WriteString(" (")
	for i, f := range fields {
		if i != 0 {
			buf.WriteByte(',')
		}
		buf.WriteString(f)
	}
	buf.WriteString(") FROM STDIN")

	stmt, err := tx.PrepareContext(ctx, buf.String())
	if err != nil {
		return nil, errors.Wrap(err, "creating bulk insert statement")
	}
	bufferpool.Put(buf)

	return stmt, nil
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

// QueryInt returns a string scanned from a single row.
func QueryInt(ctx context.Context, tx *sql.Tx, query string, args ...interface{}) (int64, error) {
	row := tx.QueryRowContext(ctx, query, args...)
	var i int64
	if err := row.Scan(&i); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, errors.Wrap(err, "value not found")
		}
		return 0, errors.Wrap(err, "scanning value")
	}

	return i, nil
}

// QueryString returns a string scanned from a single row.
func QueryString(ctx context.Context, tx *sql.Tx, query string, args ...interface{}) (string, error) {
	row := tx.QueryRowContext(ctx, query, args...)
	var str string
	if err := row.Scan(&str); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
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
//
//	SELECT [fields] FROM [table] WHERE search @@ to_tsquery($1) [pagination].
func FullTextSearch(model model.Model, params params.Query) string {
	buf := bufferpool.Get()

	buf.WriteString("SELECT ")
	WriteFields(buf, model, params.Fields)
	buf.WriteString(" FROM ")
	buf.WriteString(model.Tablename())
	buf.WriteString(" WHERE search @@ to_tsquery($1)")
	addPagination(buf, "id", params)

	q := buf.String()
	bufferpool.Put(buf)

	return q
}

// ToTSQuery formats a string to a tsquery-like syntax.
func ToTSQuery(s string) string {
	// FTS operators: "&" (AND), "<->" (FOLLOWED BY)
	// See https://www.postgresql.org/docs/13/textsearch-controls.html
	// ":*" is used to match prefixes as well
	return strings.ReplaceAll(strings.TrimSpace(s), " ", "&") + ":*"
}

// SelectInID builds a postgres select from in statement.
// [ids] mustn't be a user input.
//
// 	SELECT fields FROM table WHERE id IN ('id1','id2',...) ORDER BY id DESC
func SelectInID(model model.Model, ids, fields []string) string {
	buf := bufferpool.Get()

	buf.WriteString("SELECT ")
	WriteFields(buf, model, fields)
	buf.WriteString(" FROM ")
	buf.WriteString(model.Tablename())
	buf.WriteString(" WHERE id IN (")
	for i, id := range ids {
		if i != 0 {
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

// SelectWhere builds a select statement while receiving parameterized arguments.
//
// 	Format: "SELECT [fields] FROM [table] WHERE [whereCond] [pagination]"
//
// Pagination:
//	Standard: "ORDER BY paginationField DESC LIMIT params.Limit"
//	LookupID: "AND paginationField='params.LookupID'"
//	Cursor: "AND paginationField < 'params.Cursor' ORDER BY paginationField DESC LIMIT params.Limit"
func SelectWhere(model model.Model, whereCond, paginationField string, params params.Query) string {
	buf := bufferpool.Get()

	buf.WriteString("SELECT ")
	WriteFields(buf, model, params.Fields)
	buf.WriteString(" FROM ")
	buf.WriteString(model.Tablename())
	buf.WriteString(" WHERE ")
	buf.WriteString(whereCond)
	addPagination(buf, paginationField, params)

	q := buf.String()
	bufferpool.Put(buf)

	return q
}

// WriteFields writes the fields passed to the query.
func WriteFields(buf *bytes.Buffer, model model.Model, fields []string) {
	if fields == nil {
		// Write default fields
		buf.WriteString(model.DefaultFields())
	} else {
		for i, f := range fields {
			if i != 0 {
				buf.WriteByte(',')
			}
			buf.WriteString(f)
		}
	}
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
