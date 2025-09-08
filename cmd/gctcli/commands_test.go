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
		&cli.StringFlag{Name: "exchange", Required: true, Usage: "the required 'exchange' for the request"},
		&cli.Int64Flag{Name: "leverage", Required: true, Usage: "the required 'leverage' for the request"},
		&cli.Float64Flag{Name: "price", Usage: "the price for the order"},
		&cli.StringFlag{Name: "cryptocurrency", Aliases: []string{"c"}, Required: true, Usage: "the cryptocurrency to get the deposit address for"},
		&cli.StringFlag{Name: "asset", Aliases: []string{"a"}, Usage: "the optional 'asset' for the request"},
		&cli.Int64Flag{Name: "limit", Usage: "the optional 'limit' for the request"},
		&cli.BoolFlag{Name: "sync", Usage: "<true/false>"},
		&cli.Float64Flag{Name: "amount", Usage: "the optional 'amount' for the request"},
	}
	flags := FlagsFromStruct(&struct {
		Exchange    string  `name:"exchange" required:"true"`
		Leverage    int64   `name:"leverage" required:"t"`
		Price       float64 `name:"price" usage:"the price for the order"`
		Currency    string  `name:"cryptocurrency,c" required:"t" usage:"the cryptocurrency to get the deposit address for"`
		AssetType   string  `name:"asset,a"`
		Limit       int64   `name:"limit"`
		Sync        bool    `name:"sync" usage:"<true/false>"`
		Amount      float64 `name:"amount"`
		hiddenValue int64   `name:"hidden"`
		NoTag       bool
	}{
		Exchange: "okx",
		Leverage: 1,
		Price:    3.1415,
	})
	require.Len(t, flags, len(targets))
	for i := range targets {
		require.True(t, reflect.TypeOf(targets[i]) == reflect.TypeOf(flags[i]))
		require.Equal(t, targets[i].Names(), flags[i].Names())
		switch target := targets[i].(type) {
		case *cli.StringFlag:
			require.Equal(t, target.Required, flags[i].(*cli.StringFlag).Required)
			require.Equal(t, target.Aliases, flags[i].(*cli.StringFlag).Aliases)
			require.Equal(t, target.Usage, flags[i].(*cli.StringFlag).Usage)
			require.Equal(t, target.Usage, flags[i].(*cli.StringFlag).Usage)
		case *cli.Float64Flag:
			require.Equal(t, target.Required, flags[i].(*cli.Float64Flag).Required)
			require.Equal(t, target.Aliases, flags[i].(*cli.Float64Flag).Aliases)
			require.Equal(t, target.Usage, flags[i].(*cli.Float64Flag).Usage)
			require.Equal(t, target.Usage, flags[i].(*cli.Float64Flag).Usage)
		case *cli.Int64Flag:
			require.Equal(t, target.Required, flags[i].(*cli.Int64Flag).Required)
			require.Equal(t, target.Aliases, flags[i].(*cli.Int64Flag).Aliases)
			require.Equal(t, target.Usage, flags[i].(*cli.Int64Flag).Usage)
			require.Equal(t, target.Usage, flags[i].(*cli.Int64Flag).Usage)
		case *cli.BoolFlag:
			require.Equal(t, target.Required, flags[i].(*cli.BoolFlag).Required)
			require.Equal(t, target.Aliases, flags[i].(*cli.BoolFlag).Aliases)
			require.Equal(t, target.Usage, flags[i].(*cli.BoolFlag).Usage)
			require.Equal(t, target.DefaultText, flags[i].(*cli.BoolFlag).DefaultText)
		}
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
