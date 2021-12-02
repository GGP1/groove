package apikey

import (
	"crypto/rand"
	"net/http"
	"sync"

	"github.com/GGP1/groove/internal/log"
	"github.com/GGP1/groove/internal/validate"

	"github.com/dgrijalva/jwt-go"
	"github.com/pkg/errors"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// JWT best practices: https://datatracker.ietf.org/doc/html/rfc8725

var (
	// ErrAPIKeyNotFound is returned when no API was found.
	ErrAPIKeyNotFound = errors.New("API key not found")
	// ErrInvalidAPIKey is returned when the API was passed but is invalid.
	ErrInvalidAPIKey = errors.New("invalid API key")
	secretKey        []byte
	once             sync.Once
)

const (
	// headerName is the name of the API key header.
	headerName  = "X-Api-Key"
	prefix      = "gpTeHhB_"
	saltLen     = 8
	finalKeyLen = 167 // API key final length (prefix and salt included)
)

// Claims represents JWT claims.
type Claims struct {
	Key  string
	Salt []byte
}

// Valid validates the correctness of the token and satisfies jwt.Claims interface.
func (c Claims) Valid() error {
	if validate.ULID(c.Key) != nil || len(c.Salt) != saltLen {
		return ErrInvalidAPIKey
	}
	return nil
}

// New returns a new API key.
func New(id string) (string, error) {
	if err := validate.ULID(id); err != nil {
		return "", err
	}

	once.Do(func() {
		secretKey = []byte(viper.GetString("secrets.apikeys"))
	})

	salt := make([]byte, saltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", errors.Wrap(err, "generating key")
	}
	claims := Claims{
		Key:  id,
		Salt: salt,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	apiKey, err := token.SignedString(secretKey)
	if err != nil {
		return "", errors.Wrap(err, "signing key")
	}

	return prefix + apiKey, nil
}

// Check validates the API key received.
func Check(key string) error {
	if key[:len(prefix)] != prefix {
		return ErrInvalidAPIKey
	}

	token, err := jwt.ParseWithClaims(key[len(prefix):], &Claims{}, func(t *jwt.Token) (interface{}, error) {
		return secretKey, nil
	})
	if err != nil {
		log.Debug("failed parsing jwt", zap.Error(err))
		return ErrInvalidAPIKey
	}

	claims, ok := token.Claims.(*Claims)
	if !ok {
		return ErrInvalidAPIKey
	}

	return claims.Valid()
}

// FromRequest takes the API token from the request and validates it.
func FromRequest(r *http.Request) (string, error) {
	header := r.Header[headerName]
	if len(header) == 0 {
		return "", ErrAPIKeyNotFound
	}

	key := header[0]
	if len(key) != finalKeyLen || key[:len(prefix)] != prefix {
		return "", ErrInvalidAPIKey
	}

	if err := Check(key); err != nil {
		return "", err
	}

	return key, nil
}
