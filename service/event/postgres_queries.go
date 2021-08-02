package event

import (
	"database/sql"

	"github.com/pkg/errors"
)

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
		case "description":
			result = append(result, &e.Description)
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

func scanEvent(row *sql.Row) (Event, error) {
	var (
		event Event
		URL   sql.NullString
	)
	err := row.Scan(&event.ID, &event.Name, &event.Description, &event.Virtual, &URL,
		&event.Type, &event.Public, &event.StartTime, &event.EndTime, &event.Slots,
		&event.MinAge, &event.TicketCost, &event.CreatedAt, &event.UpdatedAt)
	if err != nil {
		return Event{}, err
	}

	event.URL = &URL.String
	if !*event.Virtual {
		// TODO: fetch location from location service
		// event.LocationID
	}

	return event, nil
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
