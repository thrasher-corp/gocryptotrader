package kraken

import (
	"net/http"
	"net/url"
	"slices"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// The orders, trades and ledger entries the account data fixtures serve, which several endpoints return
var (
	accountDataStopLossOrder = OrderInfo{
		ReferralOrderID: "OB5VMB-B4U2U-DK2WRW",
		UserReference:   45326,
		Status:          "open",
		OpenTime:        types.Time(time.UnixMicro(1780582233729133)),
		StartTime:       types.Time(time.Unix(1780582260, 0)),
		ExpireTime:      types.Time(time.Unix(1780668660, 0)),
		Description: OrderDescription{
			Pair:             "XBTUSD",
			Side:             "buy",
			OrderType:        "stop-loss-limit",
			Price:            OrderPrice{Value: 30010},
			SecondaryPrice:   OrderPrice{Value: 30020},
			Leverage:         "5:1",
			Summary:          "buy 1.25000000 XBTUSD @ stop loss 30010.0 -> limit 30020.0 with 5:1 leverage",
			ConditionalClose: "close position @ stop loss 28000.0 -> limit 27900.0",
			AssetClass:       "forex",
		},
		TimeInForce:     "gtd",
		Volume:          1.25,
		VolumeExecuted:  0.375,
		Cost:            11254.6875,
		Fee:             29.26219,
		AveragePrice:    30012.5,
		StopPrice:       30010,
		LimitPrice:      30020,
		ExternalOrderID: "EXT-8812-AB",
		OriginalOrderID: "OK3SN7-GFU3V-6E3ZBM",
		ReduceOnly:      true,
		Trigger:         "index",
		Margin:          true,
		Miscellaneous:   "stopped,partial,amended",
		SenderSubID:     "desk-7",
		OrderFlags:      "fciq",
		TradeIDs:        []string{"TCCCTY-WE2O6-P3NB37", "TJUW2K-FLX2N-AR2FLU"},
	}
	accountDataIcebergOrder = OrderInfo{
		ClientOrderID: "6d1b345e-2821-40e2-ad83-4ecb18a06876",
		Status:        "open",
		OpenTime:      types.Time(time.UnixMicro(1780582311512449)),
		Description: OrderDescription{
			Pair:       "ETHUSD",
			Side:       "sell",
			OrderType:  "iceberg",
			Price:      OrderPrice{Value: 2650.5},
			Leverage:   "none",
			Summary:    "sell 10.00000000 ETHUSD @ limit 2650.50",
			AssetClass: "forex",
		},
		TimeInForce:            "gtc",
		Volume:                 10,
		VolumeExecuted:         2,
		Cost:                   5301.4,
		Fee:                    8.48224,
		AveragePrice:           2650.7,
		DisplayVolume:          1,
		DisplayVolumeRemaining: 0.4,
		ExternalOrderID:        "EXT-8813-CD",
		Miscellaneous:          "partial",
		OrderFlags:             "post,fcib",
		TradeIDs:               []string{"THVRQM-33VKH-UCI7BS"},
	}
	accountDataCancelledOrder = OrderInfo{
		ReferralOrderID: "OSGGLD-7L567-4KBIUT",
		UserReference:   36493663,
		Status:          "canceled",
		OpenTime:        types.Time(time.UnixMicro(1688148493770800)),
		StartTime:       types.Time(time.Unix(1688148500, 0)),
		ExpireTime:      types.Time(time.Unix(1688234893, 0)),
		Description: OrderDescription{
			Pair:             "XBTGBP",
			Side:             "sell",
			OrderType:        "take-profit-limit",
			Price:            OrderPrice{Value: 27743},
			SecondaryPrice:   OrderPrice{Value: 27740},
			Leverage:         "3:1",
			Summary:          "sell 0.00100000 XBTGBP @ take profit 27743.0 -> limit 27740.0 with 3:1 leverage",
			ConditionalClose: "close position @ limit 26000.0",
			AssetClass:       "forex",
		},
		TimeInForce:     "gtd",
		Volume:          0.001,
		VolumeExecuted:  0.0004,
		Cost:            11.097,
		Fee:             0.02885,
		AveragePrice:    27742.5,
		StopPrice:       27743,
		LimitPrice:      27740,
		ExternalOrderID: "EXT-7701-EF",
		OriginalOrderID: "OTI672-HJFAO-XOIPPK",
		ReduceOnly:      true,
		Trigger:         "index",
		Margin:          true,
		Miscellaneous:   "touched,partial",
		SenderSubID:     "desk-3",
		OrderFlags:      "fciq",
		TradeIDs:        []string{"TZX2WP-XSEOP-FP7WYR"},
		CloseTime:       types.Time(time.UnixMicro(1688148610048200)),
		Reason:          "User requested",
	}
	accountDataFilledOrder = OrderInfo{
		ClientOrderID: "arb-20240509-00010",
		Status:        "closed",
		OpenTime:      types.Time(time.UnixMicro(1688592012231700)),
		Description: OrderDescription{
			Pair:       "XBTUSD",
			Side:       "buy",
			OrderType:  "iceberg",
			Price:      OrderPrice{Value: 30000},
			Leverage:   "none",
			Summary:    "buy 0.25000000 XBTUSD @ limit 30000.0",
			AssetClass: "forex",
		},
		TimeInForce:     "gtc",
		Volume:          0.25,
		VolumeExecuted:  0.25,
		Cost:            7499.5,
		Fee:             7.4995,
		AveragePrice:    29998,
		DisplayVolume:   0.05,
		ExternalOrderID: "EXT-7702-GH",
		SenderSubID:     "desk-4",
		OrderFlags:      "post,fcib",
		TradeIDs:        []string{"TJUW2K-FLX2N-AR2FLU", "TCWJEG-FL4SZ-3FKGH6"},
		CloseTime:       types.Time(time.UnixMicro(1688592012233500)),
	}
	accountDataClosingTrade = TradeInfo{
		OrderID:             "OQCLML-BW3P3-BUCMWZ",
		PositionID:          "TF5GVO-T7ZZ2-6NBKBI",
		Pair:                "XXBTZUSD",
		Time:                types.Time(time.UnixMicro(1688667796880200)),
		Side:                "buy",
		OrderType:           "limit",
		Price:               30010,
		Cost:                600.2,
		Fee:                 1.56052,
		Volume:              0.02,
		Margin:              120.04,
		Leverage:            5,
		Miscellaneous:       "closing",
		LedgerIDs:           []string{"LZWWII-S4VAD-FTRBPE", "LTKOJ5-JWDZ4-3EQOER"},
		TradeID:             40274859,
		Maker:               true,
		AssetClass:          "forex",
		ExternalExecutionID: "EXEC-55120",
		TradeOrderType:      "limit",
	}
	accountDataOpeningTrade = TradeInfo{
		OrderID:             "OH76VO-UKWAD-PSBDX6",
		PositionID:          "TKH2SE-M7IF5-CFI7LT",
		Pair:                "XXBTZUSD",
		Time:                types.Time(time.UnixMicro(1688667769639600)),
		Side:                "sell",
		OrderType:           "stop-loss",
		Price:               29950.5,
		Cost:                599.01,
		Fee:                 1.55743,
		Volume:              0.02,
		Margin:              199.67,
		Leverage:            3,
		LedgerIDs:           []string{"LMKZCZ-Z3GVL-CXKK4H", "L4UESK-KG3EQ-UFO4T5"},
		TradeID:             39482674,
		AssetClass:          "forex",
		ExternalExecutionID: "EXEC-55121",
		TradeOrderType:      "market",
		PositionStatus:      "open",
		ClosedAveragePrice:  29960.1,
		ClosedCost:          299.601,
		ClosedFee:           0.77896,
		ClosedVolume:        0.01,
		ClosedMargin:        99.867,
		NetPNL:              -12.45,
		ClosingTradeIDs:     []string{"TJUW2K-FLX2N-AR2FLU"},
	}
	accountDataTradeLedgerEntry = LedgerEntry{
		ReferenceID: "THVRQM-33VKH-UCI7BS",
		Time:        types.Time(time.UnixMicro(1688667796880200)),
		Type:        "trade",
		AssetClass:  "currency",
		Asset:       currency.ZUSD,
		Amount:      -600.2,
		Fee:         1.5605,
		Balance:     52732.1132,
	}
	accountDataTransferLedgerEntry = LedgerEntry{
		ReferenceID: "FTQcuak-V6Za8qrWnhzTx67yYHz8Tg",
		Time:        types.Time(time.UnixMicro(1688444262888800)),
		Type:        "transfer",
		Subtype:     "spotfromfutures",
		AssetClass:  "currency",
		Asset:       currency.XXBT,
		Amount:      0.5,
		Fee:         0.00005,
		Balance:     1.7435,
	}
)

func TestGetAccountBalance(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAccountBalance(t.Context(), &BalanceRequest{RebaseMultiplier: "base"})
	require.NoError(t, err, "GetAccountBalance must not error")
	if mockTests {
		exp := map[string]types.Number{"ZUSD": 171288.6158, "XXBT": 1011.19088779, "ETH2.S": 198.39708, "USD.M": 1213029.278}
		assert.Equal(t, exp, result, "GetAccountBalance should decode every field")
		return
	}
	assert.NotNil(t, result, "GetAccountBalance should return balances")
}

func TestGetExtendedBalance(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetExtendedBalance(t.Context(), &BalanceRequest{RebaseMultiplier: "rebased"})
	require.NoError(t, err, "GetExtendedBalance must not error")
	if mockTests {
		exp := map[string]ExtendedBalance{
			"ZUSD": {Balance: 25435.21, Credit: 5000, CreditUsed: 1200.5, HoldTrade: 8249.76},
			"XXBT": {Balance: 1.2435, HoldTrade: 0.8423},
		}
		assert.Equal(t, exp, result, "GetExtendedBalance should decode every field")
		return
	}
	assert.NotNil(t, result, "GetExtendedBalance should return balances")
}

func TestGetCreditLines(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCreditLines(t.Context(), &BalanceRequest{RebaseMultiplier: "base"})
	require.NoError(t, err, "GetCreditLines must not error")
	if mockTests {
		exp := &CreditLinesResponse{
			AssetDetails: map[string]CreditLineAsset{
				"USD": {Balance: 1000.5, HoldTrade: 120.25, Credit: 50000, CreditUsed: 12500, CollateralValue: 1, RolloverRate: 0.0055, ReserveRate: 0.0038},
				"EUR": {Balance: 500.25, HoldTrade: 100, Credit: 25000, CreditUsed: 5000, CollateralValue: 0.99, RolloverRate: 0.006, ReserveRate: 0.004},
			},
			LimitsMonitor: CreditLimitsMonitor{
				TotalCreditUSD:          100000,
				TotalCreditUsedUSD:      25000,
				TotalCollateralValueUSD: 150000,
				EquityUSD:               125000,
				OngoingBalance:          1.5,
				DebtToEquity:            0.2,
			},
		}
		assert.Equal(t, exp, result, "GetCreditLines should decode every field")
		_, err = e.GetCreditLines(t.Context(), &BalanceRequest{RebaseMultiplier: "rebased"})
		assert.ErrorIs(t, err, common.ErrNoResponse, "GetCreditLines should return a null result as no response")
		return
	}
	assert.NotNil(t, result.AssetDetails, "GetCreditLines should return asset details")
}

func TestGetTradeBalance(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetTradeBalance(t.Context(), &TradeBalanceRequest{Asset: currency.ZUSD, RebaseMultiplier: "base"})
	require.NoError(t, err, "GetTradeBalance must not error")
	if mockTests {
		exp := &TradeBalanceResponse{
			EquivalentBalance:   1101.3425,
			TradeBalance:        392.2264,
			MarginAmount:        7.0354,
			UnrealisedNetPNL:    -10.0232,
			CostBasis:           21.1063,
			Valuation:           31.1297,
			Equity:              382.2032,
			FreeMargin:          375.1678,
			FreeMarginForOrders: 368.441,
			MarginLevel:         5432.57,
			UnexecutedValue:     15,
		}
		assert.Equal(t, exp, result, "GetTradeBalance should decode every field")
		return
	}
	assert.GreaterOrEqual(t, result.EquivalentBalance.Float64(), 0.0, "GetTradeBalance should return a balance")
}

func TestGetOpenOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetOpenOrders(t.Context(), &OpenOrdersRequest{WithCursor: true, Limit: 101})
	require.ErrorIs(t, err, errInvalidLimit, "GetOpenOrders must reject a page limit above 100")
	_, err = e.GetOpenOrders(t.Context(), &OpenOrdersRequest{Cursor: "gqNrZXm-T3BlbjI"})
	require.ErrorIs(t, err, errCursorWithoutPaging, "GetOpenOrders must reject a cursor without cursor pagination")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name string
		req  *OpenOrdersRequest
		exp  *OpenOrdersResponse
	}{
		{
			name: "cursor page",
			req: &OpenOrdersRequest{
				IncludeTrades:             true,
				UserReference:             45326,
				WithoutTakerConsolidation: true,
				WithCursor:                true,
				Cursor:                    "gqNrZXm-T3BlbjI",
				Limit:                     1,
				RebaseMultiplier:          "base",
			},
			exp: &OpenOrdersResponse{
				Open:   map[string]OrderInfo{"OQCLML-BW3P3-BUCMWZ": accountDataStopLossOrder},
				Cursor: PageCursor{Next: "gqNrZXm-T3BlbjM"},
			},
		},
		{
			name: "client order ID",
			req:  &OpenOrdersRequest{ClientOrderID: "6d1b345e-2821-40e2-ad83-4ecb18a06876", Limit: 10},
			exp:  &OpenOrdersResponse{Open: map[string]OrderInfo{"OHYO67-6LP66-HMQ437": accountDataIcebergOrder}},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetOpenOrders(t.Context(), tc.req)
			require.NoError(t, err, "GetOpenOrders must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetOpenOrders should decode every field")
				return
			}
			assert.NotNil(t, result.Open, "GetOpenOrders should return open orders")
		})
	}
}

