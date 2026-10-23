package kucoin

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestAddCopyTradingOrder(t *testing.T) {
	t.Parallel()
	_, err := e.AddCopyTradingOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "AddCopyTradingOrder must reject a nil request")
	_, err = e.AddCopyTradingOrder(t.Context(), &CopyTradingOrderRequest{})
	require.ErrorIs(t, err, order.ErrClientOrderIDMustBeSet, "AddCopyTradingOrder must validate the request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.AddCopyTradingOrder(t.Context(), &CopyTradingOrderRequest{
		ClientOrderID: "ea391ca47fb2448095b6d1aaf968baf8", Symbol: "XBTUSDTM", MarginMode: "CROSS", Leverage: 12,
		PositionSide: "LONG", Side: "buy", Type: "market", Size: 1, TimeInForce: "GTC", FundsType: "selfFunds",
	})
	require.NoError(t, err, "AddCopyTradingOrder must not error")
	if mockTests {
		exp := &CopyTradingOrderResponse{OrderID: "368516378027724800", ClientOrderID: "ea391ca47fb2448095b6d1aaf968baf8"}
		assert.Equal(t, exp, result, "AddCopyTradingOrder should decode every field")
		return
	}
	assert.NotEmpty(t, result.OrderID, "AddCopyTradingOrder should return the order ID")
}

func TestCopyTradingOrderRequestPayload(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *CopyTradingOrderRequest
		err  error
	}{
		{name: "nil", err: common.ErrNilPointer},
		{name: "no client order ID", req: &CopyTradingOrderRequest{}, err: order.ErrClientOrderIDMustBeSet},
		{name: "no symbol", req: &CopyTradingOrderRequest{ClientOrderID: "ct-1"}, err: currency.ErrSymbolStringEmpty},
		{name: "no type", req: &CopyTradingOrderRequest{ClientOrderID: "ct-1", Symbol: "XBTUSDTM"}, err: order.ErrTypeIsInvalid},
		{name: "no side", req: &CopyTradingOrderRequest{ClientOrderID: "ct-1", Symbol: "XBTUSDTM", Type: "market"}, err: order.ErrSideIsInvalid},
		{name: "no size", req: &CopyTradingOrderRequest{ClientOrderID: "ct-1", Symbol: "XBTUSDTM", Type: "market", Side: "buy"}, err: order.ErrAmountIsInvalid},
		{name: "limit without a price", req: &CopyTradingOrderRequest{ClientOrderID: "ct-1", Symbol: "XBTUSDTM", Type: "limit", Side: "buy", Size: 1}, err: order.ErrPriceMustBeSetIfLimitOrder},
		{name: "stop without a stop price type", req: &CopyTradingOrderRequest{ClientOrderID: "ct-1", Symbol: "XBTUSDTM", Type: "market", Side: "buy", Size: 1, Stop: "up", StopPrice: 120000}, err: errInvalidStopPriceType},
		{name: "stop without a stop price", req: &CopyTradingOrderRequest{ClientOrderID: "ct-1", Symbol: "XBTUSDTM", Type: "market", Side: "buy", Size: 1, Stop: "up", StopPriceType: "MP"}, err: errStopPriceRequired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := tc.req.Payload()
			assert.ErrorIs(t, err, tc.err, "Payload should return the expected error")
		})
	}

	r := &CopyTradingOrderRequest{
		ClientOrderID: "5c52e11203aa677f33e493fb", Side: "sell", Symbol: "XBTUSDTM", Leverage: 3, Type: "limit", Stop: "down",
		StopPriceType: "TP", StopPrice: 100000, ReduceOnly: true, CloseOrder: true, MarginMode: "ISOLATED", Price: 99000.5,
		Size: 2, TimeInForce: "GTC", PostOnly: true, PositionSide: "LONG", FundsType: "leadTrial",
	}
	exp := map[string]any{
		"clientOid": "5c52e11203aa677f33e493fb", "side": "sell", "symbol": "XBTUSDTM", "leverage": uint64(3), "type": "limit",
		"stop": "down", "stopPriceType": "TP", "stopPrice": "100000", "reduceOnly": true, "closeOrder": true,
		"marginMode": "ISOLATED", "price": "99000.5", "size": uint64(2), "timeInForce": "GTC", "postOnly": true,
		"positionSide": "LONG", "fundsType": "leadTrial",
	}
	body, err := r.Payload()
	require.NoError(t, err, "Payload must not error")
	assert.Equal(t, exp, body, "Payload should send every documented parameter")

	body, err = (&CopyTradingOrderRequest{ClientOrderID: "ct-close", Symbol: "XBTUSDTM", Type: "market", CloseOrder: true}).Payload()
	require.NoError(t, err, "Payload must not error for a close order without a side or size")
	exp = map[string]any{"clientOid": "ct-close", "symbol": "XBTUSDTM", "type": "market", "closeOrder": true}
	assert.Equal(t, exp, body, "Payload should leave out what a close order does not set")
}

func TestAddCopyTradingOrderTest(t *testing.T) {
	t.Parallel()
	_, err := e.AddCopyTradingOrderTest(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "AddCopyTradingOrderTest must reject a nil request")
	_, err = e.AddCopyTradingOrderTest(t.Context(), &CopyTradingOrderRequest{ClientOrderID: "ct-1", Symbol: "XBTUSDTM", Type: "market", Side: "buy", Size: 1})
	require.ErrorIs(t, err, errInvalidLeverage, "AddCopyTradingOrderTest must reject an order without leverage")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name string
		req  *CopyTradingOrderRequest
		exp  *CopyTradingOrderResponse
	}{
		{
			name: "limit",
			req: &CopyTradingOrderRequest{
				ClientOrderID: "5c52e11203aa677f33e493fb", Symbol: "XBTUSDTM", MarginMode: "CROSS", Leverage: 12,
				PositionSide: "LONG", Side: "buy", Type: "limit", Size: 1, Price: 0.1, TimeInForce: "GTC",
			},
			exp: &CopyTradingOrderResponse{OrderID: "234125150956625920", ClientOrderID: "5c52e11203aa677f33e493fb"},
		},
		{
			name: "close order without leverage",
			req:  &CopyTradingOrderRequest{ClientOrderID: "5c52e11203aa677f33e4940c", Symbol: "XBTUSDTM", Type: "market", CloseOrder: true},
			exp:  &CopyTradingOrderResponse{OrderID: "234125150956625921", ClientOrderID: "5c52e11203aa677f33e4940c"},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.AddCopyTradingOrderTest(t.Context(), tc.req)
			require.NoError(t, err, "AddCopyTradingOrderTest must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "AddCopyTradingOrderTest should decode every field")
				return
			}
			assert.NotEmpty(t, result.OrderID, "AddCopyTradingOrderTest should return an order ID")
		})
	}
}

