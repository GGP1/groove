package event

import (
	"database/sql"

	"github.com/pkg/errors"
)

// scanLocation is the structure used to scan an event's location
type scanLocation struct {
	Country  sql.NullString `json:"country,omitempty"`
	State    sql.NullString `json:"state,omitempty"`
	ZipCode  sql.NullString `json:"zip_code,omitempty" db:"zip_code"`
	City     sql.NullString `json:"city,omitempty"`
	Address  sql.NullString `json:"address,omitempty"`
	Virtual  *bool          `json:"virtual,omitempty"`
	Platform sql.NullString `json:"platform,omitempty"`
	URL      sql.NullString `json:"url,omitempty"`
}

// eventColumns returns the event fields that will be scanned.
//
// When Scan() is called the values are stored inside the variables we passed.
func eventColumns(e *Event, columns []string) []interface{} {
	result := make([]interface{}, 0, len(columns))

	for _, c := range columns {
		switch c {
		case "id":
			result = append(result, &e.ID)
		case "name":
			result = append(result, &e.Name)
		case "type":
			result = append(result, &e.Type)
		case "public":
			result = append(result, &e.Public)
		case "start_time":
			result = append(result, &e.StartTime)
		case "end_time":
			result = append(result, &e.EndTime)
		case "min_age":
			result = append(result, &e.MinAge)
		case "ticket_cost":
			result = append(result, &e.TicketCost)
		case "slots":
			result = append(result, &e.Slots)
		case "created_at":
			result = append(result, &e.CreatedAt)
		case "updated_at":
			result = append(result, &e.UpdatedAt)
		}
	}

	return result
}

func scanEvents(rows *sql.Rows) ([]Event, error) {
	var events []Event

	cols, _ := rows.Columns()
	if len(cols) > 0 {
		// Reuse object, there's no need to reset fields as they will be always overwritten
		var event Event
		columns := eventColumns(&event, cols)

		for rows.Next() {
			if err := rows.Scan(columns...); err != nil {
				return nil, errors.Wrap(err, "scanning event rows")
			}
			events = append(events, event)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return events, nil
}

func scanEventLocation(row *sql.Row) (Location, error) {
	var l scanLocation
	err := row.Scan(&l.Virtual, &l.Country, &l.State,
		&l.ZipCode, &l.City, &l.Address, &l.Platform, &l.URL)
	if err != nil {
		return Location{}, errors.Wrap(err, "scanning event location")
	}

	return Location{
		Virtual:  l.Virtual,
		Country:  l.Country.String,
		State:    l.State.String,
		ZipCode:  l.ZipCode.String,
		City:     l.City.String,
		Address:  l.Address.String,
		Platform: l.Platform.String,
		URL:      l.URL.String,
	}, nil
}

func scanUsers(rows *sql.Rows) ([]User, error) {
	var users []User

	cols, _ := rows.Columns()
	if len(cols) > 0 {
		// Reuse object, there's no need to reset fields as they will be always overwritten
		var user User
		columns := userColumns(&user, cols)

		for rows.Next() {
			if err := rows.Scan(columns...); err != nil {
				return nil, errors.Wrap(err, "scanning rows")
			}

			users = append(users, user)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}

func userColumns(u *User, columns []string) []interface{} {
	result := make([]interface{}, 0, len(columns))

	for _, c := range columns {
		switch c {
		case "id":
			result = append(result, &u.ID)
		case "name":
			result = append(result, &u.Name)
		case "username":
			result = append(result, &u.Username)
		case "email":
			result = append(result, &u.Email)
		case "birth_date":
			result = append(result, &u.BirthDate)
		case "description":
			result = append(result, &u.Description)
		case "premium":
			result = append(result, &u.Premium)
		case "private":
			result = append(result, &u.Private)
		case "verified_email":
			result = append(result, &u.VerifiedEmail)
		case "profile_image_url":
			result = append(result, &u.ProfileImageURL)
		case "created_at":
			result = append(result, &u.CreatedAt)
		case "updated_at":
			result = append(result, &u.UpdatedAt)
		}
	}

	return result
}
