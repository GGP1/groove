package event

import (
	"context"
	"database/sql"

	"github.com/GGP1/groove/internal/params"
	"github.com/GGP1/groove/service/event/media"
	"github.com/GGP1/groove/service/event/product"
	"github.com/GGP1/groove/service/event/role"
	"github.com/GGP1/groove/service/event/zone"
)

// Important: methods inside this file must return the methods from the services
// and not themselves, otherwise it would cause an infinite recursion.

// ClonePermissions takes the permissions from the exporter event and creates them in the importer event.
func (s *service) ClonePermissions(ctx context.Context, sqlTx *sql.Tx, exporterEventID, importerEventID string) error {
	s.metrics.incMethodCalls("ClonePermissions")
	return s.roleService.ClonePermissions(ctx, sqlTx, exporterEventID, importerEventID)
}

// CloneRoles takes the role from the exporter event and creates them in the importer event.
func (s *service) CloneRoles(ctx context.Context, sqlTx *sql.Tx, exporterEventID, importerEventID string) error {
	s.metrics.incMethodCalls("CloneRoles")
	return s.roleService.CloneRoles(ctx, sqlTx, exporterEventID, importerEventID)
}

// CreateMedia adds a photo or video to the event.
func (s *service) CreateMedia(ctx context.Context, sqlTx *sql.Tx, eventID string, media media.CreateMedia) error {
	s.metrics.incMethodCalls("CreateMedia")
	return s.mediaService.CreateMedia(ctx, sqlTx, eventID, media)
}

// CreatePermission creates a permission inside the event.
func (s *service) CreatePermission(ctx context.Context, sqlTx *sql.Tx, eventID string, permission role.Permission) error {
	s.metrics.incMethodCalls("CreatePermission")
	return s.roleService.CreatePermission(ctx, sqlTx, eventID, permission)
}

// CreateProduct adds a product to the event.
func (s *service) CreateProduct(ctx context.Context, sqlTx *sql.Tx, eventID string, product product.Product) error {
	s.metrics.incMethodCalls("CreateProduct")
	return s.productService.CreateProduct(ctx, sqlTx, eventID, product)
}

// CreateRole creates a new role inside an event.
func (s *service) CreateRole(ctx context.Context, sqlTx *sql.Tx, eventID string, role role.Role) error {
	s.metrics.incMethodCalls("CreateRole")
	return s.roleService.CreateRole(ctx, sqlTx, eventID, role)
}

// CreateZone creates a new zone inside an event.
func (s *service) CreateZone(ctx context.Context, sqlTx *sql.Tx, eventID string, zone zone.Zone) error {
	s.metrics.incMethodCalls("CreateZone")
	return s.zoneService.CreateZone(ctx, sqlTx, eventID, zone)
}

// DeleteMedia removes a media from an event.
func (s *service) DeleteMedia(ctx context.Context, sqlTx *sql.Tx, eventID, mediaID string) error {
	s.metrics.incMethodCalls("DeleteMedia")
	return s.mediaService.DeleteMedia(ctx, sqlTx, eventID, mediaID)
}

// DeleteProduct removes a product from an event.
func (s *service) DeleteProduct(ctx context.Context, sqlTx *sql.Tx, eventID, productID string) error {
	s.metrics.incMethodCalls("DeleteProduct")
	return s.productService.DeleteProduct(ctx, sqlTx, eventID, productID)
}

// DeletePermission removes a permission from the event.
func (s *service) DeletePermission(ctx context.Context, sqlTx *sql.Tx, eventID, key string) error {
	s.metrics.incMethodCalls("DeletePermission")
	return s.roleService.DeletePermission(ctx, sqlTx, eventID, key)
}

// DeleteRole removes a role from the event.
func (s *service) DeleteRole(ctx context.Context, sqlTx *sql.Tx, eventID, name string) error {
	s.metrics.incMethodCalls("DeleteRole")
	return s.roleService.DeleteRole(ctx, sqlTx, eventID, name)
}

// DeleteZone removes a zone from the event.
func (s *service) DeleteZone(ctx context.Context, sqlTx *sql.Tx, eventID, name string) error {
	s.metrics.incMethodCalls("DeleteZone")
	return s.zoneService.DeleteZone(ctx, sqlTx, eventID, name)
}

func (s *service) GetMedia(ctx context.Context, sqlTx *sql.Tx, eventID string, params params.Query) ([]media.Media, error) {
	s.metrics.incMethodCalls("GetMedia")
	return s.mediaService.GetMedia(ctx, sqlTx, eventID, params)
}

