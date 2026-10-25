package lbank

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRequestError(t *testing.T) {
	t.Parallel()
	err := &requestError{code: 10008, message: "currency pair nonsupport"}
	assert.EqualError(t, err, "request failed: currency pair nonsupport (error_code 10008)", "Error should give LBank's message and error code")
	assert.ErrorIs(t, err, errRequestFailed, "a requestError should unwrap to errRequestFailed")
}
