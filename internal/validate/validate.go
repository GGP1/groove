package validate

import (
	"math"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"unicode"

	"github.com/GGP1/groove/internal/ulid"

	"github.com/pkg/errors"
)

const (
	emailStr = "^(?:(?:(?:(?:[a-zA-Z]|\\d|[!#\\$%&'\\*\\+\\-\\/=\\?\\^_`{\\|}~]|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])+(?:\\.([a-zA-Z]|\\d|[!#\\$%&'\\*\\+\\-\\/=\\?\\^_`{\\|}~]|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])+)*)|(?:(?:\\x22)(?:(?:(?:(?:\\x20|\\x09)*(?:\\x0d\\x0a))?(?:\\x20|\\x09)+)?(?:(?:[\\x01-\\x08\\x0b\\x0c\\x0e-\\x1f\\x7f]|\\x21|[\\x23-\\x5b]|[\\x5d-\\x7e]|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])|(?:(?:[\\x01-\\x09\\x0b\\x0c\\x0d-\\x7f]|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}]))))*(?:(?:(?:\\x20|\\x09)*(?:\\x0d\\x0a))?(\\x20|\\x09)+)?(?:\\x22))))@(?:(?:(?:[a-zA-Z]|\\d|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])|(?:(?:[a-zA-Z]|\\d|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])(?:[a-zA-Z]|\\d|-|\\.|~|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])*(?:[a-zA-Z]|\\d|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])))\\.)+(?:(?:[a-zA-Z]|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])|(?:(?:[a-zA-Z]|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])(?:[a-zA-Z]|\\d|-|\\.|~|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])*(?:[a-zA-Z]|[\\x{00A0}-\\x{D7FF}\\x{F900}-\\x{FDCF}\\x{FDF0}-\\x{FFEF}])))\\.?$"

	// Date cron maximums
	maxMin     = "59"
	maxH       = "23"
	maxDay     = "31"
	maxMonth   = "12"
	maxWeekDay = "6"
)

var (
	maxIntStr      = strconv.Itoa(math.MaxInt64)
	emailRegex     = regexp.MustCompile(emailStr)
	errInvalidCron = errors.New("invalid cron format, must have 6 values")
)

// Cursor returns an error if the cursor is not a ulid not a number.
func Cursor(cursor string) error {
	// The cursor must be a ulid or a number
	if err := ULID(cursor); err != nil {
		if _, err := strconv.Atoi(cursor); err != nil {
			return errors.New("invalid cursor")
		}
	}

	return nil
}

// Cron returns an error if the date cron passed is invalid.
//
// A cron contains a date of an event and its duration.
//
// Format:
//  ┌─────────────> minutes (0-59)
//  │ ┌───────────> hours (0-23)
//  │ │ ┌─────────> month day (0-31)
//  │ │ │  ┌──────> year month (0-12)
//  │ │ │  │  ┌───> week day (0-6, 0=sunday)
//  │ │ │  │  │  ┌> duration (1-∞)
//  m h md ym wd d
//
// Values may contain (*) meaning any, (-) for ranges, (L) for last occurrence or (,) for concatenations. Except for minutes and hours.
func Cron(cron string) error {
	if cron == "" || len(cron) < 11 {
		return errInvalidCron
	}

	parts := strings.Split(cron, " ")
	if len(parts) != 6 {
		return errInvalidCron
	}

	if !validNum(parts[0], maxMin, true) {
		return errors.New("invalid minutes")
	}
	if !validNum(parts[1], maxH, true) {
		return errors.New("invalid hours")
	}
	if !validNum(parts[2], maxDay, false) {
		return errors.New("invalid month day")
	}
	if !validNum(parts[3], maxMonth, false) {
		return errors.New("invalid year month")
	}
	if !validNum(parts[4], maxWeekDay, false) {
		return errors.New("invalid week day")
	}
	if !validDuration(parts[5]) {
		return errors.New("invalid duration")
	}

	return nil
}

// Email returns an error if the email passed is not valid.
func Email(email string) error {
	if len(email) < 7 || len(email) > 254 {
		return errors.New("invalid email length, must be between 7 and 254 characters long")
	}
	if !emailRegex.MatchString(email) {
		return errors.Errorf("invalid email: %q", email)
	}
	return nil
}

// Key returns an error if the key passed is invalid.
func Key(key string) error {
	if len(key) > 30 {
		return errors.New("invalid key length, maximum is 30")
	}
	for _, k := range key {
		if !unicode.IsLower(k) && k != '_' {
			return errors.New("a key can contain lower case and underscore characters only")
		}
	}
	return nil
}

// Password returns an error if the password passed is not valid.
func Password(password string) error {
	if len(password) < 10 {
		return errors.New("invalid password, it must contain 10 or more characters")
	}
	lowercase := false
	uppercase := false
	number := false
	special := false
	for _, c := range password {
		switch {
		case unicode.IsLower(c):
			lowercase = true
		case unicode.IsUpper(c):
			uppercase = true
		case unicode.IsNumber(c):
			number = true
		case unicode.IsPunct(c), unicode.IsSymbol(c):
			special = true
		}
		if lowercase && uppercase && number && special {
			return nil
		}
	}
	if !lowercase || !uppercase || !number || !special {
		return errors.New(
			"invalid password, it must contain at least one lowercase, one uppercase, one number and one special character",
		)
	}
	return nil
}

