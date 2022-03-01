package role_test

import (
	"context"
	"database/sql"
	"log"
	"testing"

	"github.com/GGP1/groove/internal/params"
	"github.com/GGP1/groove/internal/permissions"
	"github.com/GGP1/groove/internal/roles"
	"github.com/GGP1/groove/internal/txgroup"
	"github.com/GGP1/groove/internal/ulid"
	"github.com/GGP1/groove/model"
	"github.com/GGP1/groove/service/auth"
	"github.com/GGP1/groove/service/event/role"
	"github.com/GGP1/groove/test"

	"github.com/go-redis/redis/v8"
	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"
)

var (
	db     *sql.DB
	rdb    *redis.Client
	roleSv role.Service
	ctx    context.Context
)

func TestMain(m *testing.M) {
	test.Main(
		m,
		func(s *sql.DB, r *redis.Client) {
			sqlTx, err := s.BeginTx(context.Background(), nil)
			if err != nil {
				log.Fatal(err)
			}
			ctx = txgroup.NewContext(ctx, txgroup.NewSQLTx(sqlTx))
			db = s
			rdb = r
			roleSv = role.NewService(s, r)
		},
		test.Postgres, test.Redis,
	)
}

func TestPermissions(t *testing.T) {
	eventID := test.CreateEvent(t, db, "permissions")

	expectedKey := "create_permission"
	t.Run("CreatePermission", func(t *testing.T) {
		permission := model.Permission{
			Name:        "create_permission",
			Key:         expectedKey,
			Description: "TestCreatePermission",
		}
		err := roleSv.CreatePermission(ctx, eventID, permission)
		assert.NoError(t, err)
		ctx = test.CommitTx(ctx, t, db)
	})

	t.Run("GetPermissions", func(t *testing.T) {
		permissions, err := roleSv.GetPermissions(ctx, eventID)
		assert.NoError(t, err)

		assert.Equal(t, expectedKey, permissions[0].Key)
	})

	t.Run("RequirePermissions", func(t *testing.T) {
		userID := test.CreateUser(t, db, "email@mail.com", "username")
		err := roleSv.SetRole(ctx, eventID, roles.Attendant, userID)
		assert.NoError(t, err)
		ctx = test.CommitTx(ctx, t, db)

		session := auth.Session{
			ID: userID,
		}
		err = roleSv.RequirePermissions(ctx, session, eventID, permissions.ViewEvent, permissions.ModifyZones)
		assert.Error(t, err)
	})

	t.Run("UpdatePermission", func(t *testing.T) {
		name := "update_permission"
		uptPermission := model.UpdatePermission{
			Name: &name,
		}
		err := roleSv.UpdatePermission(ctx, eventID, expectedKey, uptPermission)
		assert.NoError(t, err)
		ctx = test.CommitTx(ctx, t, db)

		permissions, err := roleSv.GetPermissions(ctx, eventID)
		assert.NoError(t, err)

		assert.Equal(t, name, permissions[0].Name)
	})

	t.Run("DeletePermission", func(t *testing.T) {
		err := roleSv.DeletePermission(ctx, eventID, expectedKey)
		assert.NoError(t, err)
		ctx = test.CommitTx(ctx, t, db)

		_, err = roleSv.GetPermission(ctx, eventID, expectedKey)
		assert.Error(t, err)
		assert.ErrorIs(t, err, sql.ErrNoRows)
	})
}

func TestRoles(t *testing.T) {
	email := "role@email.com"
	userID := test.CreateUser(t, db, email, "username")
	eventID := test.CreateEvent(t, db, "roles")

	expectedRole := model.Role{
		Name:           "invitor",
		PermissionKeys: pq.StringArray{permissions.InviteUsers},
	}

	t.Run("CreateRole", func(t *testing.T) {
		err := roleSv.CreateRole(ctx, eventID, expectedRole)
		assert.NoError(t, err)
		ctx = test.CommitTx(ctx, t, db)
	})

	t.Run("SetRoles", func(t *testing.T) {
		err := roleSv.SetRole(ctx, eventID, expectedRole.Name, userID)
		assert.NoError(t, err)
		ctx = test.CommitTx(ctx, t, db)
	})

	t.Run("GetMembers", func(t *testing.T) {
		gotMembers, err := roleSv.GetMembers(ctx, eventID, params.Query{})
		assert.NoError(t, err)
		assert.Equal(t, 1, len(gotMembers))
		assert.Equal(t, userID, gotMembers[0].ID)
	})

	t.Run("GetMembersCount", func(t *testing.T) {
		count, err := roleSv.GetMembersCount(ctx, eventID)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), count)
	})

	t.Run("GetRoles", func(t *testing.T) {
		roles, err := roleSv.GetRoles(ctx, eventID)
		assert.NoError(t, err)

		assert.Equal(t, 1+len(model.ReservedRoles), len(roles))
		assert.Equal(t, expectedRole, roles[len(model.ReservedRoles)])
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

	t.Run("GetUsersByRole", func(t *testing.T) {
		gotUsers, err := roleSv.GetUsersByRole(ctx, eventID, expectedRole.Name, params.Query{})
		assert.NoError(t, err)
		assert.Equal(t, 1, len(gotUsers))
		assert.Equal(t, userID, gotUsers[0].ID)
	})

	t.Run("GetUsersCountByRole", func(t *testing.T) {
		count, err := roleSv.GetUsersCountByRole(ctx, eventID, expectedRole.Name)
		assert.NoError(t, err)
		assert.Equal(t, int64(1), count)
	})

	t.Run("UserHasRole", func(t *testing.T) {
		t.Run("True", func(t *testing.T) {
			hasRole, err := roleSv.HasRole(ctx, eventID, userID)
			assert.NoError(t, err)
			assert.True(t, hasRole)
		})

		t.Run("False", func(t *testing.T) {
			hasRole, err := roleSv.HasRole(ctx, eventID, ulid.NewString())
			assert.NoError(t, err)
			assert.False(t, hasRole)
		})
	})

	t.Run("IsHost", func(t *testing.T) {
		isHost, err := roleSv.IsHost(ctx, userID, eventID)
		assert.NoError(t, err)
		assert.False(t, isHost)
	})

	t.Run("UpdateRole", func(t *testing.T) {
		updateRole := model.UpdateRole{
			PermissionKeys: &pq.StringArray{permissions.SetUserRole},
		}

		err := roleSv.UpdateRole(ctx, eventID, expectedRole.Name, updateRole)
		assert.NoError(t, err)
		ctx = test.CommitTx(ctx, t, db)

		role, err := roleSv.GetRole(ctx, eventID, expectedRole.Name)
		assert.NoError(t, err)

		assert.Equal(t, *updateRole.PermissionKeys, role.PermissionKeys)
	})

	t.Run("UnsetRole", func(t *testing.T) {
		err := roleSv.UnsetRole(ctx, eventID, userID)
		assert.NoError(t, err)
		ctx = test.CommitTx(ctx, t, db)

		hasRole, err := roleSv.HasRole(ctx, eventID, userID)
		assert.NoError(t, err)
		assert.False(t, hasRole)
	})

	t.Run("DeleteRole", func(t *testing.T) {
		err := roleSv.DeleteRole(ctx, eventID, expectedRole.Name)
		assert.NoError(t, err)
		ctx = test.CommitTx(ctx, t, db)

		_, err = roleSv.GetRole(ctx, eventID, expectedRole.Name)
		assert.Error(t, err)
		assert.ErrorIs(t, err, sql.ErrNoRows)
	})
}
