package validate

import (
	"testing"

	"github.com/GGP1/groove/internal/ulid"

	"github.com/stretchr/testify/assert"
)

func TestCursor(t *testing.T) {
	cases := []struct {
		desc   string
		cursor string
		fail   bool
	}{
		{
			desc:   "ID",
			cursor: ulid.NewString(),
		},
		{
			desc:   "Number",
			cursor: "156",
		},
		{
			desc:   "Invalid",
			cursor: "'; SELECT * FROM users;",
			fail:   true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(t *testing.T) {
			err := Cursor(tc.cursor)
			if !tc.fail {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestEventFields(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		fields := []string{"id", "name", "type", "public", "start_time", "end_time", "created_at", "updated_at"}
		err := EventFields(fields)
		assert.NoError(t, err)
	})

	t.Run("Nil", func(t *testing.T) {
		err := EventFields(nil)
		assert.NoError(t, err)
	})

	t.Run("Invalid", func(t *testing.T) {
		err := EventFields([]string{"username"})
		assert.Error(t, err, "Expected an error and got nil")
	})

	t.Run("Empty field", func(t *testing.T) {
		err := EventFields([]string{"created_at", ""})
		assert.Error(t, err, "Expected an error and got nil")
	})
}

func TestMediaFields(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		fields := []string{"id", "event_id", "url", "created_at"}
		err := MediaFields(fields)
		assert.NoError(t, err)
	})

	t.Run("Nil", func(t *testing.T) {
		err := MediaFields(nil)
		assert.NoError(t, err)
	})

	t.Run("Invalid", func(t *testing.T) {
		err := MediaFields([]string{"username"})
		assert.Error(t, err, "Expected an error and got nil")
	})

	t.Run("Empty field", func(t *testing.T) {
		err := MediaFields([]string{"created_at", ""})
		assert.Error(t, err, "Expected an error and got nil")
	})
}

func TestProductFields(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		fields := []string{"id", "event_id", "stock", "brand", "type", "description",
			"discount", "taxes", "subtotal", "total", "created_at"}
		err := ProductFields(fields)
		assert.NoError(t, err)
	})

	t.Run("Nil", func(t *testing.T) {
		err := ProductFields(nil)
		assert.NoError(t, err)
	})

	t.Run("Invalid", func(t *testing.T) {
		err := ProductFields([]string{"username"})
		assert.Error(t, err, "Expected an error and got nil")
	})

	t.Run("Empty field", func(t *testing.T) {
		err := ProductFields([]string{"created_at", ""})
		assert.Error(t, err, "Expected an error and got nil")
	})
}

func TestUserFields(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		fields := []string{"id", "created_at", "updated_at", "name", "user_id", "username",
			"email", "description", "birth_date", "profile_image_url",
			"premium", "private", "verified_email"}
		err := UserFields(fields)
		assert.NoError(t, err)
	})

	t.Run("Nil", func(t *testing.T) {
		err := UserFields(nil)
		assert.NoError(t, err)
	})

	t.Run("Invalid", func(t *testing.T) {
		err := UserFields([]string{"type"})
		assert.Error(t, err, "Expected an error and got nil")
	})

	t.Run("Empty field", func(t *testing.T) {
		err := UserFields([]string{"created_at", ""})
		assert.Error(t, err, "Expected an error and got nil")
	})
}

func TestULID(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		id := ulid.NewString()
		err := ULID(id)
		assert.NoError(t, err)
	})

	t.Run("Invalid size", func(t *testing.T) {
		err := ULID("123")
		assert.Error(t, err)
	})

	t.Run("Invalid first character", func(t *testing.T) {
		err := ULID("81FATYTQYMDPJFEJTGC4SHXA27")
		assert.Error(t, err)
	})

	t.Run("Invalid characters", func(t *testing.T) {
		err := ULID("01FATYTQYMDPJFEJTGC4SHXA2I")
		assert.Error(t, err)
	})
}

func TestULIDs(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		ids := []string{ulid.NewString(), ulid.NewString(), ulid.NewString()}
		err := ULIDs(ids...)
		assert.NoError(t, err)
	})

	t.Run("Invalid", func(t *testing.T) {
		ids := []string{ulid.NewString(), "123", ulid.NewString()}
		err := ULIDs(ids...)
		assert.Error(t, err)
	})
}

func BenchmarkULID(b *testing.B) {
	id := ulid.NewString()
	b.SetBytes(ulid.EncodedSize)
	for i := 0; i < b.N; i++ {
		ULID(id)
	}
}