func TestGetClosedOrders(t *testing.T) {
	t.Parallel()
	start, end := time.Unix(1688140000, 0), time.Unix(1688150000, 0)
	_, err := e.GetClosedOrders(t.Context(), &ClosedOrdersRequest{Start: end, End: start})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetClosedOrders must reject a reversed range")
	_, err = e.GetClosedOrders(t.Context(), &ClosedOrdersRequest{Start: start, StartOrderID: "O37652-RJWRT-IMO74O"})
	require.ErrorIs(t, err, errTimeAndIDBound, "GetClosedOrders must reject a start that is both a time and an ID")
	_, err = e.GetClosedOrders(t.Context(), &ClosedOrdersRequest{Offset: 50, WithCursor: true})
	require.ErrorIs(t, err, errOffsetWithCursor, "GetClosedOrders must reject an offset with cursor pagination")
	_, err = e.GetClosedOrders(t.Context(), &ClosedOrdersRequest{Cursor: "gqNrZXm-Q2xvc2VkMg"})
	require.ErrorIs(t, err, errCursorWithoutPaging, "GetClosedOrders must reject a cursor without cursor pagination")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name string
		req  *ClosedOrdersRequest
		exp  *ClosedOrdersResponse
	}{
		{
			name: "offset page",
			req: &ClosedOrdersRequest{
				IncludeTrades:             true,
				UserReference:             36493663,
				Start:                     start,
				End:                       end,
				Offset:                    50,
				TimeFilter:                "close",
				WithoutTakerConsolidation: true,
				RebaseMultiplier:          "base",
			},
			exp: &ClosedOrdersResponse{Closed: map[string]OrderInfo{"O37652-RJWRT-IMO74O": accountDataCancelledOrder}, Count: 51},
		},
		{
			name: "cursor page",
			req: &ClosedOrdersRequest{
				ClientOrderID: "arb-20240509-00010",
				StartOrderID:  "O37652-RJWRT-IMO74O",
				EndOrderID:    "OMMDB2-FSB6Z-7W3HPO",
				WithCursor:    true,
				Cursor:        "gqNrZXm-Q2xvc2VkMg",
				WithoutCount:  true,
			},
			exp: &ClosedOrdersResponse{
				Closed: map[string]OrderInfo{"OMMDB2-FSB6Z-7W3HPO": accountDataFilledOrder},
				Cursor: PageCursor{Next: "gqNrZXm-Q2xvc2VkMw"},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetClosedOrders(t.Context(), tc.req)
			require.NoError(t, err, "GetClosedOrders must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetClosedOrders should decode every field")
				return
			}
			assert.NotNil(t, result.Closed, "GetClosedOrders should return closed orders")
		})
	}
}

