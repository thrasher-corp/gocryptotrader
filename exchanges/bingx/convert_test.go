package bingx

import (
	"cmp"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

func TestConvertOrderStatus(t *testing.T) {
	t.Parallel()
	for status, exp := range map[string]order.Status{
		"NEW":              order.New,
		"PENDING":          order.Pending,
		"PARTIALLY_FILLED": order.PartiallyFilled,
		"PARTIALLYFILLED":  order.PartiallyFilled,
		"FILLED":           order.Filled,
		"CANCELED":         order.Cancelled,
		"CANCELLED":        order.Cancelled,
		"EXPIRED":          order.Expired,
		"FAILED":           order.Rejected,
		"Unknown":          order.UnknownStatus,
	} {
		t.Run(status, func(t *testing.T) {
			t.Parallel()
			got, err := convertOrderStatus(status)
			assert.NoErrorf(t, err, "convertOrderStatus should not error for %s", status)
			assert.Equalf(t, exp, got, "convertOrderStatus should convert %s", status)
		})
	}
	t.Run("unsupported", func(t *testing.T) {
		t.Parallel()
		got, err := convertOrderStatus("new")
		assert.ErrorIs(t, err, order.ErrUnsupportedStatusType, "convertOrderStatus should reject a status in another case")
		assert.Equal(t, order.UnknownStatus, got, "convertOrderStatus should return no status for an unsupported one")
	})
}

func TestConvertOrderType(t *testing.T) {
	t.Parallel()
	for orderType, exp := range map[OrderType]order.Type{
		OrderTypeLimit:                        order.Limit,
		OrderTypeMarket:                       order.Market,
		OrderTypeStop:                         order.StopLimit,
		OrderTypeTakeStopLimit:                order.StopLimit,
		OrderTypeTriggerLimit:                 order.StopLimit,
		OrderTypeStopMarket:                   order.StopMarket,
		OrderTypeTakeStopMarket:               order.StopMarket,
		OrderTypeTriggerMarket:                order.StopMarket,
		OrderTypeTakeProfit:                   order.TakeProfit,
		OrderTypeTakeProfitMarket:             order.TakeProfitMarket,
		OrderTypeTrailingStopMarket:           order.TrailingStop,
		OrderTypeTrailingTakeProfitOrStopLoss: order.TrailingStop,
		OrderTypeLiquidation:                  order.Liquidation,
	} {
		t.Run(string(orderType), func(t *testing.T) {
			t.Parallel()
			got, err := convertOrderType(orderType)
			assert.NoErrorf(t, err, "convertOrderType should not error for %s", orderType)
			assert.Equalf(t, exp, got, "convertOrderType should convert %s", orderType)
		})
	}
	t.Run("unsupported", func(t *testing.T) {
		t.Parallel()
		got, err := convertOrderType("ICEBERG")
		assert.ErrorIs(t, err, order.ErrUnrecognisedOrderType, "convertOrderType should reject an unknown type")
		assert.Equal(t, order.UnknownType, got, "convertOrderType should return no type for an unknown one")
	})
}

func TestConvertOrderSide(t *testing.T) {
	t.Parallel()
	got, err := convertOrderSide(SideBuy)
	assert.NoError(t, err, "convertOrderSide should not error for BUY")
	assert.Equal(t, order.Buy, got, "convertOrderSide should convert BUY")
	got, err = convertOrderSide(SideSell)
	assert.NoError(t, err, "convertOrderSide should not error for SELL")
	assert.Equal(t, order.Sell, got, "convertOrderSide should convert SELL")
	got, err = convertOrderSide("BOTH")
	assert.ErrorIs(t, err, order.ErrSideIsInvalid, "convertOrderSide should reject an unknown side")
	assert.Equal(t, order.UnknownSide, got, "convertOrderSide should return no side for an unknown one")
}

func TestConvertTimeInForce(t *testing.T) {
	t.Parallel()
	for timeInForce, exp := range map[TimeInForce]order.TimeInForce{
		TimeInForceGoodTillCancel:    order.GoodTillCancel,
		TimeInForceImmediateOrCancel: order.ImmediateOrCancel,
		TimeInForceFillOrKill:        order.FillOrKill,
		TimeInForcePostOnly:          order.PostOnly,
		"":                           order.UnknownTIF,
	} {
		t.Run(string(timeInForce), func(t *testing.T) {
			t.Parallel()
			got, err := convertTimeInForce(timeInForce)
			assert.NoErrorf(t, err, "convertTimeInForce should not error for %q", timeInForce)
			assert.Equalf(t, exp, got, "convertTimeInForce should convert %q", timeInForce)
		})
	}
	t.Run("unsupported", func(t *testing.T) {
		t.Parallel()
		got, err := convertTimeInForce("GTX")
		assert.ErrorIs(t, err, order.ErrInvalidTimeInForce, "convertTimeInForce should reject an unknown time in force")
		assert.Equal(t, order.UnknownTIF, got, "convertTimeInForce should return no time in force for an unknown one")
	})
}

func TestConvertCoinMarginedOrderStatus(t *testing.T) {
	t.Parallel()
	for status, exp := range map[string]order.Status{
		"PENDING":   order.Pending,
		"Pending":   order.Pending,
		"filled":    order.Filled,
		"Cancelled": order.Cancelled,
		"Unknown":   order.UnknownStatus,
		"UNKNOWN":   order.UnknownStatus,
	} {
		t.Run(status, func(t *testing.T) {
			t.Parallel()
			got, err := convertCoinMarginedOrderStatus(status)
			require.NoErrorf(t, err, "convertCoinMarginedOrderStatus must not error for %s", status)
			assert.Equalf(t, exp, got, "convertCoinMarginedOrderStatus should convert %s in any case", status)
		})
	}
	t.Run("unknown status", func(t *testing.T) {
		t.Parallel()
		got, err := convertCoinMarginedOrderStatus("Settled")
		assert.ErrorIs(t, err, order.ErrUnsupportedStatusType, "convertCoinMarginedOrderStatus should reject an unknown status")
		assert.Equal(t, order.UnknownStatus, got, "convertCoinMarginedOrderStatus should return no status for an unknown one")
	})
}

func TestConvertPositionSide(t *testing.T) {
	t.Parallel()
	for side, exp := range map[string]PositionSide{"LONG": PositionSideLong, "Long": PositionSideLong, "short": PositionSideShort, "BOTH": PositionSideBoth, "": ""} {
		t.Run(cmp.Or(side, "empty"), func(t *testing.T) {
			t.Parallel()
			assert.Equalf(t, exp, convertPositionSide(side), "convertPositionSide should convert %q in any case", side)
		})
	}
}

func TestFormatOrderSide(t *testing.T) {
	t.Parallel()
	for side, exp := range map[order.Side]Side{order.Buy: SideBuy, order.Bid: SideBuy, order.Long: SideBuy, order.Sell: SideSell, order.Ask: SideSell, order.Short: SideSell} {
		t.Run(side.String(), func(t *testing.T) {
			t.Parallel()
			got, err := formatOrderSide(side)
			require.NoErrorf(t, err, "formatOrderSide must not error for %s", side)
			assert.Equalf(t, exp, got, "formatOrderSide should format %s", side)
		})
	}
	for _, side := range []order.Side{order.UnknownSide, order.AnySide, order.ClosePosition} {
		t.Run(side.String(), func(t *testing.T) {
			t.Parallel()
			_, err := formatOrderSide(side)
			assert.ErrorIsf(t, err, order.ErrSideIsInvalid, "formatOrderSide should reject %s", side)
		})
	}
}

func TestResolveOrderType(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		orderType order.Type
		price     float64
		exp       order.Type
	}{
		{order.Stop, 1, order.StopLimit},
		{order.Stop, 0, order.StopMarket},
		{order.TakeProfit, 1, order.TakeProfit},
		{order.TakeProfit, 0, order.TakeProfitMarket},
		{order.Limit, 0, order.Limit},
		{order.StopLimit, 0, order.StopLimit},
		{order.TrailingStop, 1, order.TrailingStop},
	} {
		t.Run(fmt.Sprintf("%s with price %v", tc.orderType, tc.price), func(t *testing.T) {
			t.Parallel()
			assert.Equalf(t, tc.exp, resolveOrderType(tc.orderType, tc.price), "resolveOrderType should place %s with price %v as %s", tc.orderType, tc.price, tc.exp)
		})
	}
}

