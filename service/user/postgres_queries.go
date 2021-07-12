package user

import (
	"database/sql"

	"github.com/GGP1/groove/service/event"

	"github.com/pkg/errors"
)

// Consider creating stored procedures or views to pre-compile queries in the future

// eventColumns returns the event fields that will be scanned.
//
// When Scan() is called the values are stored inside the variables we passed.
func eventColumns(e *event.Event, columns []string) []interface{} {
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
		case "virtual":
			result = append(result, &e.Virtual)
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

func scanEvents(rows *sql.Rows) ([]event.Event, error) {
	var (
		events []event.Event
		// Reuse object, there's no need to reset fields as they will be always overwritten
		event event.Event
	)

	cols, _ := rows.Columns()
	if len(cols) > 0 {
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

func scanUsers(rows *sql.Rows) ([]ListUser, error) {
	var (
		// Reuse object, there's no need to reset fields as they will be always overwritten
		user  ListUser
		users []ListUser
	)

	cols, _ := rows.Columns()
	if len(cols) > 0 {
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

// userColumns returns the user fields that will be scanned.
//
// When Scan() is called the values are stored inside the variables we passed.
func userColumns(u *ListUser, columns []string) []interface{} {
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
		case "invitations":
			result = append(result, &u.Invitations)
		case "created_at":
			result = append(result, &u.CreatedAt)
		case "updated_at":
			result = append(result, &u.UpdatedAt)
		}
	}

	return result
}
