package roles

import (
	"github.com/GGP1/groove/internal/permissions"
	"github.com/GGP1/groove/internal/romap"
)

const (
	// Host is the default role used when creating an event
	Host Name = "host"
	// Attendant is the default role used when a user is attending an event
	Attendant Name = "attendant"
	// Moderator is in charge of banning problematic users
	Moderator Name = "moderator"
	// Viewer can see the content's of an event's page
	Viewer Name = "viewer"
)

// Name represents a reserved role name.
type Name string

// Reserved is a read-only map that contains reserved roles
// and its permission keys applying to all events.
var Reserved = romap.New(map[string]interface{}{
	string(Host):      []string{permissions.All},
	string(Attendant): []string{permissions.Access},
	string(Moderator): []string{permissions.Access, permissions.BanUsers},
	string(Viewer):    []string{permissions.ViewEvent},
})
