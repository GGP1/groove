package postgres

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestBulkInsert(t *testing.T) {
	q := "INSERT INTO events_staff (event_id, role_name, user_id) VALUES"
	eventID := "1234"
	userIDs := []string{"1", "2"}

	expected := "INSERT INTO events_staff (event_id, role_name, user_id) VALUES ('1234','1','staff'), ('1234','2','staff')"
	got := BulkInsertRoles(q, eventID, "staff", userIDs)

	assert.Equal(t, expected, got)
}

func TestSelectInID(t *testing.T) {
	t.Run("Users", func(t *testing.T) {
		t.Run("Standard", func(t *testing.T) {
			uuid1 := uuid.NewString()
			uuid2 := uuid.NewString()
			q := "SELECT id, name, username, email FROM users WHERE id IN ('%s', '%s')"
			expected := fmt.Sprintf(q, uuid1, uuid2)
			got := SelectInID(Users, []string{uuid1, uuid2}, []string{"id", "name", "username", "email"})
			assert.Equal(t, expected, got)
		})

		t.Run("Default fields", func(t *testing.T) {
			uuid1 := uuid.NewString()
			uuid2 := uuid.NewString()
			q := "SELECT %s FROM users WHERE id IN ('%s', '%s')"
			expected := fmt.Sprintf(q, userDefaultFields, uuid1, uuid2)
			got := SelectInID(Users, []string{uuid1, uuid2}, nil)
			assert.Equal(t, expected, got)
		})
	})

	t.Run("Events", func(t *testing.T) {
		t.Run("Standard", func(t *testing.T) {
			uuid1 := uuid.NewString()
			uuid2 := uuid.NewString()
			q := "SELECT id, name, type, public, start_time, end_time FROM events WHERE id IN ('%s', '%s')"
			expected := fmt.Sprintf(q, uuid1, uuid2)
			got := SelectInID(Events, []string{uuid1, uuid2}, []string{"id", "name", "type", "public", "start_time", "end_time"})
			assert.Equal(t, expected, got)
		})

		t.Run("Default fields", func(t *testing.T) {
			uuid1 := uuid.NewString()
			uuid2 := uuid.NewString()
			q := "SELECT %s FROM events WHERE id IN ('%s', '%s')"
			expected := fmt.Sprintf(q, eventDefaultFields, uuid1, uuid2)
			got := SelectInID(Events, []string{uuid1, uuid2}, nil)
			assert.Equal(t, expected, got)
		})
	})
}

func BenchmarkSelectWhereID(b *testing.B) {
	fields := []string{"id", "name", "type", "public", "premium", "created_at", "slots", "ticket_cost"}
	idField := "event_id"
	id := "123456789"

	for i := 0; i < b.N; i++ {
		_ = SelectWhereID(Events, fields, idField, id)
	}
}
