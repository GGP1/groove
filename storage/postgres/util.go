package postgres

import (
	"bytes"
	"context"
	"database/sql"
	"strings"

	"github.com/GGP1/groove/internal/bufferpool"
	"github.com/GGP1/groove/internal/params"
	"github.com/GGP1/groove/internal/txgroup"
	"github.com/GGP1/groove/model"

	"github.com/pkg/errors"
)

// BeginTx returns a new sql transaction and a context with it stored.
func BeginTx(ctx context.Context, db *sql.DB) (*sql.Tx, context.Context) {
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		panic(err)
	}

	return tx, txgroup.NewContext(ctx, txgroup.NewSQLTx(tx))
}

// BeginTxOpts is like BeginTx but takes the isolation level as a parameter.
func BeginTxOpts(ctx context.Context, db *sql.DB, isolation sql.IsolationLevel) (*sql.Tx, context.Context) {
	tx, err := db.BeginTx(ctx, &sql.TxOptions{
		ReadOnly:  false,
		Isolation: isolation,
	})
	if err != nil {
		panic(err)
	}

	return tx, txgroup.NewContext(ctx, txgroup.NewSQLTx(tx))
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
func QueryBool(ctx context.Context, db *sql.DB, query string, args ...interface{}) (bool, error) {
	row := db.QueryRowContext(ctx, query, args...)
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
func QueryInt(ctx context.Context, db *sql.DB, query string, args ...interface{}) (int64, error) {
	row := db.QueryRowContext(ctx, query, args...)
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
func QueryString(ctx context.Context, db *sql.DB, query string, args ...interface{}) (string, error) {
	row := db.QueryRowContext(ctx, query, args...)
	var str string
	if err := row.Scan(&str); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", errors.Wrap(err, "value not found")
		}
		return "", errors.Wrap(err, "scanning value")
	}

	return str, nil
}

// ToTSQuery formats a string to a tsquery-like syntax.
func ToTSQuery(query string) string {
	// FTS operators: "&" (AND), "<->" (FOLLOWED BY)
	// See https://www.postgresql.org/docs/13/textsearch-controls.html
	// ":*" is used to match prefixes as well
	return strings.ReplaceAll(strings.TrimSpace(query), " ", "&") + ":*"
}

type selector struct {
	model    model.Model
	buf      *bytes.Buffer
	pagField string
	params   params.Query
	useAlias bool
}

// Select builds a query dinamically, depending on the parameters provided.
//
// It accepts three tokens enclosed by curly braces ("fields", "table" and "pag"),
// those are replaced with the model interface methods.
func Select(m model.Model, query string, params params.Query) string {
	s := &selector{
		buf:      bufferpool.Get(),
		model:    m,
		useAlias: strings.IndexByte(query, '.') != -1,
		pagField: "id",
		params:   params,
	}

	var lastWriteIdx int
	for i, c := range query {
		if c != '{' {
			continue
		}

		s.buf.WriteString(query[lastWriteIdx:i])
		// We wrote until the opening curly brace, now discard it
		i++

		closure := strings.IndexByte(query[i:], '}')
		if closure == -1 {
			// Avoid panic
			continue
		}
		switch query[i : i+closure] {
		case "fields":
			s.writeFields()
		case "table":
			s.buf.WriteString(m.Tablename())
			if s.useAlias {
				s.buf.WriteString(" AS ")
				s.buf.WriteString(m.Alias())
			}
		case "pag":
			// TODO: pagination is always at the end so it would be better
			// to look for it the other way around. A decision is needed on
			// whether the query can contain only unique tokens or repetitive ones.
			s.addPagination()
		}

		i += closure
		lastWriteIdx = i + 1
	}

	remaining := query[lastWriteIdx:]
	if len(remaining) > 0 {
		s.buf.WriteString(remaining)
	}

	q := s.buf.String()
	bufferpool.Put(s.buf)
	return q
}

func (s *selector) writeFields() {
	if len(s.params.Fields) == 0 {
		s.buf.WriteString(s.model.DefaultFields(s.useAlias))
		return
	}
	for i, f := range s.params.Fields {
		if i != 0 {
			s.buf.WriteByte(',')
		}
		s.writeField(f)
	}
}

func (s *selector) addPagination() {
	if s.params.LookupID != "" {
		s.buf.WriteString("AND ")
		s.writeField(s.pagField)
		s.buf.WriteString("='")
		s.buf.WriteString(s.params.LookupID)
		s.buf.WriteRune('\'')
		return
	}
	if s.params.Cursor != params.DefaultCursor && s.params.Cursor != "" {
		s.buf.WriteString("AND ")
		s.writeField(s.pagField)
		s.buf.WriteString(" < '")
		s.buf.WriteString(s.params.Cursor)
		s.buf.WriteString("' ")
	}
	s.buf.WriteString("ORDER BY ")
	s.writeField(s.pagField)
	s.buf.WriteString(" DESC LIMIT ")
	if s.params.Limit == "" {
		s.params.Limit = params.DefaultLimit
	}
	s.buf.WriteString(s.params.Limit)
}

func (s *selector) writeField(field string) {
	if s.useAlias {
		s.buf.WriteString(s.model.Alias())
		s.buf.WriteRune('.')
	}
	s.buf.WriteString(field)
}