func TestQueryOrdersInfo(t *testing.T) {
	t.Parallel()
	_, err := e.QueryOrdersInfo(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "QueryOrdersInfo must reject a nil request")
	_, err = e.QueryOrdersInfo(t.Context(), &QueryOrdersRequest{})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "QueryOrdersInfo must reject a request without order IDs")
	_, err = e.QueryOrdersInfo(t.Context(), &QueryOrdersRequest{OrderIDs: []string{"OQCLML-BW3P3-BUCMWZ", ""}})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "QueryOrdersInfo must reject an empty order ID")
	_, err = e.QueryOrdersInfo(t.Context(), &QueryOrdersRequest{OrderIDs: slices.Repeat([]string{"OQCLML-BW3P3-BUCMWZ"}, 51)})
	require.ErrorIs(t, err, errTooManyIDs, "QueryOrdersInfo must reject more than 50 order IDs")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name string
		req  *QueryOrdersRequest
		exp  map[string]OrderInfo
	}{
		{
			name: "open and closed orders",
			req: &QueryOrdersRequest{
				OrderIDs:                  []string{"OQCLML-BW3P3-BUCMWZ", "O37652-RJWRT-IMO74O"},
				IncludeTrades:             true,
				WithoutTakerConsolidation: true,
				RebaseMultiplier:          "base",
			},
			exp: map[string]OrderInfo{"OQCLML-BW3P3-BUCMWZ": accountDataStopLossOrder, "O37652-RJWRT-IMO74O": accountDataCancelledOrder},
		},
		{
			name: "user reference",
			req:  &QueryOrdersRequest{OrderIDs: []string{"OQCLML-BW3P3-BUCMWZ", "O37652-RJWRT-IMO74O"}, UserReference: 36493663},
			exp:  map[string]OrderInfo{"O37652-RJWRT-IMO74O": accountDataCancelledOrder},
		},
		{
			name: "client order ID",
			req:  &QueryOrdersRequest{OrderIDs: []string{"OHYO67-6LP66-HMQ437", "OMMDB2-FSB6Z-7W3HPO"}, ClientOrderID: "arb-20240509-00010"},
			exp:  map[string]OrderInfo{"OMMDB2-FSB6Z-7W3HPO": accountDataFilledOrder},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.QueryOrdersInfo(t.Context(), tc.req)
			require.NoError(t, err, "QueryOrdersInfo must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "QueryOrdersInfo should decode every field")
				return
			}
			assert.NotNil(t, result, "QueryOrdersInfo should return orders")
		})
	}
}

func TestGetOrderAmends(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderAmends(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetOrderAmends must reject a nil request")
	_, err = e.GetOrderAmends(t.Context(), &OrderAmendsRequest{})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetOrderAmends must reject an empty order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetOrderAmends(t.Context(), &OrderAmendsRequest{OrderID: "OQCLML-BW3P3-BUCMWZ", RebaseMultiplier: "base"})
	require.NoError(t, err, "GetOrderAmends must not error")
	if mockTests {
		exp := &OrderAmendsResponse{
			Count: 3,
			Amends: []OrderAmend{
				{
					AmendID:           "TSUN4B-EX2XN-WQ6GKG",
					AmendType:         "original",
					OrderQuantity:     0.01,
					DisplayQuantity:   0.005,
					RemainingQuantity: 0.01,
					LimitPrice:        61032.8,
					TriggerPrice:      61000,
					Reason:            "Order placed",
					PostOnly:          true,
					Timestamp:         types.Time(time.Unix(0, 1724158070287558000)),
				},
				{
					AmendID:           "TF6VAW-VUWMX-6SXTCH",
					AmendType:         "user",
					OrderQuantity:     0.02,
					DisplayQuantity:   0.006,
					RemainingQuantity: 0.015,
					LimitPrice:        61032.7,
					TriggerPrice:      61010,
					Reason:            "User requested",
					Timestamp:         types.Time(time.Unix(0, 1724158076936755700)),
				},
				{
					AmendID:           "TUMY4K-E4MPE-CSL2N3",
					AmendType:         "restated",
					OrderQuantity:     0.02,
					DisplayQuantity:   0.006,
					RemainingQuantity: 0.014,
					LimitPrice:        61032.6,
					TriggerPrice:      61010,
					Reason:            "Order restated",
					Timestamp:         types.Time(time.Unix(0, 1724158214879660000)),
				},
			},
		}
		assert.Equal(t, exp, result, "GetOrderAmends should decode every field")
		return
	}
	assert.NotEmpty(t, result.Amends, "GetOrderAmends should return the order as entered")
}

func TestGetTradesHistory(t *testing.T) {
	t.Parallel()
	start, end := time.Unix(1688600000, 0), time.Unix(1688700000, 0)
	_, err := e.GetTradesHistory(t.Context(), &TradesHistoryRequest{End: end, EndTradeID: "THVRQM-33VKH-UCI7BS"})
	require.ErrorIs(t, err, errTimeAndIDBound, "GetTradesHistory must reject an end that is both a time and an ID")
	_, err = e.GetTradesHistory(t.Context(), &TradesHistoryRequest{Cursor: "gqNrZXm-VHJhZGVzMg"})
	require.ErrorIs(t, err, errCursorWithoutPaging, "GetTradesHistory must reject a cursor without cursor pagination")
	for _, pair := range partialTestPairs {
		_, err = e.GetTradesHistory(t.Context(), &TradesHistoryRequest{Pair: pair})
		require.ErrorIsf(t, err, currency.ErrCurrencyCodeEmpty, "GetTradesHistory must reject %q, which lacks a base or a quote", pair)
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name string
		req  *TradesHistoryRequest
		exp  *TradesHistoryResponse
	}{
		{
			name: "offset page",
			req: &TradesHistoryRequest{
				TradeType:                 "all",
				IncludeTrades:             true,
				Start:                     start,
				End:                       end,
				Offset:                    2,
				WithoutTakerConsolidation: true,
				IncludeLedgers:            true,
				RebaseMultiplier:          "base",
				AssetClass:                "forex",
				Pair:                      spotTestPair,
				Limit:                     2,
			},
			exp: &TradesHistoryResponse{
				Count:  4,
				Trades: map[string]TradeInfo{"THVRQM-33VKH-UCI7BS": accountDataClosingTrade, "TCWJEG-FL4SZ-3FKGH6": accountDataOpeningTrade},
			},
		},
		{
			name: "cursor page",
			req: &TradesHistoryRequest{
				StartTradeID: "TKH2SE-M7IF5-CFI7LT",
				EndTradeID:   "THVRQM-33VKH-UCI7BS",
				WithoutCount: true,
				WithCursor:   true,
				Cursor:       "gqNrZXm-VHJhZGVzMg",
			},
			exp: &TradesHistoryResponse{
				Trades: map[string]TradeInfo{"TCWJEG-FL4SZ-3FKGH6": accountDataOpeningTrade},
				Cursor: PageCursor{Next: "gqNrZXm-VHJhZGVzMw"},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetTradesHistory(t.Context(), tc.req)
			require.NoError(t, err, "GetTradesHistory must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetTradesHistory should decode every field")
				return
			}
			assert.NotNil(t, result.Trades, "GetTradesHistory should return trades")
		})
	}
}

func TestQueryTradesInfo(t *testing.T) {
	t.Parallel()
	_, err := e.QueryTradesInfo(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "QueryTradesInfo must reject a nil request")
	_, err = e.QueryTradesInfo(t.Context(), &QueryTradesRequest{})
	require.ErrorIs(t, err, errTradeIDEmpty, "QueryTradesInfo must reject a request without trade IDs")
	_, err = e.QueryTradesInfo(t.Context(), &QueryTradesRequest{TradeIDs: slices.Repeat([]string{"THVRQM-33VKH-UCI7BS"}, 21)})
	require.ErrorIs(t, err, errTooManyIDs, "QueryTradesInfo must reject more than 20 trade IDs")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.QueryTradesInfo(t.Context(), &QueryTradesRequest{
		TradeIDs:         []string{"THVRQM-33VKH-UCI7BS", "TCWJEG-FL4SZ-3FKGH6"},
		IncludeTrades:    true,
		IncludeLedgers:   true,
		RebaseMultiplier: "base",
	})
	require.NoError(t, err, "QueryTradesInfo must not error")
	if mockTests {
		exp := map[string]TradeInfo{"THVRQM-33VKH-UCI7BS": accountDataClosingTrade, "TCWJEG-FL4SZ-3FKGH6": accountDataOpeningTrade}
		assert.Equal(t, exp, result, "QueryTradesInfo should decode every field")
		return
	}
	assert.NotNil(t, result, "QueryTradesInfo should return trades")
}