func TestAddCopyTradingTPSLOrder(t *testing.T) {
	t.Parallel()
	_, err := e.AddCopyTradingTPSLOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "AddCopyTradingTPSLOrder must reject a nil request")
	_, err = e.AddCopyTradingTPSLOrder(t.Context(), &CopyTradingTPSLOrderRequest{})
	require.ErrorIs(t, err, order.ErrClientOrderIDMustBeSet, "AddCopyTradingTPSLOrder must validate the request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.AddCopyTradingTPSLOrder(t.Context(), &CopyTradingTPSLOrderRequest{
		ClientOrderID: "fa928420a7a0411cb50835e14d123943", Symbol: "XBTUSDTM", MarginMode: "CROSS", Leverage: 12,
		PositionSide: "LONG", Side: "buy", Type: "market", Size: 1, StopPriceType: "TP", TriggerStopUpPrice: 111000,
		TriggerStopDownPrice: 100000, TimeInForce: "GTC", FundsType: "selfFunds",
	})
	require.NoError(t, err, "AddCopyTradingTPSLOrder must not error")
	if mockTests {
		exp := &CopyTradingOrderResponse{OrderID: "368518618981363712", ClientOrderID: "fa928420a7a0411cb50835e14d123943"}
		assert.Equal(t, exp, result, "AddCopyTradingTPSLOrder should decode every field")
		return
	}
	assert.NotEmpty(t, result.OrderID, "AddCopyTradingTPSLOrder should return the order ID")
}

func TestCopyTradingTPSLOrderRequestPayload(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *CopyTradingTPSLOrderRequest
		err  error
	}{
		{name: "nil", err: common.ErrNilPointer},
		{name: "no client order ID", req: &CopyTradingTPSLOrderRequest{}, err: order.ErrClientOrderIDMustBeSet},
		{name: "no symbol", req: &CopyTradingTPSLOrderRequest{ClientOrderID: "ct-1"}, err: currency.ErrSymbolStringEmpty},
		{name: "no type", req: &CopyTradingTPSLOrderRequest{ClientOrderID: "ct-1", Symbol: "XBTUSDTM"}, err: order.ErrTypeIsInvalid},
		{name: "no side", req: &CopyTradingTPSLOrderRequest{ClientOrderID: "ct-1", Symbol: "XBTUSDTM", Type: "market"}, err: order.ErrSideIsInvalid},
		{name: "no size", req: &CopyTradingTPSLOrderRequest{ClientOrderID: "ct-1", Symbol: "XBTUSDTM", Type: "market", Side: "buy"}, err: order.ErrAmountIsInvalid},
		{name: "no leverage", req: &CopyTradingTPSLOrderRequest{ClientOrderID: "ct-1", Symbol: "XBTUSDTM", Type: "market", Side: "buy", Size: 1}, err: errInvalidLeverage},
		{name: "limit without a price", req: &CopyTradingTPSLOrderRequest{ClientOrderID: "ct-1", Symbol: "XBTUSDTM", Type: "limit", Side: "buy", Size: 1, Leverage: 5}, err: order.ErrPriceMustBeSetIfLimitOrder},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := tc.req.Payload()
			assert.ErrorIs(t, err, tc.err, "Payload should return the expected error")
		})
	}

	r := &CopyTradingTPSLOrderRequest{
		ClientOrderID: "fa928420a7a0411cb50835e14d123943", Side: "buy", Symbol: "XBTUSDTM", Leverage: 12, Type: "limit",
		StopPriceType: "MP", ReduceOnly: true, CloseOrder: true, MarginMode: "CROSS", Price: 108000.5, Size: 1,
		TimeInForce: "GTC", PostOnly: true, TriggerStopUpPrice: 111000, TriggerStopDownPrice: 100000, PositionSide: "LONG",
		FundsType: "leadTrial",
	}
	exp := map[string]any{
		"clientOid": "fa928420a7a0411cb50835e14d123943", "side": "buy", "symbol": "XBTUSDTM", "leverage": uint64(12),
		"type": "limit", "stopPriceType": "MP", "reduceOnly": true, "closeOrder": true, "marginMode": "CROSS",
		"price": "108000.5", "size": uint64(1), "timeInForce": "GTC", "postOnly": true, "triggerStopUpPrice": "111000",
		"triggerStopDownPrice": "100000", "positionSide": "LONG", "fundsType": "leadTrial",
	}
	body, err := r.Payload()
	require.NoError(t, err, "Payload must not error")
	assert.Equal(t, exp, body, "Payload should send every documented parameter")

	body, err = (&CopyTradingTPSLOrderRequest{ClientOrderID: "ct-close", Symbol: "XBTUSDTM", Type: "market", CloseOrder: true}).Payload()
	require.NoError(t, err, "Payload must not error for a close order without a side, size or leverage")
	exp = map[string]any{"clientOid": "ct-close", "symbol": "XBTUSDTM", "type": "market", "closeOrder": true}
	assert.Equal(t, exp, body, "Payload should leave out what a close order does not set")
}

func TestCancelCopyTradingOrderByID(t *testing.T) {
	t.Parallel()
	_, err := e.CancelCopyTradingOrderByID(t.Context(), "", "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelCopyTradingOrderByID must reject an empty order ID")

	if !mockTests {
		t.Skip("the order ID is the docs' example, and cancelling one of the account's real orders is not safe in a test")
	}
	result, err := e.CancelCopyTradingOrderByID(t.Context(), "263485113055133696", "selfFunds")
	require.NoError(t, err, "CancelCopyTradingOrderByID must not error")
	exp := &CancelCopyTradingOrderByIDResponse{CancelledOrderIDs: []string{"263485113055133696"}}
	assert.Equal(t, exp, result, "CancelCopyTradingOrderByID should decode every field")
}

