package postgres_test

import (
	"context"
	"database/sql"
	"testing"

	"github.com/GGP1/groove/internal/params"
	"github.com/GGP1/groove/internal/txgroup"
	"github.com/GGP1/groove/model"
	"github.com/GGP1/groove/storage/postgres"
	"github.com/GGP1/groove/test"

	"github.com/stretchr/testify/assert"
)

func TestBeginTx(t *testing.T) {
	db := test.StartPostgres(t)

	t.Run("BeginTx", func(t *testing.T) {
		tx, ctx := postgres.BeginTx(context.Background(), db)

		got := txgroup.SQLTx(ctx)
		assert.Equal(t, got, tx)

		assert.NoError(t, tx.Rollback())
	})

	t.Run("BeginTxOpts", func(t *testing.T) {
		tx, ctx := postgres.BeginTxOpts(context.Background(), db, sql.LevelDefault)

		got := txgroup.SQLTx(ctx)
		assert.Equal(t, got, tx)

		assert.NoError(t, tx.Rollback())
	})
}

func TestToTSQuery(t *testing.T) {
	expected := "test&query:*"
	got := postgres.ToTSQuery(" test query  ")
	assert.Equal(t, expected, got)
}

func TestSelect(t *testing.T) {
	cases := []struct {
		desc     string
		model    model.Model
		query    string
		expected string
		params   params.Query
	}{
		{
			desc:  "Alias",
			model: model.T.Event,
			query: "SELECT {fields} FROM {table} WHERE e.id=$1 {pag}",
			params: params.Query{
				Fields: []string{"id", "name", "slots", "cron"},
			},
			expected: "SELECT e.id,e.name,e.slots,e.cron FROM events AS e WHERE e.id=$1 ORDER BY e.id DESC LIMIT 20",
		},
		{
			desc:     "Default fields",
			model:    model.T.Comment,
			query:    "SELECT {fields} FROM {table} WHERE id=$1 {pag}",
			params:   params.Query{},
			expected: "SELECT " + model.T.Comment.DefaultFields(false) + " FROM events_posts_comments WHERE id=$1 ORDER BY id DESC LIMIT 20",
		},
		{
			desc:  "Alias complex",
			model: model.T.Comment,
			query: `SELECT
			{fields},
			(SELECT COUNT(*) FROM events_posts_comments_likes WHERE comment_id = c.id) as likes_count,
			(SELECT EXISTS(SELECT 1 FROM events_posts_comments_likes WHERE comment_id = c.id AND user_id=$2)) as auth_user_liked
			FROM {table} WHERE post_id=$1 {pag}`,
			params: params.Query{
				Fields: []string{"id", "content"},
			},
			expected: `SELECT
			c.id,c.content,
			(SELECT COUNT(*) FROM events_posts_comments_likes WHERE comment_id = c.id) as likes_count,
			(SELECT EXISTS(SELECT 1 FROM events_posts_comments_likes WHERE comment_id = c.id AND user_id=$2)) as auth_user_liked
			FROM events_posts_comments AS c WHERE post_id=$1 ORDER BY c.id DESC LIMIT 20`,
		},
		{
			desc:  "No alias",
			model: model.T.User,
			query: "SELECT {fields} FROM {table} WHERE id=$1 {pag}",
			params: params.Query{
				Fields: []string{"id", "name"},
			},
			expected: "SELECT id,name FROM users WHERE id=$1 ORDER BY id DESC LIMIT 20",
		},
		{
			desc:  "Limit",
			model: model.T.Event,
			query: "SELECT {fields} FROM {table} WHERE search @@ to_tsquery($1) ORDER BY id DESC LIMIT 7",
			params: params.Query{
				Cursor: params.DefaultCursor,
				Fields: []string{"testing", "full", "text", "search"},
				Limit:  "7",
			},
			expected: "SELECT testing,full,text,search FROM events WHERE search @@ to_tsquery($1) ORDER BY id DESC LIMIT 7",
		},
		{
			desc:  "Cursor",
			model: model.T.Notification,
			query: "SELECT {fields} FROM {table} WHERE receiver_id=$1 {pag}",
			params: params.Query{
				Cursor: "as2d1as2",
				Fields: []string{"id"},
			},
			expected: "SELECT id FROM notifications WHERE receiver_id=$1 AND id < 'as2d1as2' ORDER BY id DESC LIMIT 20",
		},
		{
			desc:  "Lookup ID",
			model: model.T.Product,
			query: "SELECT {fields} FROM {table} WHERE event_id=$1 {pag}",
			params: params.Query{
				LookupID: "as2d1as2",
				Fields:   []string{"id"},
			},
			expected: "SELECT id FROM events_products WHERE event_id=$1 AND id='as2d1as2'",
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			got := postgres.Select(tc.model, tc.query, tc.params)
			assert.Equal(t, tc.expected, got)
		})
	}
}

func BenchmarkSelect(b *testing.B) {
	params := params.Query{
		Fields: []string{"id", "name", "type", "public", "created_at", "slots"},
	}
	query := "SELECT {fields} FROM {table} WHERE event_id=$1 AND slots IN (100, 250, 1000) {pag}"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		postgres.Select(model.T.Event, query, params)
	}
}

func BenchmarkSelectDefault(b *testing.B) {
	params := params.Query{}
	query := "SELECT {fields} FROM {table} WHERE event_id=$1 AND slots IN (100, 250, 1000) {pag}"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		postgres.Select(model.T.Event, query, params)
	}
}
