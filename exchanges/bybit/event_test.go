package bybit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// Event contracts of each type in the fixtures; they expire, so live tests look up a listed contract instead
const (
	eventTestTargetSymbol = "ETHUSDT-6NOV26-2200-BELOW"
	eventTestRangeSymbol  = "ETHUSDT-10OCT26-2580-2640-OUT"
	eventTestUpDownSymbol = "BTCUSDT-15MIN-DOWN"
	eventTestOrderID      = "746a6c87-a6dd-4008-8889-37e694046681"
	eventTestRangeOrderID = "9c1d6f10-3b9a-4d7e-a0c2-5e8f7b2d4a61"
	eventTestCursor       = "orderId%3D0f2b7c55-6e1d-4b8a-9d3f-2a4c6e8b0d12"
)

var (
	eventTestStart = time.UnixMilli(1790812800000)
	eventTestEnd   = time.UnixMilli(1791331200000)
)

// eventTestSymbol returns the fixtures' Target contract offline and a currently trading contract live
func eventTestSymbol(t *testing.T) string {
	t.Helper()
	if mockTests {
		return eventTestTargetSymbol
	}
	resp, err := e.GetEventInstrumentsInfo(t.Context(), &EventInstrumentsInfoRequest{Status: "Trading", Limit: 1})
	require.NoError(t, err, "GetEventInstrumentsInfo must not error")
	require.NotEmpty(t, resp.List, "GetEventInstrumentsInfo must return a trading event contract")
	return resp.List[0].Symbol
}