// RoleName returns an error if the name passed is invalid for a role.
func RoleName(roleName string) error {
	if len(roleName) > 40 {
		return errors.New("invalid role_name length, maximum is 40")
	}
	return nil
}

// Username returns an error if the username passed is not valid.
func Username(username string) error {
	if len(username) < 1 || len(username) > 24 {
		return errors.New("invalid username length, must be between 1 and 24 characters")
	}
	for _, c := range username {
		// Only accept lowercase, uppercase, number and (._)
		if !unicode.IsLower(c) && !unicode.IsUpper(c) && !unicode.IsNumber(c) {
			if c != '_' && c != '.' {
				return errors.New("invalid username")
			}
		}
	}
	return nil
}

// ULID returns an error if the id passed is not a ULID.
func ULID(id string) error {
	return validateULID(id)
}

// ULIDs returns an error if any of the ids passed is not a ULID.
func ULIDs(ids ...string) error {
	for i, id := range ids {
		if err := validateULID(id); err != nil {
			return errors.Wrapf(err, "id [%d]", i)
		}
	}
	return nil
}

// URL returns an error if the url received is invalid.
func URL(uri string) error {
	if len(uri) > 240 {
		return errors.New("url too long, maximum length is 240")
	}
	if _, err := url.ParseRequestURI(uri); err != nil {
		return errors.New("invalid url")
	}
	return nil
}

// numbersOnly determines if the non-numeric numbers are allowed or not.
//
// A value can be "*" | "1-59", | "1,9,24" | "3L".
func validNum(value, max string, numbersOnly bool) bool {
	if value == "*" {
		return !numbersOnly
	}

	maxLen := len(max)
	lastSpecialIdx := 0

	for i, v := range value {
		if v == '-' || v == ',' || v == 'L' {
			if numbersOnly {
				return false
			}
			// We could receive a number like "12--45" and is not checked here
			// but when compared it will always be higher so it's insignificant
			if i == 0 || (v != 'L' && i == len(value)-1) {
				return false
			}
			lastValue := value[lastSpecialIdx:i]
			if higher(lastValue, max, maxLen) {
				return false
			}
			lastSpecialIdx = i + 1 // Do not include the index
			continue
		}

		if v < '0' || v > '9' {
			return false
		}
	}

	// No special found, compare numbers
	if lastSpecialIdx == 0 {
		return !higher(value, max, maxLen)
	}

	// Check the last element in case there were special and non-L chars
	if value[len(value)-1] != 'L' {
		return !higher(value[lastSpecialIdx+1:], max, maxLen)
	}

	return true
}

// higher returns true if value is higher than max
func higher(value, max string, maxLen int) bool {
	valueLen := len(value)
	if valueLen > maxLen {
		return true
	}
	if valueLen == maxLen {
		return value > max
	}
	return false
}

func validDuration(d string) bool {
	if d == "0" || d > maxIntStr {
		return false
	}
	for _, v := range d {
		if v < '0' || v > '9' {
			return false
		}
	}
	return true
}

func validateULID(id string) error {
	// Check if a base32 encoded ULID is the right length.
	if len(id) != ulid.EncodedSize {
		return errors.New("invalid ulid: length is not 26")
	}

	// Check if the first character in a base32 encoded ULID will overflow. This
	// happens because the base32 representation encodes 130 bits, while the
	// ULID is only 128 bits.
	//
	// See https://github.com/oklog/ulid/issues/9 for details.
	if id[0] > '7' {
		return errors.New("invalid ulid: first character causes overflow")
	}

	// Check if all the characters in a base32 encoded ULID are part of the
	// expected base32 character set.
	for _, v := range id {
		if dec[v] == 0xFF {
			return errors.New("invalid ulid: contains non base32 characters")
		}
	}

	return nil
}

// Byte to index table for O(1) lookups when unmarshaling.
// We use 0xFF as sentinel value for invalid indexes.
var dec = [...]byte{
	0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
	0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
	0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
	0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
	0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x01,
	0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0xFF, 0xFF,
	0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E,
	0x0F, 0x10, 0x11, 0xFF, 0x12, 0x13, 0xFF, 0x14, 0x15, 0xFF,
	0x16, 0x17, 0x18, 0x19, 0x1A, 0xFF, 0x1B, 0x1C, 0x1D, 0x1E,
	0x1F, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0x0A, 0x0B, 0x0C,
	0x0D, 0x0E, 0x0F, 0x10, 0x11, 0xFF, 0x12, 0x13, 0xFF, 0x14,
	0x15, 0xFF, 0x16, 0x17, 0x18, 0x19, 0x1A, 0xFF, 0x1B, 0x1C,
	0x1D, 0x1E, 0x1F, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
	0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
	0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
	0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
	0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
	0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
	0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
	0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
	0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
	0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
	0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
	0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
	0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
	0xFF, 0xFF, 0xFF, 0xFF, 0xFF, 0xFF,
}
