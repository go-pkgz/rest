package requestid

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWithFrom(t *testing.T) {
	assert.Empty(t, From(context.Background()))
	assert.Equal(t, "abc", From(With(context.Background(), "abc")))
}
