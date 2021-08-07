package ulid

import (
	"crypto/rand"

	"github.com/oklog/ulid/v2"
)

// EncodedSize is the length of a text encoded ULID.
const EncodedSize = 26

// New returns a new ULID.
func New() ulid.ULID {
	return ulid.MustNew(ulid.Now(), rand.Reader)
}

// NewString returns a new lexicographically sortable string encoded ULID.
func NewString() string {
	return New().String()
}