func TestFormatOrderType(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		a         asset.Item
		orderType order.Type
		exp       OrderType
	}{
		{asset.Spot, order.Limit, OrderTypeLimit},
		{asset.Spot, order.Market, OrderTypeMarket},
		{asset.Spot, order.StopLimit, OrderTypeTakeStopLimit},
		{asset.Spot, order.StopMarket, OrderTypeTakeStopMarket},
		{asset.USDTMarginedFutures, order.Limit, OrderTypeLimit},
		{asset.USDTMarginedFutures, order.StopLimit, OrderTypeStop},
		{asset.USDTMarginedFutures, order.StopMarket, OrderTypeStopMarket},
		{asset.USDTMarginedFutures, order.TakeProfit, OrderTypeTakeProfit},
		{asset.USDTMarginedFutures, order.TakeProfitMarket, OrderTypeTakeProfitMarket},
		{asset.USDTMarginedFutures, order.TrailingStop, OrderTypeTrailingStopMarket},
		{asset.CoinMarginedFutures, order.Market, OrderTypeMarket},
		{asset.CoinMarginedFutures, order.StopLimit, OrderTypeStop},
		{asset.CoinMarginedFutures, order.StopMarket, OrderTypeStopMarket},
		{asset.CoinMarginedFutures, order.TakeProfit, OrderTypeTakeProfit},
		{asset.CoinMarginedFutures, order.TakeProfitMarket, OrderTypeTakeProfitMarket},
	} {
		t.Run(tc.a.String()+" "+tc.orderType.String(), func(t *testing.T) {
			t.Parallel()
			got, err := formatOrderType(tc.a, tc.orderType)
			require.NoErrorf(t, err, "formatOrderType must not error for %s %s", tc.a, tc.orderType)
			assert.Equalf(t, tc.exp, got, "formatOrderType should format %s %s", tc.a, tc.orderType)
		})
	}
	for _, tc := range []struct {
		a         asset.Item
		orderType order.Type
	}{
		{asset.Spot, order.TakeProfit},
		{asset.Spot, order.TrailingStop},
		{asset.CoinMarginedFutures, order.TrailingStop},
		{asset.USDTMarginedFutures, order.Stop},
		{asset.Futures, order.StopLimit},
	} {
		t.Run(tc.a.String()+" "+tc.orderType.String(), func(t *testing.T) {
			t.Parallel()
			_, err := formatOrderType(tc.a, tc.orderType)
			assert.ErrorIsf(t, err, order.ErrUnsupportedOrderType, "formatOrderType should reject %s %s", tc.a, tc.orderType)
		})
	}
}

