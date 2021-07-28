package postgres

import (
	"bytes"
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

func TestBulkInsert(t *testing.T) {
	q := "INSERT INTO events_staff (event_id, role_name, user_id) VALUES"
	eventID := "1234"
	userIDs := []string{"1", "2"}

	expected := "INSERT INTO events_staff (event_id, role_name, user_id) VALUES ('1234','1','staff'), ('1234','2','staff')"
	got := BulkInsertRoles(q, eventID, "staff", userIDs)

	assert.Equal(t, expected, got)
}

func TestFullTextSearch(t *testing.T) {
	expected := "SELECT testing,full,text,search FROM events WHERE search @@ to_tsquery('query&text:*') ORDER BY id DESC LIMIT 7"
	query := "query text"
	got := FullTextSearch(Events, query, params.Query{
		Cursor: params.DefaultCursor,
		Fields: []string{"testing", "full", "text", "search"},
		Limit:  "7",
	})

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

func TestSelectWhereID(t *testing.T) {
	t.Run("Standard", func(t *testing.T) {
		expected := "SELECT id, name FROM events WHERE event_id='123456' ORDER BY id DESC LIMIT 20"

		params := params.Query{
			Fields: []string{"id", "name"},
			Cursor: params.DefaultCursor,
			Limit:  "20",
		}

		got := SelectWhereID(Events, "event_id", "123456", "id", params)
		assert.Equal(t, expected, got)
	})
	t.Run("Lookup ID", func(t *testing.T) {
		expected := "SELECT email, username, birth_date FROM users WHERE user_id='012345' AND id='abcdefgh'"
		params := params.Query{
			Fields:   []string{"email", "username", "birth_date"},
			LookupID: "abcdefgh",
			Limit:    "20",
		}

		got := SelectWhereID(Users, "user_id", "012345", "id", params)
		assert.Equal(t, expected, got)
	})

	t.Run("Cursor", func(t *testing.T) {
		expected := "SELECT id, url FROM events_media WHERE event_id='qwertyu' AND id < 'asdfghj' ORDER BY id DESC LIMIT 5"
		params := params.Query{
			Fields: []string{"id", "url"},
			Cursor: "asdfghj",
			Limit:  "5",
		}

		got := SelectWhereID(Media, "event_id", "qwertyu", "id", params)
		assert.Equal(t, expected, got)
	})
}

func TestReplaceAllWithBuf(t *testing.T) {
	var buf bytes.Buffer
	buf.WriteString("SELECT * FROM users WHERE search @@ to_tsquery('")
	replaceAll(&buf, "replace all strings')", " ", " & ")

	assert.Equal(t, "SELECT * FROM users WHERE search @@ to_tsquery('replace & all & strings')", buf.String())
}

func BenchmarkSelectInID(b *testing.B) {
	ids := []string{ulid.NewString(), ulid.NewString(), ulid.NewString()}
	fields := []string{"id", "name", "type", "public", "premium", "created_at", "slots", "ticket_cost"}

	for i := 0; i < b.N; i++ {
		SelectInID(Users, ids, fields)
	}
}

func BenchmarkSelectWhereID(b *testing.B) {
	params := params.Query{
		LookupID: ulid.NewString(),
		Fields:   []string{"id", "name", "type", "public", "premium", "created_at", "slots", "ticket_cost"},
	}
	idField := "event_id"
	id := ulid.NewString()
	paginationField := "id"

	for i := 0; i < b.N; i++ {
		SelectWhereID(Events, idField, id, paginationField, params)
	}
}
