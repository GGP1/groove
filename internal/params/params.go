package params

import (
	"context"
	"net/url"
	"strconv"
	"strings"

	"github.com/GGP1/groove/internal/validate"

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
	if err := validate.ULID(id); err != nil {
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
		if err := validate.ULID(lookupID); err != nil {
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
	} else {
		if err := validate.Cursor(cursor); err != nil {
			return Query{}, err
		}
	}

	params := Query{
		Cursor: cursor,
		Limit:  limit,
		Fields: fields,
	}
	return params, nil
}

func parseBool(value string) (bool, error) {
	if value == "" {
		return false, nil
	}

	b, err := strconv.ParseBool(value)
	if err != nil {
		return false, errors.Errorf("invalid boolean (%q)", value)
	}

	return b, nil
}

func parseFields(obj obj, values url.Values) ([]string, error) {
	var fields []string
	switch obj {
	case User:
		fields = split(values.Get("user.fields"))
		if err := validate.UserFields(fields); err != nil {
			return nil, err
		}

	case Event:
		fields = split(values.Get("event.fields"))
		if err := validate.EventFields(fields); err != nil {
			return nil, err
		}

	case Media:
		fields = split(values.Get("media.fields"))
		if err := validate.MediaFields(fields); err != nil {
			return nil, err
		}

	case Product:
		fields = split(values.Get("product.fields"))
		if err := validate.ProductFields(fields); err != nil {
			return nil, err
		}

	default:
		// Just in case obj is not valid
		fields = nil
	}

	return fields, nil
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

// split is like strings.Split but returns nil if the slice is empty
func split(s string) []string {
	if s == "" {
		return nil
	}
	return strings.Split(s, ",")
}
