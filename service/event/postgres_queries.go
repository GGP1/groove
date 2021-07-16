package event

import (
	"database/sql"
	"strconv"

	"github.com/GGP1/groove/internal/bufferpool"
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

func updateEventQuery(e UpdateEvent) string {
	buf := bufferpool.Get()
	buf.WriteString("UPDATE events SET")

	if e.Name != nil {
		buf.WriteString(" name='")
		buf.WriteString(*e.Name)
		buf.WriteString("',")
	}
	if e.Type != nil {
		buf.WriteString(" type=")
		buf.WriteString(strconv.Itoa(int(*e.Type)))
		buf.WriteByte(',')
	}
	if e.StartTime != nil {
		buf.WriteString(" start_time=")
		buf.WriteString(strconv.Itoa(int(*e.StartTime)))
		buf.WriteByte(',')
	}
	if e.EndTime != nil {
		buf.WriteString(" end_time=")
		buf.WriteString(strconv.Itoa(int(*e.EndTime)))
		buf.WriteByte(',')
	}
	if e.Slots != nil {
		buf.WriteString(" slots=")
		buf.WriteString(strconv.Itoa(int(*e.Slots)))
		buf.WriteByte(',')
	}
	if e.TicketCost != nil {
		buf.WriteString(" ticket_cost=")
		buf.WriteString(strconv.Itoa(int(*e.TicketCost)))
		buf.WriteByte(',')
	}
	if e.MinAge != nil {
		buf.WriteString(" min_age=")
		buf.WriteString(strconv.Itoa(int(*e.MinAge)))
		buf.WriteByte(',')
	}

	buf.WriteString(" updated_at=$2 WHERE id=$1")

	q := buf.String()
	bufferpool.Put(buf)

	return q
}