func TestOrderTypeTakesPrice(t *testing.T) {
	t.Parallel()
	for orderType, exp := range map[OrderType]bool{
		OrderTypeLimit:              true,
		OrderTypeStop:               true,
		OrderTypeTakeProfit:         true,
		OrderTypeTakeStopLimit:      true,
		OrderTypeTriggerLimit:       true,
		OrderTypeMarket:             false,
		OrderTypeStopMarket:         false,
		OrderTypeTakeProfitMarket:   false,
		OrderTypeTakeStopMarket:     false,
		OrderTypeTriggerMarket:      false,
		OrderTypeTrailingStopMarket: false,
	} {
		t.Run(string(orderType), func(t *testing.T) {
			t.Parallel()
			assert.Equalf(t, exp, orderTypeTakesPrice(orderType), "orderTypeTakesPrice should report whether %s takes a price", orderType)
		})
	}
}

func TestOrderTypeTakesTrigger(t *testing.T) {
	t.Parallel()
	for orderType, exp := range map[OrderType]bool{
		OrderTypeStop:               true,
		OrderTypeStopMarket:         true,
		OrderTypeTakeProfit:         true,
		OrderTypeTakeProfitMarket:   true,
		OrderTypeTakeStopLimit:      true,
		OrderTypeTakeStopMarket:     true,
		OrderTypeTriggerLimit:       true,
		OrderTypeTriggerMarket:      true,
		OrderTypeLimit:              false,
		OrderTypeMarket:             false,
		OrderTypeTrailingStopMarket: false,
	} {
		t.Run(string(orderType), func(t *testing.T) {
			t.Parallel()
			assert.Equalf(t, exp, orderTypeTakesTrigger(orderType), "orderTypeTakesTrigger should report whether %s triggers at a stop price", orderType)
		})
	}
}