// GetPermissions returns all event's permissions.
func (s *service) GetPermissions(ctx context.Context, sqlTx *sql.Tx, eventID string) ([]role.Permission, error) {
	s.metrics.incMethodCalls("GetPermissions")
	return s.roleService.GetPermissions(ctx, sqlTx, eventID)
}

// GetProducts returns the product from an event.
func (s *service) GetProducts(ctx context.Context, sqlTx *sql.Tx, eventID string, params params.Query) ([]product.Product, error) {
	s.metrics.incMethodCalls("GetProducts")
	return s.productService.GetProducts(ctx, sqlTx, eventID, params)
}

// GetRole returns a role in a given event.
func (s *service) GetRole(ctx context.Context, sqlTx *sql.Tx, eventID, name string) (role.Role, error) {
	s.metrics.incMethodCalls("GetRole")
	return s.roleService.GetRole(ctx, sqlTx, eventID, name)
}

// GetRoles returns all event's role.
func (s *service) GetRoles(ctx context.Context, sqlTx *sql.Tx, eventID string) ([]role.Role, error) {
	s.metrics.incMethodCalls("GetRoles")
	return s.roleService.GetRoles(ctx, sqlTx, eventID)
}

// GetUserRole returns user's role inside the event.
func (s *service) GetUserRole(ctx context.Context, sqlTx *sql.Tx, eventID, userID string) (role.Role, error) {
	s.metrics.incMethodCalls("GetUserRole")
	return s.roleService.GetUserRole(ctx, sqlTx, eventID, userID)
}

// GetZoneByName returns the permission keys required to enter a zone.
func (s *service) GetZoneByName(ctx context.Context, sqlTx *sql.Tx, eventID, name string) (zone.Zone, error) {
	s.metrics.incMethodCalls("GetZoneByName")
	return s.zoneService.GetZoneByName(ctx, sqlTx, eventID, name)
}

// GetZones gets an event's zones.
func (s *service) GetZones(ctx context.Context, sqlTx *sql.Tx, eventID string) ([]zone.Zone, error) {
	s.metrics.incMethodCalls("GetZones")
	return s.zoneService.GetZones(ctx, sqlTx, eventID)
}

// IsHost returns if the user's role in the events passed is host.
func (s *service) IsHost(ctx context.Context, sqlTx *sql.Tx, userID string, eventIDs ...string) (bool, error) {
	s.metrics.incMethodCalls("IsHost")
	return s.roleService.IsHost(ctx, sqlTx, userID, eventIDs...)
}

// SetRoles assigns a role to n users inside an event.
func (s *service) SetRoles(ctx context.Context, sqlTx *sql.Tx, eventID, roleName string, userIDs ...string) error {
	s.metrics.incMethodCalls("SetRoles")
	return s.roleService.SetRoles(ctx, sqlTx, eventID, roleName, userIDs...)
}

// SetViewerRole assigns the viewer role to a user.
func (s service) SetViewerRole(ctx context.Context, sqlTx *sql.Tx, eventID, userID string) error {
	s.metrics.incMethodCalls("SetViewerRole")
	return s.roleService.SetViewerRole(ctx, sqlTx, eventID, userID)
}

// UpdateMedia updates event's media.
func (s *service) UpdateMedia(ctx context.Context, sqlTx *sql.Tx, eventID string, media media.Media) error {
	s.metrics.incMethodCalls("UpdateMedia")
	return s.mediaService.UpdateMedia(ctx, sqlTx, eventID, media)
}

// UpdatePermission ..
func (s *service) UpdatePermission(ctx context.Context, sqlTx *sql.Tx, eventID, key string, permission role.UpdatePermission) error {
	s.metrics.incMethodCalls("UpdatePermission")
	return s.roleService.UpdatePermission(ctx, sqlTx, eventID, key, permission)
}

// UpdateProduct updates an event product.
func (s *service) UpdateProduct(ctx context.Context, sqlTx *sql.Tx, eventID string, product product.UpdateProduct) error {
	s.metrics.incMethodCalls("UpdateProduct")
	return s.productService.UpdateProduct(ctx, sqlTx, eventID, product)
}

// UpdateRole ..
func (s *service) UpdateRole(ctx context.Context, sqlTx *sql.Tx, eventID, name string, role role.UpdateRole) error {
	s.metrics.incMethodCalls("UpdateRole")
	return s.roleService.UpdateRole(ctx, sqlTx, eventID, name, role)
}

// UserHasRole returns if the user has a role inside the event or not.
func (s *service) UserHasRole(ctx context.Context, sqlTx *sql.Tx, eventID, userID string) (bool, error) {
	s.metrics.incMethodCalls("UserHasRole")
	return s.roleService.UserHasRole(ctx, sqlTx, eventID, userID)
}
