package location

import "database/sql"

// Service provides a locations service.
type Service interface {
	Autocomplete(input string) ([]Location, error)
	GetLocation(locationID string) (Location, error)
}

type service struct {
	db *sql.DB
}

// NewService returns a new location service.
func NewService(db *sql.DB) Service {
	return service{db}
}

// Autocomplete returns a list of locations given an input.
func (s service) Autocomplete(input string) ([]Location, error) {
	// TODO: finish
	return nil, nil
}

// GetLocation fetches the location from an internal database.
func (s service) GetLocation(locationID string) (Location, error) {
	// TODO: finish
	return Location{}, nil
}
