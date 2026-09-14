package htx

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestError(t *testing.T) {
	t.Parallel()
	const message htxError = "test error"
	assert.Equal(t, "test error", message.Error(), "Error should return the HTX error message")
}
