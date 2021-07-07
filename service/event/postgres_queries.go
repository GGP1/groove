package event

import (
	"database/sql"

	"github.com/pkg/errors"
)

func scanUser(rows *sql.Rows) (User, error) {
	var u User
	cols, _ := rows.Columns()
	if len(cols) > 0 {
		columns := userColumns(&u, cols)
		if err := rows.Scan(columns...); err != nil {
			return User{}, errors.Wrap(err, "scanning rows")
		}
	}

	return u, nil
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
