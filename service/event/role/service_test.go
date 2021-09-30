package role_test

import (
	"context"
	"log"
	"os"
	"testing"
	"time"

	"github.com/GGP1/groove/internal/cache"
	"github.com/GGP1/groove/internal/permissions"
	"github.com/GGP1/groove/internal/roles"
	"github.com/GGP1/groove/internal/sqltx"
	"github.com/GGP1/groove/internal/ulid"
	"github.com/GGP1/groove/service/event/role"
	"github.com/GGP1/groove/test"

	"github.com/dgraph-io/dgo/v210"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
)

var (
	roleSv      role.Service
	dc          *dgo.Dgraph
	ctx         context.Context
	cacheClient cache.Client
)

func TestMain(m *testing.M) {
	poolPg, resourcePg, postgres, err := test.RunPostgres()
	if err != nil {
		log.Fatal(err)
	}
	poolDc, resourceDc, dgraph, conn, err := test.RunDgraph()
	if err != nil {
		log.Fatal(err)
	}
	poolMc, resourceMc, memcached, err := test.RunMemcached()
	if err != nil {
		log.Fatal(err)
	}

	sqlTx, err := postgres.BeginTx(context.Background(), nil)
	if err != nil {
		log.Fatal(err)
	}
	ctx = sqltx.NewContext(ctx, sqlTx)
	cacheClient = memcached

	roleSv = role.NewService(postgres, dgraph, cacheClient)

	code := m.Run()

	if err := sqlTx.Rollback(); err != nil {
		log.Fatal(err)
	}

	if err := poolPg.Purge(resourcePg); err != nil {
		log.Fatal(err)
	}
	if err := poolDc.Purge(resourceDc); err != nil {
		log.Fatal(err)
	}
	if err := conn.Close(); err != nil {
		log.Fatal(err)
	}
	if err := poolMc.Purge(resourceMc); err != nil {
		log.Fatal(err)
	}

	os.Exit(code)
}

func TestCreatePermission(t *testing.T) {
	eventID := ulid.NewString()

	err := createEvent(eventID, "create_permission")
	assert.NoError(t, err)

	permission := role.Permission{
		Name:        "create_permission",
		Key:         "create_permission",
		Description: "TestCreatePermission",
	}
	err = roleSv.CreatePermission(ctx, eventID, permission)
	assert.NoError(t, err)
}

func TestGetPermissions(t *testing.T) {
	eventID := ulid.NewString()

	err := createEvent(eventID, "permissions")
	assert.NoError(t, err)

	expectedKey := "create_permission"
	t.Run("CreatePermission", func(t *testing.T) {
		permission := role.Permission{
			Name:        "create_permission",
			Key:         expectedKey,
			Description: "TestCreatePermission",
		}
		err = roleSv.CreatePermission(ctx, eventID, permission)
		assert.NoError(t, err)
	})

	t.Run("GetPermissions", func(t *testing.T) {
		permissions, err := roleSv.GetPermissions(ctx, eventID)
		assert.NoError(t, err)

		assert.Equal(t, expectedKey, permissions[0].Key)
	})

	t.Run("UpdatePermission", func(t *testing.T) {
		name := "update_permission"
		uptPermission := role.UpdatePermission{
			Name: &name,
		}
		err := roleSv.UpdatePermission(ctx, eventID, expectedKey, uptPermission)
		assert.NoError(t, err)

		permissions, err := roleSv.GetPermissions(ctx, eventID)
		assert.NoError(t, err)

		assert.Equal(t, name, permissions[0].Name)
	})

	t.Run("DeletePermission", func(t *testing.T) {
		err := roleSv.DeletePermission(ctx, eventID, expectedKey)
		assert.NoError(t, err)

		permissions, err := roleSv.GetPermissions(ctx, eventID)
		assert.NoError(t, err)

		assert.Equal(t, 0, len(permissions))
	})
}

func TestRoles(t *testing.T) {
	eventID := ulid.NewString()
	userID := ulid.NewString()

	email := "role@email.com"
	err := createUser(userID, email, "role")
	assert.NoError(t, err)

	err = createEvent(eventID, "roles")
	assert.NoError(t, err)

	expectedRole := role.Role{
		Name:           string(roles.Attendant),
		PermissionKeys: pq.StringArray{permissions.InviteUsers},
	}

	t.Run("CreateRole", func(t *testing.T) {
		err = roleSv.CreateRole(ctx, eventID, expectedRole)
		assert.NoError(t, err)
	})

	t.Run("DeleteRole", func(t *testing.T) {
		name := "delete"
		err := roleSv.CreateRole(ctx, eventID, role.Role{Name: name, PermissionKeys: pq.StringArray{"abc"}})
		assert.NoError(t, err)

		err = roleSv.DeleteRole(ctx, eventID, name)
		assert.NoError(t, err)

		_, err = roleSv.GetRole(ctx, eventID, name)
		assert.Error(t, err)
	})

	t.Run("SetRoles", func(t *testing.T) {
		err = roleSv.SetRoles(ctx, eventID, expectedRole.Name, userID)
		assert.NoError(t, err)
	})

	t.Run("GetRoles", func(t *testing.T) {
		roles, err := roleSv.GetRoles(ctx, eventID)
		assert.NoError(t, err)

		assert.Equal(t, 1, len(roles))
		assert.Equal(t, expectedRole, roles[0])
	})

	t.Run("GetRole", func(t *testing.T) {
		role, err := roleSv.GetRole(ctx, eventID, expectedRole.Name)
		assert.NoError(t, err)

		assert.Equal(t, expectedRole, role)
	})

	t.Run("GetUserRole", func(t *testing.T) {
		gotRole, err := roleSv.GetUserRole(ctx, eventID, userID)
		assert.NoError(t, err)

		assert.Equal(t, expectedRole, gotRole)
	})

	t.Run("SetReservedRole", func(t *testing.T) {
		expectedRole := roles.Viewer
		err := roleSv.SetReservedRole(ctx, eventID, userID, expectedRole)
		assert.NoError(t, err)

		gotRole, err := roleSv.GetUserRole(ctx, eventID, userID)
		assert.NoError(t, err)

		assert.Equal(t, expectedRole, gotRole.Name)
	})

	t.Run("UserHasRole", func(t *testing.T) {
		t.Run("True", func(t *testing.T) {
			ok, err := roleSv.HasRole(ctx, eventID, userID)
			assert.NoError(t, err)

			assert.True(t, ok)
		})

		t.Run("False", func(t *testing.T) {
			ok, err := roleSv.HasRole(ctx, eventID, ulid.NewString())
			assert.NoError(t, err)

			assert.False(t, ok)
		})
	})
}

func createEvent(id, name string) error {
	sqlTx := sqltx.FromContext(ctx)
	q := `INSERT INTO events 
	(id, name, type, public, virtual, slots, start_time, end_Time) 
	VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`
	_, err := sqlTx.ExecContext(ctx, q, id, name, 1, true, false, 100, 15000, 320000)
	return err
}

func createUser(id, email, username string) error {
	sqlTx := sqltx.FromContext(ctx)
	q := "INSERT INTO users (id, name, email, username, password, birth_date) VALUES ($1,$2,$3,$4,$5,$6)"
	_, err := sqlTx.ExecContext(ctx, q, id, "test", email, username, "password", time.Now())

	return err
}
