package permissions

import (
	"strings"

	"github.com/GGP1/groove/internal/bufferpool"

	"github.com/pkg/errors"
)

// Endpoint contains a map with the endpoints and the permissions required to access them.
var Endpoint = map[string][]string{
	"/delete":          {All},
	"/bans/add":        {BanUsers},
	"/permission":      {CreatePermission},
	"/role":            {CreateRole},
	"/invited/add":     {InviteUsers},
	"/role/set":        {SetUserRole},
	"/update":          {UpdateEvent},
	"/update/media":    {UpdateMedia},
	"/update/products": {UpdateProduct},
}

const (
	// Host is the default role used when creating an event
	Host = "host"
	// Attendant is the default role used when a user is attending an event
	Attendant = "attendant"
	// Separator used to parse and unparse keys
	Separator = "/"
)

// TODO: users can create permissions and roles but they can't require them in specific cases, maybe make the client send
// data about the resource that will be accessed and make the user set the permissions required beforhand. For example:
// The server receives: zone_1 and requires: zone_1_permissions_required

// Permission
const (
	All              = "*"
	BanUsers         = "ban_users"
	CreatePermission = "create_permission"
	CreateRole       = "create_role"
	InviteUsers      = "invite_users"
	SetUserRole      = "set_user_role"
	UpdateEvent      = "update_event"
	UpdateMedia      = "update_media"
	UpdateProduct    = "update_product"
)

// Set is a group of permissions that the user saved for later use.
// TODO: let users save sets of permissions to re-utilize in other events. users_permissions_sets
type Set struct {
	PermissionKeys []string
}

// TODO: create pre-set permissions like "invite_users", "edit_description" and let clients create custom ones for different purposes.
// The pre-set permissions should refer to control over the API resources

// Require makes sure the user has all the permissions required
func Require(userPermKeys map[string]struct{}, required ...string) error {
	for _, r := range required {
		if _, ok := userPermKeys[r]; !ok {
			return errors.Errorf("permission key %q missing", r)
		}
	}

	return nil
}

// Required returns the permission keys required to access the endpoint passed.
func Required(url string) []string {
	for u, pk := range Endpoint {
		if strings.HasSuffix(url, u) {
			return pk
		}
	}
	return []string{}
}

// ParseKeys ..
func ParseKeys(permissions map[string]struct{}) string {
	buf := bufferpool.Get()

	for p := range permissions {
		buf.WriteString(p)
		buf.WriteString(Separator)
	}
	parsed := buf.String()
	bufferpool.Put(buf)

	return parsed[:len(parsed)-1]
}

// UnparseKeys takes a string with parsed permissions keys and returns a map.
//
// If the string is empty it returns an empty map.
func UnparseKeys(s string) map[string]struct{} {
	n := strings.Count(s, Separator) + 1
	mp := make(map[string]struct{}, n)
	n--
	i := 0
	for i < n {
		idx := strings.Index(s, Separator)
		if idx < 0 {
			break
		}
		mp[s[:idx]] = struct{}{}
		s = s[idx+len(Separator):]
		i++
	}
	mp[s] = struct{}{}

	return mp
}
