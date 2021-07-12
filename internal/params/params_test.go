package params

import (
	"context"
	"strconv"
	"testing"

	"github.com/google/uuid"
	"github.com/julienschmidt/httprouter"
	"github.com/stretchr/testify/assert"
)

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
			rawQuery: "lookup.id=1573b020-be65-4691-ab99-d744f8febbc4",
			expected: Query{
				Fields:   nil,
				LookupID: "1573b020-be65-4691-ab99-d744f8febbc4",
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
				Cursor: "0",
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

func TestParseInt(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		expected := "20"
		got, err := parseInt("20", "12", 50)
		assert.NoError(t, err)
		assert.Equal(t, expected, got)
	})
	t.Run("Default", func(t *testing.T) {
		expected := "12"
		got, err := parseInt("", "12", 50)
		assert.NoError(t, err)
		assert.Equal(t, expected, got)
	})
	t.Run("Invalid", func(t *testing.T) {
		_, err := parseInt("abc", "12", 50)
		assert.Error(t, err)
	})
	t.Run("Maximum exceeded", func(t *testing.T) {
		_, err := parseInt("20", "12", 15)
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

func TestUUID(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		id := uuid.NewString()
		assert.NoError(t, ValidateUUID(id))
	})

	t.Run("Invalid", func(t *testing.T) {
		id := "asdfhasdfhqu8123hjdquh"
		assert.Error(t, ValidateUUID(id), "Expected an error and got nil")
	})
}

func TestUUIDFromCtx(t *testing.T) {
	id := uuid.NewString()
	params := httprouter.Params{
		{Key: "id", Value: id},
	}
	ctx := context.WithValue(context.Background(), httprouter.ParamsKey, params)

	got, err := UUIDFromCtx(ctx)
	assert.NoError(t, err)

	assert.Equal(t, id, got)
}

func TestUUIDs(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		ids := []string{uuid.NewString(), uuid.NewString(), uuid.NewString()}
		assert.NoError(t, ValidateUUIDs(ids...))
	})

	t.Run("Invalid", func(t *testing.T) {
		ids := []string{uuid.NewString(), uuid.NewString(), "as6d45sa6dasda"}
		assert.Error(t, ValidateUUIDs(ids...), "Expected an error and got nil")
	})
}

func TestValidateEventFields(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		fields := []string{"id", "name", "type", "public", "start_time", "end_time", "created_at", "updated_at"}
		err := validateEventFields(fields)
		assert.NoError(t, err)
	})

	t.Run("Invalid", func(t *testing.T) {
		err := validateEventFields([]string{"username"})
		assert.Error(t, err, "Expected an error and got nil")
	})
}

func TestValidateMediaFields(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		fields := []string{"id", "event_id", "url", "created_at"}
		err := validateMediaFields(fields)
		assert.NoError(t, err)
	})

	t.Run("Invalid", func(t *testing.T) {
		err := validateMediaFields([]string{"username"})
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

	t.Run("Invalid", func(t *testing.T) {
		err := validateProductFields([]string{"username"})
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

	t.Run("Invalid", func(t *testing.T) {
		err := validateUserFields([]string{"type"})
		assert.Error(t, err, "Expected an error and got nil")
	})
}

func BenchmarkParseQuery(b *testing.B) {
	rawQuery := "cursor=2&limit=20&user.fields=id,username,email,birth_date"
	for i := 0; i < b.N; i++ {
		ParseQuery(rawQuery, User)
	}
}

func BenchmarkUUID(b *testing.B) {
	id := uuid.NewString()
	for i := 0; i < b.N; i++ {
		ValidateUUID(id)
	}
}

func BenchmarkUUIDFromCtx(b *testing.B) {
	id := uuid.NewString()
	params := httprouter.Params{
		{Key: "id", Value: id},
	}
	ctx := context.WithValue(context.Background(), httprouter.ParamsKey, params)

	for i := 0; i < b.N; i++ {
		UUIDFromCtx(ctx)
	}
}
