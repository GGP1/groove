package permissions

import (
	"fmt"
	"strings"
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

func TestParseKeys(t *testing.T) {
	mp := map[string]struct{}{
		BanUsers:    {},
		InviteUsers: {},
		UpdateEvent: {},
	}

	got := ParseKeys(mp)

	for _, s := range strings.Split(got, Separator) {
		if s != BanUsers && s != InviteUsers && s != UpdateEvent {
			t.Fail()
		}
	}
}

func TestUnparseKeys(t *testing.T) {
	expected := map[string]struct{}{
		BanUsers:          {},
		ModifyPermissions: {},
		ModifyRoles:       {},
		InviteUsers:       {},
		ModifyMedia:       {},
		ModifyProducts:    {},
		SetUserRole:       {},
		UpdateEvent:       {},
	}
	permissionsKeys := fmt.Sprintf("%s/%s/%s/%s/%s/%s/%s/%s",
		BanUsers, ModifyPermissions, ModifyRoles, InviteUsers,
		SetUserRole, UpdateEvent, ModifyMedia, ModifyProducts)
	got := UnparseKeys(permissionsKeys)
	assert.Equal(t, expected, got)
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

func BenchmarkParseKeys(b *testing.B) {
	mp := map[string]struct{}{
		BanUsers:          {},
		InviteUsers:       {},
		ModifyMedia:       {},
		ModifyPermissions: {},
		ModifyProducts:    {},
		ModifyRoles:       {},
		ModifyZones:       {},
		SetUserRole:       {},
		UpdateEvent:       {},
	}
	for i := 0; i < b.N; i++ {
		_ = ParseKeys(mp)
	}
}

func BenchmarkUnparsePermissionsKeys(b *testing.B) {
	permissionsKeys := fmt.Sprintf("%s/%s/%s/%s/%s/%s/%s/%s",
		BanUsers, ModifyMedia, ModifyPermissions, ModifyProducts, ModifyRoles, InviteUsers, SetUserRole, UpdateEvent)
	for i := 0; i < b.N; i++ {
		_ = UnparseKeys(permissionsKeys)
	}
}
