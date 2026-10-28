package coinbase

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/fill"
	"github.com/thrasher-corp/gocryptotrader/exchanges/margin"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
)

var testSOLUSDCPerpetual = currency.NewPairWithDelimiter("SOL", "USDC-PERPETUAL", currency.UnderscoreDelimiter)

func TestWebsocketHandleDerivativesAccount(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	ex.Features.Enabled.FillsFeed = true
	ex.Websocket.Fills.Setup(true, ex.Websocket.DataHandler)
	conn, err := ex.Websocket.CreateTestConnection(derivativesPrivateConnection)
	require.NoError(t, err, "CreateTestConnection must not error")
	require.Empty(t, replayDerivativesFixture(t, ex, conn, "wsDerivativesAccount.jsonl"), "replaying the gateway's account fixture must not error")

	var gotOrders [][]order.Detail
	var gotFills [][]fill.Data
	relays := map[string]any{}
	for {
		var p any
		select {
		case payload := <-ex.Websocket.DataHandler.C:
			p = payload.Data
		default:
		}
		if p == nil {
			break
		}
		switch v := p.(type) {
		case []order.Detail:
			gotOrders = append(gotOrders, v)
		case []fill.Data:
			gotFills = append(gotFills, v)
		case *DerivativesAccountSummary:
			relays["portfolio"] = v
		case *DerivativesWebsocketUserChanges:
			relays["changes"] = v
		case *DerivativesAccessLogEntry:
			relays["access log"] = v
		case *DerivativesWebsocketUserLock:
			relays["lock"] = v
		case *DerivativesWebsocketPositionLock:
			relays["position lock"] = v
		case *DerivativesWebsocketIsolatedPortfolio:
			relays["isolated portfolio"] = v
		case *DerivativesWebsocketUserLiquidation:
			relays["isolated liquidation"] = v
		case *DerivativesWebsocketTrailingTriggerPrices:
			relays["trailing trigger prices"] = v
		}
	}

	expOrders := [][]order.Detail{
		{{
			Exchange: ex.Name, OrderID: "USDC-1234567", ClientOrderID: "gct-1", Pair: testUSDCPerpetual, AssetType: asset.USDCMarginedFutures,
			Side: order.Buy, Type: order.Limit, Status: order.Open, TimeInForce: order.GoodTillCancel | order.PostOnly, Price: 80000, Amount: 0.01,
			RemainingAmount: 0.01, MarginType: margin.Multi, Date: time.UnixMilli(1792374000000), LastUpdated: time.UnixMilli(1792374000000),
		}},
		{{
			Exchange: ex.Name, OrderID: "7654321", Pair: testCoinPerpetual, AssetType: asset.CoinMarginedFutures, Side: order.Sell, Type: order.Market,
			Status: order.PartiallyFilledCancelled, TimeInForce: order.ImmediateOrCancel, ReduceOnly: true, Price: 82000, Amount: 100,
			ExecutedAmount: 40, RemainingAmount: 60, AverageExecutedPrice: 82400.5, MarginType: margin.Multi, Date: time.UnixMilli(1792374001000),
			LastUpdated: time.UnixMilli(1792374002000),
		}},
		// An isolated subaccount's order arrives alone, and without its type as the docs show it
		{{
			Exchange: ex.Name, OrderID: "12345678", Pair: testSOLUSDCPerpetual, AssetType: asset.USDCMarginedFutures, Side: order.Buy,
			Status: order.Open, Price: 100, Amount: 10, RemainingAmount: 10, MarginType: margin.Isolated,
		}},
	}
	assert.Equal(t, expOrders, gotOrders, "the order notifications should be relayed as order details")
	expFills := [][]fill.Data{
		{{
			ID: "USDC-80635730", TradeID: "USDC-80635730", Timestamp: time.UnixMilli(1792374003000), Exchange: ex.Name, AssetType: asset.USDCMarginedFutures,
			CurrencyPair: testUSDCPerpetual, Side: order.Buy, OrderID: "USDC-1234567", ClientOrderID: "gct-1", Price: 80000, Amount: 0.004,
		}},
		{{
			ID: "T-998811", TradeID: "T-998811", Timestamp: time.UnixMilli(1787109600000), Exchange: ex.Name, AssetType: asset.USDCMarginedFutures,
			CurrencyPair: testSOLUSDCPerpetual, Side: order.Buy, OrderID: "12345678", Price: 100, Amount: 10,
		}},
	}
	assert.Equal(t, expFills, gotFills, "the trade notifications should be relayed as fills")

	trailing := DerivativesWebsocketTrailingTriggerPrices{"eth-usdt-buy-98765": 3450.75}
	expRelays := map[string]any{
		"portfolio": &DerivativesAccountSummary{
			Currency: currency.USDC, Balance: 2000, Equity: 2001.25, MarginBalance: 2001.25, AvailableFunds: 1960.5, AvailableWithdrawalFunds: 1950.25,
			InitialMargin: 40.75, MaintenanceMargin: 20.4, TotalProfitLoss: 1.25, SessionUnrealisedProfitLoss: 1.25, MarginModel: "cross_sm",
		},
		"changes": &DerivativesWebsocketUserChanges{
			InstrumentName: "BTC-PERPETUAL",
			Orders: []DerivativesOrderResponse{{
				OrderID: "7654321", OrderState: DerivativesOrderStateCancelled, OrderType: DerivativesOrderTypeMarket, TimeInForce: DerivativesTimeInForceImmediateOrCancel,
				InstrumentName: "BTC-PERPETUAL", CreationTimestamp: testMarketDataMilli(1792374001000), LastUpdateTimestamp: testMarketDataMilli(1792374002000),
				Direction: DerivativesDirectionSell, Price: DerivativesOrderPrice{Value: 82000}, ReduceOnly: true, Amount: 100, FilledAmount: 40, AveragePrice: 82400.5,
			}},
			Trades: []DerivativesUserTradeResponse{{
				TradeID: "453630200", TradeSequence: 307194900, InstrumentName: "BTC-PERPETUAL", Timestamp: testMarketDataMilli(1792374001500), OrderType: "market",
				OrderID: "7654321", Direction: DerivativesDirectionSell, IndexPrice: 82380.1, Price: 82400.5, MarkPrice: 82390.4, Amount: 40, Liquidity: "T",
				Fee: 0.00000024, FeeCurrency: currency.BTC, State: DerivativesOrderStateCancelled,
			}},
			Positions: []DerivativesPositionResponse{{
				InstrumentName: "BTC-PERPETUAL", Kind: DerivativesKindFuture, Direction: DerivativesDirectionSell, Size: -40, SizeCurrency: -0.000485,
				AveragePrice: 82400.5, MarkPrice: 82390.4, IndexPrice: 82380.1, Delta: -0.000485, InitialMargin: 0.0000097, MaintenanceMargin: 0.0000049,
				Leverage: 50, TotalProfitLoss: 0.00000006,
			}},
		},
		"access log":         &DerivativesAccessLogEntry{ID: 45, Timestamp: testMarketDataMilli(1792374004000), IP: "203.0.113.7", Country: "Australia", City: "Sydney", Log: "success"},
		"lock":               &DerivativesWebsocketUserLock{Currency: currency.USDC, Locked: true},
		"position lock":      &DerivativesWebsocketPositionLock{Locked: true},
		"isolated portfolio": &DerivativesWebsocketIsolatedPortfolio{Currency: currency.USDC, Equity: 50, MarginBalance: 50, AvailableFunds: 40, InitialMargin: 10, MaintenanceMargin: 5, MainUserID: 1000, UserID: 123456},
		"isolated liquidation": &DerivativesWebsocketUserLiquidation{
			State: "liquidation_started", UserID: 123456, Isolated: true, Currency: currency.USDC, MarginBalance: 50, MaintenanceMargin: 45,
			InitialMargin: 60, CloseOutMargin: 30, Equity: 50, Timestamp: testMarketDataMilli(1792374005000),
		},
		"trailing trigger prices": &trailing,
	}
	assert.Equal(t, expRelays, relays, "every other account notification should be relayed decoded")
}

