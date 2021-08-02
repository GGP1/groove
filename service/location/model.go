package location

// Location represents the place where the event will take place, it could be on-site or virtual.
type Location struct {
	Country     *string `json:"country,omitempty"`
	CountryCode *string `json:"country_code,omitempty"`
	State       *string `json:"state,omitempty"`
	City        *string `json:"city,omitempty"`
	Address     *string `json:"address,omitempty"`
	Latitude    float64 `json:"latitude,omitempty"`
	Longitude   float64 `json:"longitude,omitempty"`
}
