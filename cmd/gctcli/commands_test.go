package main

import (
	"errors"
	"fmt"
	"os"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/collateral"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
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
	assert.Equal(t, SampleTest{Exchange: "Okx", OrderID: 4321, ClientOrderID: "9012", PostOnly: true}, target)
}

func TestFunctionsAndStructHandling(t *testing.T) {
	t.Parallel()

	funcAndValue := []struct {
		l                   string
		function            func(c *cli.Context) error
		val                 any
		args                []string
		err                 error
		missingRequiredFlag string
	}{
		{l: "withdrawlRequestByDate", function: withdrawlRequestByDate, val: &WithdrawalRequestByDate{Start: time.Now().AddDate(0, -1, 0).Format(time.DateTime), End: time.Now().Format(time.DateTime)}, args: []string{"test", "--exchange", "binance"}},

		// Futures commands handled
		{l: "getManagedPosition-ErrNotFuturesAsset", function: getManagedPosition, val: &GetManagedPositionsParams{}, args: []string{"test", "-e", "Okx", "-a", "spot", "-p", "btc-usdt"}, err: futures.ErrNotFuturesAsset},
		{l: "getManagedPosition-MissingRequiredFlag", function: getManagedPosition, val: &GetManagedPositionsParams{}, args: []string{"test", "-e", "Okx", "-a", "spot"}, missingRequiredFlag: "pair"},
		{l: "getManagedPosition", function: getManagedPosition, val: &GetManagedPositionsParams{}, args: []string{"test", "-e", "Okx", "-a", "futures", "-p", "btc-usdt"}},
		{l: "getAllManagedPositions", function: getAllManagedPositions, val: &GetAllManagedPositions{}},
		{l: "getCollateral", function: getCollateral, val: &GetCollateralParams{}, args: []string{"test", "-e", "okx", "-a", "futures"}},
		{l: "getLatestFundingRate", function: getLatestFundingRate, val: &GetLatestFundingRateParams{}, args: []string{"test", "-e", "Binance", "-a", "futures", "-p", "btc-usdt"}},
		{l: "getLatestFundingRate-RequiredValueMissing-asset", function: getLatestFundingRate, val: &GetLatestFundingRateParams{}, args: []string{"test", "-e", "Binance", "-a", "", "-p", "btc-usdt"}, err: ErrRequiredValueMissing},
		{l: "getLatestFundingRate-RequiredValueMissing-pair", function: getLatestFundingRate, val: &GetLatestFundingRateParams{}, args: []string{"test", "-e", "Binance", "-a", "futures", "-p", ""}, err: ErrRequiredValueMissing},
		{l: "getCollateralMode", function: getCollateralMode, val: &GetCollateralMode{}, args: []string{"test", "-e", "Binance", "-a", "spot"}, err: futures.ErrNotFuturesAsset},
		{l: "getCollateralMode", function: getCollateralMode, val: &GetCollateralMode{}, args: []string{"test", "-e", "Binance", "-a", "futures"}},
		{l: "setCollateralMode", function: setCollateralMode, val: &SetCollateralMode{}, args: []string{"test", "-e", "kucoin", "--asset", "perpetual_swap", "-c", "multi"}, err: asset.ErrNotSupported},
		{l: "setCollateralMode", function: setCollateralMode, val: &SetCollateralMode{}, args: []string{"test", "-e", "kucoin", "--asset", "delivery", "-c", "abcd"}, err: collateral.ErrInvalidCollateralMode},
		{l: "setCollateralMode", function: setCollateralMode, val: &SetCollateralMode{}, args: []string{"test", "-e", "kucoin", "--asset", "delivery", "-c", "multi"}, err: asset.ErrNotSupported},
		{l: "setLeverage", function: setLeverage, val: &SetLeverage{}, args: []string{"test", "--exchange", "binance", "-a", "spot", "-p", "btc_usdt", "-margintype", "multi", "-l", "2312"}, err: futures.ErrNotFuturesAsset},
		{l: "setLeverage", function: setLeverage, val: &SetLeverage{}, args: []string{"test", "--exchange", "binance", "-a", "futures", "-p", "btc_usdt", "-margintype", "multi", "-l", "2312"}},
		{l: "getLeverage", function: getLeverage, val: &LeverageInfo{}, args: []string{"test", "--exchange", "okx", "-a", "something", "-p", "btc_usdt", "-margintype", "multi"}, err: asset.ErrNotSupported},
		{l: "getLeverage", function: getLeverage, val: &LeverageInfo{}, args: []string{"test", "--exchange", "okx", "-a", "spot", "-p", "btc_usdt", "-margintype", "multi"}, err: futures.ErrNotFuturesAsset},
		{l: "getLeverage", function: getLeverage, val: &LeverageInfo{}, args: []string{"test", "--exchange", "okx", "-a", "futures", "-p", "btc_usdt", "-margintype", "multi"}},
		{l: "changePositionMargin", function: changePositionMargin, val: &ChangePositionMargin{}, args: []string{"test", "--exchange", "okx", "--asset", "spot", "--pair", "btc-usd", "--margintype", "cross", "--originalallocatedmargin", "123.", "--newallocatedmargin", "456"}, err: futures.ErrNotFuturesAsset},
		{l: "changePositionMargin", function: changePositionMargin, val: &ChangePositionMargin{}, args: []string{"test", "--exchange", "okx", "--asset", "futures", "--pair", "btc-usd", "--margintype", "cross", "--originalallocatedmargin", "123.", "--newallocatedmargin", "456"}},
		{l: "getFuturesPositionSummary", function: getFuturesPositionSummary, val: &GetFuturesPositionSummary{}, args: []string{"test", "-e", "deribit", "-a", "spot", "-p", "btc-eth"}, err: futures.ErrNotFuturesAsset},
		{l: "getFuturesPositionSummary", function: getFuturesPositionSummary, val: &GetFuturesPositionSummary{}, args: []string{"test", "-e", "deribit", "-a", "coinmarginedfutures", "-p", "btc-eth"}},
		{l: "getFuturePositionOrders", function: getFuturePositionOrders, val: &GetFuturePositionOrders{}, args: []string{"test", "-e", "deribit", "-a", "coinmarginedfutures", "-p", "btc-eth"}},
		{l: "setMarginType", function: setMarginType, val: &SetMarginType{}, args: []string{"test", "-e", "deribit", "-a", "coinmarginedfutures", "-margintype", "multi", "-p", "btc-eth"}},
		{l: "getOpenInterest", function: getOpenInterest, val: &GetOpenInterest{}, args: []string{"test", "-e", "kucoin"}},

		// Trade commands handler
		{function: setExchangeTradeProcessing, val: &SetExchangeTradeProcessing{}, args: []string{"setexchangetradeprocessing", "-e", "binance", "-status"}},
	}

	for a := range funcAndValue {
		t.Run(funcAndValue[a].l, func(t *testing.T) {
			t.Parallel()
			app := &cli.App{
				Flags:  FlagsFromStruct(funcAndValue[a].val),
				Action: funcAndValue[a].function,
			}

			err := app.Run(funcAndValue[a].args)
			if funcAndValue[a].missingRequiredFlag != "" {
				require.ErrorContains(t, err, fmt.Sprintf("Required flag %q not set", funcAndValue[a].missingRequiredFlag))
			} else if !errors.Is(err, os.ErrNotExist) {
				require.ErrorIs(t, err, funcAndValue[a].err)
			}
		})
	}
}