func TestWebsocketHandleDerivativesAccountErrors(t *testing.T) {
	t.Parallel()
	ex := newWebsocketTestExchange(t)
	ex.Features.Enabled.FillsFeed = true
	ex.Websocket.Fills.Setup(true, ex.Websocket.DataHandler)
	conn, err := ex.Websocket.CreateTestConnection(derivativesPrivateConnection)
	require.NoError(t, err, "CreateTestConnection must not error")
	for name, tc := range map[string]struct {
		message string
		err     error
	}{
		"orders data":      {`{"method":"subscription","params":{"channel":"user.orders.BTC-PERPETUAL.100ms","data":"x"}}`, errDerivativesMalformedNotification},
		"order list":       {`{"method":"subscription","params":{"channel":"user.orders.BTC-PERPETUAL.100ms","data":[1]}}`, errDerivativesMalformedNotification},
		"order":            {`{"method":"subscription","params":{"channel":"user.orders.BTC-PERPETUAL.raw","data":{"amount":"x"}}}`, errDerivativesMalformedNotification},
		"order instrument": {`{"method":"subscription","params":{"channel":"user.orders.BTC-PERPETUAL.raw","data":{"instrument_name":"BTC-26DEC26"}}}`, errDerivativesUnknownInstrument},
		"order direction":  {`{"method":"subscription","params":{"channel":"user.orders.BTC-PERPETUAL.raw","data":{"instrument_name":"BTC-PERPETUAL","direction":"hold"}}}`, order.ErrSideIsInvalid},
		"order type":       {`{"method":"subscription","params":{"channel":"user.orders.BTC-PERPETUAL.raw","data":{"instrument_name":"BTC-PERPETUAL","direction":"buy","order_type":"iceberg"}}}`, order.ErrUnrecognisedOrderType},
		"order state":      {`{"method":"subscription","params":{"channel":"user.orders.BTC-PERPETUAL.raw","data":{"instrument_name":"BTC-PERPETUAL","direction":"buy","order_type":"limit","order_state":"lost"}}}`, errDerivativesUnknownOrderState},
		"order tif":        {`{"method":"subscription","params":{"channel":"user.orders.BTC-PERPETUAL.raw","data":{"instrument_name":"BTC-PERPETUAL","direction":"buy","order_type":"limit","order_state":"open","time_in_force":"forever"}}}`, order.ErrInvalidTimeInForce},
		"trades data":      {`{"method":"subscription","params":{"channel":"user.trades.BTC-PERPETUAL.100ms","data":{}}}`, errDerivativesMalformedNotification},
		"trade instrument": {`{"method":"subscription","params":{"channel":"user.trades.BTC-PERPETUAL.100ms","data":[{"instrument_name":"BTC-26DEC26"}]}}`, errDerivativesUnknownInstrument},
		"trade direction":  {`{"method":"subscription","params":{"channel":"user.trades.BTC-PERPETUAL.100ms","data":[{"instrument_name":"BTC-PERPETUAL","direction":"hold"}]}}`, order.ErrSideIsInvalid},
	} {
		assert.ErrorIsf(t, ex.wsHandleDerivatives(t.Context(), conn, []byte(tc.message)), tc.err, "wsHandleDerivatives should error for a bad %s", name)
	}
	assert.Empty(t, relayed[[]order.Detail](ex), "nothing should be relayed for orders that cannot be converted")

	ex.Features.Enabled.FillsFeed = false
	assert.NoError(t, ex.wsHandleDerivatives(t.Context(), conn, []byte(`{"method":"subscription","params":{"channel":"user.trades.BTC-PERPETUAL.100ms","data":[{"instrument_name":"BTC-26DEC26"}]}}`)), "trades should be ignored with the fills feed off")
}