func TestGetOpenPositions(t *testing.T) {
	t.Parallel()
	_, err := e.GetOpenPositions(t.Context(), &OpenPositionsRequest{TradeIDs: []string{""}})
	require.ErrorIs(t, err, errTradeIDEmpty, "GetOpenPositions must reject an empty trade ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetOpenPositions(t.Context(), &OpenPositionsRequest{
		TradeIDs:            []string{"TF5GVO-T7ZZ2-6NBKBI", "T24DOR-TAFLM-ID3NYP"},
		IncludeCalculations: true,
		RebaseMultiplier:    "base",
	})
	require.NoError(t, err, "GetOpenPositions must not error")
	if mockTests {
		exp := map[string]OpenPosition{
			"TF5GVO-T7ZZ2-6NBKBI": {
				OrderID:          "OLWNFG-LLH4R-D6SFFP",
				AssetClass:       "forex",
				Status:           "open",
				Pair:             "XXBTZUSD",
				Time:             types.Time(time.UnixMicro(1605280097829400)),
				Side:             "buy",
				OrderType:        "limit",
				Cost:             104610.52842,
				Fee:              289.06565,
				Volume:           8.82412861,
				VolumeClosed:     0.202,
				Margin:           20922.10568,
				Value:            258797.5,
				UnrealisedPNL:    154186.9728,
				Terms:            "0.0100% per 4 hours",
				NextRolloverTime: types.Time(time.Unix(1616672637, 0)),
			},
			"T24DOR-TAFLM-ID3NYP": {
				OrderID:          "OIVYGZ-M5EHU-ZRUQXX",
				AssetClass:       "forex",
				Status:           "open",
				Pair:             "XETHZUSD",
				Time:             types.Time(time.UnixMicro(1607943827317200)),
				Side:             "sell",
				OrderType:        "market",
				Cost:             4575.2,
				Fee:              11.89552,
				Volume:           2,
				VolumeClosed:     0.5,
				Margin:           915.04,
				Value:            3450,
				UnrealisedPNL:    -187.65,
				Terms:            "0.0200% per 4 hours",
				NextRolloverTime: types.Time(time.Unix(1616672700, 0)),
				Miscellaneous:    "partial",
				OrderFlags:       "fciq",
			},
		}
		assert.Equal(t, exp, result, "GetOpenPositions should decode every field")
		return
	}
	assert.NotNil(t, result, "GetOpenPositions should return positions")
}

func TestGetConsolidatedOpenPositions(t *testing.T) {
	t.Parallel()
	_, err := e.GetConsolidatedOpenPositions(t.Context(), &OpenPositionsRequest{TradeIDs: []string{""}})
	require.ErrorIs(t, err, errTradeIDEmpty, "GetConsolidatedOpenPositions must reject an empty trade ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetConsolidatedOpenPositions(t.Context(), &OpenPositionsRequest{
		TradeIDs:            []string{"TF5GVO-T7ZZ2-6NBKBI", "T24DOR-TAFLM-ID3NYP"},
		IncludeCalculations: true,
		RebaseMultiplier:    "base",
	})
	require.NoError(t, err, "GetConsolidatedOpenPositions must not error")
	if mockTests {
		exp := []ConsolidatedPosition{
			{
				Pair:          "XXBTZUSD",
				AssetClass:    "forex",
				Positions:     2,
				Side:          "buy",
				Leverage:      "5.00",
				Cost:          250367.29698,
				Fee:           624.30622,
				Volume:        16.82412861,
				VolumeClosed:  0.202,
				Margin:        50073.45939,
				Value:         498921.5,
				UnrealisedPNL: 248554.2042,
			},
			{
				Pair:          "XETHZUSD",
				AssetClass:    "forex",
				Positions:     1,
				Side:          "sell",
				Leverage:      "n/a",
				Cost:          4575.2,
				Fee:           11.89552,
				Volume:        2,
				VolumeClosed:  0.5,
				Margin:        915.04,
				Value:         3450,
				UnrealisedPNL: -187.65,
			},
		}
		assert.Equal(t, exp, result, "GetConsolidatedOpenPositions should decode every field")
		return
	}
	assert.NotNil(t, result, "GetConsolidatedOpenPositions should return positions")
}

func TestGetLedgers(t *testing.T) {
	t.Parallel()
	start, end := time.Unix(1688400000, 0), time.Unix(1688700000, 0)
	tokenised := []ClassifiedAsset{{Name: "AAPLx", AssetClass: "tokenized_asset"}}
	_, err := e.GetLedgers(t.Context(), &LedgersRequest{Assets: []currency.Code{currency.XBT}, ClassifiedAssets: tokenised})
	require.ErrorIs(t, err, errFilterFormConflict, "GetLedgers must reject assets with classified assets")
	_, err = e.GetLedgers(t.Context(), &LedgersRequest{Assets: []currency.Code{currency.EMPTYCODE}})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetLedgers must reject an empty asset")
	_, err = e.GetLedgers(t.Context(), &LedgersRequest{ClassifiedAssets: []ClassifiedAsset{{AssetClass: "tokenized_asset"}}})
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetLedgers must reject a classified asset without a name")
	_, err = e.GetLedgers(t.Context(), &LedgersRequest{ClassifiedAssets: []ClassifiedAsset{{Name: "AAPLx"}}})
	require.ErrorIs(t, err, errAssetClassEmpty, "GetLedgers must reject a classified asset without a class")
	_, err = e.GetLedgers(t.Context(), &LedgersRequest{Start: start, StartLedgerID: "LTKOJ5-JWDZ4-3EQOER"})
	require.ErrorIs(t, err, errTimeAndIDBound, "GetLedgers must reject a start that is both a time and an ID")
	_, err = e.GetLedgers(t.Context(), &LedgersRequest{Offset: 2, WithCursor: true})
	require.ErrorIs(t, err, errOffsetWithCursor, "GetLedgers must reject an offset with cursor pagination")

	// The VCR server cannot match a JSON array in a request body, so a local server checks the classified form
	ex := newHTTPTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&body), "Decode should not error")
		assert.Contains(t, body, "nonce", "body should carry the nonce")
		delete(body, "nonce")
		exp := map[string]any{"asset": []any{map[string]any{"asset": "AAPLx", "aclass": "tokenized_asset"}}}
		assert.Equal(t, exp, body, "body should carry the classified assets as a list of objects")
		_, _ = w.Write([]byte(`{"error":[],"result":{"ledger":{"LTKOJ5-JWDZ4-3EQOER":{"refid":"TQ6ZXA-PLM2N-RS6TYU","time":1782587828.681409,"type":"trade","subtype":"","aclass":"tokenized_asset","asset":"AAPLx","amount":"0.50000000","fee":"0.00100000","balance":"2.75000000"}},"count":1}}`))
	})
	result, err := ex.GetLedgers(t.Context(), &LedgersRequest{ClassifiedAssets: tokenised})
	require.NoError(t, err, "GetLedgers must not error for classified assets")
	exp := &LedgersResponse{
		Ledger: map[string]LedgerEntry{
			"LTKOJ5-JWDZ4-3EQOER": {
				ReferenceID: "TQ6ZXA-PLM2N-RS6TYU",
				Time:        types.Time(time.UnixMicro(1782587828681409)),
				Type:        "trade",
				AssetClass:  "tokenized_asset",
				Asset:       currency.NewCode("AAPLx"),
				Amount:      0.5,
				Fee:         0.001,
				Balance:     2.75,
			},
		},
		Count: 1,
	}
	assert.Equal(t, exp, result, "GetLedgers should decode the classified assets' entries")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name string
		req  *LedgersRequest
		exp  *LedgersResponse
	}{
		{
			name: "offset page",
			req: &LedgersRequest{
				Assets:           []currency.Code{currency.XBT, currency.USD},
				AssetClass:       "currency",
				LedgerType:       "trade",
				Start:            start,
				End:              end,
				Offset:           2,
				RebaseMultiplier: "base",
			},
			exp: &LedgersResponse{Ledger: map[string]LedgerEntry{"L4UESK-KG3EQ-UFO4T5": accountDataTradeLedgerEntry}, Count: 3},
		},
		{
			name: "cursor page",
			req: &LedgersRequest{
				StartLedgerID: "LTKOJ5-JWDZ4-3EQOER",
				EndLedgerID:   "L4UESK-KG3EQ-UFO4T5",
				WithCursor:    true,
				Cursor:        "gqNrZXm-TGVkZ2VyMg",
				WithoutCount:  true,
			},
			exp: &LedgersResponse{
				Ledger: map[string]LedgerEntry{"LMKZCZ-Z3GVL-CXKK4H": accountDataTransferLedgerEntry},
				Cursor: PageCursor{Next: "gqNrZXm-TGVkZ2VyMw"},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetLedgers(t.Context(), tc.req)
			require.NoError(t, err, "GetLedgers must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetLedgers should decode every field")
				return
			}
			assert.NotNil(t, result.Ledger, "GetLedgers should return ledger entries")
		})
	}
}

