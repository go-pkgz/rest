package c

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCache_Keys(t *testing.T) {
	tbl := []struct {
		key    string
		scopes []string
		full   string
	}{
		{"key1", []string{"s1"}, "s1@@key1@@site"},
		{"key2", []string{"s11", "s2"}, "s11$$s2@@key2@@site"},
		{"key3", []string{}, "@@key3@@site"},
	}

	for n, tt := range tbl {
		k := NewKey("site").ID(tt.key).Scopes(tt.scopes...)
		full := k.Merge()
		assert.Equal(t, tt.full, full, "making key, #%d", n)

		k, e := ParseKey(full)
		assert.Nil(t, e)
		assert.Equal(t, tt.scopes, k.scopes)
		assert.Equal(t, tt.key, k.id)
	}

	_, err := ParseKey("abc")
	assert.Error(t, err)
	_, err = ParseKey("")
	assert.Error(t, err)
}
