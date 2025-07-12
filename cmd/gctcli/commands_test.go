package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
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

func TestUnmarshalCLIFields(t *testing.T) {
	t.Parallel()
	type SampleTest struct {
		Exchange      string `cli:"exchange,required"`
		OrderID       string `cli:"order_id,required"`
		ClientOrderID string `cli:"client_order_id"`
		PostOnly      bool   `cli:"post_only"`
		ReduceOnly    bool   `cli:"reduce_only"`
	}
	sample1 := &SampleTest{Exchange: "Okx", OrderID: "1234", ClientOrderID: "5678", PostOnly: true}

	target := &SampleTest{}
	app := &cli.App{
		Flags: FlagsFromStruct(sample1, nil),
		Action: func(ctx *cli.Context) error {
			return UnmarshalCLIFields(ctx, target)
		},
	}
	err := app.Run([]string{"test", "-exchange", "", "-order_id", "1234", "-client_order_id", "5678"})
	require.ErrorIs(t, err, ErrRequiredValueMissing)

	err = app.Run([]string{"test", "-exchange", "Okx", "-order_id", "1234", "-client_order_id", "5678", "-post_only", "true"})
	require.NoError(t, err)
	assert.Equal(t, *sample1, *target)
}
