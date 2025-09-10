package main

import (
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/collateral"
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
		{getLatestFundingRate, &GetLatestFundingRateParams{}, []string{"test", "-e", "Binance", "-a", "futures", "-p", "btc-usdt"}, nil},
		{getCollateralMode, &GetCollateralMode{}, []string{"test", "-e", "Binance", "-a", "spot"}, futures.ErrNotFuturesAsset},
		{getCollateralMode, &GetCollateralMode{}, []string{"test", "-e", "Binance", "-a", "futures"}, nil},
		{setCollateralMode, &SetCollateralMode{}, []string{"test", "-e", "kucoin", "--asset", "perpetual_swap", "-c", "multi"}, asset.ErrNotSupported},
		{setCollateralMode, &SetCollateralMode{}, []string{"test", "-e", "kucoin", "--asset", "delivery", "-c", "abcd"}, collateral.ErrInvalidCollateralMode},
		{setCollateralMode, &SetCollateralMode{}, []string{"test", "-e", "kucoin", "--asset", "delivery", "-c", "multi"}, asset.ErrNotSupported},
		{setLeverage, &SetLeverage{}, []string{"test", "--exchange", "binance", "-a", "spot", "-p", "btc_usdt", "-margintype", "multi", "-l", "2312"}, futures.ErrNotFuturesAsset},
		{setLeverage, &SetLeverage{}, []string{"test", "--exchange", "binance", "-a", "futures", "-p", "btc_usdt", "-margintype", "multi", "-l", "2312"}, nil},
		{getLeverage, &LeverageInfo{}, []string{"test", "--exchange", "okx", "-a", "something", "-p", "btc_usdt", "-margintype", "multi"}, asset.ErrNotSupported},
		{getLeverage, &LeverageInfo{}, []string{"test", "--exchange", "okx", "-a", "spot", "-p", "btc_usdt", "-margintype", "multi"}, futures.ErrNotFuturesAsset},
		{getLeverage, &LeverageInfo{}, []string{"test", "--exchange", "okx", "-a", "futures", "-p", "btc_usdt", "-margintype", "multi"}, nil},
		{changePositionMargin, &ChangePositionMargin{}, []string{"test", "--exchange", "okx", "--asset", "spot", "--pair", "btc-usd", "--margintype", "cross", "--originalallocatedmargin", "123.", "--newallocatedmargin", "456"}, futures.ErrNotFuturesAsset},
		{changePositionMargin, &ChangePositionMargin{}, []string{"test", "--exchange", "okx", "--asset", "futures", "--pair", "btc-usd", "--margintype", "cross", "--originalallocatedmargin", "123.", "--newallocatedmargin", "456"}, nil},
		{getFuturesPositionSummary, &GetFuturesPositionSummary{}, []string{"test", "-e", "deribit", "-a", "spot", "-p", "btc-eth"}, futures.ErrNotFuturesAsset},
		{getFuturesPositionSummary, &GetFuturesPositionSummary{}, []string{"test", "-e", "deribit", "-a", "coinmarginedfutures", "-p", "btc-eth"}, nil},
		{getFuturePositionOrders, &GetFuturePositionOrders{}, []string{"test", "-e", "deribit", "-a", "coinmarginedfutures", "-p", "btc-eth"}, nil},
		{setMarginType, &SetMarginType{}, []string{"test", "-e", "deribit", "-a", "coinmarginedfutures", "-margintype", "multi", "-p", "btc-eth"}, nil},
		{getOpenInterest, &GetOpenInterest{}, []string{"test", "-e", "kucoin"}, nil},
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