func TestCancelCopyTradingOrderByClientOrderID(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *CancelCopyTradingOrderByClientOrderIDRequest
		err  error
	}{
		{name: "nil", err: common.ErrNilPointer},
		{name: "no symbol", req: &CancelCopyTradingOrderByClientOrderIDRequest{ClientOrderID: "ct-1"}, err: currency.ErrSymbolStringEmpty},
		{name: "no client order ID", req: &CancelCopyTradingOrderByClientOrderIDRequest{Symbol: "XBTUSDTM"}, err: order.ErrClientOrderIDMustBeSet},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.CancelCopyTradingOrderByClientOrderID(t.Context(), tc.req)
			assert.ErrorIs(t, err, tc.err, "CancelCopyTradingOrderByClientOrderID should reject the request")
		})
	}

	if !mockTests {
		t.Skip("the client order ID is the docs' example, and cancelling one of the account's real orders is not safe in a test")
	}
	result, err := e.CancelCopyTradingOrderByClientOrderID(t.Context(), &CancelCopyTradingOrderByClientOrderIDRequest{Symbol: "XBTUSDTM", ClientOrderID: "5c52e11203aa677f331e4913fb", FundsType: "selfFunds"})
	require.NoError(t, err, "CancelCopyTradingOrderByClientOrderID must not error")
	exp := &CancelCopyTradingOrderByClientOrderIDResponse{ClientOrderID: "5c52e11203aa677f331e4913fb"}
	assert.Equal(t, exp, result, "CancelCopyTradingOrderByClientOrderID should decode every field")
}

func TestGetCopyTradingMaxOpenSize(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *CopyTradingMaxOpenSizeRequest
		err  error
	}{
		{name: "nil", err: common.ErrNilPointer},
		{name: "no symbol", req: &CopyTradingMaxOpenSizeRequest{Price: 108000, Leverage: 10}, err: currency.ErrSymbolStringEmpty},
		{name: "no price", req: &CopyTradingMaxOpenSizeRequest{Symbol: "XBTUSDTM", Leverage: 10}, err: limits.ErrPriceBelowMin},
		{name: "no leverage", req: &CopyTradingMaxOpenSizeRequest{Symbol: "XBTUSDTM", Price: 108000}, err: errInvalidLeverage},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.GetCopyTradingMaxOpenSize(t.Context(), tc.req)
			assert.ErrorIs(t, err, tc.err, "GetCopyTradingMaxOpenSize should reject the request")
		})
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCopyTradingMaxOpenSize(t.Context(), &CopyTradingMaxOpenSizeRequest{Symbol: "XBTUSDTM", Price: 108000, Leverage: 10, FundsType: "selfFunds"})
	require.NoError(t, err, "GetCopyTradingMaxOpenSize must not error")
	if mockTests {
		exp := &CopyTradingMaxOpenSizeResponse{Symbol: "XBTUSDTM", MaxBuyOpenSize: 1000000, MaxSellOpenSize: 51}
		assert.Equal(t, exp, result, "GetCopyTradingMaxOpenSize should decode every field")
		return
	}
	assert.Equal(t, "XBTUSDTM", result.Symbol, "GetCopyTradingMaxOpenSize should return the symbol it was asked for")
}

func TestGetCopyTradingMaxWithdrawMargin(t *testing.T) {
	t.Parallel()
	_, err := e.GetCopyTradingMaxWithdrawMargin(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetCopyTradingMaxWithdrawMargin must reject a nil request")
	_, err = e.GetCopyTradingMaxWithdrawMargin(t.Context(), &CopyTradingMaxWithdrawMarginRequest{})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "GetCopyTradingMaxWithdrawMargin must reject an empty symbol")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	cancelled, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = e.GetCopyTradingMaxWithdrawMargin(cancelled, &CopyTradingMaxWithdrawMarginRequest{Symbol: "XBTUSDTM"})
	require.ErrorIs(t, err, context.Canceled, "GetCopyTradingMaxWithdrawMargin must return the request's error")

	result, err := e.GetCopyTradingMaxWithdrawMargin(t.Context(), &CopyTradingMaxWithdrawMarginRequest{Symbol: "XBTUSDTM", PositionSide: "LONG", FundsType: "selfFunds"})
	require.NoError(t, err, "GetCopyTradingMaxWithdrawMargin must not error")
	if mockTests {
		assert.Equal(t, 65.4982871262, result, "GetCopyTradingMaxWithdrawMargin should return the margin KuCoin sent")
		return
	}
	assert.GreaterOrEqual(t, result, 0.0, "GetCopyTradingMaxWithdrawMargin should not return a negative margin")
}

func TestAddCopyTradingIsolatedMargin(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *AddCopyTradingIsolatedMarginRequest
		err  error
	}{
		{name: "nil", err: common.ErrNilPointer},
		{name: "no symbol", req: &AddCopyTradingIsolatedMarginRequest{Margin: 3, BusinessNumber: "112233"}, err: currency.ErrSymbolStringEmpty},
		{name: "no margin", req: &AddCopyTradingIsolatedMarginRequest{Symbol: "XBTUSDTM", BusinessNumber: "112233"}, err: limits.ErrAmountBelowMin},
		{name: "no business number", req: &AddCopyTradingIsolatedMarginRequest{Symbol: "XBTUSDTM", Margin: 3}, err: errBusinessNumberRequired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.AddCopyTradingIsolatedMargin(t.Context(), tc.req)
			assert.ErrorIs(t, err, tc.err, "AddCopyTradingIsolatedMargin should reject the request")
		})
	}

	sent := sentRequest(t, func(ex *Exchange) error {
		_, err := ex.AddCopyTradingIsolatedMargin(t.Context(), &AddCopyTradingIsolatedMarginRequest{Symbol: "XBTUSDTM", Margin: 3.5, BusinessNumber: "112233", PositionSide: "LONG", FundsType: "selfFunds"})
		return err
	})
	assert.JSONEq(t, `{"symbol":"XBTUSDTM","margin":3.5,"bizNo":"112233","positionSide":"LONG","fundsType":"selfFunds"}`, sent.body, "AddCopyTradingIsolatedMargin should send the margin as the number KuCoin documents")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.AddCopyTradingIsolatedMargin(t.Context(), &AddCopyTradingIsolatedMarginRequest{Symbol: "XBTUSDTM", Margin: 3, BusinessNumber: "112233", PositionSide: "LONG", FundsType: "selfFunds"})
	require.NoError(t, err, "AddCopyTradingIsolatedMargin must not error")
	if mockTests {
		exp := &AddCopyTradingIsolatedMarginResponse{
			ID:                           "400000000001993893",
			Symbol:                       "XBTUSDTM",
			AutoDeposit:                  true,
			MaintenanceMarginRequirement: 0.004,
			RiskLimit:                    100000,
			RealLeverage:                 7.2,
			CrossMode:                    true,
			MarginMode:                   "ISOLATED",
			PositionSide:                 "LONG",
			Leverage:                     7.2,
			DeleveragePercentage:         0.25,
			OpeningTimestamp:             types.Time(time.UnixMilli(1760667597842)),
			CurrentTimestamp:             types.Time(time.UnixMilli(1760667659286)),
			CurrentQuantity:              1,
			CurrentCost:                  108.4412,
			CurrentCommission:            0.06506472,
			UnrealisedCost:               108.4412,
			RealisedGrossCost:            -0.5,
			RealisedCost:                 -0.43493528,
			IsOpen:                       true,
			MarkPrice:                    108459.69,
			MarkValue:                    108.45969,
			PositionCost:                 108.4412,
			PositionCross:                6,
			PositionInitial:              9.036766663,
			PositionCommission:           0.0563,
			PositionLoss:                 0.0012,
			PositionMargin:               15.091866663,
			PositionMaintenance:          0.498914574,
			MaintenanceMargin:            15.055256663,
			RealisedGrossPNL:             0.5,
			RealisedPNL:                  0.43493528,
			UnrealisedPNL:                0.01849,
			UnrealisedPNLPercent:         0.0002,
			UnrealisedROEPercent:         0.002,
			AverageEntryPrice:            108441.2,
			LiquidationPrice:             93836.08,
			BankruptPrice:                93404.44,
			SettleCurrency:               currency.USDT,
		}
		assert.Equal(t, exp, result, "AddCopyTradingIsolatedMargin should decode every field")
		return
	}
	require.NotNil(t, result, "AddCopyTradingIsolatedMargin must return a result")
	assert.Equal(t, "XBTUSDTM", result.Symbol, "AddCopyTradingIsolatedMargin should return the position it added margin to")
}