func TestQueryLedgers(t *testing.T) {
	t.Parallel()
	_, err := e.QueryLedgers(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "QueryLedgers must reject a nil request")
	_, err = e.QueryLedgers(t.Context(), &QueryLedgersRequest{LedgerIDs: []string{""}})
	require.ErrorIs(t, err, errLedgerIDEmpty, "QueryLedgers must reject an empty ledger ID")
	_, err = e.QueryLedgers(t.Context(), &QueryLedgersRequest{LedgerIDs: slices.Repeat([]string{"L4UESK-KG3EQ-UFO4T5"}, 21)})
	require.ErrorIs(t, err, errTooManyIDs, "QueryLedgers must reject more than 20 ledger IDs")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.QueryLedgers(t.Context(), &QueryLedgersRequest{
		LedgerIDs:        []string{"L4UESK-KG3EQ-UFO4T5", "LTKOJ5-JWDZ4-3EQOER"},
		IncludeTrades:    true,
		RebaseMultiplier: "base",
	})
	require.NoError(t, err, "QueryLedgers must not error")
	if mockTests {
		amountToken, err := types.NewPreciseNumberFromString("50000000")
		require.NoError(t, err, "NewPreciseNumberFromString must not error")
		exp := map[string]LedgerEntry{
			"L4UESK-KG3EQ-UFO4T5": accountDataTradeLedgerEntry,
			"LTKOJ5-JWDZ4-3EQOER": {
				ReferenceID: "TQ6ZXA-PLM2N-RS6TYU",
				Time:        types.Time(time.UnixMicro(1782587828681409)),
				Type:        "trade",
				AssetClass:  "tokenized_asset",
				Asset:       currency.NewCode("AAPLx"),
				Amount:      0.5,
				Fee:         0.001,
				Balance:     2.75,
				AmountToken: amountToken,
			},
		}
		assert.Equal(t, exp, result, "QueryLedgers should decode every field")
		return
	}
	assert.NotNil(t, result, "QueryLedgers should return ledger entries")
}

func TestGetTradeVolume(t *testing.T) {
	t.Parallel()
	classified := []ClassifiedAsset{{Name: "AAPLxUSD", AssetClass: "equity_pair"}}
	_, err := e.GetTradeVolume(t.Context(), &TradeVolumeRequest{Pairs: currency.Pairs{spotTestPair}, ClassifiedPairs: classified})
	require.ErrorIs(t, err, errFilterFormConflict, "GetTradeVolume must reject pairs with classified pairs")
	_, err = e.GetTradeVolume(t.Context(), &TradeVolumeRequest{Pairs: currency.Pairs{currency.EMPTYPAIR}})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetTradeVolume must reject an empty pair")
	_, err = e.GetTradeVolume(t.Context(), &TradeVolumeRequest{ClassifiedPairs: []ClassifiedAsset{{AssetClass: "equity_pair"}}})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetTradeVolume must reject a classified pair without a name")
	_, err = e.GetTradeVolume(t.Context(), &TradeVolumeRequest{ClassifiedPairs: []ClassifiedAsset{{Name: "AAPLxUSD"}}})
	require.ErrorIs(t, err, errAssetClassEmpty, "GetTradeVolume must reject a classified pair without a class")

	// The VCR server cannot match a JSON array in a request body, so a local server checks the classified form
	ex := newHTTPTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&body), "Decode should not error")
		assert.Contains(t, body, "nonce", "body should carry the nonce")
		delete(body, "nonce")
		exp := map[string]any{"pair": []any{map[string]any{"asset": "AAPLxUSD", "aclass": "equity_pair"}}}
		assert.Equal(t, exp, body, "body should carry the classified pairs as a list of objects")
		_, _ = w.Write([]byte(`{"error":[],"result":{"currency":"ZUSD","asset_class":"currency","volume":"1250.5000","inputs":{"domain_spot_volume_30d":"1250.5000","domain_futures_volume_30d":"0.0000","domain_assets_on_platform":"0.0000"},"fees":{"AAPLxUSD":{"fee":"0.2600","minfee":"0.1000","maxfee":"0.2600","nextfee":"0.2400","tiervolume":"0.0000","nextvolume":"10000.0000"}},"fees_maker":null}}`))
	})
	result, err := ex.GetTradeVolume(t.Context(), &TradeVolumeRequest{ClassifiedPairs: classified})
	require.NoError(t, err, "GetTradeVolume must not error for classified pairs")
	exp := &TradeVolumeResponse{
		Currency:   currency.ZUSD,
		AssetClass: "currency",
		Volume:     1250.5,
		Inputs:     TradeVolumeInputs{DomainSpotVolume30Day: 1250.5},
		Fees:       map[string]TradeVolumeFee{"AAPLxUSD": {Fee: 0.26, MinimumFee: 0.1, MaximumFee: 0.26, NextFee: 0.24, NextVolume: 10000}},
	}
	assert.Equal(t, exp, result, "GetTradeVolume should decode the classified pairs' fees")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err = e.GetTradeVolume(t.Context(), &TradeVolumeRequest{
		Pairs:            currency.Pairs{spotTestPair, currency.NewPair(currency.ETH, currency.USD)},
		FeeInfo:          true,
		FeeSchedule:      true,
		RebaseMultiplier: "base",
	})
	require.NoError(t, err, "GetTradeVolume must not error")
	if mockTests {
		exp := &TradeVolumeResponse{
			Currency:   currency.ZUSD,
			AssetClass: "currency",
			Volume:     200709587.4223,
			Inputs: TradeVolumeInputs{
				DomainSpotVolume30Day:    200709587.4223,
				DomainFuturesVolume30Day: 1500000,
				DomainAssetsOnPlatform:   35000000,
			},
			Fees: map[string]TradeVolumeFee{
				"XXBTZUSD": {
					Fee:               0.1,
					MinimumFee:        0.08,
					MaximumFee:        0.4,
					NextFee:           0.09,
					TierVolume:        100000000,
					TierFuturesVolume: 1000000,
					NextVolume:        250000000,
					NextFuturesVolume: 5000000,
					VolumeOffset:      1250,
				},
				"XETHZUSD": {Fee: 0.12, MinimumFee: 0.08, MaximumFee: 0.4, TierVolume: 50000000},
			},
			FeesMaker: map[string]TradeVolumeFee{
				"XXBTZUSD": {
					Fee:               0.02,
					MaximumFee:        0.25,
					NextFee:           0.01,
					TierVolume:        100000000,
					TierFuturesVolume: 1000000,
					NextVolume:        250000000,
					NextFuturesVolume: 5000000,
					VolumeOffset:      1250,
				},
				"XETHZUSD": {Fee: 0.04, MaximumFee: 0.25, TierVolume: 50000000},
			},
			Subaccounts: []TradeVolumeSubaccount{
				{IIBAN: "AA88 N84G WOAK NMOI", Volume: 120000000},
				{IIBAN: "BB12 K45T ZPQR STUV", Volume: 80709587.4223},
			},
			Schedules: []TradeVolumeFeeSchedule{
				{
					Pair:  "XXBTZUSD",
					Class: "forex",
					Tiers: []TradeVolumeFeeTier{
						{MakerFee: 0.25, TakerFee: 0.4},
						{
							MakerFee:                0.02,
							TakerFee:                0.1,
							MinimumSpotVolume:       100000000,
							MinimumFuturesVolume:    1000000,
							MinimumAssetsOnPlatform: 25000000,
							Active:                  true,
						},
					},
				},
			},
		}
		assert.Equal(t, exp, result, "GetTradeVolume should decode every field")
		return
	}
	assert.NotEmpty(t, result.Fees, "GetTradeVolume should return the requested pairs' fees")
}

func TestRequestExportReport(t *testing.T) {
	t.Parallel()
	start, end := time.Unix(1683556800, 0), time.Unix(1688669085, 0)
	_, err := e.RequestExportReport(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "RequestExportReport must reject a nil request")
	_, err = e.RequestExportReport(t.Context(), &ExportReportRequest{Description: "my_trades_1"})
	require.ErrorIs(t, err, errExportReportTypeEmpty, "RequestExportReport must reject an empty report type")
	_, err = e.RequestExportReport(t.Context(), &ExportReportRequest{Report: "trades"})
	require.ErrorIs(t, err, errExportDescriptionEmpty, "RequestExportReport must reject an empty description")
	_, err = e.RequestExportReport(t.Context(), &ExportReportRequest{Report: "trades", Description: "my_trades_1", Start: end, End: start})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "RequestExportReport must reject a reversed range")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.RequestExportReport(t.Context(), &ExportReportRequest{
		Report:      "trades",
		Format:      "TSV",
		Description: "my_trades_1",
		Fields:      []string{"ordertxid", "time", "price", "cost"},
		Start:       start,
		End:         end,
	})
	require.NoError(t, err, "RequestExportReport must not error")
	if mockTests {
		assert.Equal(t, &ExportReportResponse{ID: "TCJA"}, result, "RequestExportReport should decode every field")
		return
	}
	assert.NotEmpty(t, result.ID, "RequestExportReport should return the report's ID")
}

