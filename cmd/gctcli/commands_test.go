package main

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v2"
)

func TestFlagsFromStruct(t *testing.T) {
	t.Parallel()
	targets := []cli.Flag{
		&cli.StringFlag{Name: "exchange", Required: true, Usage: "the required 'exchange' for the request", Value: "okx"},
		&cli.Int64Flag{Name: "leverage", Required: true, Usage: "the required 'leverage' for the request", Value: 1},
		&cli.Float64Flag{Name: "price", Usage: "the price for the order", Value: 3.141529},
		&cli.StringFlag{Name: "cryptocurrency", Aliases: []string{"c"}, Required: true, Usage: "the cryptocurrency to get the deposit address for"},
		&cli.StringFlag{Name: "asset", Aliases: []string{"a"}, Usage: "the optional 'asset' for the request"},
		&cli.Int64Flag{Name: "limit", Usage: "the optional 'limit' for the request"},
		&cli.BoolFlag{Name: "sync", Usage: "<true/false>", Value: true},
		&cli.Float64Flag{Name: "amount", Usage: "the optional 'amount' for the request"},
	}
	flags := FlagsFromStruct(&struct {
		Exchange    string  `name:"exchange"         required:"true"`
		Leverage    int64   `name:"leverage"         required:"t"`
		Price       float64 `name:"price"            usage:"the price for the order"`
		Currency    string  `name:"cryptocurrency,c" required:"t"                    usage:"the cryptocurrency to get the deposit address for"`
		AssetType   string  `name:"asset,a"`
		Limit       int64   `name:"limit"`
		Sync        bool    `name:"sync"             usage:"<true/false>"`
		Amount      float64 `name:"amount"`
		hiddenValue int64   `name:"hidden"`
		NoTag       bool
	}{
		Exchange: "okx",
		Leverage: 1,
		Price:    3.141529,
		Sync:     true,
	})
	require.Len(t, flags, len(targets))
	for i := range targets {
		require.Equal(t, reflect.TypeOf(targets[i]), reflect.TypeOf(flags[i]))
		require.Equal(t, targets[i].Names(), flags[i].Names())
		switch target := targets[i].(type) {
		case *cli.StringFlag:
			flag, ok := flags[i].(*cli.StringFlag)
			require.True(t, ok)
			require.Equal(t, target.Required, flag.Required)
			require.Equal(t, target.Aliases, flag.Aliases)
			require.Equal(t, target.Usage, flag.Usage)
			require.Equal(t, target.Usage, flag.Usage)
		case *cli.Float64Flag:
			flag, ok := flags[i].(*cli.Float64Flag)
			require.True(t, ok)

			require.Equal(t, target.Required, flag.Required)
			require.Equal(t, target.Aliases, flag.Aliases)
			require.Equal(t, target.Usage, flag.Usage)
			require.Equal(t, target.Usage, flag.Usage)
		case *cli.Int64Flag:
			flag, ok := flags[i].(*cli.Int64Flag)
			require.True(t, ok)

			require.Equal(t, target.Required, flag.Required)
			require.Equal(t, target.Aliases, flag.Aliases)
			require.Equal(t, target.Usage, flag.Usage)
			require.Equal(t, target.Usage, flag.Usage)
		case *cli.BoolFlag:
			flag, ok := flags[i].(*cli.BoolFlag)
			require.True(t, ok)

			require.Equal(t, target.Required, flag.Required)
			require.Equal(t, target.Aliases, flag.Aliases)
			require.Equal(t, target.Usage, flag.Usage)
			require.Equal(t, target.DefaultText, flag.DefaultText)
		}
	}
}

func TestUnmarshalCLIFields(t *testing.T) {
	t.Parallel()
	type SampleTest struct {
		Exchange      string `name:"exchange"        required:"t"`
		OrderID       int64  `name:"order_id"        required:"true"`
		ClientOrderID string `name:"client_order_id"`
		PostOnly      bool   `name:"post_only"`
		ReduceOnly    bool   `name:"reduce_only"`
	}

	flags := FlagsFromStruct(&SampleTest{Exchange: "Okx", OrderID: 1234, ClientOrderID: "5678", PostOnly: true})

	var target SampleTest
	app := &cli.App{
		Flags: flags,
		Action: func(ctx *cli.Context) error {
			return UnmarshalCLIFields(ctx, &target)
		},
	}
	err := app.Run([]string{"test", "-exchange", "", "-order_id", "1234", "-client_order_id", "5678"})
	require.ErrorIs(t, err, ErrRequiredValueMissing)

	err = app.Run([]string{"test", "-exchange", "Okx", "-order_id", "4321", "-client_order_id", "9012", "-post_only", "true"})
	require.NoError(t, err)
	assert.Equal(t,
		SampleTest{Exchange: "Okx", OrderID: 4321, ClientOrderID: "9012", PostOnly: true},
		target)
}