func TestDerivativesOrderType(t *testing.T) {
	t.Parallel()
	for orderType, exp := range map[string]order.Type{
		"": order.UnknownType, DerivativesOrderTypeLimit: order.Limit, DerivativesOrderTypeMarket: order.Market, DerivativesOrderTypeMarketLimit: order.Market,
		DerivativesOrderTypeStopLimit: order.StopLimit, DerivativesOrderTypeStopMarket: order.StopMarket, DerivativesOrderTypeTakeLimit: order.TakeProfit,
		DerivativesOrderTypeTakeMarket: order.TakeProfitMarket, DerivativesOrderTypeTrailingStop: order.TrailingStop, "liquidation": order.Liquidation,
	} {
		got, err := derivativesOrderType(orderType)
		require.NoErrorf(t, err, "derivativesOrderType must not error for %q", orderType)
		assert.Equalf(t, exp, got, "derivativesOrderType should convert %q", orderType)
	}
	_, err := derivativesOrderType("iceberg")
	assert.ErrorIs(t, err, order.ErrUnrecognisedOrderType, "derivativesOrderType should error for an unknown order type")
}

func TestDerivativesOriginalOrderType(t *testing.T) {
	t.Parallel()
	ex := &Exchange{}
	for _, tc := range []struct {
		orderType, originalOrderType string
		exp                          order.Type
	}{
		{DerivativesOrderTypeLimit, "", order.Limit},
		{DerivativesOrderTypeLimit, DerivativesOrderTypeMarket, order.Market},
		{DerivativesOrderTypeLimit, DerivativesOrderTypeMarketLimit, order.Market},
	} {
		detail, err := ex.derivativesOrderDetail(&DerivativesOrderResponse{OrderID: "USDC-1", InstrumentName: "BTC_USDC-PERPETUAL", Direction: DerivativesDirectionBuy, OrderState: DerivativesOrderStateFilled, OrderType: tc.orderType, OriginalOrderType: tc.originalOrderType, Amount: 1, FilledAmount: 1})
		require.NoErrorf(t, err, "derivativesOrderDetail must not error for %s placed as %q", tc.orderType, tc.originalOrderType)
		assert.Equalf(t, tc.exp, detail.Type, "derivativesOrderDetail should report a %s order placed as %q as %s", tc.orderType, tc.originalOrderType, tc.exp)
		trade, err := ex.derivativesTradeHistory(&DerivativesUserTradeResponse{TradeID: "USDC-2", Direction: DerivativesDirectionBuy, OrderType: tc.orderType, OriginalOrderType: tc.originalOrderType})
		require.NoErrorf(t, err, "derivativesTradeHistory must not error for %s placed as %q", tc.orderType, tc.originalOrderType)
		assert.Equalf(t, tc.exp, trade.Type, "derivativesTradeHistory should report a trade of a %s order placed as %q as %s", tc.orderType, tc.originalOrderType, tc.exp)
	}
}

