package postgres_test

import (
	"bytes"
	"context"
	"database/sql"
	"fmt"
	"testing"

	"github.com/GGP1/groove/internal/params"
	"github.com/GGP1/groove/internal/sqltx"
	"github.com/GGP1/groove/internal/ulid"
	"github.com/GGP1/groove/model"
	"github.com/GGP1/groove/storage/postgres"
	"github.com/GGP1/groove/test"

	"github.com/stretchr/testify/assert"
)

func TestAddPagination(t *testing.T) {
	q := "SELECT * FROM users WHERE id='123456'"
	t.Run("Limit", func(t *testing.T) {
		expected := "SELECT * FROM users WHERE id='123456' ORDER BY name DESC LIMIT 6"
		params := params.Query{
			Cursor: params.DefaultCursor,
			Limit:  "6",
		}

		got := postgres.AddPagination(q, "name", params)
		assert.Equal(t, expected, got)
	})

	t.Run("Lookup ID", func(t *testing.T) {
		expected := "SELECT * FROM users WHERE id='123456' AND name='banana'"
		params := params.Query{LookupID: "banana"}

		got := postgres.AddPagination(q, "name", params)
		assert.Equal(t, expected, got)
	})

	t.Run("Cursor", func(t *testing.T) {
		expected := "SELECT * FROM users WHERE id='123456' AND name > 'banana' ORDER BY name DESC LIMIT 14"
		params := params.Query{
			Cursor: "banana",
			Limit:  "14",
		}

		got := postgres.AddPagination(q, "name", params)
		assert.Equal(t, expected, got)
	})
}

func TestAppendInIDs(t *testing.T) {
	ids := []string{"1", "2", "3"}

	t.Run("IN", func(t *testing.T) {
		q := "SELECT * FROM users WHERE event_id='abc' AND id"
		expected := "SELECT * FROM users WHERE event_id='abc' AND id IN ('1','2','3')"
		got := postgres.AppendInIDs(q, ids)

		assert.Equal(t, expected, got)
	})

	t.Run("NOT IN", func(t *testing.T) {
		q := "SELECT * FROM events WHERE id='abc' AND user_id NOT"
		expected := "SELECT * FROM events WHERE id='abc' AND user_id NOT IN ('1','2','3')"
		got := postgres.AppendInIDs(q, ids)

		assert.Equal(t, expected, got)
	})
}

func TestBeginTx(t *testing.T) {
	pool, rsrc, db, err := test.RunPostgres()
	if err != nil {
		t.Fatal(err)
	}

	t.Run("BeginTx", func(t *testing.T) {
		tx, ctx := postgres.BeginTx(context.Background(), db)

		got := sqltx.FromContext(ctx)
		assert.Equal(t, got, tx)

		assert.NoError(t, tx.Rollback())
	})

	t.Run("BeginTxOpts", func(t *testing.T) {
		tx, ctx := postgres.BeginTxOpts(context.Background(), db, sql.LevelDefault)

		got := sqltx.FromContext(ctx)
		assert.Equal(t, got, tx)

		assert.NoError(t, tx.Rollback())
	})

	assert.NoError(t, pool.Purge(rsrc))
}

func TestFullTextSearch(t *testing.T) {
	t.Run("Event", func(t *testing.T) {
		expected := "SELECT testing,full,text,search FROM events WHERE search @@ to_tsquery($1) AND public=true ORDER BY id DESC LIMIT 7"
		got := postgres.FullTextSearch(model.Event, params.Query{
			Cursor: params.DefaultCursor,
			Fields: []string{"testing", "full", "text", "search"},
			Limit:  "7",
		})

		assert.Equal(t, expected, got)
	})
	t.Run("User", func(t *testing.T) {
		expected := "SELECT testing,full,text,search FROM users WHERE search @@ to_tsquery($1) AND private=false ORDER BY id DESC LIMIT 7"
		got := postgres.FullTextSearch(model.User, params.Query{
			Cursor: params.DefaultCursor,
			Fields: []string{"testing", "full", "text", "search"},
			Limit:  "7",
		})

		assert.Equal(t, expected, got)
	})
}

func TestToTSQuery(t *testing.T) {
	expected := "test&query:*"
	got := postgres.ToTSQuery(" test query  ")
	assert.Equal(t, expected, got)
}

