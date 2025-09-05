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
		Exchange string  `name:"exchange"`
		Leverage int64   `name:"leverage"`
		Price    float64 `name:"price" usage:"the price for the order"`
	}{
		Exchange: "okx",
		Leverage: 1,
		Price:    3.1415,
	})
	require.Len(t, flags, 3)
	for e := range flags {
		assert.Contains(t, []string{"exchange", "leverage", "price"}, flags[e].Names()[0])
	}
}

func TestUnmarshalCLIFields(t *testing.T) {
	t.Parallel()
	// FlagsFromStringNew
	type SampleTest struct {
		Exchange      string `name:"exchange" required:"t"`
		OrderID       string `name:"order_id" required:"true"`
		ClientOrderID string `name:"client_order_id"`
		PostOnly      bool   `name:"post_only"`
		ReduceOnly    bool   `name:"reduce_only"`
	}
	sample1 := &SampleTest{Exchange: "Okx", OrderID: "1234", ClientOrderID: "5678", PostOnly: true}

	target := &SampleTest{}
	flags := FlagsFromStruct(target)

	app := &cli.App{
		Flags: flags,
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
