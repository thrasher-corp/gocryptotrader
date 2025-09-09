package main

import (
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/urfave/cli/v2"
)

func TestUnmarshalCLIFieldsA(t *testing.T) {
	t.Parallel()

	funcAndValue := []struct {
		function func(c *cli.Context) error
		value    any
		args     []string
		err      error
	}{
		{getManagedPosition, &GetManagedPositionsParams{}, []string{"test", "-e", "Okx", "-a", "spot", "-p", "btc-usdt"}, futures.ErrNotFuturesAsset},
		{getManagedPosition, &GetManagedPositionsParams{}, []string{"test", "-e", "Okx", "-a", "futures", "-p", "btc-usdt"}, nil},
		{getAllManagedPositions, &GetAllManagedPositions{}, []string{}, nil},
		{getCollateral, &GetCollateralParams{}, []string{"test", "-e", "okx", "-a", "futures"}, nil},
	}

	for a := range funcAndValue {
		app := &cli.App{
			Flags:  FlagsFromStruct(funcAndValue[a].value),
			Action: funcAndValue[a].function,
		}

		err := app.Run(funcAndValue[a].args)
		if !errors.Is(err, os.ErrNotExist) {
			require.ErrorIs(t, err, funcAndValue[a].err)
		}
	}
}
