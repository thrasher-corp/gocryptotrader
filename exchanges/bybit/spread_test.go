package bybit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// spreadTestSymbol is a perpetual and spot combination, which has no expiry
const spreadTestSymbol = "BTCUSDT_BTC/USDT"

var (
	spreadTestStart = time.UnixMilli(1790812800000)
	spreadTestEnd   = time.UnixMilli(1791331200000)
)

func TestGetSpreadInstrumentsInfo(t *testing.T) {
	t.Parallel()
	_, err := e.GetSpreadInstrumentsInfo(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetSpreadInstrumentsInfo must reject a nil request")

	result, err := e.GetSpreadInstrumentsInfo(t.Context(), &SpreadInstrumentsInfoRequest{BaseCoin: currency.BTC, Limit: 2})
	require.NoError(t, err, "GetSpreadInstrumentsInfo must not error")
	if mockTests {
		exp := &SpreadInstrumentsInfoResponse{
			List: []SpreadInstrument{
				{
					Symbol:       "BTCUSDT_BTC/USDT",
					ContractType: "FundingRateArb",
					Status:       "Trading",
					BaseCoin:     currency.BTC,
					QuoteCoin:    currency.USDT,
					SettleCoin:   currency.USDT,
					TickSize:     0.1,
					MinPrice:     -19999.9,
					MaxPrice:     19999.9,
					LotSize:      0.001,
					MinSize:      0.001,
					MaxSize:      230,
					LaunchTime:   types.Time(time.UnixMilli(1745376900000)),
					Legs: []SpreadInstrumentLeg{
						{Symbol: "BTCUSDT", ContractType: "LinearPerpetual"},
						{Symbol: "BTCUSDT", ContractType: "Spot"},
					},
				},
				{
					Symbol:       "BTCUSDT-25DEC26_BTC/USDT",
					ContractType: "CarryTrade",
					Status:       "Trading",
					BaseCoin:     currency.BTC,
					QuoteCoin:    currency.USDT,
					SettleCoin:   currency.USDT,
					TickSize:     0.1,
					MinPrice:     -19999.9,
					MaxPrice:     19999.9,
					LotSize:      0.001,
					MinSize:      0.001,
					MaxSize:      10,
					LaunchTime:   types.Time(time.UnixMilli(1767947400000)),
					DeliveryTime: types.Time(time.UnixMilli(1798182000000)),
					Legs: []SpreadInstrumentLeg{
						{Symbol: "BTCUSDT-25DEC26", ContractType: "LinearFutures"},
						{Symbol: "BTCUSDT", ContractType: "Spot"},
					},
				},
			},
			NextPageCursor: "first%3D100074%26last%3D100970",
		}
		assert.Equal(t, exp, result, "GetSpreadInstrumentsInfo should decode every field")
	} else {
		require.NotEmpty(t, result.List, "GetSpreadInstrumentsInfo must return instruments")
	}

	result, err = e.GetSpreadInstrumentsInfo(t.Context(), &SpreadInstrumentsInfoRequest{Symbol: spreadTestSymbol, Cursor: "first%3D100074%26last%3D100970"})
	require.NoError(t, err, "GetSpreadInstrumentsInfo must not error for a symbol and cursor")
	if mockTests {
		exp := &SpreadInstrumentsInfoResponse{List: []SpreadInstrument{{
			Symbol:       "BTCUSDT_BTC/USDT",
			ContractType: "FundingRateArb",
			Status:       "Trading",
			BaseCoin:     currency.BTC,
			QuoteCoin:    currency.USDT,
			SettleCoin:   currency.USDT,
			TickSize:     0.1,
			MinPrice:     -19999.9,
			MaxPrice:     19999.9,
			LotSize:      0.001,
			MinSize:      0.001,
			MaxSize:      230,
			LaunchTime:   types.Time(time.UnixMilli(1745376900000)),
			Legs: []SpreadInstrumentLeg{
				{Symbol: "BTCUSDT", ContractType: "LinearPerpetual"},
				{Symbol: "BTCUSDT", ContractType: "Spot"},
			},
		}}}
		assert.Equal(t, exp, result, "GetSpreadInstrumentsInfo should decode a symbol filtered page")
	}
}

func TestGetSpreadOrderbook(t *testing.T) {
	t.Parallel()
	_, err := e.GetSpreadOrderbook(t.Context(), "", 25)
	require.ErrorIs(t, err, errSymbolMissing, "GetSpreadOrderbook must reject an empty symbol")

	result, err := e.GetSpreadOrderbook(t.Context(), spreadTestSymbol, 25)
	require.NoError(t, err, "GetSpreadOrderbook must not error")
	if mockTests {
		exp := &SpreadOrderbookResponse{
			Symbol:            spreadTestSymbol,
			Bids:              orderbook.LevelsArrayPriceAmount{{Price: -44.9, Amount: 7.268}},
			Asks:              orderbook.LevelsArrayPriceAmount{{Price: -37.8, Amount: 7.266}},
			Timestamp:         types.Time(time.UnixMilli(1791535023555)),
			UpdateID:          642479,
			Sequence:          3303640867,
			CreationTimestamp: types.Time(time.UnixMilli(1791534942561)),
		}
		assert.Equal(t, exp, result, "GetSpreadOrderbook should decode every field")
		return
	}
	assert.Equal(t, spreadTestSymbol, result.Symbol, "GetSpreadOrderbook should return the symbol's book")
}

func TestGetSpreadTickers(t *testing.T) {
	t.Parallel()
	_, err := e.GetSpreadTickers(t.Context(), "")
	require.ErrorIs(t, err, errSymbolMissing, "GetSpreadTickers must reject an empty symbol")

	result, err := e.GetSpreadTickers(t.Context(), spreadTestSymbol)
	require.NoError(t, err, "GetSpreadTickers must not error")
	if mockTests {
		exp := &SpreadTickersResponse{
			List: []SpreadTicker{
				{
					Symbol:              spreadTestSymbol,
					BidPrice:            -44.9,
					BidSize:             7.268,
					AskPrice:            -37.8,
					AskSize:             7.266,
					LastPrice:           -43.5,
					HighPrice24Hour:     -43.3,
					LowPrice24Hour:      -43.5,
					PreviousPrice24Hour: -43.4,
					Volume24Hour:        0.238,
				},
			},
		}
		assert.Equal(t, exp, result, "GetSpreadTickers should decode every field")
		return
	}
	require.NotEmpty(t, result.List, "GetSpreadTickers must return a ticker")
	assert.Equal(t, spreadTestSymbol, result.List[0].Symbol, "GetSpreadTickers should return the symbol's ticker")
}

func TestGetSpreadRecentPublicTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetSpreadRecentPublicTrades(t.Context(), "", 2)
	require.ErrorIs(t, err, errSymbolMissing, "GetSpreadRecentPublicTrades must reject an empty symbol")

	result, err := e.GetSpreadRecentPublicTrades(t.Context(), spreadTestSymbol, 2)
	require.NoError(t, err, "GetSpreadRecentPublicTrades must not error")
	if mockTests {
		exp := &SpreadRecentPublicTradesResponse{
			List: []SpreadPublicTrade{
				{
					ExecutionID: "b311eff0-b8a1-546c-992d-981b5c81f857",
					Symbol:      spreadTestSymbol,
					Price:       -43.5,
					Size:        0.119,
					Side:        "Sell",
					Time:        types.Time(time.UnixMilli(1791362633856)),
					Sequence:    3302117152,
				},
				{
					ExecutionID: "c122c90f-4955-51ba-a47d-9a51a96b5c4e",
					Symbol:      spreadTestSymbol,
					Price:       -43.3,
					Size:        0.119,
					Side:        "Sell",
					Time:        types.Time(time.UnixMilli(1791362628878)),
					Sequence:    3302117114,
				},
			},
		}
		assert.Equal(t, exp, result, "GetSpreadRecentPublicTrades should decode every field")
		return
	}
	require.NotEmpty(t, result.List, "GetSpreadRecentPublicTrades must return trades")
	assert.Equal(t, spreadTestSymbol, result.List[0].Symbol, "GetSpreadRecentPublicTrades should return the symbol's trades")
}

func TestCreateSpreadOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CreateSpreadOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CreateSpreadOrder must reject a nil request")
	_, err = e.CreateSpreadOrder(t.Context(), &CreateSpreadOrderRequest{Side: "Buy", OrderType: "Limit", Quantity: 0.119})
	require.ErrorIs(t, err, errSymbolMissing, "CreateSpreadOrder must reject an empty symbol")
	_, err = e.CreateSpreadOrder(t.Context(), &CreateSpreadOrderRequest{Symbol: spreadTestSymbol, OrderType: "Limit", Quantity: 0.119})
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "CreateSpreadOrder must reject an empty side")
	_, err = e.CreateSpreadOrder(t.Context(), &CreateSpreadOrderRequest{Symbol: spreadTestSymbol, Side: "Buy", Quantity: 0.119})
	require.ErrorIs(t, err, order.ErrTypeIsInvalid, "CreateSpreadOrder must reject an empty order type")
	_, err = e.CreateSpreadOrder(t.Context(), &CreateSpreadOrderRequest{Symbol: spreadTestSymbol, Side: "Buy", OrderType: "Limit"})
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "CreateSpreadOrder must reject a zero quantity")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	price := -44.9
	result, err := e.CreateSpreadOrder(t.Context(), &CreateSpreadOrderRequest{
		Symbol:      spreadTestSymbol,
		Side:        "Buy",
		OrderType:   "Limit",
		Quantity:    0.119,
		Price:       &price,
		OrderLinkID: "spread-gct-001",
		TimeInForce: "PostOnly",
	})
	require.NoError(t, err, "CreateSpreadOrder must not error")
	if mockTests {
		assert.Equal(t, &SpreadOrderResponse{OrderID: "1b00b997-d825-465e-ad1d-80b0eb1955af", OrderLinkID: "spread-gct-001"}, result, "CreateSpreadOrder should decode every field")
		return
	}
	assert.NotEmpty(t, result.OrderID, "CreateSpreadOrder should return an order ID")
}