func TestRemoveCopyTradingIsolatedMargin(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *RemoveCopyTradingIsolatedMarginRequest
		err  error
	}{
		{name: "nil", err: common.ErrNilPointer},
		{name: "no symbol", req: &RemoveCopyTradingIsolatedMarginRequest{WithdrawAmount: 0.1}, err: currency.ErrSymbolStringEmpty},
		{name: "no withdraw amount", req: &RemoveCopyTradingIsolatedMarginRequest{Symbol: "XBTUSDTM"}, err: limits.ErrAmountBelowMin},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.RemoveCopyTradingIsolatedMargin(t.Context(), tc.req)
			assert.ErrorIs(t, err, tc.err, "RemoveCopyTradingIsolatedMargin should reject the request")
		})
	}

	sent := sentRequest(t, func(ex *Exchange) error {
		_, err := ex.RemoveCopyTradingIsolatedMargin(t.Context(), &RemoveCopyTradingIsolatedMarginRequest{Symbol: "XBTUSDTM", WithdrawAmount: 0.1, PositionSide: "LONG", FundsType: "selfFunds"})
		return err
	})
	assert.JSONEq(t, `{"symbol":"XBTUSDTM","withdrawAmount":0.1,"positionSide":"LONG","fundsType":"selfFunds"}`, sent.body, "RemoveCopyTradingIsolatedMargin should send the amount as the number KuCoin documents")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	cancelled, cancel := context.WithCancel(t.Context())
	cancel()
	_, err := e.RemoveCopyTradingIsolatedMargin(cancelled, &RemoveCopyTradingIsolatedMarginRequest{Symbol: "XBTUSDTM", WithdrawAmount: 0.1})
	require.ErrorIs(t, err, context.Canceled, "RemoveCopyTradingIsolatedMargin must return the request's error")

	result, err := e.RemoveCopyTradingIsolatedMargin(t.Context(), &RemoveCopyTradingIsolatedMarginRequest{Symbol: "XBTUSDTM", WithdrawAmount: 0.1, PositionSide: "LONG", FundsType: "selfFunds"})
	require.NoError(t, err, "RemoveCopyTradingIsolatedMargin must not error")
	assert.Equal(t, 0.1, result, "RemoveCopyTradingIsolatedMargin should return the amount withdrawn")
}

func TestModifyCopyTradingIsolatedMarginRiskLimit(t *testing.T) {
	t.Parallel()
	_, err := e.ModifyCopyTradingIsolatedMarginRiskLimit(t.Context(), "", 1)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "ModifyCopyTradingIsolatedMarginRiskLimit must reject an empty symbol")
	_, err = e.ModifyCopyTradingIsolatedMarginRiskLimit(t.Context(), "XBTUSDTM", 0)
	require.ErrorIs(t, err, errRiskLimitLevelRequired, "ModifyCopyTradingIsolatedMarginRiskLimit must reject a zero level")

	sent := sentRequest(t, func(ex *Exchange) error {
		_, err := ex.ModifyCopyTradingIsolatedMarginRiskLimit(t.Context(), "XBTUSDTM", 2)
		return err
	})
	assert.JSONEq(t, `{"symbol":"XBTUSDTM","level":2}`, sent.body, "ModifyCopyTradingIsolatedMarginRiskLimit should send the level as the integer KuCoin documents")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.ModifyCopyTradingIsolatedMarginRiskLimit(t.Context(), "XBTUSDTM", 1)
	require.NoError(t, err, "ModifyCopyTradingIsolatedMarginRiskLimit must not error")
	assert.True(t, result, "ModifyCopyTradingIsolatedMarginRiskLimit should report the request was accepted")
}

func TestModifyCopyTradingAutoDepositStatus(t *testing.T) {
	t.Parallel()
	_, err := e.ModifyCopyTradingAutoDepositStatus(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "ModifyCopyTradingAutoDepositStatus must reject a nil request")
	_, err = e.ModifyCopyTradingAutoDepositStatus(t.Context(), &CopyTradingAutoDepositStatusRequest{Status: true})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "ModifyCopyTradingAutoDepositStatus must reject an empty symbol")

	sent := sentRequest(t, func(ex *Exchange) error {
		_, err := ex.ModifyCopyTradingAutoDepositStatus(t.Context(), &CopyTradingAutoDepositStatusRequest{Symbol: "XBTUSDTM", PositionSide: "LONG", FundsType: "selfFunds"})
		return err
	})
	assert.JSONEq(t, `{"symbol":"XBTUSDTM","status":false,"positionSide":"LONG","fundsType":"selfFunds"}`, sent.body, "ModifyCopyTradingAutoDepositStatus should send the status as the boolean KuCoin documents, false included")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.ModifyCopyTradingAutoDepositStatus(t.Context(), &CopyTradingAutoDepositStatusRequest{Symbol: "XBTUSDTM", Status: true, PositionSide: "LONG", FundsType: "selfFunds"})
	require.NoError(t, err, "ModifyCopyTradingAutoDepositStatus must not error")
	assert.True(t, result, "ModifyCopyTradingAutoDepositStatus should report the change succeeded")
}

