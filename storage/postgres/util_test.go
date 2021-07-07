package postgres

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

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
