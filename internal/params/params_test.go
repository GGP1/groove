package params_test

import (
	"context"
	"testing"

	"github.com/GGP1/groove/internal/params"
	"github.com/GGP1/groove/internal/ulid"
	"github.com/GGP1/groove/model"

	"github.com/julienschmidt/httprouter"
	"github.com/stretchr/testify/assert"
)

func TestIDFromCtx(t *testing.T) {
	id := ulid.NewString()
	cases := []struct {
		desc string
		key  string
		tag  []string
	}{
		{
			desc: "Without tag",
			key:  "id",
		},
		{
			desc: "With tag",
			key:  "event_id",
			tag:  []string{"event_id"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			httpParams := httprouter.Params{
				{Key: tc.key, Value: id},
			}
			ctx := context.WithValue(context.Background(), httprouter.ParamsKey, httpParams)

			got, err := params.IDFromCtx(ctx, tc.tag...)
			assert.NoError(t, err)
			assert.Equal(t, got, id)
		})
	}
}

func TestIDAndNameFromCtx(t *testing.T) {
	id := ulid.NewString()
	name := "name"
	httpParams := httprouter.Params{
		{Key: "id", Value: id},
		{Key: "name", Value: name},
	}
	ctx := context.WithValue(context.Background(), httprouter.ParamsKey, httpParams)

	gotID, gotName, err := params.IDAndNameFromCtx(ctx)
	assert.NoError(t, err)
	assert.Equal(t, id, gotID)
	assert.Equal(t, name, gotName)
}

func TestIDAndKeyFromCtx(t *testing.T) {
	id := ulid.NewString()
	key := "key"
	httpParams := httprouter.Params{
		{Key: "id", Value: id},
		{Key: "key", Value: key},
	}
	ctx := context.WithValue(context.Background(), httprouter.ParamsKey, httpParams)

	gotID, gotKey, err := params.IDAndKeyFromCtx(ctx)
	assert.NoError(t, err)
	assert.Equal(t, id, gotID)
	assert.Equal(t, key, gotKey)
}

func TestParse(t *testing.T) {
	cases := []struct {
		desc     string
		model    model.Model
		rawQuery string
		expected params.Query
	}{
		{
			desc:     "User",
			model:    model.T.User,
			rawQuery: "cursor=2&limit=20&user.fields=id,username,email,birth_date",
			expected: params.Query{
				Cursor: "2",
				Fields: []string{"id", "username", "email", "birth_date"},
				Limit:  "20",
			},
		},
		{
			desc:     "Event",
			model:    model.T.Event,
			rawQuery: "cursor=15&limit=3&event.fields=id,name,created_at",
			expected: params.Query{
				Cursor: "15",
				Fields: []string{"id", "name", "created_at"},
				Limit:  "3",
			},
		},
		{
			desc:     "Post",
			model:    model.T.Post,
			rawQuery: "cursor=39&limit=8&post.fields=id,media,created_at",
			expected: params.Query{
				Cursor: "39",
				Fields: []string{"id", "media", "created_at"},
				Limit:  "8",
			},
		},
		{
			desc:     "Product",
			model:    model.T.Product,
			rawQuery: "cursor=2&limit=50&product.fields=stock,brand,type",
			expected: params.Query{
				Cursor: "2",
				Fields: []string{"stock", "brand", "type"},
				Limit:  "50",
			},
		},
		{
			desc:     "Lookup ID",
			model:    model.T.User,
			rawQuery: "lookup.id=01FATW8S0BMJ053XZ779Q025PC",
			expected: params.Query{
				Fields:   nil,
				LookupID: "01FATW8S0BMJ053XZ779Q025PC",
			},
		},
		{
			desc:     "Count true",
			rawQuery: "count=t",
			expected: params.Query{Count: true},
		},
		{
			desc:     "Count false",
			rawQuery: "count=false",
			model:    model.T.User,
			expected: params.Query{
				Count:  false,
				Cursor: params.DefaultCursor,
				Limit:  "20",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			got, err := params.Parse(tc.rawQuery, tc.model)
			assert.NoError(t, err)
			assert.Equal(t, tc.expected, got)
		})
	}

	t.Run("Invalid boolean", func(t *testing.T) {
		rawQuery := "count=invalid"

		_, err := params.Parse(rawQuery, model.T.User)
		assert.Error(t, err)
	})
	t.Run("Invalid lookup ID", func(t *testing.T) {
		rawQuery := "lookup.id=4691-ab99-d744f8febbc4"

		_, err := params.Parse(rawQuery, model.T.User)
		assert.Error(t, err)
	})
	t.Run("Maximum exceeded", func(t *testing.T) {
		rawQuery := "limit=100"

		_, err := params.Parse(rawQuery, model.T.Event)
		assert.Error(t, err)
	})
	t.Run("Invalid cursor", func(t *testing.T) {
		rawQuery := "cursor=4691-ab99-d744f8febbc4"

		_, err := params.Parse(rawQuery, model.T.User)
		assert.Error(t, err)
	})
}

// func TestParseLimit(t *testing.T) {
// 	t.Run("Valid", func(t *testing.T) {
// 		expected := "20"
// 		got, err := parseLimit("20")
// 		assert.NoError(t, err)
// 		assert.Equal(t, expected, got)
// 	})
// 	t.Run("Default", func(t *testing.T) {
// 		got, err := parseLimit("")
// 		assert.NoError(t, err)
// 		assert.Equal(t, DefaultLimit, got)

// 		got2, err := parseLimit("-5")
// 		assert.NoError(t, err)
// 		assert.Equal(t, DefaultLimit, got2)
// 	})
// 	t.Run("Invalid", func(t *testing.T) {
// 		_, err := parseLimit("abc")
// 		assert.Error(t, err)
// 	})
// 	t.Run("Maximum exceeded", func(t *testing.T) {
// 		_, err := parseLimit("70")
// 		assert.Error(t, err)
// 	})
// }

func BenchmarkParse(b *testing.B) {
	rawQuery := "cursor=2&limit=20&user.fields=id,username,email,birth_date"
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		params.Parse(rawQuery, model.T.User)
	}
}

func BenchmarkIDFromCtx(b *testing.B) {
	id := ulid.NewString()
	httpParams := httprouter.Params{
		{Key: "id", Value: id},
	}
	ctx := context.WithValue(context.Background(), httprouter.ParamsKey, httpParams)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		params.IDFromCtx(ctx)
	}
}