func TestSwitchCopyTradingMarginMode(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *SwitchCopyTradingMarginModeRequest
		err  error
	}{
		{name: "nil", err: common.ErrNilPointer},
		{name: "no symbol", req: &SwitchCopyTradingMarginModeRequest{MarginMode: "CROSS"}, err: currency.ErrSymbolStringEmpty},
		{name: "no margin mode", req: &SwitchCopyTradingMarginModeRequest{Symbol: "XBTUSDTM"}, err: errMarginModeRequired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.SwitchCopyTradingMarginMode(t.Context(), tc.req)
			assert.ErrorIs(t, err, tc.err, "SwitchCopyTradingMarginMode should reject the request")
		})
	}

	sent := sentRequest(t, func(ex *Exchange) error {
		_, err := ex.SwitchCopyTradingMarginMode(t.Context(), &SwitchCopyTradingMarginModeRequest{Symbol: "XBTUSDTM", MarginMode: "CROSS", FundsType: "selfFunds"})
		return err
	})
	assert.JSONEq(t, `{"symbol":"XBTUSDTM","marginMode":"CROSS","fundsType":"selfFunds"}`, sent.body, "SwitchCopyTradingMarginMode should send the documented strings")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.SwitchCopyTradingMarginMode(t.Context(), &SwitchCopyTradingMarginModeRequest{Symbol: "XBTUSDTM", MarginMode: "CROSS", FundsType: "selfFunds"})
	require.NoError(t, err, "SwitchCopyTradingMarginMode must not error")
	exp := &SwitchCopyTradingMarginModeResponse{Symbol: "XBTUSDTM", MarginMode: "CROSS"}
	assert.Equal(t, exp, result, "SwitchCopyTradingMarginMode should decode every field")
}

func TestModifyCopyTradingCrossMarginLeverage(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *CopyTradingCrossMarginLeverageRequest
		err  error
	}{
		{name: "nil", err: common.ErrNilPointer},
		{name: "no symbol", req: &CopyTradingCrossMarginLeverageRequest{Leverage: 15}, err: currency.ErrSymbolStringEmpty},
		{name: "no leverage", req: &CopyTradingCrossMarginLeverageRequest{Symbol: "XBTUSDTM"}, err: errInvalidLeverage},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.ModifyCopyTradingCrossMarginLeverage(t.Context(), tc.req)
			assert.ErrorIs(t, err, tc.err, "ModifyCopyTradingCrossMarginLeverage should reject the request")
		})
	}

	sent := sentRequest(t, func(ex *Exchange) error {
		_, err := ex.ModifyCopyTradingCrossMarginLeverage(t.Context(), &CopyTradingCrossMarginLeverageRequest{Symbol: "XBTUSDTM", Leverage: 15, FundsType: "selfFunds"})
		return err
	})
	assert.JSONEq(t, `{"symbol":"XBTUSDTM","leverage":"15","fundsType":"selfFunds"}`, sent.body, "ModifyCopyTradingCrossMarginLeverage should send the leverage as the string KuCoin documents")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.ModifyCopyTradingCrossMarginLeverage(t.Context(), &CopyTradingCrossMarginLeverageRequest{Symbol: "XBTUSDTM", Leverage: 15, FundsType: "selfFunds"})
	require.NoError(t, err, "ModifyCopyTradingCrossMarginLeverage must not error")
	assert.True(t, result, "ModifyCopyTradingCrossMarginLeverage should report the change succeeded")
}

func TestGetCopyTradingCrossMarginRequirement(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *CopyTradingCrossMarginRequirementRequest
		err  error
	}{
		{name: "nil", err: common.ErrNilPointer},
		{name: "no symbols", req: &CopyTradingCrossMarginRequirementRequest{Leverage: 15, PositionValue: 10000}, err: currency.ErrSymbolStringEmpty},
		{name: "an empty symbol", req: &CopyTradingCrossMarginRequirementRequest{Symbols: []string{"XBTUSDTM", ""}, Leverage: 15, PositionValue: 10000}, err: currency.ErrSymbolStringEmpty},
		{name: "no leverage", req: &CopyTradingCrossMarginRequirementRequest{Symbols: []string{"XBTUSDTM"}, PositionValue: 10000}, err: errInvalidLeverage},
		{name: "no position value", req: &CopyTradingCrossMarginRequirementRequest{Symbols: []string{"XBTUSDTM"}, Leverage: 15}, err: errPositionValueRequired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.GetCopyTradingCrossMarginRequirement(t.Context(), tc.req)
			assert.ErrorIs(t, err, tc.err, "GetCopyTradingCrossMarginRequirement should reject the request")
		})
	}

	sent := sentRequest(t, func(ex *Exchange) error {
		_, err := ex.GetCopyTradingCrossMarginRequirement(t.Context(), &CopyTradingCrossMarginRequirementRequest{Symbols: []string{"XBTUSDTM", "ETHUSDTM"}, Leverage: 15, PositionValue: 10000.5})
		return err
	})
	assert.JSONEq(t, `{"symbol":"XBTUSDTM,ETHUSDTM","leverage":"15","positionValue":"10000.5"}`, sent.body, "GetCopyTradingCrossMarginRequirement should send the leverage and position value as the strings KuCoin documents")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCopyTradingCrossMarginRequirement(t.Context(), &CopyTradingCrossMarginRequirementRequest{Symbols: []string{"XBTUSDTM", "ETHUSDTM"}, Leverage: 15, PositionValue: 10000})
	require.NoError(t, err, "GetCopyTradingCrossMarginRequirement must not error")
	if mockTests {
		exp := []CopyTradingCrossMarginRequirementResponse{
			{Symbol: "XBTUSDTM", Size: 87, PositionValue: 10000, Price: 113876.3, InitialMarginRate: 0.06666667, MaintenanceMarginRate: 0.00400117},
			{Symbol: "ETHUSDTM", Size: 249, PositionValue: 10000, Price: 4012.35, InitialMarginRate: 0.06666667, MaintenanceMarginRate: 0.005},
		}
		assert.Equal(t, exp, result, "GetCopyTradingCrossMarginRequirement should decode every field")
		return
	}
	assert.Len(t, result, 2, "GetCopyTradingCrossMarginRequirement should return a requirement per symbol")
}

func TestSwitchCopyTradingPositionMode(t *testing.T) {
	t.Parallel()
	_, err := e.SwitchCopyTradingPositionMode(t.Context(), "", "")
	require.ErrorIs(t, err, errPositionModeRequired, "SwitchCopyTradingPositionMode must reject an empty position mode")

	sent := sentRequest(t, func(ex *Exchange) error {
		_, err := ex.SwitchCopyTradingPositionMode(t.Context(), "1", "selfFunds")
		return err
	})
	assert.JSONEq(t, `{"positionMode":"1","fundsType":"selfFunds"}`, sent.body, "SwitchCopyTradingPositionMode should send the position mode as the string KuCoin documents")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.SwitchCopyTradingPositionMode(t.Context(), "1", "selfFunds")
	require.NoError(t, err, "SwitchCopyTradingPositionMode must not error")
	assert.Equal(t, &SwitchCopyTradingPositionModeResponse{PositionMode: "1"}, result, "SwitchCopyTradingPositionMode should decode every field")
}