func TestOrderTypeTrails(t *testing.T) {
	t.Parallel()
	for orderType, exp := range map[OrderType]bool{
		OrderTypeTrailingStopMarket:           true,
		OrderTypeTrailingTakeProfitOrStopLoss: true,
		OrderTypeStopMarket:                   false,
		OrderTypeLimit:                        false,
		OrderTypeTakeStopMarket:               false,
	} {
		t.Run(string(orderType), func(t *testing.T) {
			t.Parallel()
			assert.Equalf(t, exp, orderTypeTrails(orderType), "orderTypeTrails should report whether %s trails the price", orderType)
		})
	}
}

func TestFormatTimeInForce(t *testing.T) {
	t.Parallel()
	for timeInForce, exp := range map[order.TimeInForce]TimeInForce{
		order.UnknownTIF:                      "",
		order.GoodTillCancel:                  TimeInForceGoodTillCancel,
		order.ImmediateOrCancel:               TimeInForceImmediateOrCancel,
		order.FillOrKill:                      TimeInForceFillOrKill,
		order.PostOnly:                        TimeInForcePostOnly,
		order.PostOnly | order.GoodTillCancel: TimeInForcePostOnly,
	} {
		t.Run(cmp.Or(timeInForce.String(), "unset"), func(t *testing.T) {
			t.Parallel()
			got, err := formatTimeInForce(timeInForce)
			require.NoErrorf(t, err, "formatTimeInForce must not error for %s", timeInForce)
			assert.Equalf(t, exp, got, "formatTimeInForce should format %s", timeInForce)
		})
	}
	for _, timeInForce := range []order.TimeInForce{order.GoodTillDay, order.PostOnly | order.ImmediateOrCancel} {
		t.Run(timeInForce.String(), func(t *testing.T) {
			t.Parallel()
			_, err := formatTimeInForce(timeInForce)
			assert.ErrorIsf(t, err, order.ErrUnsupportedTimeInForce, "formatTimeInForce should reject %s", timeInForce)
		})
	}
}

func TestFormatWorkingType(t *testing.T) {
	t.Parallel()
	for priceType, exp := range map[order.PriceType]WorkingType{
		order.LastPrice:  WorkingTypeContractPrice,
		order.MarkPrice:  WorkingTypeMarkPrice,
		order.IndexPrice: WorkingTypeIndexPrice,
	} {
		t.Run(priceType.String(), func(t *testing.T) {
			t.Parallel()
			got, err := formatWorkingType(priceType)
			require.NoErrorf(t, err, "formatWorkingType must not error for %s", priceType)
			assert.Equalf(t, exp, got, "formatWorkingType should format %s", priceType)
		})
	}
	t.Run("unknown price type", func(t *testing.T) {
		t.Parallel()
		_, err := formatWorkingType(order.PriceType(99))
		assert.ErrorIs(t, err, order.ErrUnknownPriceType, "formatWorkingType should reject an unknown price type")
	})
}