func TestGetExportReportStatus(t *testing.T) {
	t.Parallel()
	_, err := e.GetExportReportStatus(t.Context(), "")
	require.ErrorIs(t, err, errExportReportTypeEmpty, "GetExportReportStatus must reject an empty report type")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetExportReportStatus(t.Context(), "trades")
	require.NoError(t, err, "GetExportReportStatus must not error")
	if mockTests {
		exp := []ExportReport{
			{
				ID:            "VSKC",
				Description:   "my_trades_1",
				Format:        "CSV",
				Report:        "trades",
				Subtype:       "all",
				Status:        "Processed",
				Flags:         "0",
				Fields:        "all",
				CreatedTime:   types.Time(time.Unix(1688669085, 0)),
				ExpireTime:    types.Time(time.Unix(1688878685, 0)),
				StartTime:     types.Time(time.Unix(1688669090, 0)),
				CompletedTime: types.Time(time.Unix(1688669093, 0)),
				DataStartTime: types.Time(time.Unix(1683556800, 0)),
				DataEndTime:   types.Time(time.Unix(1688669085, 0)),
				AssetClass:    "forex",
				Asset:         "all",
				AssetClasses:  []string{"forex", "tokenized_asset"},
				EndTime:       types.Time(time.Unix(1688669086, 0)),
			},
			{
				ID:              "TCJA",
				Description:     "my_trades_2",
				Format:          "TSV",
				Report:          "trades",
				Subtype:         "all",
				Status:          "Processed",
				Error:           "EExport:Exported size too big",
				Flags:           "1",
				Fields:          "ordertxid,time,price,cost",
				CreatedTime:     types.Time(time.Unix(1688363637, 0)),
				ExpireTime:      types.Time(time.Unix(1688573237, 0)),
				StartTime:       types.Time(time.Unix(1688363650, 0)),
				CompletedTime:   types.Time(time.Unix(1688363664, 0)),
				DataStartTime:   types.Time(time.Unix(1683235200, 0)),
				DataEndTime:     types.Time(time.Unix(1688363637, 0)),
				AssetClass:      "forex",
				Asset:           "all",
				AssetClasses:    []string{"forex"},
				EndTime:         types.Time(time.Unix(1688363638, 0)),
				PendingDeletion: true,
			},
		}
		assert.Equal(t, exp, result, "GetExportReportStatus should decode every field")
		return
	}
	assert.NotNil(t, result, "GetExportReportStatus should return reports")
}

func TestRetrieveDataExport(t *testing.T) {
	t.Parallel()
	_, err := e.RetrieveDataExport(t.Context(), "")
	require.ErrorIs(t, err, errExportIDEmpty, "RetrieveDataExport must reject an empty ID")

	// The VCR server only serves JSON, so a local server serves the archive and the failures
	ex := newHTTPTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/0/private/RetrieveExport", r.URL.Path, "RetrieveDataExport should request the export")
		var body struct {
			ID string `json:"id"`
		}
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&body), "Decode should not error")
		switch body.ID {
		case "VSKC":
			w.Header().Set("Content-Type", "application/octet-stream")
			_, _ = w.Write([]byte("PK\x03\x04report"))
		case "TCJA":
			_, _ = w.Write([]byte(`{"error":["EGeneral:Invalid arguments:id"]}`))
		case "QWER":
			w.WriteHeader(http.StatusServiceUnavailable)
			_, _ = w.Write([]byte(`{"error":["EService:Unavailable"]}`))
		case "ZXCV":
			_, _ = w.Write([]byte(`{"error":[],"result":{"id":"ZXCV"}}`))
		}
	})
	result, err := ex.RetrieveDataExport(t.Context(), "VSKC")
	require.NoError(t, err, "RetrieveDataExport must not error for an archive")
	assert.Equal(t, []byte("PK\x03\x04report"), result, "RetrieveDataExport should return the archive as sent")

	_, err = ex.RetrieveDataExport(t.Context(), "TCJA")
	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr, "RetrieveDataExport must return the error Kraken replied with")
	assert.Equal(t, []string{"EGeneral:Invalid arguments:id"}, apiErr.Errors, "APIError should hold the replied error")

	_, err = ex.RetrieveDataExport(t.Context(), "QWER")
	assert.ErrorIs(t, err, errAPIResponse, "RetrieveDataExport should return the error of an error status")
	assert.ErrorIs(t, err, request.ErrBadStatus, "RetrieveDataExport should keep the status error")

	_, err = ex.RetrieveDataExport(t.Context(), "ZXCV")
	assert.ErrorIs(t, err, errExportNotArchive, "RetrieveDataExport should reject a JSON reply without an error")

	_, err = ex.RetrieveDataExport(t.Context(), "ASDF")
	assert.ErrorIs(t, err, common.ErrNoResponse, "RetrieveDataExport should reject an empty reply")
}

func TestDeleteExportReport(t *testing.T) {
	t.Parallel()
	_, err := e.DeleteExportReport(t.Context(), "", "delete")
	require.ErrorIs(t, err, errExportIDEmpty, "DeleteExportReport must reject an empty ID")
	_, err = e.DeleteExportReport(t.Context(), "VSKC", "")
	require.ErrorIs(t, err, errExportRemovalTypeEmpty, "DeleteExportReport must reject an empty removal type")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tc := range []struct {
		id, removalType string
		exp             *DeleteExportReportResponse
	}{
		{id: "VSKC", removalType: "delete", exp: &DeleteExportReportResponse{Deleted: true}},
		{id: "TCJA", removalType: "cancel", exp: &DeleteExportReportResponse{Cancelled: true}},
	} {
		t.Run(tc.removalType, func(t *testing.T) {
			t.Parallel()
			result, err := e.DeleteExportReport(t.Context(), tc.id, tc.removalType)
			require.NoError(t, err, "DeleteExportReport must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "DeleteExportReport should decode every field")
				return
			}
			assert.True(t, result.Deleted || result.Cancelled, "DeleteExportReport should remove the report")
		})
	}
}

func TestGetAPIKeyInfo(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAPIKeyInfo(t.Context(), &APIKeyInfoRequest{OneTimePassword: "123456"})
	require.NoError(t, err, "GetAPIKeyInfo must not error")
	if mockTests {
		exp := &APIKeyInfoResponse{
			Name:         "my-api-key",
			Key:          "mock-api-key",
			Nonce:        1772627060997,
			NonceWindow:  5,
			Permissions:  []string{"query-funds", "withdraw-funds", "query-open-trades", "modify-trades"},
			IIBAN:        "AA88 N84G WOAK NMOI",
			ValidUntil:   types.Time(time.Unix(1804163060, 0)),
			QueryFrom:    types.Time(time.Unix(1767225600, 0)),
			QueryTo:      types.Time(time.Unix(1798761600, 0)),
			CreatedTime:  types.Time(time.Unix(1772542900, 0)),
			ModifiedTime: types.Time(time.Unix(1772543095, 0)),
			IPAllowlist:  []string{"203.0.113.0/24", "198.51.100.7"},
			LastUsed:     types.Time(time.Unix(1772627061, 0)),
		}
		assert.Equal(t, exp, result, "GetAPIKeyInfo should decode every field")
		return
	}
	assert.NotZero(t, result.Nonce, "GetAPIKeyInfo should return the key's nonce")
}

func TestAPIKeyInfoResponseUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var result APIKeyInfoResponse
	err := result.UnmarshalJSON([]byte(`{"apiKeyName":"my-api-key","apiKey":"mock-api-key","nonce":"1772627060997","nonceWindow":0,"permissions":["query-funds","withdraw-funds","query-open-trades","modify-trades"],"iban":"AA88 N84G WOAK NMOI","validUntil":"0","queryFrom":"0","queryTo":"0","createdTime":"1772542900","modifiedTime":"1772543095","ipAllowlist":[],"lastUsed":"1772627061"}`))
	require.NoError(t, err, "UnmarshalJSON must not error for Kraken's camelCase example")
	exp := APIKeyInfoResponse{
		Name:         "my-api-key",
		Key:          "mock-api-key",
		Nonce:        1772627060997,
		Permissions:  []string{"query-funds", "withdraw-funds", "query-open-trades", "modify-trades"},
		IIBAN:        "AA88 N84G WOAK NMOI",
		CreatedTime:  types.Time(time.Unix(1772542900, 0)),
		ModifiedTime: types.Time(time.Unix(1772543095, 0)),
		IPAllowlist:  []string{},
		LastUsed:     types.Time(time.Unix(1772627061, 0)),
	}
	assert.Equal(t, exp, result, "UnmarshalJSON should decode the camelCase names as the schema's")
	assert.Error(t, result.UnmarshalJSON([]byte(`[]`)), "UnmarshalJSON should reject an array")
	assert.Error(t, result.UnmarshalJSON([]byte(`{"nonce":"next"}`)), "UnmarshalJSON should reject a nonce that is not an integer")
}

