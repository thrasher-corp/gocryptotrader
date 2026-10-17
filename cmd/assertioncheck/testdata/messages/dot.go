package messages

import (
	"testing"

	. "github.com/stretchr/testify/assert"
)

func dot(t *testing.T) {
	Equalf(t, 1, 1, "should match") // want "has a plain message; use Equal"
	Equal(t, "must match", "must match", "values should match")
}
