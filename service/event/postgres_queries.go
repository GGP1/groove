package event

import (
	"database/sql"

	"github.com/pkg/errors"
)

func scanMedia(rows *sql.Rows) ([]Media, error) {
	var (
		// Reuse object, there's no need to reset fields as they will be always overwritten
		media  Media
		medias []Media
	)

	cols, _ := rows.Columns()
	if len(cols) > 0 {
		columns := mediaColumns(&media, cols)

		for rows.Next() {
			if err := rows.Scan(columns...); err != nil {
				return nil, errors.Wrap(err, "scanning rows")
			}

			medias = append(medias, media)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return medias, nil
}

func mediaColumns(m *Media, columns []string) []interface{} {
	result := make([]interface{}, 0, len(columns))

	for _, c := range columns {
		switch c {
		case "id":
			result = append(result, &m.ID)
		case "event_id":
			result = append(result, &m.EventID)
		case "url":
			result = append(result, &m.URL)
		}
	}

	return result
}

func scanProducts(rows *sql.Rows) ([]Product, error) {
	var (
		// Reuse object, there's no need to reset fields as they will be always overwritten
		product  Product
		products []Product
	)

	cols, _ := rows.Columns()
	if len(cols) > 0 {
		columns := productColumns(&product, cols)

		for rows.Next() {
			if err := rows.Scan(columns...); err != nil {
				return nil, errors.Wrap(err, "scanning rows")
			}

			products = append(products, product)
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return products, nil
}

func productColumns(p *Product, columns []string) []interface{} {
	result := make([]interface{}, 0, len(columns))

	for _, c := range columns {
		switch c {
		case "id":
			result = append(result, &p.ID)
		case "event_id":
			result = append(result, &p.EventID)
		case "stock":
			result = append(result, &p.Stock)
		case "brand":
			result = append(result, &p.Brand)
		case "description":
			result = append(result, &p.Description)
		case "discount":
			result = append(result, &p.Discount)
		case "taxes":
			result = append(result, &p.Taxes)
		case "subtotal":
			result = append(result, &p.Subtotal)
		case "total":
			result = append(result, &p.Total)
		}
	}

	return result
}

func scanUsers(rows *sql.Rows) ([]User, error) {
	var (
		// Reuse object, there's no need to reset fields as they will be always overwritten
		user  User
		users []User
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
