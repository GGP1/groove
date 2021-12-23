package model_test

import (
	"strings"
	"testing"

	"github.com/GGP1/groove/model"
)

var models = []model.Model{
	model.T.Comment,
	model.T.Event,
	model.T.Notification,
	model.T.Post,
	model.T.Product,
	model.T.User,
}

func TestAlias(t *testing.T) {
	aliases := map[string]struct{}{}

	for _, m := range models {
		k := m.Alias()
		if _, ok := aliases[k]; ok {
			t.Fatalf("alias %s is duplicated", k)
		}
		aliases[k] = struct{}{}
	}
}

func TestDefaultFields(t *testing.T) {
	for _, m := range models {
		defaultFields := strings.ReplaceAll(m.DefaultFields(false), "\n\t", " ")
		fields := strings.Split(defaultFields, ", ")
		for i, field := range fields {
			if !m.ValidField(field) {
				t.Fatalf("%q default field [%d] from %s is invalid", field, i, m.Tablename())
			}
		}
	}
}

func TestCacheKey(t *testing.T) {
	id := "1"
	keys := map[string]struct{}{}

	for _, m := range models {
		k := m.CacheKey(id)
		if _, ok := keys[k]; ok {
			t.Fatalf("cache key %s is duplicated", k)
		}
		keys[k] = struct{}{}
	}
}

func TestTablename(t *testing.T) {
	tables := map[string]struct{}{}

	for _, m := range models {
		k := m.Tablename()
		if _, ok := tables[k]; ok {
			t.Fatalf("table name %s is duplicated", k)
		}
		tables[k] = struct{}{}
	}
}

func TestURLQueryKey(t *testing.T) {
	keys := map[string]struct{}{}

	for _, m := range models {
		k := m.URLQueryKey()
		if _, ok := keys[k]; ok {
			t.Fatalf("URL query key %s is duplicated", k)
		}
		keys[k] = struct{}{}
	}
}
