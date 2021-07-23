package permissions

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRequire(t *testing.T) {
	userPermsKeys := map[string]struct{}{
		InviteUsers: {},
		SetUserRole: {},
	}

	t.Run("Success", func(t *testing.T) {
		err := Require(userPermsKeys, InviteUsers)
		assert.NoError(t, err)

		hostPerm := map[string]struct{}{All: {}}
		err = Require(hostPerm, ModifyRoles, ModifyZones)
		assert.NoError(t, err)
	})

	t.Run("Fail", func(t *testing.T) {
		err := Require(userPermsKeys, ModifyMedia)
		assert.Error(t, err)

		err = Require(userPermsKeys, Access, BanUsers, ModifyPermissions)
		assert.Error(t, err)
	})
}

func BenchmarkRequire(b *testing.B) {
	userPermsKeys := map[string]struct{}{
		Access:            {},
		BanUsers:          {},
		ModifyPermissions: {},
		ModifyRoles:       {},
		ModifyZones:       {},
		InviteUsers:       {},
		ModifyMedia:       {},
		ModifyProducts:    {},
		SetUserRole:       {},
		UpdateEvent:       {},
	}
	required := []string{
		Access,
		BanUsers,
		ModifyPermissions,
		ModifyRoles,
		ModifyZones,
		InviteUsers,
		ModifyMedia,
		ModifyProducts,
		SetUserRole,
		UpdateEvent,
	}
	for i := 0; i < b.N; i++ {
		_ = Require(userPermsKeys, required...)
	}
}
