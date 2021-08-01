package params

import (
	"context"
	"net/url"
	"strconv"
	"strings"

	"github.com/GGP1/groove/internal/ulid"

	"github.com/julienschmidt/httprouter"
	"github.com/pkg/errors"
)

// Object type
const (
	User obj = iota
	Event
	Media
	Product

	// DefaultCursor is the one used in case it isn't provided by the client
	DefaultCursor = "0"

	// maxLimit is the maximum number of objects returned
	maxLimit = 50
	// defaultLimit is the number of objects returned in case none is specified
	defaultLimit = "20"
)

type obj uint8

// Query contains the request parameters provided by the client.
type Query struct {
	Count    bool
	Cursor   string
	Fields   []string
	Limit    string
	LookupID string
}

// IDFromCtx takes the id parameter from context and validates it.
func IDFromCtx(ctx context.Context) (string, error) {
	id := httprouter.ParamsFromContext(ctx).ByName("id")
	if err := ulid.Validate(id); err != nil {
		return "", err
	}
	return id, nil
}

// ParseQuery returns the url params received after validating them.
func ParseQuery(rawQuery string, obj obj) (Query, error) {
	// Note: values.Get() retrieves only the first parameter, it's better to avoid accessing
	// the map manually, also validate the input to avoid HTTP parameter pollution.
	values, err := url.ParseQuery(rawQuery)
	if err != nil {
		return Query{}, err
	}
	count, err := parseBool(values.Get("count"))
	if err != nil {
		return Query{}, err
	}
	if count {
		// As the other fields won't be used, just return here
		return Query{Count: count}, nil
	}

	fields, err := parseFields(obj, values)
	if err != nil {
		return Query{}, err
	}

	if lookupID := values.Get("lookup.id"); lookupID != "" {
		if err := ulid.Validate(lookupID); err != nil {
			return Query{}, err
		}
		// As the other fields won't be used, just return here
		return Query{Fields: fields, LookupID: lookupID}, nil
	}

	limit, err := parseLimit(values.Get("limit"))
	if err != nil {
		return Query{}, errors.Wrap(err, "limit")
	}
	cursor := values.Get("cursor")
	if cursor == "" {
		cursor = DefaultCursor
	}

	params := Query{
		Cursor: cursor,
		Limit:  limit,
		Fields: fields,
	}
	return params, nil
}

// ValidateSearchQuery returns an error if the query contains invalid characters.
func ValidateSearchQuery(query string) error {
	if strings.ContainsAny(query, ";-\\'\":*#$%/|@,¬<>_()[]}{¡~€^") {
		return errors.New("query contains invalid characters")
	}
	return nil
}

func parseFields(obj obj, values url.Values) ([]string, error) {
	var fields []string
	switch obj {
	case User:
		fields = split(values.Get("user.fields"))
		if err := validateUserFields(fields); err != nil {
			return nil, err
		}

	case Event:
		fields = split(values.Get("event.fields"))
		if err := validateEventFields(fields); err != nil {
			return nil, err
		}

	case Media:
		fields = split(values.Get("media.fields"))
		if err := validateMediaFields(fields); err != nil {
			return nil, err
		}

	case Product:
		fields = split(values.Get("product.fields"))
		if err := validateProductFields(fields); err != nil {
			return nil, err
		}

	default:
		// Just in case obj is not valid
		fields = nil
	}

	return fields, nil
}

func parseBool(value string) (bool, error) {
	switch value {
	case "true", "True", "TRUE", "t", "T", "1":
		return true, nil
	case "", "false", "False", "FALSE", "f", "F", "0":
		return false, nil
	}
	return false, errors.Errorf("invalid boolean (%q)", value)
}

// parseLimit parses an integer from a url value and validates it.
//
// The returned value is a string because it will be used in database queries only.
func parseLimit(value string) (string, error) {
	switch value {
	case "":
		return defaultLimit, nil
	default:
		i, err := strconv.Atoi(value)
		if err != nil {
			return "", errors.Wrap(err, "invalid number")
		}
		if i < 1 {
			return defaultLimit, nil
		}
		if i > maxLimit {
			return "", errors.Errorf("number provided (%d) exceeded maximum (%d)", i, maxLimit)
		}
		return value, nil
	}
}

// validateEventFields validates the fields requested.
func validateEventFields(fields []string) error {
	if fields == nil {
		return nil
	}
	for i, f := range fields {
		switch f {
		case "":
			return errors.Errorf("invalid empty field at index %d", i)
		case "id", "created_at", "updated_at", "name", "description", "event_id",
			"type", "public", "virtual", "ticket_cost", "slots", "attending",
			"start_time", "end_time", "min_age":
			continue
		default:
			return errors.Errorf("unrecognized field (%s)", f)
		}
	}
	return nil
}

func validateMediaFields(fields []string) error {
	if fields == nil {
		return nil
	}
	for i, f := range fields {
		switch f {
		case "":
			return errors.Errorf("invalid empty field at index %d", i)
		case "id", "event_id", "url", "created_at":
			continue
		default:
			return errors.Errorf("unrecognized field %q", f)
		}
	}

	return nil
}

func validateProductFields(fields []string) error {
	if fields == nil {
		return nil
	}
	for i, f := range fields {
		switch f {
		case "":
			return errors.Errorf("invalid empty field at index %d", i)
		case "id", "event_id", "stock", "brand", "type", "description",
			"discount", "taxes", "subtotal", "total", "created_at":
			continue
		default:
			return errors.Errorf("unrecognized field %q", f)
		}
	}

	return nil
}

// validateUserFields validates the correctness of the user fields passed.
func validateUserFields(fields []string) error {
	if fields == nil {
		return nil
	}
	for i, f := range fields {
		switch f {
		case "":
			return errors.Errorf("invalid empty field at index %d", i)
		case "id", "created_at", "updated_at", "name", "user_id", "username",
			"email", "description", "birth_date", "profile_image_url",
			"premium", "private", "verified_email":
			continue
		default:
			return errors.Errorf("unrecognized field %q", f)
		}
	}
	return nil
}

// split is like strings.Split but returns nil if the slice is empty
func split(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, ",")
}
