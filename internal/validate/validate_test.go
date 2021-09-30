package validate

import (
	"strconv"
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

func TestEmail(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		err := Email("testing_email_regexp@test.com")
		assert.NoError(t, err)
	})

	t.Run("Invalid", func(t *testing.T) {
		emails := []string{"testing_email_regexptest.com", "@test.com", "%·$·@#2t.com", "invalid_email@test"}

		for i, email := range emails {
			t.Run(strconv.Itoa(i), func(t *testing.T) {
				err := Email(email)
				assert.Error(t, err)
			})
		}
	})
}

func TestPassword(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		pwd := "eC#fnz}18A"
		err := Password(pwd)
		assert.NoError(t, err)
	})

	t.Run("Invalid", func(t *testing.T) {
		passwords := []string{"asc1I_", "1nv4lidpassword", "123456789+123A"}

		for i, password := range passwords {
			t.Run(strconv.Itoa(i), func(t *testing.T) {
				err := Password(password)
				assert.Error(t, err)
			})
		}
	})
}

func TestRoleName(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		err := RoleName("chef")
		assert.NoError(t, err)
	})

	t.Run("Invalid", func(t *testing.T) {
		roleNames := []string{"n'tall&why", "invalid-name"}

		for i, roleName := range roleNames {
			t.Run(strconv.Itoa(i), func(t *testing.T) {
				err := RoleName(roleName)
				assert.Error(t, err)
			})
		}
	})
}

func TestUsername(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		err := Username("gastonpalomeque")
		assert.NoError(t, err)
	})

	t.Run("Invalid", func(t *testing.T) {
		usernames := []string{"n'tall&wse", "contains_invalid-chars", "actuallytoolongforausername"}

		for i, username := range usernames {
			t.Run(strconv.Itoa(i), func(t *testing.T) {
				err := Username(username)
				assert.Error(t, err)
			})
		}
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

func BenchmarkEmail(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Email("testing_email_regex@test.com")
	}
}

func BenchmarkPassword(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Password("8tOnVgK]/#ET{")
	}
}

func BenchmarkUsername(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Username("testing_username_regex")
	}
}

func BenchmarkULID(b *testing.B) {
	id := ulid.NewString()
	b.SetBytes(ulid.EncodedSize)
	for i := 0; i < b.N; i++ {
		ULID(id)
	}
}
