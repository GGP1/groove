package params

import (
	"context"
	"net/url"
	"strconv"
	"strings"

	"github.com/GGP1/groove/internal/validate"
	"github.com/GGP1/groove/model"

	"github.com/julienschmidt/httprouter"
	"github.com/pkg/errors"
)

const (
	// DefaultCursor is the one used in case it isn't provided by the client
	DefaultCursor = "0"
	// DefaultLimit is the number of objects returned in case none is specified
	DefaultLimit = "20"

	// maxLimit is the maximum number of objects returned
	maxLimit = 50
)

// Query contains the request parameters provided by the client.
type Query struct {
	Cursor   string
	Limit    string
	LookupID string
	Fields   []string
	Count    bool
}

// IDFromCtx takes the id parameter from context and validates it.
func IDFromCtx(ctx context.Context, tag ...string) (string, error) {
	tagName := "id"
	if len(tag) > 0 {
		tagName = tag[0]
	}
	id := httprouter.ParamsFromContext(ctx).ByName(tagName)
	if err := validate.ULID(id); err != nil {
		return "", err
	}
	return id, nil
}

// IDAndNameFromCtx returns the id and name parameters from the endpoint's route.
func IDAndNameFromCtx(ctx context.Context) (id, name string, err error) {
	ctxParams := httprouter.ParamsFromContext(ctx)
	id = ctxParams.ByName("id")
	if err = validate.ULID(id); err != nil {
		return "", "", err
	}
	name = strings.ToLower(ctxParams.ByName("name"))
	return
}

// Parse returns the url params received after validating them.
func Parse(rawQuery string, model model.Model) (Query, error) {
	values, err := url.ParseQuery(rawQuery)
	if err != nil {
		return Query{}, err
	}

	return ParseQuery(values, model)
}

// ParseQuery returns the parameters from the url values passed.
func ParseQuery(values url.Values, model model.Model) (Query, error) {
	count, err := parseBool(values.Get("count"))
	if err != nil {
		return Query{}, err
	}
	if count {
		// As the other fields won't be used, just return here
		return Query{Count: count}, nil
	}

	fields, err := parseFields(model, values)
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

func parseFields(model model.Model, values url.Values) ([]string, error) {
	fieldsValue := values.Get(model.URLQueryKey())
	if fieldsValue == "" {
		return nil, nil
	}

	fields := strings.Split(fieldsValue, ",")
	for i, field := range fields {
		if field == "" {
			return nil, errors.Errorf("invalid empty field at index [%d]", i)
		}
		if !model.ValidField(field) {
			return nil, errors.Errorf("unrecognized field (%s)", field)
		}
	}

	return fields, nil
}

// parseLimit parses an integer from a url value and validates it.
//
// The returned value is a string because it will be used in database queries only.
func parseLimit(value string) (string, error) {
	switch value {
	case "":
		return DefaultLimit, nil
	default:
		i, err := strconv.Atoi(value)
		if err != nil {
			return "", errors.Wrap(err, "invalid number")
		}
		if i < 1 {
			return DefaultLimit, nil
		}
		if i > maxLimit {
			return "", errors.Errorf("number provided (%d) exceeded maximum (%d)", i, maxLimit)
		}
		return value, nil
	}
}
