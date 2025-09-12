package bybit

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

func TestDeriveSubmitOrderArguments(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		submit *order.Submit
		exp    *PlaceOrderRequest
		err    error
	}{
		{err: order.ErrSubmissionIsNil},
		{
			submit: &order.Submit{
				Exchange:  e.GetName(),
				Pair:      currency.NewBTCUSDT(),
				AssetType: asset.Binary,
				Side:      order.Buy,
				Type:      order.Market,
				Amount:    1,
			},
			err: asset.ErrNotSupported,
		},
		{
			submit: &order.Submit{
				Exchange:  e.GetName(),
				Pair:      currency.NewBTCUSDT(),
				AssetType: asset.USDCMarginedFutures,
				Side:      order.Buy,
				Type:      order.Market,
				Amount:    1,
			},
			exp: &PlaceOrderRequest{
				Category:      getCategoryName(asset.USDCMarginedFutures),
				Symbol:        currency.NewBTCUSDT().Format(currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter}),
				Side:          sideBuy,
				OrderType:     orderTypeToString(order.Market),
				OrderQuantity: 1,
				TimeInForce:   "IOC",
			},
		},
		{
			submit: &order.Submit{
				Exchange:  e.GetName(),
				Pair:      currency.NewBTCUSDT(),
				AssetType: asset.USDCMarginedFutures,
				Side:      order.Buy,
				Type:      order.Market,
				Amount:    1,
				RiskManagementModes: order.RiskManagementModes{
					TakeProfit: order.RiskManagement{Price: 100},
					StopLoss:   order.RiskManagement{Price: 200},
				},
			},
			exp: &PlaceOrderRequest{
				Category:            getCategoryName(asset.USDCMarginedFutures),
				Symbol:              currency.NewBTCUSDT().Format(currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter}),
				Side:                sideBuy,
				OrderType:           orderTypeToString(order.Market),
				OrderQuantity:       1,
				TimeInForce:         "IOC",
				TakeProfitPrice:     100,
				TakeProfitTriggerBy: "LastPrice",
				StopLossPrice:       200,
				StopLossTriggerBy:   "LastPrice",
			},
		},
		{
			submit: &order.Submit{
				Exchange:     e.GetName(),
				Pair:         currency.NewBTCUSDT(),
				AssetType:    asset.USDCMarginedFutures,
				Side:         order.Sell,
				Type:         order.Limit,
				Amount:       1,
				Price:        5000,
				TriggerPrice: 150,
				TimeInForce:  order.FillOrKill,
			},
			exp: &PlaceOrderRequest{
				Category:         getCategoryName(asset.USDCMarginedFutures),
				Symbol:           currency.NewBTCUSDT().Format(currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter}),
				Side:             sideSell,
				OrderType:        orderTypeToString(order.Limit),
				OrderQuantity:    1,
				TimeInForce:      "FOK",
				TriggerPrice:     150,
				Price:            5000,
				TriggerPriceType: "LastPrice",
			},
		},
		{
			submit: &order.Submit{
				Exchange:     e.GetName(),
				Pair:         currency.NewBTCUSDT(),
				AssetType:    asset.USDCMarginedFutures,
				Side:         order.Sell,
				Type:         order.Limit,
				Amount:       1,
				Price:        5000,
				TriggerPrice: 150,
				TimeInForce:  order.PostOnly,
			},
			exp: &PlaceOrderRequest{
				Category:         getCategoryName(asset.USDCMarginedFutures),
				Symbol:           currency.NewBTCUSDT().Format(currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter}),
				Side:             sideSell,
				OrderType:        orderTypeToString(order.Limit),
				OrderQuantity:    1,
				TimeInForce:      "PostOnly",
				TriggerPrice:     150,
				Price:            5000,
				TriggerPriceType: "LastPrice",
			},
		},
		{
			submit: &order.Submit{
				Exchange:     e.GetName(),
				Pair:         currency.NewBTCUSDT(),
				AssetType:    asset.Spot,
				Side:         order.Sell,
				Type:         order.Limit,
				Amount:       1,
				Price:        5000,
				TriggerPrice: 150,
				TimeInForce:  order.ImmediateOrCancel,
			},
			exp: &PlaceOrderRequest{
				Category:         getCategoryName(asset.Spot),
				Symbol:           currency.NewBTCUSDT().Format(currency.PairFormat{Uppercase: true}),
				Side:             sideSell,
				OrderType:        orderTypeToString(order.Limit),
				OrderQuantity:    1,
				TimeInForce:      "IOC",
				TriggerPrice:     150,
				Price:            5000,
				OrderFilter:      "tpslOrder",
				TriggerPriceType: "LastPrice",
			},
		},
	} {
		got, err := e.deriveSubmitOrderArguments(tc.submit)
		if tc.err != nil {
			require.ErrorIs(t, err, tc.err)
			continue
		}
		require.NoError(t, err)
		assert.Equal(t, tc.exp, got)
	}
}

func TestDeriveAmendOrderArguments(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		action *order.Modify
		exp    *AmendOrderRequest
		err    error
	}{
		{err: order.ErrModifyOrderIsNil},
		{
			action: &order.Modify{
				OrderID:   "69420",
				Pair:      currency.NewBTCUSDT(),
				AssetType: asset.Binary,
			},
			err: asset.ErrNotSupported,
		},
		{
			action: &order.Modify{
				OrderID:   "69420",
				Pair:      currency.NewBTCUSDT(),
				AssetType: asset.USDCMarginedFutures,
			},
			exp: &AmendOrderRequest{
				Category:          getCategoryName(asset.USDCMarginedFutures),
				Symbol:            currency.NewBTCUSDT().Format(currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter}),
				OrderID:           "69420",
				StopLossTriggerBy: "LastPrice",
				TriggerPriceType:  "LastPrice",
			},
		},
	} {
		got, err := e.deriveAmendOrderArguments(tc.action)
		if tc.err != nil {
			require.ErrorIs(t, err, tc.err)
			continue
		}
		require.NoError(t, err)
		assert.Equal(t, tc.exp, got)
	}
}

func TestDeriveCancelOrderArguments(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		action *order.Cancel
		exp    *CancelOrderRequest
		err    error
	}{
		{err: order.ErrCancelOrderIsNil},
		{
			action: &order.Cancel{
				OrderID:   "69420",
				Pair:      currency.NewBTCUSDT(),
				AssetType: asset.Binary,
			},
			err: asset.ErrNotSupported,
		},
		{
			action: &order.Cancel{
				OrderID:   "69420",
				Pair:      currency.NewBTCUSDT(),
				AssetType: asset.USDCMarginedFutures,
			},
			exp: &CancelOrderRequest{
				Category: getCategoryName(asset.USDCMarginedFutures),
				Symbol:   currency.NewBTCUSDT().Format(currency.PairFormat{Uppercase: true, Delimiter: currency.DashDelimiter}),
				OrderID:  "69420",
			},
		},
	} {
		got, err := e.deriveCancelOrderArguments(tc.action)
		if tc.err != nil {
			require.ErrorIs(t, err, tc.err)
			continue
		}
		require.NoError(t, err)
		assert.Equal(t, tc.exp, got)
	}
}
