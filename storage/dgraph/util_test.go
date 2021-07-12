package dgraph

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseCount(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		expected := new(uint64)
		*expected = 470
		rdf := []byte("<0x8> <invited> \"470\" .")

		got, err := ParseCount(rdf)
		assert.NoError(t, err)
		assert.Equal(t, expected, got)
	})

	t.Run("Error", func(t *testing.T) {
		rdf := []byte("<0x8> <invited> \"groove\" .")

		_, err := ParseCount(rdf)
		assert.Error(t, err)
	})
}

func TestParseCountWithMap(t *testing.T) {
	t.Run("Valid", func(t *testing.T) {
		expected := new(uint64)
		*expected = 470
		rdf := []byte("<0x8> <count(invited)> \"470\" .\n")

		got, err := ParseCountWithMap(rdf)
		assert.NoError(t, err)
		assert.Equal(t, expected, got["invited"])
	})

	t.Run("Error", func(t *testing.T) {
		rdf := []byte("<0x8> <count(invited)> \"groove\" .\n")

		_, err := ParseCountWithMap(rdf)
		assert.Error(t, err)
	})
}

func TestParseRDF(t *testing.T) {
	expected := []string{"8d371ac6-350b-4d63-b43f-57a42042f817", "721bb10c-123c-40f9-af53-6631f45a1aa4"}
	rdf := []byte(`<0x2> <~invited> <0x1> .
<0x1> <event_id> "8d371ac6-350b-4d63-b43f-57a42042f817" .
<0x1> <event_id> "721bb10c-123c-40f9-af53-6631f45a1aa4" .
`)
	got := ParseRDFUUIDs(rdf)
	assert.Equal(t, expected, got)
}

func TestParseRDFWithMap(t *testing.T) {
	expected := []string{"8d371ac6-350b-4d63-b43f-57a42042f817", "721bb10c-123c-40f9-af53-6631f45a1aa4"}
	rdf := []byte(`<0x2> <invited> <0x1> .
<0x1> <event_id> "8d371ac6-350b-4d63-b43f-57a42042f817" .
<0x1> <event_id> "721bb10c-123c-40f9-af53-6631f45a1aa4" .
`)
	got, err := ParseRDFWithMap(rdf)
	assert.NoError(t, err)
	assert.Equal(t, expected, got["invited"])
}

func TestTriple(t *testing.T) {
	t.Run("No UID", func(t *testing.T) {
		expected := []byte("uid(1234) <invited> \"7568\" .")
		got := Triple("uid(1234)", "invited", "7568")
		assert.Equal(t, expected, got)
	})

	t.Run("UID", func(t *testing.T) {
		expected := []byte("uid(1234) <invited> uid(user) .")
		got := Triple("uid(1234)", "invited", "uid(user)")
		assert.Equal(t, expected, got)
	})
}

func BenchmarkParseRDFResponse(b *testing.B) {
	rdf := []byte(`<0x2> <~invited> <0x1> .
<0x1> <event_id> "8d371ac6-350b-4d63-b43f-57a42042f817" .
<0x1> <event_id> "721bb10c-123c-40f9-af53-6631f45a1aa4" .
`)
	for i := 0; i < b.N; i++ {
		_ = ParseRDFUUIDs(rdf)
	}
}

func BenchmarkParseJSON(b *testing.B) {
	q := []byte(`
	{
		"q": [
			{
				"~invited": [
					{
						"event_id": "8d371ac6-350b-4d63-b43f-57a42042f817"
					},
					{
						"event_id": "721bb10c-123c-40f9-af53-6631f45a1aa4"
					}
				]
			}
		]
	}`)
	var str struct {
		Q []struct {
			Host []struct {
				EventID string `json:"event_id,omitempty"`
			} `json:"~invited,omitempty"`
		}
	}
	for i := 0; i < b.N; i++ {
		_ = json.Unmarshal(q, &str)
	}
}

func BenchmarkParseRDFResponseWithMap(b *testing.B) {
	rdf := []byte(`<0x2> <~invited> <0x1> .
<0x1> <event_id> "8d371ac6-350b-4d63-b43f-57a42042f817" .
<0x1> <event_id> "721bb10c-123c-40f9-af53-6631f45a1aa4" .
`)
	for i := 0; i < b.N; i++ {
		_, _ = ParseRDFWithMap(rdf)
	}
}

func BenchmarkTriple(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Triple("uid(1234)", "invited", "7568")
	}
}
