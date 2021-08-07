package postgres

import (
	"fmt"
	"testing"

	"github.com/GGP1/groove/internal/params"
	"github.com/GGP1/groove/internal/ulid"

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

		got := AddPagination(q, "name", params)
		assert.Equal(t, expected, got)
	})

	t.Run("Lookup ID", func(t *testing.T) {
		expected := "SELECT * FROM users WHERE id='123456' AND name='banana'"
		params := params.Query{LookupID: "banana"}

		got := AddPagination(q, "name", params)
		assert.Equal(t, expected, got)
	})

	t.Run("Cursor", func(t *testing.T) {
		expected := "SELECT * FROM users WHERE id='123456' AND name > 'banana' ORDER BY name DESC LIMIT 14"
		params := params.Query{
			Cursor: "banana",
			Limit:  "14",
		}

		got := AddPagination(q, "name", params)
		assert.Equal(t, expected, got)
	})
}

func TestFullTextSearch(t *testing.T) {
	expected := "SELECT testing,full,text,search FROM events WHERE search @@ to_tsquery($1) ORDER BY id DESC LIMIT 7"
	got := FullTextSearch(Events, params.Query{
		Cursor: params.DefaultCursor,
		Fields: []string{"testing", "full", "text", "search"},
		Limit:  "7",
	})

	assert.Equal(t, expected, got)
}

func TestToTSQuery(t *testing.T) {
	expected := "test&query:*"
	got := ToTSQuery(" test query  ")
	assert.Equal(t, expected, got)
}

func TestSelectInID(t *testing.T) {
	t.Run("Users", func(t *testing.T) {
		t.Run("Standard", func(t *testing.T) {
			id1 := ulid.NewString()
			id2 := ulid.NewString()
			q := "SELECT id, name, username, email FROM users WHERE id IN ('%s', '%s')"
			expected := fmt.Sprintf(q, id1, id2)
			got := SelectInID(Users, []string{id1, id2}, []string{"id", "name", "username", "email"})
			assert.Equal(t, expected, got)
		})

		t.Run("Default fields", func(t *testing.T) {
			id1 := ulid.NewString()
			id2 := ulid.NewString()
			q := "SELECT %s FROM users WHERE id IN ('%s', '%s')"
			expected := fmt.Sprintf(q, userDefaultFields, id1, id2)
			got := SelectInID(Users, []string{id1, id2}, nil)
			assert.Equal(t, expected, got)
		})
	})

	t.Run("Events", func(t *testing.T) {
		t.Run("Standard", func(t *testing.T) {
			id1 := ulid.NewString()
			id2 := ulid.NewString()
			q := "SELECT id, name, type, public, start_time, end_time FROM events WHERE id IN ('%s', '%s')"
			expected := fmt.Sprintf(q, id1, id2)
			got := SelectInID(Events, []string{id1, id2}, []string{"id", "name", "type", "public", "start_time", "end_time"})
			assert.Equal(t, expected, got)
		})

		t.Run("Default fields", func(t *testing.T) {
			id1 := ulid.NewString()
			id2 := ulid.NewString()
			q := "SELECT %s FROM events WHERE id IN ('%s', '%s')"
			expected := fmt.Sprintf(q, eventDefaultFields, id1, id2)
			got := SelectInID(Events, []string{id1, id2}, nil)
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

		got := SelectWhere(Events, "event_id=$1", "id", params)
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

		got := SelectWhere(Users, "user_id=$1", "id", params)
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

		got := SelectWhere(Media, "event_id=$1", "id", params)
		assert.Equal(t, expected, got)
	})
}

func BenchmarkSelectInID(b *testing.B) {
	ids := []string{ulid.NewString(), ulid.NewString(), ulid.NewString()}
	fields := []string{"id", "name", "type", "public", "premium", "created_at", "slots", "ticket_cost"}

	for i := 0; i < b.N; i++ {
		SelectInID(Users, ids, fields)
	}
}

func BenchmarkSelectWhere(b *testing.B) {
	params := params.Query{
		LookupID: ulid.NewString(),
		Fields:   []string{"id", "name", "type", "public", "premium", "created_at", "slots", "ticket_cost"},
	}
	whereCond := "event_id=$1"
	paginationField := "id"

	for i := 0; i < b.N; i++ {
		SelectWhere(Events, whereCond, paginationField, params)
	}
}