func TestSelectInID(t *testing.T) {
	t.Run("Users", func(t *testing.T) {
		t.Run("Standard", func(t *testing.T) {
			id1 := ulid.NewString()
			id2 := ulid.NewString()
			q := "SELECT id,name,username,email FROM users WHERE id IN ('%s','%s') ORDER BY id DESC"
			expected := fmt.Sprintf(q, id1, id2)
			got := postgres.SelectInID(model.User, []string{"id", "name", "username", "email"}, []string{id1, id2})
			assert.Equal(t, expected, got)
		})

		t.Run("Default fields", func(t *testing.T) {
			id1 := ulid.NewString()
			id2 := ulid.NewString()
			q := "SELECT %s FROM users WHERE id IN ('%s','%s') ORDER BY id DESC"
			expected := fmt.Sprintf(q, model.User.DefaultFields(), id1, id2)
			got := postgres.SelectInID(model.User, nil, []string{id1, id2})
			assert.Equal(t, expected, got)
		})
	})

	t.Run("Events", func(t *testing.T) {
		t.Run("Standard", func(t *testing.T) {
			id1 := ulid.NewString()
			id2 := ulid.NewString()
			q := "SELECT id,name,type,public,cron FROM events WHERE id IN ('%s','%s') ORDER BY id DESC"
			expected := fmt.Sprintf(q, id1, id2)
			got := postgres.SelectInID(model.Event, []string{"id", "name", "type", "public", "cron"}, []string{id1, id2})
			assert.Equal(t, expected, got)
		})

		t.Run("Default fields", func(t *testing.T) {
			id1 := ulid.NewString()
			id2 := ulid.NewString()
			q := "SELECT %s FROM events WHERE id IN ('%s','%s') ORDER BY id DESC"
			expected := fmt.Sprintf(q, model.Event.DefaultFields(), id1, id2)
			got := postgres.SelectInID(model.Event, nil, []string{id1, id2})
			assert.Equal(t, expected, got)
		})
	})
}

func TestSelectWhere(t *testing.T) {
	t.Run("Standard", func(t *testing.T) {
		expected := "SELECT id,name FROM events WHERE event_id=$1 ORDER BY id DESC LIMIT 20"

		params := params.Query{
			Fields: []string{"id", "name"},
			Cursor: params.DefaultCursor,
			Limit:  "20",
		}

		got := postgres.SelectWhere(model.Event, "event_id=$1", "id", params)
		assert.Equal(t, expected, got)
	})
	t.Run("Lookup ID", func(t *testing.T) {
		id := ulid.NewString()
		expected := "SELECT email,username,birth_date FROM users WHERE user_id=$1 AND id='" + id + "'"
		params := params.Query{
			Fields:   []string{"email", "username", "birth_date"},
			LookupID: id,
			Limit:    "20",
		}

		got := postgres.SelectWhere(model.User, "user_id=$1", "id", params)
		assert.Equal(t, expected, got)
	})

	t.Run("Cursor", func(t *testing.T) {
		cursor := ulid.NewString()
		expected := "SELECT id,url FROM events_media WHERE event_id=$1 AND id < '" + cursor + "' ORDER BY id DESC LIMIT 5"
		params := params.Query{
			Fields: []string{"id", "url"},
			Cursor: cursor,
			Limit:  "5",
		}

		got := postgres.SelectWhere(model.Post, "event_id=$1", "id", params)
		assert.Equal(t, expected, got)
	})
}

func TestWriteFields(t *testing.T) {
	fields := []string{"id", "cron", "virtual"}
	expected := "SELECT id,cron,virtual FROM events"

	buf := bytes.NewBufferString("SELECT ")
	postgres.WriteFields(buf, model.Event, fields)
	buf.WriteString(" FROM events")

	assert.Equal(t, expected, buf.String())
}

func TestWriteIDs(t *testing.T) {
	ids := []string{ulid.NewString(), ulid.NewString(), ulid.NewString()}
	expected := fmt.Sprintf("SELECT * FROM users WHERE id IN ('%s','%s','%s')", ids[0], ids[1], ids[2])

	buf := bytes.NewBufferString("SELECT * FROM users WHERE id IN ")
	postgres.WriteIDs(buf, ids)

	assert.Equal(t, expected, buf.String())
}

func BenchmarkSelectInID(b *testing.B) {
	ids := []string{ulid.NewString(), ulid.NewString(), ulid.NewString()}
	fields := []string{"id", "name", "type", "public", "created_at", "slots"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		postgres.SelectInID(model.User, ids, fields)
	}
}

func BenchmarkSelectWhere(b *testing.B) {
	params := params.Query{
		LookupID: ulid.NewString(),
		Fields:   []string{"id", "name", "type", "public", "created_at", "slots"},
	}
	whereCond := "event_id=$1"
	paginationField := "id"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		postgres.SelectWhere(model.Event, whereCond, paginationField, params)
	}
}
