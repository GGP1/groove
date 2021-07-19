package params

import (
	"context"
	"strconv"
	"testing"

	"github.com/GGP1/groove/internal/ulid"

	"github.com/julienschmidt/httprouter"
	"github.com/stretchr/testify/assert"
)

func TestIDFromCtx(t *testing.T) {
	id := ulid.NewString()
	params := httprouter.Params{
		{Key: "id", Value: id},
	}
	ctx := context.WithValue(context.Background(), httprouter.ParamsKey, params)

	got, err := IDFromCtx(ctx)
	assert.NoError(t, err)
	assert.Equal(t, id, got)
}

func TestParseQuery(t *testing.T) {
	cases := []struct {
		desc     string
		obj      obj
		rawQuery string
		expected Query
	}{
		{
			desc:     "User",
			obj:      User,
			rawQuery: "cursor=2&limit=20&user.fields=id,username,email,birth_date",
			expected: Query{
				Cursor: "2",
				Fields: []string{"id", "username", "email", "birth_date"},
				Limit:  "20",
			},
		},
		{
			desc:     "Event",
			obj:      Event,
			rawQuery: "cursor=15&limit=3&event.fields=id,name,created_at",
			expected: Query{
				Cursor: "15",
				Fields: []string{"id", "name", "created_at"},
				Limit:  "3",
			},
		},
		{
			desc:     "Media",
			obj:      Media,
			rawQuery: "cursor=39&limit=8&media.fields=id,url,created_at",
			expected: Query{
				Cursor: "39",
				Fields: []string{"id", "url", "created_at"},
				Limit:  "8",
			},
		},
		{
			desc:     "Product",
			obj:      Product,
			rawQuery: "cursor=2&limit=50&product.fields=stock,brand,type",
			expected: Query{
				Cursor: "2",
				Fields: []string{"stock", "brand", "type"},
				Limit:  "50",
			},
		},
		{
			desc:     "Lookup ID",
			obj:      User,
			rawQuery: "lookup.id=01FATW8S0BMJ053XZ779Q025PC",
			expected: Query{
				Fields:   nil,
				LookupID: "01FATW8S0BMJ053XZ779Q025PC",
			},
		},
		{
			desc:     "Count true",
			rawQuery: "count=t",
			expected: Query{Count: true},
		},
		{
			desc:     "Count false",
			rawQuery: "count=false",
			obj:      User,
			expected: Query{
				Count:  false,
				Cursor: "",
				Limit:  "20",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			got, err := ParseQuery(tc.rawQuery, tc.obj)
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, got)
		})
	}

	t.Run("Invalid boolean", func(t *testing.T) {
		rawQuery := "count=invalid"

		_, err := ParseQuery(rawQuery, User)
		assert.Error(t, err)
	})
	t.Run("Invalid lookup ID", func(t *testing.T) {
		rawQuery := "lookup.id=4691-ab99-d744f8febbc4"

		_, err := ParseQuery(rawQuery, User)
		assert.Error(t, err)
	})
	t.Run("Maximum exceeded", func(t *testing.T) {
		rawQuery := "limit=100"

		_, err := ParseQuery(rawQuery, Event)
		assert.Error(t, err)
	})
}

func TestParseBool(t *testing.T) {
	cases := []struct {
		desc     string
		expected bool
		input    string
	}{
		{
			desc:     "True",
			expected: true,
			input:    "t",
		},
		{
			desc:     "False",
			expected: false,
			input:    "0",
		},
	}

	for i, tc := range cases {
		t.Run(strconv.Itoa(i), func(t *testing.T) {
			got, err := parseBool(tc.input)
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, got)
		})
	}

	t.Run("Invalid", func(t *testing.T) {
		_, err := parseBool("abcdefg")
		assert.Error(t, err)
	})
}

func TestParseLimit(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		expected := "20"
		got, err := parseLimit("20")
		assert.NoError(t, err)
		assert.Equal(t, expected, got)
	})
	t.Run("Default", func(t *testing.T) {
		got, err := parseLimit("")
		assert.NoError(t, err)
		assert.Equal(t, defaultLimit, got)

		got2, err := parseLimit("-5")
		assert.NoError(t, err)
		assert.Equal(t, defaultLimit, got2)
	})
	t.Run("Invalid", func(t *testing.T) {
		_, err := parseLimit("abc")
		assert.Error(t, err)
	})
	t.Run("Maximum exceeded", func(t *testing.T) {
		_, err := parseLimit("70")
		assert.Error(t, err)
	})
}