func TestAmendSpreadOrder(t *testing.T) {
	t.Parallel()
	_, err := e.AmendSpreadOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "AmendSpreadOrder must reject a nil request")
	_, err = e.AmendSpreadOrder(t.Context(), &AmendSpreadOrderRequest{OrderLinkID: "spread-gct-001", Quantity: 0.238})
	require.ErrorIs(t, err, errSymbolMissing, "AmendSpreadOrder must reject an empty symbol")
	_, err = e.AmendSpreadOrder(t.Context(), &AmendSpreadOrderRequest{Symbol: spreadTestSymbol, Quantity: 0.238})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "AmendSpreadOrder must reject a request without an order ID or order link ID")
	_, err = e.AmendSpreadOrder(t.Context(), &AmendSpreadOrderRequest{Symbol: spreadTestSymbol, OrderLinkID: "spread-gct-001"})
	require.ErrorIs(t, err, errQuantityOrPriceRequired, "AmendSpreadOrder must reject a request changing neither quantity nor price")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	price := 0.0
	result, err := e.AmendSpreadOrder(t.Context(), &AmendSpreadOrderRequest{
		Symbol:      spreadTestSymbol,
		OrderID:     "1b00b997-d825-465e-ad1d-80b0eb1955af",
		OrderLinkID: "spread-gct-001",
		Quantity:    0.238,
		Price:       &price,
	})
	require.NoError(t, err, "AmendSpreadOrder must not error")
	if mockTests {
		assert.Equal(t, &SpreadOrderResponse{OrderID: "1b00b997-d825-465e-ad1d-80b0eb1955af", OrderLinkID: "spread-gct-001"}, result, "AmendSpreadOrder should decode every field")
		return
	}
	assert.NotEmpty(t, result.OrderID, "AmendSpreadOrder should return an order ID")
}

func TestCancelSpreadOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelSpreadOrder(t.Context(), "", "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelSpreadOrder must reject a request without an order ID or order link ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelSpreadOrder(t.Context(), "1b00b997-d825-465e-ad1d-80b0eb1955af", "spread-gct-001")
	require.NoError(t, err, "CancelSpreadOrder must not error")
	if mockTests {
		assert.Equal(t, &SpreadOrderResponse{OrderID: "1b00b997-d825-465e-ad1d-80b0eb1955af", OrderLinkID: "spread-gct-001"}, result, "CancelSpreadOrder should decode every field")
		return
	}
	assert.NotEmpty(t, result.OrderID, "CancelSpreadOrder should return an order ID")
}