func TestListWalletAccounts(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.ListWalletAccounts(t.Context())
	require.NoError(t, err, "ListWalletAccounts must not error")
	if mockTests {
		exp := &WalletAccountsResponse{
			Accounts: []WalletAccount{
				{AccountID: "WX6V-JUKW-KKPB-QE36", Flags: WalletAccountFlags{Active: true}, Status: "active", Type: "main"},
				{AccountID: "QZ3D-8RTM-PL2K-NV7C", Flags: WalletAccountFlags{UserDefined: true}, Status: "disabled", Type: "spot", Name: "Trading bot"},
			},
			Cursor: PageCursor{Next: "gqNrZXm-V2FsbGV0czI"},
		}
		assert.Equal(t, exp, result, "ListWalletAccounts should decode every field")
		return
	}
	assert.NotEmpty(t, result.Accounts, "ListWalletAccounts should return the default wallet")
}

// accountDataRequest is a request the account data test server received, its nonce removed from the body
type accountDataRequest struct {
	path  string
	query url.Values
	body  map[string]any
}

// accountDataRecorder holds the requests its test server received
type accountDataRecorder struct {
	mu       sync.Mutex
	requests []accountDataRequest
}

// take returns the requests received since the last take
func (a *accountDataRecorder) take() []accountDataRequest {
	a.mu.Lock()
	defer a.mu.Unlock()
	requests := a.requests
	a.requests = nil
	return requests
}

// newAccountDataRecorder returns an exchange whose test server records each request it receives and replies with an
// empty result: an array for consolidated positions, an object otherwise
func newAccountDataRecorder(t *testing.T) (*Exchange, *accountDataRecorder) {
	t.Helper()
	recorder := new(accountDataRecorder)
	ex := newHTTPTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		var body map[string]any
		assert.NoError(t, json.NewDecoder(r.Body).Decode(&body), "Decode should not error")
		assert.Contains(t, body, "nonce", "body should carry the nonce")
		delete(body, "nonce")
		recorder.mu.Lock()
		recorder.requests = append(recorder.requests, accountDataRequest{path: r.URL.Path, query: r.URL.Query(), body: body})
		recorder.mu.Unlock()
		if body["consolidation"] == "market" {
			_, _ = w.Write([]byte(`{"error":[],"result":[]}`))
			return
		}
		_, _ = w.Write([]byte(`{"error":[],"result":{}}`))
	})
	return ex, recorder
}

func TestAccountIDQueryParameter(t *testing.T) {
	t.Parallel()
	ex, recorder := newAccountDataRecorder(t)
	const accountID = "WX6V-JUKW-KKPB-QE36"
	for _, tc := range []struct {
		path string
		call func() error
	}{
		{"/0/private/Balance", func() error {
			_, err := ex.GetAccountBalance(t.Context(), &BalanceRequest{AccountID: accountID})
			return err
		}},
		{"/0/private/BalanceEx", func() error {
			_, err := ex.GetExtendedBalance(t.Context(), &BalanceRequest{AccountID: accountID})
			return err
		}},
		{"/0/private/CreditLines", func() error {
			_, err := ex.GetCreditLines(t.Context(), &BalanceRequest{AccountID: accountID})
			return err
		}},
		{"/0/private/TradeBalance", func() error {
			_, err := ex.GetTradeBalance(t.Context(), &TradeBalanceRequest{AccountID: accountID})
			return err
		}},
		{"/0/private/OpenOrders", func() error {
			_, err := ex.GetOpenOrders(t.Context(), &OpenOrdersRequest{AccountID: accountID})
			return err
		}},
		{"/0/private/ClosedOrders", func() error {
			_, err := ex.GetClosedOrders(t.Context(), &ClosedOrdersRequest{AccountID: accountID})
			return err
		}},
		{"/0/private/QueryOrders", func() error {
			_, err := ex.QueryOrdersInfo(t.Context(), &QueryOrdersRequest{AccountID: accountID, OrderIDs: []string{"OQCLML-BW3P3-BUCMWZ"}})
			return err
		}},
		{"/0/private/OrderAmends", func() error {
			_, err := ex.GetOrderAmends(t.Context(), &OrderAmendsRequest{AccountID: accountID, OrderID: "OQCLML-BW3P3-BUCMWZ"})
			return err
		}},
		{"/0/private/TradesHistory", func() error {
			_, err := ex.GetTradesHistory(t.Context(), &TradesHistoryRequest{AccountID: accountID})
			return err
		}},
		{"/0/private/QueryTrades", func() error {
			_, err := ex.QueryTradesInfo(t.Context(), &QueryTradesRequest{AccountID: accountID, TradeIDs: []string{"THVRQM-33VKH-UCI7BS"}})
			return err
		}},
		{"/0/private/OpenPositions", func() error {
			_, err := ex.GetOpenPositions(t.Context(), &OpenPositionsRequest{AccountID: accountID})
			return err
		}},
		{"/0/private/OpenPositions", func() error {
			_, err := ex.GetConsolidatedOpenPositions(t.Context(), &OpenPositionsRequest{AccountID: accountID})
			return err
		}},
		{"/0/private/Ledgers", func() error {
			_, err := ex.GetLedgers(t.Context(), &LedgersRequest{AccountID: accountID})
			return err
		}},
		{"/0/private/QueryLedgers", func() error {
			_, err := ex.QueryLedgers(t.Context(), &QueryLedgersRequest{AccountID: accountID, LedgerIDs: []string{"L4UESK-KG3EQ-UFO4T5"}})
			return err
		}},
	} {
		require.NoErrorf(t, tc.call(), "%s must not error", tc.path)
		got := recorder.take()
		require.Lenf(t, got, 1, "%s must send one request", tc.path)
		assert.Equalf(t, tc.path, got[0].path, "%s should be requested", tc.path)
		assert.Equalf(t, url.Values{"account_id": {accountID}}, got[0].query, "%s should send account_id in the query", tc.path)
		assert.NotContainsf(t, got[0].body, "account_id", "%s should not send account_id in the body", tc.path)
	}
}

func TestAccountDataNilRequests(t *testing.T) {
	t.Parallel()
	ex, recorder := newAccountDataRecorder(t)
	for _, tc := range []struct {
		path string
		call func() error
		body map[string]any
	}{
		{"/0/private/Balance", func() error { _, err := ex.GetAccountBalance(t.Context(), nil); return err }, map[string]any{}},
		{"/0/private/BalanceEx", func() error { _, err := ex.GetExtendedBalance(t.Context(), nil); return err }, map[string]any{}},
		{"/0/private/CreditLines", func() error { _, err := ex.GetCreditLines(t.Context(), nil); return err }, map[string]any{}},
		{"/0/private/TradeBalance", func() error { _, err := ex.GetTradeBalance(t.Context(), nil); return err }, map[string]any{}},
		{"/0/private/OpenOrders", func() error { _, err := ex.GetOpenOrders(t.Context(), nil); return err }, map[string]any{}},
		{"/0/private/ClosedOrders", func() error { _, err := ex.GetClosedOrders(t.Context(), nil); return err }, map[string]any{}},
		{"/0/private/TradesHistory", func() error { _, err := ex.GetTradesHistory(t.Context(), nil); return err }, map[string]any{}},
		{"/0/private/OpenPositions", func() error { _, err := ex.GetOpenPositions(t.Context(), nil); return err }, map[string]any{}},
		{"/0/private/OpenPositions", func() error {
			_, err := ex.GetConsolidatedOpenPositions(t.Context(), nil)
			return err
		}, map[string]any{"consolidation": "market"}},
		{"/0/private/Ledgers", func() error { _, err := ex.GetLedgers(t.Context(), nil); return err }, map[string]any{}},
		{"/0/private/TradeVolume", func() error { _, err := ex.GetTradeVolume(t.Context(), nil); return err }, map[string]any{}},
		{"/0/private/GetApiKeyInfo", func() error { _, err := ex.GetAPIKeyInfo(t.Context(), nil); return err }, map[string]any{}},
	} {
		require.NoErrorf(t, tc.call(), "%s must not error for a nil request", tc.path)
		got := recorder.take()
		require.Lenf(t, got, 1, "%s must send one request", tc.path)
		assert.Equalf(t, tc.path, got[0].path, "%s should be requested", tc.path)
		assert.Emptyf(t, got[0].query, "%s should send no query for a nil request", tc.path)
		assert.Equalf(t, tc.body, got[0].body, "%s should send only the nonce for a nil request", tc.path)
	}
}