func TestGetEventInstrumentsInfo(t *testing.T) {
	t.Parallel()
	_, err := e.GetEventInstrumentsInfo(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetEventInstrumentsInfo must reject a nil request")

	result, err := e.GetEventInstrumentsInfo(t.Context(), &EventInstrumentsInfoRequest{Status: "Trading", Limit: 3})
	require.NoError(t, err, "GetEventInstrumentsInfo must not error")
	if mockTests {
		exp := &EventInstrumentsInfoResponse{
			NextPageCursor: "0%2C3",
			List: []EventInstrument{
				{
					Symbol:            "BTCUSDT-15MIN-DOWN",
					SymbolID:          500017,
					BaseCoin:          currency.BTC,
					QuoteCoin:         currency.USDT,
					SettleCoin:        currency.USDT,
					EventContractType: "Up_Down",
					DurationWindow:    900,
					RVThreshold:       0.6,
					TakerFeeRate:      0.0088,
					MakerFeeRate:      0.0005,
					LaunchTime:        types.Time(time.Unix(1786000980, 0)),
					DeliveryTime:      types.Time(time.Unix(1798779602, 0)),
					Status:            "Trading",
					FeeRateRule: []EventFeeRateRule{
						{MaxSecondsToDelivery: 315360000, BaseMarkup: 0.06},
					},
					LotSizeFilter: EventLotSizeFilter{
						MinOrderAmount:                      5,
						MaxOrderAmount:                      800,
						OrderAmountTickSize:                 1,
						QuoteMaxValidMilliseconds:           60000,
						QuoteMinIntervalPerUserMilliseconds: 200,
					},
					QuotationLimit: []EventQuotationLimit{
						{
							MaxSecondsToDelivery:  315360000,
							QuoteMaxAmountPerUser: 20000,
							QuoteMinAmountPerUser: 10000,
							QuoteMinAmount:        50,
						},
					},
				},
				{
					Symbol:             "ETHUSDT-10OCT26-2580-2640-OUT",
					SymbolID:           5001709,
					BaseCoin:           currency.ETH,
					QuoteCoin:          currency.USDT,
					SettleCoin:         currency.USDT,
					EventContractType:  "Range",
					RVThreshold:        0.62,
					Rebate:             0.2,
					TakerFeeRate:       0.045,
					LaunchTime:         types.Time(time.Unix(1791368460, 0)),
					DeliveryTime:       types.Time(time.Unix(1791619200, 0)),
					Status:             "Trading",
					LowerBound:         2580,
					UpperBound:         2640,
					CallOptionSymbolID: 407620,
					PutOptionSymbolID:  407615,
					FeeRateRule: []EventFeeRateRule{
						{MaxSecondsToDelivery: 259200, BaseMarkup: 5},
						{MinSecondsToDelivery: 259200, MaxSecondsToDelivery: 315360000, BaseMarkup: 5},
					},
					LotSizeFilter: EventLotSizeFilter{
						MinOrderAmount:                      5,
						MaxOrderAmount:                      500,
						OrderAmountTickSize:                 1,
						QuoteMaxValidMilliseconds:           300000,
						QuoteMinIntervalPerUserMilliseconds: 5000,
					},
					QuotationLimit: []EventQuotationLimit{
						{
							MaxSecondsToDelivery:  259200,
							QuoteMaxAmountPerUser: 10000000,
							QuoteMinAmountPerUser: 5000,
							QuoteMinAmount:        500,
						},
						{
							MinSecondsToDelivery:  259200,
							MaxSecondsToDelivery:  315360000,
							QuoteMaxAmountPerUser: 10000000,
							QuoteMinAmountPerUser: 3000,
							QuoteMinAmount:        500,
						},
					},
				},
				{
					Symbol:            "ETHUSDT-6NOV26-2200-BELOW",
					SymbolID:          5001801,
					BaseCoin:          currency.ETH,
					QuoteCoin:         currency.USDT,
					SettleCoin:        currency.USDT,
					EventContractType: "Target",
					RVThreshold:       0.62,
					Rebate:            0.2,
					TakerFeeRate:      0.045,
					LaunchTime:        types.Time(time.Unix(1791476580, 0)),
					DeliveryTime:      types.Time(time.Unix(1793952000, 0)),
					Status:            "Trading",
					TargetPrice:       2200,
					PutOptionSymbolID: 386741,
					FeeRateRule: []EventFeeRateRule{
						{MaxSecondsToDelivery: 259200, BaseMarkup: 5},
						{MinSecondsToDelivery: 259200, MaxSecondsToDelivery: 315360000, BaseMarkup: 5},
					},
					LotSizeFilter: EventLotSizeFilter{
						MinOrderAmount:                      5,
						MaxOrderAmount:                      500,
						OrderAmountTickSize:                 1,
						QuoteMaxValidMilliseconds:           300000,
						QuoteMinIntervalPerUserMilliseconds: 5000,
					},
					QuotationLimit: []EventQuotationLimit{
						{
							MaxSecondsToDelivery:  259200,
							QuoteMaxAmountPerUser: 10000000,
							QuoteMinAmountPerUser: 5000,
							QuoteMinAmount:        500,
						},
						{
							MinSecondsToDelivery:  259200,
							MaxSecondsToDelivery:  315360000,
							QuoteMaxAmountPerUser: 10000000,
							QuoteMinAmountPerUser: 3000,
							QuoteMinAmount:        500,
						},
					},
				},
			},
		}
		assert.Equal(t, exp, result, "GetEventInstrumentsInfo should decode every field")
	} else {
		require.NotEmpty(t, result.List, "GetEventInstrumentsInfo must return trading event contracts")
	}

	result, err = e.GetEventInstrumentsInfo(t.Context(), &EventInstrumentsInfoRequest{Symbol: eventTestSymbol(t), Cursor: "0%2C3"})
	require.NoError(t, err, "GetEventInstrumentsInfo must not error for a symbol and cursor")
	if mockTests {
		exp := &EventInstrumentsInfoResponse{
			List: []EventInstrument{
				{
					Symbol:            "ETHUSDT-6NOV26-2200-BELOW",
					SymbolID:          5001801,
					BaseCoin:          currency.ETH,
					QuoteCoin:         currency.USDT,
					SettleCoin:        currency.USDT,
					EventContractType: "Target",
					RVThreshold:       0.62,
					Rebate:            0.2,
					TakerFeeRate:      0.045,
					LaunchTime:        types.Time(time.Unix(1791476580, 0)),
					DeliveryTime:      types.Time(time.Unix(1793952000, 0)),
					Status:            "Trading",
					TargetPrice:       2200,
					PutOptionSymbolID: 386741,
					FeeRateRule: []EventFeeRateRule{
						{MaxSecondsToDelivery: 259200, BaseMarkup: 5},
						{MinSecondsToDelivery: 259200, MaxSecondsToDelivery: 315360000, BaseMarkup: 5},
					},
					LotSizeFilter: EventLotSizeFilter{
						MinOrderAmount:                      5,
						MaxOrderAmount:                      500,
						OrderAmountTickSize:                 1,
						QuoteMaxValidMilliseconds:           300000,
						QuoteMinIntervalPerUserMilliseconds: 5000,
					},
					QuotationLimit: []EventQuotationLimit{
						{
							MaxSecondsToDelivery:  259200,
							QuoteMaxAmountPerUser: 10000000,
							QuoteMinAmountPerUser: 5000,
							QuoteMinAmount:        500,
						},
						{
							MinSecondsToDelivery:  259200,
							MaxSecondsToDelivery:  315360000,
							QuoteMaxAmountPerUser: 10000000,
							QuoteMinAmountPerUser: 3000,
							QuoteMinAmount:        500,
						},
					},
				},
			},
		}
		assert.Equal(t, exp, result, "GetEventInstrumentsInfo should decode every field of case 2")
	}
}

func TestGetEventOrderbook(t *testing.T) {
	t.Parallel()
	_, err := e.GetEventOrderbook(t.Context(), "")
	require.ErrorIs(t, err, errSymbolMissing, "GetEventOrderbook must reject an empty symbol")

	symbol := eventTestSymbol(t)
	result, err := e.GetEventOrderbook(t.Context(), symbol)
	require.NoError(t, err, "GetEventOrderbook must not error")
	if mockTests {
		exp := &EventOrderbookResponse{
			Symbol: "ETHUSDT-6NOV26-2200-BELOW",
			Entries: []EventOrderbookEntry{
				{PayoutRatio: 3.4907, OrderValue: 500},
				{PayoutRatio: 3.3021, OrderValue: 1000},
			},
			Timestamp:         types.Time(time.UnixMilli(1791535752122)),
			UpdateID:          11558,
			Sequence:          101174421,
			CreationTimestamp: types.Time(time.UnixMilli(1791535749200)),
		}
		assert.Equal(t, exp, result, "GetEventOrderbook should decode every field")
		return
	}
	assert.Equal(t, symbol, result.Symbol, "GetEventOrderbook should return the symbol's book")
}

func TestEventOrderbookEntryUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var entry EventOrderbookEntry
	require.NoError(t, entry.UnmarshalJSON([]byte(`["3.4907","500"]`)), "UnmarshalJSON must decode an entry")
	assert.Equal(t, EventOrderbookEntry{PayoutRatio: 3.4907, OrderValue: 500}, entry, "UnmarshalJSON should decode the payout ratio and order value")
	assert.Error(t, entry.UnmarshalJSON([]byte(`{"r":"3.4907"}`)), "UnmarshalJSON should reject an object")
	assert.Error(t, entry.UnmarshalJSON([]byte(`["ratio","500"]`)), "UnmarshalJSON should reject a payout ratio that is not a number")
}

func TestGetEventOrderHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetEventOrderHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetEventOrderHistory must reject a nil request")
	_, err = e.GetEventOrderHistory(t.Context(), &EventOrderHistoryRequest{StartTime: eventTestEnd, EndTime: eventTestStart})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetEventOrderHistory must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetEventOrderHistory(t.Context(), &EventOrderHistoryRequest{StartTime: eventTestStart, EndTime: eventTestEnd, Limit: 3, Cursor: eventTestCursor})
	require.NoError(t, err, "GetEventOrderHistory must not error")
	if mockTests {
		exp := &EventOrderHistoryResponse{
			List: []EventOrder{
				{
					OrderID:                 "746a6c87-a6dd-4008-8889-37e694046681",
					OrderLinkID:             "event-gct-001",
					OrderStatus:             "Filled",
					Symbol:                  "ETHUSDT-6NOV26-2200-BELOW",
					SymbolID:                5001801,
					BaseCoin:                currency.ETH,
					QuoteCoin:               currency.USDT,
					SettleCoin:              currency.USDT,
					ECDirection:             "BELOW",
					ECTargetPrice:           2200,
					ECContractType:          "Target",
					SettleTime:              types.Time(time.Unix(1793952000, 0)),
					ECOrderValue:            500,
					ECIndexPrice:            2391.27,
					OrderAveragePayoutRatio: 3.4907,
					CumulativeExecutedValue: 500,
					PayoutRatio:             3.4907,
					TradeFee:                22.5,
					TransactionTime:         types.Time(time.UnixMilli(1791008160374)),
					CreateTime:              types.Time(time.UnixMilli(1791008100917)),
					UpdateTime:              types.Time(time.UnixMilli(1791008160374)),
					CumulativePayout:        1745.35,
					CumulativeExecutedFee:   22.5,
					CreateType:              "CreateByUser",
					ExtraFees:               "[{\"feeCoin\":\"USDT\",\"feeType\":\"GST\",\"subFeeType\":\"IND_GST\",\"feeRate\":\"0.0000675\",\"fee\":\"0.003375\"}]",
				},
				{
					OrderID:         "9c1d6f10-3b9a-4d7e-a0c2-5e8f7b2d4a61",
					OrderLinkID:     "event-gct-002",
					OrderStatus:     "Cancelled",
					Symbol:          "ETHUSDT-10OCT26-2580-2640-OUT",
					SymbolID:        5001709,
					BaseCoin:        currency.ETH,
					QuoteCoin:       currency.USDT,
					SettleCoin:      currency.USDT,
					ECDirection:     "RANGE_OUT",
					ECContractType:  "Range",
					ECLowerBound:    2580,
					ECUpperBound:    2640,
					SettleTime:      types.Time(time.Unix(1791619200, 0)),
					ECOrderValue:    1000,
					ECIndexPrice:    2391.27,
					PayoutRatio:     1.18,
					TransactionTime: types.Time(time.UnixMilli(1791008200511)),
					CreateTime:      types.Time(time.UnixMilli(1791008190511)),
					UpdateTime:      types.Time(time.UnixMilli(1791008200511)),
					CancelType:      "CancelByUser",
					CreateType:      "CreateByUser",
					RejectReason:    "EC_PerCancelRequest",
				},
				{
					OrderID:          "0f2b7c55-6e1d-4b8a-9d3f-2a4c6e8b0d12",
					OrderStatus:      "Rejected",
					Symbol:           "BTCUSDT-15MIN-DOWN",
					SymbolID:         500017,
					BaseCoin:         currency.BTC,
					QuoteCoin:        currency.USDT,
					SettleCoin:       currency.USDT,
					ECDirection:      "DOWN",
					ECDurationWindow: 900,
					ECContractType:   "UpDown",
					SettleTime:       types.Time(time.Unix(1791009000, 0)),
					ECOrderValue:     50,
					ECIndexPrice:     121480.2,
					PayoutRatio:      1.95,
					TransactionTime:  types.Time(time.UnixMilli(1791008100120)),
					CreateTime:       types.Time(time.UnixMilli(1791008100118)),
					UpdateTime:       types.Time(time.UnixMilli(1791008100120)),
					CreateType:       "CreateByUser",
					RejectReason:     "EC_NoImmediateQtyToFill",
				},
			},
			Cursor: "orderId%3D0f2b7c55-6e1d-4b8a-9d3f-2a4c6e8b0d12",
		}
		assert.Equal(t, exp, result, "GetEventOrderHistory should decode every field")
	}
	result, err = e.GetEventOrderHistory(t.Context(), &EventOrderHistoryRequest{Symbol: eventTestTargetSymbol, OrderID: eventTestOrderID, OrderLinkID: "event-gct-001"})
	require.NoError(t, err, "GetEventOrderHistory must not error for a symbol and order IDs")
	if mockTests {
		exp := &EventOrderHistoryResponse{
			List: []EventOrder{
				{
					OrderID:                 "746a6c87-a6dd-4008-8889-37e694046681",
					OrderLinkID:             "event-gct-001",
					OrderStatus:             "Filled",
					Symbol:                  "ETHUSDT-6NOV26-2200-BELOW",
					SymbolID:                5001801,
					BaseCoin:                currency.ETH,
					QuoteCoin:               currency.USDT,
					SettleCoin:              currency.USDT,
					ECDirection:             "BELOW",
					ECTargetPrice:           2200,
					ECContractType:          "Target",
					SettleTime:              types.Time(time.Unix(1793952000, 0)),
					ECOrderValue:            500,
					ECIndexPrice:            2391.27,
					OrderAveragePayoutRatio: 3.4907,
					CumulativeExecutedValue: 500,
					PayoutRatio:             3.4907,
					TradeFee:                22.5,
					TransactionTime:         types.Time(time.UnixMilli(1791008160374)),
					CreateTime:              types.Time(time.UnixMilli(1791008100917)),
					UpdateTime:              types.Time(time.UnixMilli(1791008160374)),
					CumulativePayout:        1745.35,
					CumulativeExecutedFee:   22.5,
					CreateType:              "CreateByUser",
					ExtraFees:               "[{\"feeCoin\":\"USDT\",\"feeType\":\"GST\",\"subFeeType\":\"IND_GST\",\"feeRate\":\"0.0000675\",\"fee\":\"0.003375\"}]",
				},
			},
		}
		assert.Equal(t, exp, result, "GetEventOrderHistory should decode every field of case 2")
	}
}

func TestGetEventActiveOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetEventActiveOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetEventActiveOrders must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetEventActiveOrders(t.Context(), &EventActiveOrdersRequest{Limit: 3, Cursor: eventTestCursor})
	require.NoError(t, err, "GetEventActiveOrders must not error")
	if mockTests {
		exp := &EventActiveOrdersResponse{
			List: []EventActiveOrder{
				{
					OrderID:        "746a6c87-a6dd-4008-8889-37e694046681",
					OrderLinkID:    "event-gct-001",
					Symbol:         "ETHUSDT-6NOV26-2200-BELOW",
					SymbolID:       5001801,
					Side:           "Sell",
					OrderStatus:    "New",
					PayoutRatio:    3.4907,
					LeavesValue:    500,
					ECContractType: "Target",
					ECDirection:    "BELOW",
					ECOrderValue:   500,
					ECSettleTime:   types.Time(time.Unix(1793952000, 0)),
					ECTargetPrice:  2200,
					ECIndexPrice:   2391.27,
					CreateType:     "CreateByUser",
					CreatedTime:    types.Time(time.UnixMilli(1791008100917)),
					UpdatedTime:    types.Time(time.UnixMilli(1791008100917)),
				},
				{
					OrderID:                 "9c1d6f10-3b9a-4d7e-a0c2-5e8f7b2d4a61",
					OrderLinkID:             "event-gct-002",
					Symbol:                  "ETHUSDT-10OCT26-2580-2640-OUT",
					SymbolID:                5001709,
					Side:                    "Sell",
					OrderStatus:             "PartiallyFilled",
					PayoutRatio:             1.18,
					CumulativeExecutedValue: 200,
					CumulativeExecutedFee:   9,
					LeavesValue:             800,
					ECContractType:          "Range",
					ECDirection:             "RANGE_OUT",
					ECOrderValue:            1000,
					ECSettleTime:            types.Time(time.Unix(1791619200, 0)),
					ECLowerBound:            2580,
					ECUpperBound:            2640,
					ECIndexPrice:            2391.27,
					OrderAveragePayoutRatio: 1.18,
					CreateType:              "CreateByUser",
					CreatedTime:             types.Time(time.UnixMilli(1791008190511)),
					UpdatedTime:             types.Time(time.UnixMilli(1791008195002)),
				},
				{
					OrderID:          "0f2b7c55-6e1d-4b8a-9d3f-2a4c6e8b0d12",
					Symbol:           "BTCUSDT-15MIN-DOWN",
					SymbolID:         500017,
					Side:             "Sell",
					OrderStatus:      "New",
					PayoutRatio:      1.95,
					LeavesValue:      50,
					ECContractType:   "UpDown",
					ECDirection:      "DOWN",
					ECOrderValue:     50,
					ECSettleTime:     types.Time(time.Unix(1791009000, 0)),
					ECDurationWindow: 900,
					ECIndexPrice:     121480.2,
					CreateType:       "CreateByUser",
					CreatedTime:      types.Time(time.UnixMilli(1791008100118)),
					UpdatedTime:      types.Time(time.UnixMilli(1791008100118)),
				},
			},
			NextPageCursor: "orderId%3D0f2b7c55-6e1d-4b8a-9d3f-2a4c6e8b0d12",
		}
		assert.Equal(t, exp, result, "GetEventActiveOrders should decode every field")
	}
	result, err = e.GetEventActiveOrders(t.Context(), &EventActiveOrdersRequest{Symbol: eventTestTargetSymbol, OrderID: eventTestOrderID, OrderLinkID: "event-gct-001"})
	require.NoError(t, err, "GetEventActiveOrders must not error for a symbol and order IDs")
	if mockTests {
		exp := &EventActiveOrdersResponse{
			List: []EventActiveOrder{
				{
					OrderID:        "746a6c87-a6dd-4008-8889-37e694046681",
					OrderLinkID:    "event-gct-001",
					Symbol:         "ETHUSDT-6NOV26-2200-BELOW",
					SymbolID:       5001801,
					Side:           "Sell",
					OrderStatus:    "New",
					PayoutRatio:    3.4907,
					LeavesValue:    500,
					ECContractType: "Target",
					ECDirection:    "BELOW",
					ECOrderValue:   500,
					ECSettleTime:   types.Time(time.Unix(1793952000, 0)),
					ECTargetPrice:  2200,
					ECIndexPrice:   2391.27,
					CreateType:     "CreateByUser",
					CreatedTime:    types.Time(time.UnixMilli(1791008100917)),
					UpdatedTime:    types.Time(time.UnixMilli(1791008100917)),
				},
			},
		}
		assert.Equal(t, exp, result, "GetEventActiveOrders should decode every field of case 2")
	}
}