func TestCancelAllSpreadOrders(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelAllSpreadOrders(t.Context(), spreadTestSymbol, true)
	require.NoError(t, err, "CancelAllSpreadOrders must not error")
	if mockTests {
		exp := &CancelAllSpreadOrdersResponse{
			List: []SpreadOrderResponse{
				{OrderID: "1b00b997-d825-465e-ad1d-80b0eb1955af", OrderLinkID: "spread-gct-001"},
				{OrderID: "11ec47f3-f0a2-4b2a-b302-236f2a2d53a2"},
			},
			Success: true,
		}
		assert.Equal(t, exp, result, "CancelAllSpreadOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "CancelAllSpreadOrders should return a result")
}

func TestGetSpreadOpenOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetSpreadOpenOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetSpreadOpenOrders must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSpreadOpenOrders(t.Context(), &SpreadOpenOrdersRequest{
		Symbol:      spreadTestSymbol,
		BaseCoin:    currency.BTC,
		OrderID:     "1b00b997-d825-465e-ad1d-80b0eb1955af",
		OrderLinkID: "spread-gct-001",
		Limit:       1,
		Cursor:      "aaaee090-fab3-42ea-aea0-c9fbfe6c4bc4%3A1744096099767",
	})
	require.NoError(t, err, "GetSpreadOpenOrders must not error")
	if mockTests {
		exp := &SpreadOpenOrdersResponse{
			List: []SpreadOpenOrder{
				{
					Symbol:                     spreadTestSymbol,
					BaseCoin:                   currency.BTC,
					OrderType:                  "Limit",
					OrderLinkID:                "spread-gct-001",
					Side:                       "Buy",
					TimeInForce:                "GTC",
					OrderID:                    "1b00b997-d825-465e-ad1d-80b0eb1955af",
					LeavesQuantity:             0.1,
					OrderStatus:                "PartiallyFilled",
					CumulativeExecutedQuantity: 0.019,
					Price:                      -44.9,
					Quantity:                   0.119,
					CreatedTime:                types.Time(time.UnixMilli(1791008100508)),
					UpdatedTime:                types.Time(time.UnixMilli(1791008160123)),
				},
			},
			NextPageCursor: "1b00b997-d825-465e-ad1d-80b0eb1955af%3A1791008100508%2C1b00b997-d825-465e-ad1d-80b0eb1955af%3A1791008100508",
		}
		assert.Equal(t, exp, result, "GetSpreadOpenOrders should decode every field")
	}
}

func TestGetSpreadOrderHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetSpreadOrderHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetSpreadOrderHistory must reject a nil request")
	_, err = e.GetSpreadOrderHistory(t.Context(), &SpreadOrderHistoryRequest{StartTime: spreadTestEnd, EndTime: spreadTestStart})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetSpreadOrderHistory must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSpreadOrderHistory(t.Context(), &SpreadOrderHistoryRequest{
		Symbol:      spreadTestSymbol,
		BaseCoin:    currency.BTC,
		OrderID:     "1b00b997-d825-465e-ad1d-80b0eb1955af",
		OrderLinkID: "spread-gct-001",
		StartTime:   spreadTestStart,
		EndTime:     spreadTestEnd,
		Limit:       1,
		Cursor:      "aaaee090-fab3-42ea-aea0-c9fbfe6c4bc4%3A1744096099767",
	})
	require.NoError(t, err, "GetSpreadOrderHistory must not error")
	if mockTests {
		exp := &SpreadOrderHistoryResponse{
			List: []SpreadHistoricOrder{
				{
					Symbol:                     spreadTestSymbol,
					OrderType:                  "Limit",
					OrderLinkID:                "spread-gct-001",
					OrderID:                    "1b00b997-d825-465e-ad1d-80b0eb1955af",
					ContractType:               "FundingRateArb",
					CancelRejectReason:         "EC_PerCancelRequest",
					OrderStatus:                "Cancelled",
					Price:                      -44.9,
					OrderQuantity:              0.119,
					TimeInForce:                "PostOnly",
					BaseCoin:                   currency.BTC,
					CreatedAt:                  types.Time(time.UnixMilli(1791008100508)),
					UpdatedAt:                  types.Time(time.UnixMilli(1791008396079)),
					Side:                       "Buy",
					LeavesQuantity:             0.1,
					SettleCoin:                 currency.USDT,
					CumulativeExecutedQuantity: 0.019,
					Quantity:                   0.119,
					Leg1Symbol:                 "BTCUSDT",
					Leg1ProductType:            "Futures",
					Leg1OrderID:                "82335b0a-b7d9-4ea5-9230-e71271a65100",
					Leg1Side:                   "Buy",
					Leg2ProductType:            "Spot",
					Leg2OrderID:                "1924011967786517249",
					Leg2Symbol:                 "BTCUSDT",
					Leg2Side:                   "Sell",
				},
			},
			NextPageCursor: "1b00b997-d825-465e-ad1d-80b0eb1955af%3A1791008100508%2C1b00b997-d825-465e-ad1d-80b0eb1955af%3A1791008100508",
		}
		assert.Equal(t, exp, result, "GetSpreadOrderHistory should decode every field")
	}
}

func TestGetSpreadTradeHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetSpreadTradeHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetSpreadTradeHistory must reject a nil request")
	_, err = e.GetSpreadTradeHistory(t.Context(), &SpreadTradeHistoryRequest{StartTime: spreadTestEnd, EndTime: spreadTestStart})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetSpreadTradeHistory must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSpreadTradeHistory(t.Context(), &SpreadTradeHistoryRequest{
		Symbol:      spreadTestSymbol,
		OrderID:     "1b00b997-d825-465e-ad1d-80b0eb1955af",
		OrderLinkID: "spread-gct-001",
		StartTime:   spreadTestStart,
		EndTime:     spreadTestEnd,
		Limit:       1,
		Cursor:      "aaaee090-fab3-42ea-aea0-c9fbfe6c4bc4%3A1744096099767",
	})
	require.NoError(t, err, "GetSpreadTradeHistory must not error")
	if mockTests {
		exp := &SpreadTradeHistoryResponse{
			List: []SpreadExecution{
				{
					Symbol:            spreadTestSymbol,
					OrderLinkID:       "spread-gct-001",
					Side:              "Buy",
					OrderID:           "1b00b997-d825-465e-ad1d-80b0eb1955af",
					ExecutionPrice:    -44.9,
					ExecutionTime:     types.Time(time.UnixMilli(1791008160120)),
					ExecutionType:     "Trade",
					ExecutionQuantity: 0.019,
					ExecutionID:       "82c82077-0caa-5304-894d-58a50a342bd7",
					Legs: []SpreadExecutionLeg{
						{
							Symbol:            "BTCUSDT",
							Side:              "Buy",
							ExecutionPrice:    121955.1,
							ExecutionTime:     types.Time(time.UnixMilli(1791008160124)),
							ExecutionValue:    2317.1469,
							ExecutionType:     "FutureSpread",
							Category:          "linear",
							ExecutionQuantity: 0.019,
							ExecutionFee:      1.2744308,
							FeeCurrency:       currency.USDT,
							ExecutionID:       "99a18f80-d3b5-4c6f-a1f1-8c5870e3f3bc",
						},
						{
							Symbol:            "BTCUSDT",
							Side:              "Sell",
							ExecutionPrice:    122000,
							ExecutionTime:     types.Time(time.UnixMilli(1791008160124)),
							ExecutionValue:    2318,
							ExecutionType:     "FutureSpread",
							Category:          "spot",
							ExecutionQuantity: 0.019,
							ExecutionFeeV2:    2.5498,
							FeeCurrency:       currency.USDT,
							ExecutionID:       "2110000000061481958",
						},
					},
				},
			},
			NextPageCursor: "82c82077-0caa-5304-894d-58a50a342bd7%3A1791008160120%2C82c82077-0caa-5304-894d-58a50a342bd7%3A1791008160120",
		}
		assert.Equal(t, exp, result, "GetSpreadTradeHistory should decode every field")
	}
}

func TestGetSpreadMaxQuantity(t *testing.T) {
	t.Parallel()
	_, err := e.GetSpreadMaxQuantity(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetSpreadMaxQuantity must reject a nil request")
	_, err = e.GetSpreadMaxQuantity(t.Context(), &SpreadMaxQuantityRequest{Side: "1", OrderPrice: -44.9})
	require.ErrorIs(t, err, errSymbolMissing, "GetSpreadMaxQuantity must reject an empty symbol")
	_, err = e.GetSpreadMaxQuantity(t.Context(), &SpreadMaxQuantityRequest{Symbol: spreadTestSymbol, OrderPrice: -44.9})
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "GetSpreadMaxQuantity must reject an empty side")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetSpreadMaxQuantity(t.Context(), &SpreadMaxQuantityRequest{Symbol: spreadTestSymbol, Side: "1", OrderPrice: -44.9})
	require.NoError(t, err, "GetSpreadMaxQuantity must not error")
	if mockTests {
		assert.Equal(t, &SpreadMaxQuantityResponse{AvailableBalance: 1992.5519949}, result, "GetSpreadMaxQuantity should decode every field")
	}
}
