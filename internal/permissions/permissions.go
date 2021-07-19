package permissions

import (
	"strings"

	"github.com/GGP1/groove/internal/bufferpool"
	"github.com/GGP1/groove/internal/romap"

	"github.com/pkg/errors"
)

// ReservedKeys is a read-only map containing all reserved permissions keys.
var ReservedKeys = romap.New(map[string]interface{}{
	All:              struct{}{},
	Access:           struct{}{},
	BanUsers:         struct{}{},
	CreatePermission: struct{}{},
	CreateRole:       struct{}{},
	CreateZone:       struct{}{},
	InviteUsers:      struct{}{},
	SetUserRole:      struct{}{},
	UpdateEvent:      struct{}{},
	UpdateMedia:      struct{}{},
	UpdateProduct:    struct{}{},
})

// Pre-defined permission key.
const (
	All              = "*"
	Access           = "access"
	BanUsers         = "ban_users"
	CreatePermission = "create_permission"
	CreateRole       = "create_role"
	CreateZone       = "create_zone"
	InviteUsers      = "invite_users"
	SetUserRole      = "set_user_role"
	UpdateEvent      = "update_event"
	UpdateMedia      = "update_media"
	UpdateProduct    = "update_product"

	// Separator is used to parse and unparse keys.
	Separator = "/"
)

// Require makes sure the user has all the permissions required.
func Require(userPermKeys map[string]struct{}, required ...string) error {
	if _, ok := userPermKeys[All]; ok {
		return nil
	}

	if len(required) > len(userPermKeys) {
		return errors.New("permission keys missing")
	}

	for _, r := range required {
		if _, ok := userPermKeys[r]; !ok {
			return errors.Errorf("permission key %q missing", r)
		}
	}

	return nil
}

// ParseKeys parses the permissions keys to a string.
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
