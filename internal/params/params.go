package params

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"github.com/julienschmidt/httprouter"
	"github.com/pkg/errors"
)

var errInvalidUUIDFormat = errors.New("invalid UUID format")

// Object types
const (
	User = iota
	Event
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
	var fields []string
	switch obj {
	case User:
		fields = split(values.Get("user.fields"))
		if err := validateUserFields(fields); err != nil {
			return Query{}, err
		}
	case Event:
		fields = split(values.Get("event.fields"))
		if err := validateEventFields(fields); err != nil {
			return Query{}, err
		}
	default:
		// Just in case obj is not valid
		fields = nil
	}
	lookupID := values.Get("lookup.id")
	if lookupID != "" {
		if err := ValidateUUID(lookupID); err != nil {
			return Query{}, err
		}
		// As the other fields won't be used, just return here
		return Query{Fields: fields, LookupID: lookupID}, nil
	}

	cursor, err := parseInt(values.Get("cursor"), "0", 200)
	if err != nil {
		return Query{}, errors.Wrap(err, "cursor")
	}
	limit, err := parseInt(values.Get("limit"), "20", 100)
	if err != nil {
		return Query{}, errors.Wrap(err, "limit")
	}

	params := Query{
		Cursor: cursor,
		Limit:  limit,
		Fields: fields,
	}
	return params, nil
}

// GetEventPredicate validates that the predicate passed corresponds to an event node and returns it.
func GetEventPredicate(params httprouter.Params) (string, error) {
	predicate := params.ByName("predicate")
	switch predicate {
	case "banned", "confirmed", "invited", "liked_by":
		return predicate, nil
	default:
		// Predicates accepted: banned, confirmed, invited, liked_by
		return "", errors.Errorf("invalid predicate (%s).", predicate)
	}
}

// UUIDFromCtx takes the id parameter from context and validates it.
func UUIDFromCtx(ctx context.Context) (string, error) {
	id := httprouter.ParamsFromContext(ctx).ByName("id")
	if err := ValidateUUID(id); err != nil {
		return "", err
	}
	return id, nil
}

// ValidateUUID validates that the passed id is a valid UUIDv4 according to RFC4122.
//
// Useful to avoid making a database query with an invalid ValidateUUID.
func ValidateUUID(id string) error {
	switch len(id) {
	// Standard: xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
	case 36:

	// urn:uuid:xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
	case 45:
		if strings.ToLower(id[:9]) != "urn:uuid:" {
			return fmt.Errorf("invalid urn prefix: %q", id[:9])
		}
		id = id[9:]

		// Microsoft: {xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx}
	case 38:
		if id[0] != '{' && id[37] != '}' {
			return errInvalidUUIDFormat
		}
		id = id[1:37]

		// Raw hex: xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
	case 32:
		for i := 0; i < 16; i++ {
			if ok := validBytes(id[i*2], id[i*2+1]); !ok {
				return errInvalidUUIDFormat
			}
		}
		return nil
	default:
		return errors.Errorf("invalid UUID length: %d", len(id))
	}
	// id is now at least 36 bytes long
	// it must be of the form  xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx
	if id[8] != '-' || id[13] != '-' || id[18] != '-' || id[23] != '-' {
		return errInvalidUUIDFormat
	}
	for _, x := range [16]int{
		0, 2, 4, 6,
		9, 11,
		14, 16,
		19, 21,
		24, 26, 28, 30, 32, 34} {
		if ok := validBytes(id[x], id[x+1]); !ok {
			return errInvalidUUIDFormat
		}
	}
	return nil
}

// ValidateUUIDs takes multiple ids and validates them all.
func ValidateUUIDs(ids ...string) error {
	for _, id := range ids {
		if err := ValidateUUID(id); err != nil {
			return errors.Wrapf(err, "%q", id)
		}
	}
	return nil
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
		case "id":
			continue
		case "creator_id":
			continue
		case "created_at":
			continue
		case "updated_at":
			continue
		case "name":
			continue
		case "event_id":
			continue
		case "type":
			continue
		case "public":
			continue
		case "virtual":
			continue
		case "ticket_cost":
			continue
		case "slots":
			continue
		case "attending":
			continue
		case "start_time":
			continue
		case "end_time":
			continue
		case "min_age":
			continue
		default:
			return errors.Errorf("unrecognized field (%s)", f)
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
		case "id":
			continue
		case "created_at":
			continue
		case "updated_at":
			continue
		case "name":
			continue
		case "user_id":
			continue
		case "username":
			continue
		case "email":
			continue
		case "description":
			continue
		case "birth_date":
			continue
		case "profile_image_url":
			continue
		case "premium":
			continue
		case "verified_email":
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

func parseBool(str string) (bool, error) {
	switch str {
	case "true", "True", "TRUE", "t", "T", "1":
		return true, nil
	case "", "false", "False", "FALSE", "f", "F", "0":
		return false, nil
	}
	return false, errors.Errorf("invalid bool %q", str)
}

// parseInt parses an integer from a url value and validates it.
//
// Value and default are strings as both the received (params) and
// used (dgraph query) values are also strings.
func parseInt(value, def string, max int) (string, error) {
	switch value {
	case "":
		return def, nil
	default:
		i, err := strconv.Atoi(value)
		if err != nil {
			return "", errors.Wrap(err, "invalid number")
		}
		if i < 0 {
			return def, nil // TODO: when receiving negative values change orderasc to orderdesc in service funcs
		}
		if max > 0 && i > max {
			return "", errors.Errorf("number provided (%d) exceeded maximum (%d)", i, max)
		}
		return value, nil
	}
}

// UUID utils

// validBytes makes sure the bytes provided are valid.
func validBytes(x1, x2 byte) bool {
	return xvalues[x1] != 255 && xvalues[x2] != 255
}

// xvalues returns the value of a byte as a hexadecimal digit or 255.
var xvalues = [256]byte{
	255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255,
	0, 1, 2, 3, 4, 5, 6, 7, 8, 9, 255, 255, 255, 255, 255, 255,
	255, 10, 11, 12, 13, 14, 15, 255, 255, 255, 255, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255,
	255, 10, 11, 12, 13, 14, 15, 255, 255, 255, 255, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255,
	255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255, 255,
}