func TestGetEventPositionInfo(t *testing.T) {
	t.Parallel()
	_, err := e.GetEventPositionInfo(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetEventPositionInfo must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetEventPositionInfo(t.Context(), &EventPositionInfoRequest{BaseCoin: currency.ETH, SettleCoin: currency.USDT, Limit: 2, Cursor: eventTestCursor})
	require.NoError(t, err, "GetEventPositionInfo must not error")
	if mockTests {
		exp := &EventPositionInfoResponse{
			List: []EventPosition{
				{
					Symbol:             "ETHUSDT-6NOV26-2200-BELOW",
					BaseCoin:           currency.ETH,
					SettleCoin:         currency.USDT,
					Side:               "Sell",
					ECContractType:     "Target",
					ECDirection:        "BELOW",
					AveragePayoutRatio: 3.4907,
					PositionValue:      500,
					ECSettleTime:       types.Time(time.Unix(1793952000, 0)),
					CreatedTime:        types.Time(time.UnixMilli(1791008160374)),
					UpdatedTime:        types.Time(time.UnixMilli(1791008160374)),
					Sequence:           101174421,
				},
				{
					Symbol:              "ETHUSDT-10OCT26-2580-2640-OUT",
					BaseCoin:            currency.ETH,
					SettleCoin:          currency.USDT,
					Side:                "Sell",
					ECContractType:      "Range",
					ECDirection:         "RANGE_OUT",
					AveragePayoutRatio:  1.18,
					PositionValue:       200,
					ECSettleTime:        types.Time(time.Unix(1791619200, 0)),
					CumulativeClosedPNL: -36,
					CreatedTime:         types.Time(time.UnixMilli(1791008195002)),
					UpdatedTime:         types.Time(time.UnixMilli(1791619200015)),
					Sequence:            101180113,
				},
			},
			NextPageCursor: "symbol%3DETHUSDT-10OCT26-2580-2640-OUT",
		}
		assert.Equal(t, exp, result, "GetEventPositionInfo should decode every field")
	}
	result, err = e.GetEventPositionInfo(t.Context(), &EventPositionInfoRequest{Symbol: eventTestTargetSymbol})
	require.NoError(t, err, "GetEventPositionInfo must not error for a symbol")
	if mockTests {
		exp := &EventPositionInfoResponse{
			List: []EventPosition{
				{
					Symbol:             "ETHUSDT-6NOV26-2200-BELOW",
					BaseCoin:           currency.ETH,
					SettleCoin:         currency.USDT,
					Side:               "Sell",
					ECContractType:     "Target",
					ECDirection:        "BELOW",
					AveragePayoutRatio: 3.4907,
					PositionValue:      500,
					ECSettleTime:       types.Time(time.Unix(1793952000, 0)),
					CreatedTime:        types.Time(time.UnixMilli(1791008160374)),
					UpdatedTime:        types.Time(time.UnixMilli(1791008160374)),
					Sequence:           101174421,
				},
			},
		}
		assert.Equal(t, exp, result, "GetEventPositionInfo should decode every field of case 2")
	}
}

func TestGetEventSettlementRecords(t *testing.T) {
	t.Parallel()
	_, err := e.GetEventSettlementRecords(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetEventSettlementRecords must reject a nil request")
	_, err = e.GetEventSettlementRecords(t.Context(), &EventSettlementRecordsRequest{})
	require.ErrorIs(t, err, errSymbolMissing, "GetEventSettlementRecords must reject an empty symbol")
	_, err = e.GetEventSettlementRecords(t.Context(), &EventSettlementRecordsRequest{Symbol: eventTestTargetSymbol, StartTime: eventTestEnd, EndTime: eventTestStart})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetEventSettlementRecords must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetEventSettlementRecords(t.Context(), &EventSettlementRecordsRequest{Symbol: eventTestTargetSymbol, StartTime: eventTestStart, EndTime: eventTestEnd, Limit: 1, Cursor: eventTestCursor})
	require.NoError(t, err, "GetEventSettlementRecords must not error for a Target contract")
	if mockTests {
		exp := &EventSettlementRecordsResponse{
			List: []EventSettlement{
				{
					TransactionID:      "0176bba4-66fa-542b-af33-cea000a83dc0",
					Symbol:             "ETHUSDT-6NOV26-2200-BELOW",
					SymbolID:           5001801,
					BaseCoin:           currency.ETH,
					SettleCoin:         currency.USDT,
					Side:               "Sell",
					ECContractType:     "Target",
					ECDirection:        "BELOW",
					SettlePrice:        2412.51,
					EntryPrice:         3.4907,
					GrossPayoutRatio:   3.4907,
					ECTargetPrice:      2200,
					TransactionTime:    types.Time(time.UnixMilli(1793952000015)),
					CrossSequence:      101190001,
					ExecutionFee:       22.5,
					ExecutionFeeRate:   0.045,
					SessionRealisedPNL: 477.5,
					OrderID:            "746a6c87-a6dd-4008-8889-37e694046681",
				},
			},
			Cursor: "transId%3D0176bba4-66fa-542b-af33-cea000a83dc0",
		}
		assert.Equal(t, exp, result, "GetEventSettlementRecords should decode every field")
	}
	result, err = e.GetEventSettlementRecords(t.Context(), &EventSettlementRecordsRequest{Symbol: eventTestRangeSymbol, OrderID: eventTestRangeOrderID})
	require.NoError(t, err, "GetEventSettlementRecords must not error for a Range contract")
	if mockTests {
		exp := &EventSettlementRecordsResponse{
			List: []EventSettlement{
				{
					TransactionID:      "5b1e9d3a-2c4f-5a6b-8d7e-1f0a2b3c4d5e",
					Symbol:             "ETHUSDT-10OCT26-2580-2640-OUT",
					SymbolID:           5001709,
					BaseCoin:           currency.ETH,
					SettleCoin:         currency.USDT,
					Side:               "Sell",
					ECContractType:     "Range",
					ECDirection:        "RANGE_OUT",
					SettlePrice:        2601.32,
					EntryPrice:         1.18,
					GrossPayoutRatio:   1.18,
					ECLowerBound:       2580,
					ECUpperBound:       2640,
					TransactionTime:    types.Time(time.UnixMilli(1791619200015)),
					CrossSequence:      101180113,
					ExecutionFee:       9,
					ExecutionFeeRate:   0.045,
					SessionRealisedPNL: -36,
					OrderID:            "9c1d6f10-3b9a-4d7e-a0c2-5e8f7b2d4a61",
				},
			},
		}
		assert.Equal(t, exp, result, "GetEventSettlementRecords should decode every field of case 2")
	}
	result, err = e.GetEventSettlementRecords(t.Context(), &EventSettlementRecordsRequest{Symbol: eventTestUpDownSymbol})
	require.NoError(t, err, "GetEventSettlementRecords must not error for an UpDown contract")
	if mockTests {
		exp := &EventSettlementRecordsResponse{
			List: []EventSettlement{
				{
					TransactionID:      "7c2d4e6f-8a0b-4c1d-9e2f-3a4b5c6d7e8f",
					Symbol:             "BTCUSDT-15MIN-DOWN",
					SymbolID:           500017,
					BaseCoin:           currency.BTC,
					SettleCoin:         currency.USDT,
					Side:               "Sell",
					ECContractType:     "UpDown",
					ECDirection:        "DOWN",
					SettlePrice:        121512.4,
					EntryPrice:         1.95,
					GrossPayoutRatio:   1.95,
					ECDurationWindow:   900,
					TransactionTime:    types.Time(time.UnixMilli(1791009000012)),
					CrossSequence:      1311,
					ExecutionFee:       2.25,
					ExecutionFeeRate:   0.045,
					Payout:             97.5,
					SessionRealisedPNL: 45.25,
					OrderID:            "0f2b7c55-6e1d-4b8a-9d3f-2a4c6e8b0d12",
				},
			},
		}
		assert.Equal(t, exp, result, "GetEventSettlementRecords should decode every field of case 3")
	}
}

func TestGetEventTradeHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetEventTradeHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetEventTradeHistory must reject a nil request")
	_, err = e.GetEventTradeHistory(t.Context(), &EventTradeHistoryRequest{})
	require.ErrorIs(t, err, errSymbolMissing, "GetEventTradeHistory must reject an empty symbol")
	_, err = e.GetEventTradeHistory(t.Context(), &EventTradeHistoryRequest{Symbol: eventTestTargetSymbol, StartTime: eventTestEnd, EndTime: eventTestStart})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetEventTradeHistory must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetEventTradeHistory(t.Context(), &EventTradeHistoryRequest{Symbol: eventTestTargetSymbol, StartTime: eventTestStart, EndTime: eventTestEnd, Limit: 1, Cursor: eventTestCursor})
	require.NoError(t, err, "GetEventTradeHistory must not error for a Target contract")
	if mockTests {
		exp := &EventTradeHistoryResponse{
			List: []EventTrade{
				{
					OrderID:          "746a6c87-a6dd-4008-8889-37e694046681",
					OrderLinkID:      "event-gct-001",
					ExecutionID:      "ea89cadf-6471-5fd4-8248-20b470d8478b",
					Symbol:           "ETHUSDT-6NOV26-2200-BELOW",
					BaseCoin:         currency.ETH,
					SettleCoin:       currency.USDT,
					Side:             "Sell",
					ExecutionType:    "Trade",
					ExecutionPrice:   3.4907,
					ExecutionValue:   500,
					ExecutionFee:     22.5,
					ExecutionFeeRate: 0.045,
					OrderPrice:       3.4907,
					FeeCoin:          currency.USDT,
					TransactionTime:  types.Time(time.UnixMilli(1791008160374)),
					TimeInForce:      "GTC",
					CrossSequence:    101174421,
					ECPayout:         1745.35,
					SettleTime:       types.Time(time.Unix(1793952000, 0)),
					ECOrderValue:     500,
					ECContractType:   "Target",
					ECDirection:      "BELOW",
					ECTargetPrice:    2200,
					ExtraFees:        "[{\"feeCoin\":\"USDT\",\"feeType\":\"GST\",\"subFeeType\":\"IND_GST\",\"feeRate\":\"0.0000675\",\"fee\":\"0.003375\"}]",
				},
			},
			Cursor: "execId%3Dea89cadf-6471-5fd4-8248-20b470d8478b",
		}
		assert.Equal(t, exp, result, "GetEventTradeHistory should decode every field")
	}
	result, err = e.GetEventTradeHistory(t.Context(), &EventTradeHistoryRequest{Symbol: eventTestRangeSymbol, OrderID: eventTestRangeOrderID, OrderLinkID: "event-gct-002"})
	require.NoError(t, err, "GetEventTradeHistory must not error for a Range contract")
	if mockTests {
		exp := &EventTradeHistoryResponse{
			List: []EventTrade{
				{
					OrderID:          "9c1d6f10-3b9a-4d7e-a0c2-5e8f7b2d4a61",
					OrderLinkID:      "event-gct-002",
					ExecutionID:      "3f6a1c2e-9d8b-5e7f-a1b2-c3d4e5f60718",
					Symbol:           "ETHUSDT-10OCT26-2580-2640-OUT",
					BaseCoin:         currency.ETH,
					SettleCoin:       currency.USDT,
					Side:             "Sell",
					ExecutionType:    "Trade",
					ExecutionPrice:   1.18,
					ExecutionValue:   200,
					ExecutionFee:     9,
					ExecutionFeeRate: 0.045,
					OrderPrice:       1.18,
					FeeCoin:          currency.USDT,
					TransactionTime:  types.Time(time.UnixMilli(1791008195002)),
					TimeInForce:      "GTC",
					CrossSequence:    101174788,
					ECPayout:         236,
					SettleTime:       types.Time(time.Unix(1791619200, 0)),
					ECOrderValue:     1000,
					ECContractType:   "Range",
					ECDirection:      "RANGE_OUT",
					ECLowerBound:     2580,
					ECUpperBound:     2640,
				},
			},
		}
		assert.Equal(t, exp, result, "GetEventTradeHistory should decode every field of case 2")
	}
	result, err = e.GetEventTradeHistory(t.Context(), &EventTradeHistoryRequest{Symbol: eventTestUpDownSymbol})
	require.NoError(t, err, "GetEventTradeHistory must not error for an UpDown contract")
	if mockTests {
		exp := &EventTradeHistoryResponse{
			List: []EventTrade{
				{
					OrderID:          "0f2b7c55-6e1d-4b8a-9d3f-2a4c6e8b0d12",
					ExecutionID:      "8b9c0d1e-2f3a-5b4c-8d6e-7f8091a2b3c4",
					Symbol:           "BTCUSDT-15MIN-DOWN",
					BaseCoin:         currency.BTC,
					SettleCoin:       currency.USDT,
					Side:             "Sell",
					ExecutionType:    "Trade",
					ExecutionPrice:   1.95,
					ExecutionValue:   50,
					ExecutionFee:     2.25,
					ExecutionFeeRate: 0.045,
					OrderPrice:       1.95,
					FeeCoin:          currency.USDT,
					TransactionTime:  types.Time(time.UnixMilli(1791008150631)),
					TimeInForce:      "IOC",
					CrossSequence:    1307,
					ECPayout:         97.5,
					ECDurationWindow: 900,
					SettleTime:       types.Time(time.Unix(1791009000, 0)),
					ECOrderValue:     50,
					ECContractType:   "UpDown",
					ECDirection:      "DOWN",
				},
			},
		}
		assert.Equal(t, exp, result, "GetEventTradeHistory should decode every field of case 3")
	}
}

func TestSubmitEventQuote(t *testing.T) {
	t.Parallel()
	_, err := e.SubmitEventQuote(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "SubmitEventQuote must reject a nil request")
	_, err = e.SubmitEventQuote(t.Context(), &SubmitEventQuoteRequest{OrderLinkID: "event-gct-001", GrossPayoutRatio: 3.4907, Amount: 500})
	require.ErrorIs(t, err, errSymbolMissing, "SubmitEventQuote must reject an empty symbol")
	_, err = e.SubmitEventQuote(t.Context(), &SubmitEventQuoteRequest{Symbol: eventTestTargetSymbol, GrossPayoutRatio: 3.4907, Amount: 500})
	require.ErrorIs(t, err, errOrderLinkIDMissing, "SubmitEventQuote must reject an empty order link ID")
	_, err = e.SubmitEventQuote(t.Context(), &SubmitEventQuoteRequest{Symbol: eventTestTargetSymbol, OrderLinkID: "event-gct-001", Amount: 500})
	require.ErrorIs(t, err, errInvalidGrossPayoutRatio, "SubmitEventQuote must reject a zero gross payout ratio")
	_, err = e.SubmitEventQuote(t.Context(), &SubmitEventQuoteRequest{Symbol: eventTestTargetSymbol, OrderLinkID: "event-gct-001", GrossPayoutRatio: 3.4907})
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "SubmitEventQuote must reject a zero amount")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.SubmitEventQuote(t.Context(), &SubmitEventQuoteRequest{Symbol: eventTestTargetSymbol, OrderLinkID: "event-gct-001", GrossPayoutRatio: 3.4907, Amount: 500})
	require.NoError(t, err, "SubmitEventQuote must not error")
	if mockTests {
		exp := &EventQuoteResponse{OrderLinkID: "event-gct-001", OrderID: "746a6c87-a6dd-4008-8889-37e694046681"}
		assert.Equal(t, exp, result, "SubmitEventQuote should decode every field")
	}
}

func TestCancelEventQuote(t *testing.T) {
	t.Parallel()
	_, err := e.CancelEventQuote(t.Context(), "", "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelEventQuote must reject a request without an order ID or order link ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelEventQuote(t.Context(), eventTestOrderID, "event-gct-001")
	require.NoError(t, err, "CancelEventQuote must not error")
	if mockTests {
		exp := &EventQuoteResponse{OrderLinkID: "event-gct-001", OrderID: "746a6c87-a6dd-4008-8889-37e694046681"}
		assert.Equal(t, exp, result, "CancelEventQuote should decode every field")
	}
}
