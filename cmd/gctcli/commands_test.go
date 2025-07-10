package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFlagsFromStruct(t *testing.T) {
	t.Parallel()
	flags := FlagsFromStruct(&struct {
		Exchange string  `cli:"exchange"`
		Leverage int64   `cli:"leverage"`
		Price    float64 `cli:"price"`
	}{
		Exchange: "okx",
		Leverage: 1,
		Price:    3.1415,
	}, map[string]string{"price": "the price for the order"})
	require.Len(t, flags, 3)
	for e := range flags {
		assert.Contains(t, []string{"exchange", "leverage", "price"}, flags[e].Names()[0])
	}
}
