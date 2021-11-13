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
		rdf := []byte("<0x8> <count(invited)> \"470\" .\n")

		got, err := ParseCount(rdf)
		assert.NoError(t, err)
		assert.Equal(t, expected, got)
	})

	t.Run("Invalid", func(t *testing.T) {
		expected := new(uint64)
		*expected = 470
		rdf := []byte("<0x8> <invited> .")

		_, err := ParseCount(rdf)
		assert.Error(t, err)
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

func TestParseRDFULIDs(t *testing.T) {
	expected := []string{"01FATYNXRDPTPSJNEJ0DQ5KBAB", "01FATYMXV9M5K093CK5NX0Y4K9"}
	rdf := []byte(`<0x2> <~invited> <0x1> .
<0x1> <event_id> "01FATYNXRDPTPSJNEJ0DQ5KBAB" .
<0x1> <event_id> "01FATYMXV9M5K093CK5NX0Y4K9" .
`)
	got := ParseRDFULIDs(rdf)
	assert.Equal(t, expected, got)
}

func TestParseRDF(t *testing.T) {
	expected := []string{"01FATYNXRDPTPSJNEJ0DQ5KBAB", "01FATYMXV9M5K093CK5NX0Y4K9"}
	rdf := []byte(`<0x2> <invited> <0x1> .
<0x1> <event_id> "01FATYNXRDPTPSJNEJ0DQ5KBAB" .
<0x1> <event_id> "01FATYMXV9M5K093CK5NX0Y4K9" .
`)
	got, err := ParseRDF(rdf)
	assert.NoError(t, err)
	assert.Equal(t, expected, got["event_id"])
}

func TestParseRDFByPredicate(t *testing.T) {
	expected := []string{"01FATYNXRDPTPSJNEJ0DQ5KBAB", "01FATYMXV9M5K093CK5NX0Y4K9"}
	rdf := []byte(`<0x2> <invited> <0x1> .
<0x1> <event_id> "01FATYNXRDPTPSJNEJ0DQ5KBAB" .
<0x1> <event_id> "01FATYMXV9M5K093CK5NX0Y4K9" .
`)
	got, err := ParseRDFByPredicate(rdf)
	assert.NoError(t, err)
	assert.Equal(t, expected, got["invited"])
}

func TestParseRDFPredicate(t *testing.T) {
	expected := []string{"01FATYNXRDPTPSJNEJ0DQ5KBAB", "01FATYMXV9M5K093CK5NX0Y4K9", "01FATYNXRDPTPSJNEJ0DQ5KBAC", "01FATYMXV9M5K093CK5NX0Y4K0"}
	rdf := []byte(`<0x2> <invited> <0x1> .
<0x1> <event_id> "01FATYNXRDPTPSJNEJ0DQ5KBAB" .
<0x1> <event_id> "01FATYMXV9M5K093CK5NX0Y4K9" .
<0x2> <banned> <0x1> .
<0x1> <event_id> "01FATYNXRDPTPSJNEJ0DQ5KBAC" .
<0x1> <event_id> "01FATYMXV9M5K093CK5NX0Y4K0" .
`)
	got, err := ParseRDFPredicate(rdf, "event_id")
	assert.NoError(t, err)
	assert.Equal(t, expected, got)
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

func BenchmarkParseCount(b *testing.B) {
	rdf := []byte(`<0x1> <count(predicate)> "15" .`)
	for i := 0; i < b.N; i++ {
		_, _ = ParseCount(rdf)
	}
}

func BenchmarkParseCountWithMap(b *testing.B) {
	rdf := []byte(`<0x1> <count(predicate)> "15" .`)
	for i := 0; i < b.N; i++ {
		_, _ = ParseCountWithMap(rdf)
	}
}

func BenchmarkParseRDFULIDs(b *testing.B) {
	rdf := []byte(`<0x2> <~invited> <0x1> .
<0x1> <event_id> "01FATYNXRDPTPSJNEJ0DQ5KBAB" .
<0x1> <event_id> "01FATYMXV9M5K093CK5NX0Y4K9" .
`)
	for i := 0; i < b.N; i++ {
		_ = ParseRDFULIDs(rdf)
	}
}

func BenchmarkParseJSON(b *testing.B) {
	q := []byte(`
	{
		"q": [
			{
				"~invited": [
					{
						"event_id": "01FATYNXRDPTPSJNEJ0DQ5KBAB"
					},
					{
						"event_id": "01FATYMXV9M5K093CK5NX0Y4K9"
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

func BenchmarkParseRDF(b *testing.B) {
	rdf := []byte(`<0x2> <~invited> <0x1> .
<0x1> <event_id> "01FATYNXRDPTPSJNEJ0DQ5KBAB" .
<0x1> <event_id> "01FATYMXV9M5K093CK5NX0Y4K9" .
`)
	for i := 0; i < b.N; i++ {
		_, _ = ParseRDF(rdf)
	}
}

func BenchmarkTriple(b *testing.B) {
	for i := 0; i < b.N; i++ {
		Triple("uid(1234)", "invited", "7568")
	}
}
