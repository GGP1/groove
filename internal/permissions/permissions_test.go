package permissions

import (
	"fmt"
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
	})

	t.Run("Fail", func(t *testing.T) {
		err := Require(userPermsKeys, UpdateMedia)
		assert.Error(t, err)
	})
}

func TestRequired(t *testing.T) {
	url := "/update"
	got := Required(url)
	assert.Equal(t, Endpoint[url], got)

}

func TestParseKeys(t *testing.T) {
	mp := map[string]struct{}{
		BanUsers:    {},
		InviteUsers: {},
		UpdateEvent: {},
	}
	expected := "ban_users/invite_users/update_event"

	got := ParseKeys(mp)
	assert.Equal(t, expected, got)
}

func TestUnparseKeys(t *testing.T) {
	expected := map[string]struct{}{
		BanUsers:         {},
		CreatePermission: {},
		CreateRole:       {},
		InviteUsers:      {},
		SetUserRole:      {},
		UpdateEvent:      {},
		UpdateMedia:      {},
		UpdateProduct:    {},
	}
	permissionsKeys := fmt.Sprintf("%s/%s/%s/%s/%s/%s/%s/%s",
		BanUsers, CreatePermission, CreateRole, InviteUsers, SetUserRole, UpdateEvent, UpdateMedia, UpdateProduct)
	got := UnparseKeys(permissionsKeys)
	assert.Equal(t, expected, got)
}

func BenchmarkParseKeys(b *testing.B) {
	mp := map[string]struct{}{
		BanUsers:         {},
		CreatePermission: {},
		CreateRole:       {},
		InviteUsers:      {},
		SetUserRole:      {},
		UpdateEvent:      {},
		UpdateMedia:      {},
		UpdateProduct:    {},
	}
	for i := 0; i < b.N; i++ {
		_ = ParseKeys(mp)
	}
}

func BenchmarkUnparsePermissionsKeys(b *testing.B) {
	permissionsKeys := fmt.Sprintf("%s/%s/%s/%s/%s/%s/%s/%s",
		BanUsers, CreatePermission, CreateRole, InviteUsers, SetUserRole, UpdateEvent, UpdateMedia, UpdateProduct)
	for i := 0; i < b.N; i++ {
		_ = UnparseKeys(permissionsKeys)
	}
}
