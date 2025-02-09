package roles

import (
	"github.com/GGP1/groove/internal/permissions"
	"github.com/GGP1/groove/internal/romap"
)

const (
	// Host is the default role used when creating an event
	Host = "host"
	// Attendant is the default role used when a user is attending an event
	Attendant = "attendant"
	// Moderator is in charge of banning problematic users
	Moderator = "moderator"
	// Viewer can see the content's of an event's page
	Viewer = "viewer"
)

// Reserved is a read-only map that contains reserved roles
// and its permission keys applying to all events.
var Reserved = romap.New(map[string][]string{
	Host:      {permissions.All},
	Attendant: {permissions.Access},
	Moderator: {permissions.Access, permissions.BanUsers},
	Viewer:    {permissions.ViewEvent},
})