func TestAccountDataPageParams(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name       string
		withCursor bool
		cursor     string
		offset     uint64
		exp        map[string]any
		err        error
	}{
		{name: "none", exp: map[string]any{}},
		{name: "offset", offset: 50, exp: map[string]any{"ofs": uint64(50)}},
		{name: "first cursor page", withCursor: true, exp: map[string]any{"with_cursor": true}},
		{name: "next cursor page", withCursor: true, cursor: "gqNrZXm-T3BlbjI", exp: map[string]any{"with_cursor": true, "cursor": "gqNrZXm-T3BlbjI"}},
		{name: "cursor without cursor pagination", cursor: "gqNrZXm-T3BlbjI", err: errCursorWithoutPaging},
		{name: "offset with cursor pagination", withCursor: true, offset: 50, err: errOffsetWithCursor},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			body := map[string]any{}
			err := accountDataPageParams(body, tc.withCursor, tc.cursor, tc.offset)
			if tc.err != nil {
				assert.ErrorIs(t, err, tc.err, "accountDataPageParams should reject the combination")
				return
			}
			require.NoError(t, err, "accountDataPageParams must not error")
			assert.Equal(t, tc.exp, body, "accountDataPageParams should add the pagination parameters")
		})
	}
}

func TestAccountDataRangeParams(t *testing.T) {
	t.Parallel()
	start, end := time.Unix(1688140000, 0), time.Unix(1688150000, 0)
	for _, tc := range []struct {
		name           string
		start, end     time.Time
		startID, endID string
		exp            map[string]any
		err            error
	}{
		{name: "none", exp: map[string]any{}},
		{name: "times", start: start, end: end, exp: map[string]any{"start": int64(1688140000), "end": int64(1688150000)}},
		{name: "IDs", startID: "O37652-RJWRT-IMO74O", endID: "OMMDB2-FSB6Z-7W3HPO", exp: map[string]any{"start": "O37652-RJWRT-IMO74O", "end": "OMMDB2-FSB6Z-7W3HPO"}},
		{name: "time start and ID end", start: start, endID: "OMMDB2-FSB6Z-7W3HPO", exp: map[string]any{"start": int64(1688140000), "end": "OMMDB2-FSB6Z-7W3HPO"}},
		{name: "start as time and ID", start: start, startID: "O37652-RJWRT-IMO74O", err: errTimeAndIDBound},
		{name: "end as time and ID", end: end, endID: "OMMDB2-FSB6Z-7W3HPO", err: errTimeAndIDBound},
		{name: "reversed", start: end, end: start, err: common.ErrStartAfterEnd},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			body := map[string]any{}
			err := accountDataRangeParams(body, tc.start, tc.end, tc.startID, tc.endID)
			if tc.err != nil {
				assert.ErrorIs(t, err, tc.err, "accountDataRangeParams should reject the range")
				return
			}
			require.NoError(t, err, "accountDataRangeParams must not error")
			assert.Equal(t, tc.exp, body, "accountDataRangeParams should add the range")
		})
	}
}

func TestAccountDataIDList(t *testing.T) {
	t.Parallel()
	_, err := accountDataIDList(nil, 2, errLedgerIDEmpty)
	assert.ErrorIs(t, err, errLedgerIDEmpty, "accountDataIDList should reject no IDs")
	_, err = accountDataIDList([]string{"L4UESK-KG3EQ-UFO4T5", ""}, 2, errLedgerIDEmpty)
	assert.ErrorIs(t, err, errLedgerIDEmpty, "accountDataIDList should reject an empty ID")
	_, err = accountDataIDList([]string{"L4UESK-KG3EQ-UFO4T5", "LMKZCZ-Z3GVL-CXKK4H", "LTKOJ5-JWDZ4-3EQOER"}, 2, errLedgerIDEmpty)
	assert.ErrorIs(t, err, errTooManyIDs, "accountDataIDList should reject more IDs than the limit")
	list, err := accountDataIDList([]string{"L4UESK-KG3EQ-UFO4T5", "LMKZCZ-Z3GVL-CXKK4H"}, 2, errLedgerIDEmpty)
	require.NoError(t, err, "accountDataIDList must not error")
	assert.Equal(t, "L4UESK-KG3EQ-UFO4T5,LMKZCZ-Z3GVL-CXKK4H", list, "accountDataIDList should join the IDs with commas")
}

func TestAccountDataCheckClassified(t *testing.T) {
	t.Parallel()
	err := accountDataCheckClassified([]ClassifiedAsset{{Name: "AAPLx", AssetClass: "tokenized_asset"}, {AssetClass: "tokenized_asset"}}, currency.ErrCurrencyCodeEmpty)
	assert.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "accountDataCheckClassified should reject a name that is empty")
	err = accountDataCheckClassified([]ClassifiedAsset{{Name: "AAPLx"}}, currency.ErrCurrencyCodeEmpty)
	assert.ErrorIs(t, err, errAssetClassEmpty, "accountDataCheckClassified should reject a class that is empty")
	assert.NoError(t, accountDataCheckClassified([]ClassifiedAsset{{Name: "AAPLx", AssetClass: "tokenized_asset"}}, currency.ErrCurrencyCodeEmpty), "accountDataCheckClassified should accept named, classified names")
}

func TestAccountDataParams(t *testing.T) {
	t.Parallel()
	query, body := accountDataParams("", "")
	assert.Empty(t, query, "accountDataParams should set no account without one")
	assert.Empty(t, body, "accountDataParams should set no rebase multiplier without one")
	query, body = accountDataParams("AA24 N84G HSBB 7ZDO", "base")
	assert.Equal(t, url.Values{"account_id": {"AA24 N84G HSBB 7ZDO"}}, query, "accountDataParams should query the account")
	assert.Equal(t, map[string]any{"rebase_multiplier": "base"}, body, "accountDataParams should send the rebase multiplier")
}

func TestAccountDataOrderParams(t *testing.T) {
	t.Parallel()
	body := map[string]any{}
	accountDataOrderParams(body, false, 0, "", false)
	assert.Empty(t, body, "accountDataOrderParams should add nothing unasked")
	accountDataOrderParams(body, true, 123, "6d1b345e-2821-40e2-ad83-4ecb18a06876", true)
	exp := map[string]any{"trades": true, "userref": int32(123), "cl_ord_id": "6d1b345e-2821-40e2-ad83-4ecb18a06876", "consolidate_taker": false}
	assert.Equal(t, exp, body, "accountDataOrderParams should add every parameter asked for")
}

func TestAccountDataPositionParams(t *testing.T) {
	t.Parallel()
	query, body, err := accountDataPositionParams(nil)
	require.NoError(t, err, "accountDataPositionParams must not error for a nil request")
	assert.Empty(t, query, "accountDataPositionParams should query no account for a nil request")
	assert.Empty(t, body, "accountDataPositionParams should send nothing for a nil request")
	_, _, err = accountDataPositionParams(&OpenPositionsRequest{TradeIDs: []string{"TF5GVO-T7ZZ2-6NBKBI", ""}})
	assert.ErrorIs(t, err, errTradeIDEmpty, "accountDataPositionParams should reject an empty trade ID")
	query, body, err = accountDataPositionParams(&OpenPositionsRequest{AccountID: "AA24 N84G HSBB 7ZDO", TradeIDs: []string{"TF5GVO-T7ZZ2-6NBKBI", "T24DOR-TAFLM-ID3NYP"}, IncludeCalculations: true, RebaseMultiplier: "base"})
	require.NoError(t, err, "accountDataPositionParams must not error")
	assert.Equal(t, url.Values{"account_id": {"AA24 N84G HSBB 7ZDO"}}, query, "accountDataPositionParams should query the account")
	assert.Equal(t, map[string]any{"txid": "TF5GVO-T7ZZ2-6NBKBI,T24DOR-TAFLM-ID3NYP", "docalcs": true, "rebase_multiplier": "base"}, body, "accountDataPositionParams should send the trade IDs, calculations and rebase multiplier")
}
