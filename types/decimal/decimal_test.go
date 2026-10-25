package decimal

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDecimalIsInteger(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name     string
		value    Decimal
		expected bool
	}{
		{name: "positive integer", value: NewFromInt(42), expected: true},
		{name: "negative integer", value: MustFromString("-42.000"), expected: true},
		{name: "zero", value: Zero, expected: true},
		{name: "positive fraction", value: MustFromString("42.1"), expected: false},
		{name: "negative fraction", value: MustFromString("-0.1"), expected: false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, tc.value.IsInteger(),
				"IsInteger should identify values without a fractional component")
		})
	}
}