func TestCopyTradingPageRequestValues(t *testing.T) {
	t.Parallel()
	_, err := (*CopyTradingPageRequest)(nil).values()
	require.ErrorIs(t, err, common.ErrNilPointer, "values must reject a nil request")

	params, err := (&CopyTradingPageRequest{}).values()
	require.NoError(t, err, "values must not error for an empty request")
	assert.Empty(t, params, "values should leave out unset parameters")

	params, err = (&CopyTradingPageRequest{FundsType: "leadTrial", CurrentPage: 2, PageSize: 50}).values()
	require.NoError(t, err, "values must not error")
	assert.Equal(t, "currentPage=2&fundsType=leadTrial&pageSize=50", params.Encode(), "values should send every parameter")
}

func TestGetCopyTradingActiveOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetCopyTradingActiveOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetCopyTradingActiveOrders must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCopyTradingActiveOrders(t.Context(), &CopyTradingPageRequest{FundsType: "all", CurrentPage: 1, PageSize: 20})
	require.NoError(t, err, "GetCopyTradingActiveOrders must not error")
	if mockTests {
		exp := &CopyTradingOrdersResponse{
			CurrentPage: 1,
			PageSize:    20,
			TotalNumber: 3,
			TotalPage:   1,
			Items: []CopyTradingOrder{
				{
					ID:             "497570415959482368",
					Symbol:         "XBTUSDTM",
					Type:           "limit",
					Side:           "sell",
					Price:          85000,
					Size:           1,
					Value:          85,
					TimeInForce:    "GTC",
					PostOnly:       true,
					Leverage:       25,
					CloseOrder:     true,
					ReduceOnly:     true,
					CancelAfter:    -1,
					Channel:        "COPY_TRADING",
					ClientOrderID:  "6ac74121329b88000718d02b",
					Remark:         "order_remarks",
					Tags:           "copy_trading",
					CreatedAt:      types.Time(time.UnixMilli(1791443233602)),
					IsActive:       true,
					SettleCurrency: currency.USDT,
					TriggerOrderID: "0",
					FromReduce:     true,
					MarginMode:     "CROSS",
					DisplayType:    "limit",
					PositionSide:   "BOTH",
				},
				{
					ID:                  "497571002316394496",
					Symbol:              "XBTUSDTM",
					Type:                "limit",
					Side:                "buy",
					Price:               80000,
					Size:                3,
					Value:               240,
					DealValue:           80,
					DealSize:            1,
					SelfTradePrevention: "CN",
					Stop:                "down",
					StopPriceType:       "TP",
					StopTriggered:       true,
					StopPrice:           80500,
					TimeInForce:         "GTC",
					Iceberg:             true,
					Leverage:            10,
					ForceHold:           true,
					VisibleSize:         1,
					CancelAfter:         -1,
					Channel:             "COPY_TRADING",
					ClientOrderID:       "6ac7414c329b88000718d0a1",
					Tags:                "copy_trading",
					CancelExist:         true,
					CreatedAt:           types.Time(time.UnixMilli(1791443372113)),
					IsActive:            true,
					SettleCurrency:      currency.USDT,
					TriggerOrderID:      "0",
					MarginMode:          "ISOLATED",
					DisplayType:         "limit",
					PositionSide:        "BOTH",
				},
				{
					ID:                   "497571587161759744",
					Symbol:               "ETHUSDTM",
					Type:                 "limit",
					Side:                 "sell",
					Price:                4100,
					Size:                 1,
					Value:                41,
					StopPriceType:        "MP",
					TimeInForce:          "GTC",
					Hidden:               true,
					Leverage:             5,
					CancelAfter:          3600,
					Channel:              "COPY_TRADING",
					ClientOrderID:        "6ac7417a329b88000718d1f3",
					Tags:                 "copy_trading",
					CreatedAt:            types.Time(time.UnixMilli(1791443511486)),
					IsActive:             true,
					SettleCurrency:       currency.USDT,
					Shortcut:             true,
					TriggerStopUpPrice:   4200,
					TriggerStopDownPrice: 3850,
					TriggerOrderID:       "497571587161759700",
					IsTrialFunds:         true,
					MarginMode:           "ISOLATED",
					DisplayType:          "limit",
					AdvancedDisplayType:  "limit",
					PositionSide:         "LONG",
					AdvancedType:         "TRAIL",
					ActivationPrice:      4050,
					CallbackRate:         0.01,
					AdvancedStatus:       "ACTIVE",
					TrialCoupon: &CopyTradingTrialCoupon{
						ID:               10002,
						Code:             "TRIAL-002",
						Currency:         currency.USDT,
						FaceValue:        100,
						ExpiredPeriod:    types.Time(time.Unix(1798761600, 0)),
						WithdrawHelpText: "Profits can be withdrawn once the trading volume threshold is reached",
					},
				},
			},
		}
		assert.Equal(t, exp, result, "GetCopyTradingActiveOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "GetCopyTradingActiveOrders should return a page")
}

func TestGetCopyTradingFills(t *testing.T) {
	t.Parallel()
	_, err := e.GetCopyTradingFills(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetCopyTradingFills must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCopyTradingFills(t.Context(), &CopyTradingPageRequest{FundsType: "leadTrial", CurrentPage: 1, PageSize: 20})
	require.NoError(t, err, "GetCopyTradingFills must not error")
	if mockTests {
		exp := &CopyTradingFillsResponse{
			Items: []CopyTradingFill{
				{
					TradeID:      "1946343102217",
					OrderID:      "497570128993693696",
					Symbol:       "ETHUSDTM",
					Side:         "buy",
					PositionSide: "LONG",
					Price:        4015.5,
					Size:         2,
					Fee:          0.048186,
					FeeCurrency:  currency.USDT,
					Liquidity:    "taker",
					FundsType:    "leadTrial",
					TradeTime:    types.Time(time.UnixMilli(1791443164851)),
					IsTrialFunds: true,
				},
				{
					TradeID:      "1946343215861",
					OrderID:      "497570342218129408",
					Symbol:       "ETHUSDTM",
					Side:         "sell",
					PositionSide: "LONG",
					Price:        4030.2,
					Size:         1,
					Fee:          0.0080604,
					FeeCurrency:  currency.USDT,
					Liquidity:    "maker",
					FundsType:    "leadTrial",
					TradeTime:    types.Time(time.UnixMilli(1791443240117)),
					IsTrialFunds: true,
				},
			},
			Offset:      2,
			TrialOffset: 2,
			HasNext:     true,
		}
		assert.Equal(t, exp, result, "GetCopyTradingFills should decode every field")
		return
	}
	assert.NotNil(t, result, "GetCopyTradingFills should return a page")
}

func TestGetCopyTradingPositions(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCopyTradingPositions(t.Context(), "all")
	require.NoError(t, err, "GetCopyTradingPositions must not error")
	if mockTests {
		exp := []CopyTradingPositionResponse{
			{
				ID:                           "500000000002517039",
				Symbol:                       "XBTUSDTM",
				MaintenanceMarginRequirement: 0.0040000133,
				MarginMode:                   "CROSS",
				PositionSide:                 "BOTH",
				Leverage:                     25,
				DeleveragePercentage:         0.53,
				OpeningTimestamp:             types.Time(time.UnixMilli(1791443029243)),
				CurrentTimestamp:             types.Time(time.UnixMilli(1791443048230)),
				CurrentQuantity:              1,
				CurrentCost:                  82.6908,
				CurrentCommission:            0.04961448,
				UnrealisedCost:               82.6908,
				RealisedCost:                 0.04961448,
				IsOpen:                       true,
				MarkPrice:                    82709.29,
				MarkValue:                    82.70929,
				PositionCost:                 82.6908,
				PositionInitial:              3.307632,
				PositionMargin:               3.307632,
				PositionMaintenance:          0.3307642998,
				RealisedPNL:                  -0.04961448,
				UnrealisedPNL:                0.01849,
				UnrealisedPNLPercent:         0.0002,
				UnrealisedROEPercent:         0.0056,
				AverageEntryPrice:            82690.8,
				LiquidationPrice:             76120.58,
				BankruptPrice:                75770.42,
				SettleCurrency:               currency.USDT,
				CrossMode:                    true,
			},
			{
				ID:                           "500000000002517102",
				Symbol:                       "ETHUSDTM",
				AutoDeposit:                  true,
				MaintenanceMarginRequirement: 0.005,
				RiskLimit:                    1,
				RealLeverage:                 4.98,
				MarginMode:                   "ISOLATED",
				PositionSide:                 "LONG",
				Leverage:                     5,
				DeleveragePercentage:         0.1,
				OpeningTimestamp:             types.Time(time.UnixMilli(1791443164851)),
				CurrentTimestamp:             types.Time(time.UnixMilli(1791443260117)),
				CurrentQuantity:              1,
				CurrentCost:                  40.155,
				CurrentCommission:            0.0562464,
				UnrealisedCost:               40.155,
				RealisedGrossCost:            -0.147,
				RealisedCost:                 -0.0907536,
				IsOpen:                       true,
				MarkPrice:                    4005.2,
				MarkValue:                    40.052,
				PositionCost:                 40.155,
				PositionCross:                1.5,
				PositionInitial:              8.031,
				PositionCommission:           0.0241,
				PositionLoss:                 0.0012,
				PositionMargin:               9.5539,
				PositionMaintenance:          0.2244,
				PositionProfit:               0.0034,
				PositionCommissionFunding:    0.0005,
				MaintenanceMargin:            9.4509,
				RealisedGrossPNL:             0.147,
				RealisedPNL:                  0.0907536,
				UnrealisedPNL:                -0.103,
				UnrealisedPNLPercent:         -0.0026,
				UnrealisedROEPercent:         -0.0128,
				AverageEntryPrice:            4015.5,
				LiquidationPrice:             3082.47,
				BankruptPrice:                3060.31,
				SettleCurrency:               currency.USDT,
				TrialCode:                    "TRIAL-002",
				IsTrialFunds:                 true,
			},
		}
		assert.Equal(t, exp, result, "GetCopyTradingPositions should decode every field")
		return
	}
	for i := range result {
		assert.NotEmptyf(t, result[i].Symbol, "GetCopyTradingPositions should return the symbol of position %d", i)
	}
}

func TestGetCopyTradingOrderHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetCopyTradingOrderHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetCopyTradingOrderHistory must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCopyTradingOrderHistory(t.Context(), &CopyTradingPageRequest{FundsType: "all", CurrentPage: 1, PageSize: 20})
	require.NoError(t, err, "GetCopyTradingOrderHistory must not error")
	if mockTests {
		exp := &CopyTradingOrdersResponse{
			CurrentPage: 1,
			PageSize:    20,
			TotalNumber: 3,
			TotalPage:   1,
			Items: []CopyTradingOrder{
				{
					ID:             "497569558740131840",
					Symbol:         "XBTUSDTM",
					Type:           "limit",
					Side:           "buy",
					Price:          82698.4,
					Size:           1,
					Value:          82.6984,
					DealValue:      82.6908,
					DealSize:       1,
					TimeInForce:    "GTC",
					Hidden:         true,
					Leverage:       25,
					CancelAfter:    -1,
					Channel:        "COPY_TRADING",
					ClientOrderID:  "6ac740555741d70007d85cc8",
					Tags:           "copy_trading",
					CreatedAt:      types.Time(time.UnixMilli(1791443029199)),
					SettleCurrency: currency.USDT,
					TriggerOrderID: "0",
					MarginMode:     "CROSS",
					DisplayType:    "limit",
					PositionSide:   "BOTH",
				},
				{
					ID:                  "497569901142138880",
					Symbol:              "XBTUSDTM",
					Type:                "limit",
					Side:                "sell",
					Price:               90000,
					Size:                2,
					Value:               180,
					DealValue:           90,
					DealSize:            1,
					SelfTradePrevention: "CO",
					Stop:                "up",
					StopPriceType:       "MP",
					StopTriggered:       true,
					StopPrice:           89500,
					TimeInForce:         "GTC",
					PostOnly:            true,
					Leverage:            25,
					ForceHold:           true,
					CloseOrder:          true,
					ReduceOnly:          true,
					CancelAfter:         60,
					Channel:             "COPY_TRADING",
					ClientOrderID:       "6ac7407b5741d70007d85d12",
					Remark:              "order_remarks",
					Tags:                "copy_trading",
					CancelExist:         true,
					CreatedAt:           types.Time(time.UnixMilli(1791443110525)),
					IsActive:            true,
					SettleCurrency:      currency.USDT,
					TriggerOrderID:      "0",
					FromReduce:          true,
					MarginMode:          "CROSS",
					DisplayType:         "limit",
					PositionSide:        "BOTH",
				},
				{
					ID:                   "497570128993693696",
					Symbol:               "ETHUSDTM",
					Type:                 "limit",
					Side:                 "buy",
					Price:                4015.5,
					Size:                 2,
					Value:                80.31,
					DealValue:            80.31,
					DealSize:             2,
					StopPriceType:        "TP",
					TimeInForce:          "GTC",
					Iceberg:              true,
					Leverage:             5,
					VisibleSize:          1,
					CancelAfter:          -1,
					Channel:              "COPY_TRADING",
					ClientOrderID:        "6ac740a95741d70007d85e40",
					Tags:                 "copy_trading",
					CreatedAt:            types.Time(time.UnixMilli(1791443164836)),
					SettleCurrency:       currency.USDT,
					Shortcut:             true,
					TriggerStopUpPrice:   4300,
					TriggerStopDownPrice: 3800,
					TriggerOrderID:       "497570128993693600",
					IsTrialFunds:         true,
					MarginMode:           "ISOLATED",
					DisplayType:          "limit",
					AdvancedDisplayType:  "limit",
					PositionSide:         "LONG",
					AdvancedType:         "TRAIL",
					ActivationPrice:      4050,
					CallbackRate:         0.02,
					AdvancedStatus:       "TRIGGERED",
					TrialCoupon: &CopyTradingTrialCoupon{
						ID:               10002,
						Code:             "TRIAL-002",
						Currency:         currency.USDT,
						FaceValue:        100,
						ExpiredPeriod:    types.Time(time.Unix(1798761600, 0)),
						WithdrawHelpText: "Profits can be withdrawn once the trading volume threshold is reached",
					},
				},
			},
		}
		assert.Equal(t, exp, result, "GetCopyTradingOrderHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetCopyTradingOrderHistory should return a page")
}

func TestGetCopyTradingAccountOverview(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCopyTradingAccountOverview(t.Context())
	require.NoError(t, err, "GetCopyTradingAccountOverview must not error")
	if mockTests {
		exp := &CopyTradingAccountOverviewResponse{
			Currency: currency.USDT,
			SelfFunds: CopyTradingFundsOverview{
				AccountEquity:    6.98849,
				AvailableBalance: 3.680858,
				MarginUsed:       3.307632,
				UnrealisedPNL:    0.01849,
			},
			LeadTrial: CopyTradingFundsOverview{
				AccountEquity:    100.044,
				AvailableBalance: 90.4901,
				MarginUsed:       9.5539,
				UnrealisedPNL:    -0.103,
			},
			TrialFund: &CopyTradingTrialFund{
				TrialFundID:             "10001",
				RewardID:                "20001",
				Amount:                  100,
				Status:                  "ACTIVE",
				ExpiresAt:               types.Time(time.Unix(1798761600, 0)),
				MaxLeverage:             20,
				SupportContracts:        []string{"XBTUSDTM", "ETHUSDTM"},
				WithdrawAllowed:         true,
				MaxWithdrawAmount:       20,
				WithdrawThresholdTrade:  1000,
				WithdrawThresholdProfit: 10,
			},
		}
		assert.Equal(t, exp, result, "GetCopyTradingAccountOverview should decode every field")
		return
	}
	assert.False(t, result.Currency.IsEmpty(), "GetCopyTradingAccountOverview should return the settlement currency")
}

func TestBatchSwitchCopyTradingMarginMode(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *BatchSwitchCopyTradingMarginModeRequest
		err  error
	}{
		{name: "nil", err: common.ErrNilPointer},
		{name: "no items", req: &BatchSwitchCopyTradingMarginModeRequest{FundsType: "selfFunds"}, err: errMarginModeChangesRequired},
		{name: "an item without a symbol", req: &BatchSwitchCopyTradingMarginModeRequest{Items: []CopyTradingMarginModeChange{{MarginMode: "CROSS"}}}, err: errMarginModeChangeIncomplete},
		{name: "an item without a margin mode", req: &BatchSwitchCopyTradingMarginModeRequest{Items: []CopyTradingMarginModeChange{{Symbol: "XBTUSDTM"}}}, err: errMarginModeChangeIncomplete},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.BatchSwitchCopyTradingMarginMode(t.Context(), tc.req)
			assert.ErrorIs(t, err, tc.err, "BatchSwitchCopyTradingMarginMode should reject the request")
		})
	}

	sent := sentRequest(t, func(ex *Exchange) error {
		_, err := ex.BatchSwitchCopyTradingMarginMode(t.Context(), &BatchSwitchCopyTradingMarginModeRequest{
			FundsType: "leadTrial",
			Items: []CopyTradingMarginModeChange{
				{Symbol: "ETHUSDTM", MarginMode: "CROSS"},
				{Symbol: "XRPUSDTM", MarginMode: "ISOLATED", FundsType: "leadTrial"},
			},
		})
		return err
	})
	assert.JSONEq(t, `{"items":[{"symbol":"ETHUSDTM","marginMode":"CROSS"},{"symbol":"XRPUSDTM","marginMode":"ISOLATED","fundsType":"leadTrial"}],"fundsType":"leadTrial"}`, sent.body, "BatchSwitchCopyTradingMarginMode should send the documented items")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.BatchSwitchCopyTradingMarginMode(t.Context(), &BatchSwitchCopyTradingMarginModeRequest{
		FundsType: "leadTrial",
		Items: []CopyTradingMarginModeChange{
			{Symbol: "ETHUSDTM", MarginMode: "CROSS"},
			{Symbol: "XRPUSDTM", MarginMode: "ISOLATED", FundsType: "leadTrial"},
		},
	})
	require.NoError(t, err, "BatchSwitchCopyTradingMarginMode must not error")
	if mockTests {
		exp := &BatchSwitchCopyTradingMarginModeResponse{
			FundsType: "leadTrial",
			Items: []CopyTradingMarginModeResult{
				{Symbol: "ETHUSDTM", RequestedMarginMode: "CROSS", MarginMode: "CROSS", Status: "SUCCESS"},
				{Symbol: "XRPUSDTM", RequestedMarginMode: "ISOLATED", MarginMode: "CROSS", Status: "FAILED", ErrorCode: "400100", ErrorMessage: "Invalid margin mode"},
			},
		}
		assert.Equal(t, exp, result, "BatchSwitchCopyTradingMarginMode should return each item's outcome")
		return
	}
	require.NotNil(t, result, "BatchSwitchCopyTradingMarginMode must return a result")
	assert.Len(t, result.Items, 2, "BatchSwitchCopyTradingMarginMode should return an outcome per item")
}
