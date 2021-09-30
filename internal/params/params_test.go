package params

import (
	"context"
	"testing"

	"github.com/GGP1/groove/internal/ulid"
	"github.com/GGP1/groove/model"

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

func TestParse(t *testing.T) {
	cases := []struct {
		desc     string
		model    model.Model
		rawQuery string
		expected Query
	}{
		{
			desc:     "User",
			model:    model.User,
			rawQuery: "cursor=2&limit=20&user.fields=id,username,email,birth_date",
			expected: Query{
				Cursor: "2",
				Fields: []string{"id", "username", "email", "birth_date"},
				Limit:  "20",
			},
		},
		{
			desc:     "Event",
			model:    model.Event,
			rawQuery: "cursor=15&limit=3&event.fields=id,name,created_at",
			expected: Query{
				Cursor: "15",
				Fields: []string{"id", "name", "created_at"},
				Limit:  "3",
			},
		},
		{
			desc:     "Post",
			model:    model.Post,
			rawQuery: "cursor=39&limit=8&media.fields=id,url,created_at",
			expected: Query{
				Cursor: "39",
				Fields: []string{"id", "event_id", "created_at"},
				Limit:  "8",
			},
		},
		{
			desc:     "Product",
			model:    model.Product,
			rawQuery: "cursor=2&limit=50&product.fields=stock,brand,type",
			expected: Query{
				Cursor: "2",
				Fields: []string{"stock", "brand", "type"},
				Limit:  "50",
			},
		},
		{
			desc:     "Lookup ID",
			model:    model.User,
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
			model:    model.User,
			expected: Query{
				Count:  false,
				Cursor: DefaultCursor,
				Limit:  "20",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			got, err := Parse(tc.rawQuery, tc.model)
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, got)
		})
	}

	t.Run("Invalid boolean", func(t *testing.T) {
		rawQuery := "count=invalid"

		_, err := Parse(rawQuery, model.User)
		assert.Error(t, err)
	})
	t.Run("Invalid lookup ID", func(t *testing.T) {
		rawQuery := "lookup.id=4691-ab99-d744f8febbc4"

		_, err := Parse(rawQuery, model.User)
		assert.Error(t, err)
	})
	t.Run("Maximum exceeded", func(t *testing.T) {
		rawQuery := "limit=100"

		_, err := Parse(rawQuery, model.Event)
		assert.Error(t, err)
	})
	t.Run("Invalid cursor", func(t *testing.T) {
		rawQuery := "cursor=4691-ab99-d744f8febbc4"

		_, err := Parse(rawQuery, model.User)
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

func BenchmarkParse(b *testing.B) {
	rawQuery := "cursor=2&limit=20&user.fields=id,username,email,birth_date"
	for i := 0; i < b.N; i++ {
		Parse(rawQuery, model.User)
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