func TestSplit(t *testing.T) {
	cases := []struct {
		desc     string
		expected []string
		input    string
	}{
		{
			desc:     "Non-nil",
			expected: []string{"name", "username", "email", "birth_date"},
			input:    "name,username,email,birth_date",
		},
		{
			desc:     "Nil",
			expected: nil,
			input:    "",
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			got := split(tc.input)
			assert.Equal(t, tc.expected, got)
		})
	}
}

func TestValidateEventFields(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		fields := []string{"id", "name", "type", "public", "start_time", "end_time", "created_at", "updated_at"}
		err := validateEventFields(fields)
		assert.NoError(t, err)
	})

	t.Run("Nil", func(t *testing.T) {
		err := validateEventFields(nil)
		assert.NoError(t, err)
	})

	t.Run("Invalid", func(t *testing.T) {
		err := validateEventFields([]string{"username"})
		assert.Error(t, err, "Expected an error and got nil")
	})

	t.Run("Empty field", func(t *testing.T) {
		err := validateEventFields([]string{"created_at", ""})
		assert.Error(t, err, "Expected an error and got nil")
	})
}

func TestValidateMediaFields(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		fields := []string{"id", "event_id", "url", "created_at"}
		err := validateMediaFields(fields)
		assert.NoError(t, err)
	})

	t.Run("Nil", func(t *testing.T) {
		err := validateMediaFields(nil)
		assert.NoError(t, err)
	})

	t.Run("Invalid", func(t *testing.T) {
		err := validateMediaFields([]string{"username"})
		assert.Error(t, err, "Expected an error and got nil")
	})

	t.Run("Empty field", func(t *testing.T) {
		err := validateMediaFields([]string{"created_at", ""})
		assert.Error(t, err, "Expected an error and got nil")
	})
}

func TestValidateProductFields(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		fields := []string{"id", "event_id", "stock", "brand", "type", "description",
			"discount", "taxes", "subtotal", "total", "created_at"}
		err := validateProductFields(fields)
		assert.NoError(t, err)
	})

	t.Run("Nil", func(t *testing.T) {
		err := validateProductFields(nil)
		assert.NoError(t, err)
	})

	t.Run("Invalid", func(t *testing.T) {
		err := validateProductFields([]string{"username"})
		assert.Error(t, err, "Expected an error and got nil")
	})

	t.Run("Empty field", func(t *testing.T) {
		err := validateProductFields([]string{"created_at", ""})
		assert.Error(t, err, "Expected an error and got nil")
	})
}

func TestValidateUserFields(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		fields := []string{"id", "created_at", "updated_at", "name", "user_id", "username",
			"email", "description", "birth_date", "profile_image_url",
			"premium", "private", "verified_email"}
		err := validateUserFields(fields)
		assert.NoError(t, err)
	})

	t.Run("Nil", func(t *testing.T) {
		err := validateUserFields(nil)
		assert.NoError(t, err)
	})

	t.Run("Invalid", func(t *testing.T) {
		err := validateUserFields([]string{"type"})
		assert.Error(t, err, "Expected an error and got nil")
	})

	t.Run("Empty field", func(t *testing.T) {
		err := validateUserFields([]string{"created_at", ""})
		assert.Error(t, err, "Expected an error and got nil")
	})
}

func BenchmarkParseQuery(b *testing.B) {
	rawQuery := "cursor=2&limit=20&user.fields=id,username,email,birth_date"
	for i := 0; i < b.N; i++ {
		ParseQuery(rawQuery, User)
	}
}

func BenchmarkIDFromCtx(b *testing.B) {
	id := ulid.NewString()
	params := httprouter.Params{
		{Key: "id", Value: id},
	}
	ctx := context.WithValue(context.Background(), httprouter.ParamsKey, params)

	for i := 0; i < b.N; i++ {
		IDFromCtx(ctx)
	}
}