func TestDerivativesOrderStatus(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		state  string
		filled float64
		exp    order.Status
	}{
		{DerivativesOrderStateOpen, 0, order.Open},
		{DerivativesOrderStateOpen, 1, order.PartiallyFilled},
		{DerivativesOrderStateTriggered, 0, order.Closed},
		{DerivativesOrderStateTriggered, 1, order.Closed},
		{DerivativesOrderStateUntriggered, 0, order.Pending},
		{DerivativesOrderStateFilled, 1, order.Filled},
		{DerivativesOrderStateCancelled, 0, order.Cancelled},
		{DerivativesOrderStateCancelled, 1, order.PartiallyFilledCancelled},
		{DerivativesOrderStateRejected, 0, order.Rejected},
		{"archive", 1, order.Closed},
	} {
		got, err := derivativesOrderStatus(tc.state, tc.filled)
		require.NoErrorf(t, err, "derivativesOrderStatus must not error for %s", tc.state)
		assert.Equalf(t, tc.exp, got, "derivativesOrderStatus should convert %s with %v filled", tc.state, tc.filled)
	}
	_, err := derivativesOrderStatus("lost", 0)
	assert.ErrorIs(t, err, errDerivativesUnknownOrderState, "derivativesOrderStatus should error for an unknown state")
}
