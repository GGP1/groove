package test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/GGP1/groove/internal/ulid"
	"github.com/GGP1/groove/model"

	"github.com/stretchr/testify/assert"
)

// CreateEvent creates a new user for testing purposes.
func CreateEvent(t testing.TB, db *sql.DB, name string) string {
	ctx := context.Background()
	id := ulid.NewString()
	q := `INSERT INTO events 
	(id, name, type, public, virtual, slots, cron, start_date, end_date, ticket_type) 
	VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)`
	_, err := db.ExecContext(ctx, q,
		id, name, model.GrandPrix, true, false, 100, "48 12 * * * 15", time.Now(), time.Now().Add(time.Hour*2400), 1)
	assert.NoError(t, err)

	return id
}

// CreateUser creates a new user for testing purposes and returns its id.
func CreateUser(t testing.TB, db *sql.DB, email, username string) string {
	id := ulid.NewString()

	q := "INSERT INTO users (id, name, email, username, password, birth_date, type, invitations) VALUES ($1,$2,$3,$4,$5,$6,$7,$8)"
	_, err := db.ExecContext(context.Background(), q, id, "test", email, username, "1", time.Now(), model.Personal, model.Friends)
	assert.NoError(t, err)

	return id
}

// CreatePermission creates a new permission inside an event. The event must exist in the database.
func CreatePermission(t testing.TB, db *sql.DB, eventID string, permission model.Permission) {
	q := "INSERT INTO events_permissions (event_id, key, name, description) VALUES ($1, $2, $3, $4)"
	_, err := db.ExecContext(context.Background(), q, eventID, permission.Key, permission.Name, permission.Description)
	assert.NoError(t, err)
}

// CreateRole creates a new role inside an event. The event must exist in the database.
func CreateRole(t testing.TB, db *sql.DB, eventID string, role model.Role) {
	q := "INSERT INTO events_roles (event_id, name, permission_keys) VALUES ($1, $2, $3)"
	_, err := db.ExecContext(context.Background(), q, eventID, role.Name, role.PermissionKeys)
	assert.NoError(t, err)
}
