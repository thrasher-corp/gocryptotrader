package bybit

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// preUpgradeTestStart and preUpgradeTestEnd bound the 7 day window the pre-upgrade tests query
var (
	preUpgradeTestStart = time.Date(2023, 4, 20, 0, 0, 0, 0, time.UTC)
	preUpgradeTestEnd   = time.Date(2023, 4, 27, 0, 0, 0, 0, time.UTC)
)

func TestGetPreUpgradeOrderHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetPreUpgradeOrderHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetPreUpgradeOrderHistory must reject a nil request")
	_, err = e.GetPreUpgradeOrderHistory(t.Context(), &PreUpgradeOrderHistoryRequest{Symbol: "BTCUSDT"})
	require.ErrorIs(t, err, errCategoryNotSet, "GetPreUpgradeOrderHistory must reject an empty category")
	_, err = e.GetPreUpgradeOrderHistory(t.Context(), &PreUpgradeOrderHistoryRequest{Category: cLinear, StartTime: preUpgradeTestStart})
	require.ErrorIs(t, err, common.ErrDateUnset, "GetPreUpgradeOrderHistory must reject a start time without an end time")
	_, err = e.GetPreUpgradeOrderHistory(t.Context(), &PreUpgradeOrderHistoryRequest{Category: cLinear, StartTime: preUpgradeTestEnd, EndTime: preUpgradeTestStart})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetPreUpgradeOrderHistory must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name string
		req  *PreUpgradeOrderHistoryRequest
		exp  *PreUpgradeOrderHistoryResponse
	}{
		{name: "linear", req: &PreUpgradeOrderHistoryRequest{Category: cLinear, Symbol: "BTCUSDT", OrderFilter: "StopOrder", OrderStatus: "Filled", StartTime: preUpgradeTestStart, EndTime: preUpgradeTestEnd, Limit: 1}, exp: &PreUpgradeOrderHistoryResponse{
			Category: "linear",
			List: []PreUpgradeOrder{
				{
					OrderID:                    "67836246-460e-4c52-a009-af0c3e1d12bc",
					OrderLinkID:                "linear-stop-1",
					Symbol:                     "BTCUSDT",
					Price:                      27203.4,
					Quantity:                   0.2,
					Side:                       "Sell",
					PositionIndex:              2,
					OrderStatus:                "Filled",
					CancelType:                 "UNKNOWN",
					RejectReason:               "EC_NoError",
					AveragePrice:               28632.126,
					CumulativeExecutedQuantity: 0.2,
					CumulativeExecutedValue:    5726.4252,
					CumulativeExecutedFee:      3.43585512,
					TimeInForce:                "IOC",
					OrderType:                  "Market",
					StopOrderType:              "StopLoss",
					TriggerPrice:               28650,
					TakeProfit:                 30000,
					StopLoss:                   28650,
					TPTriggerBy:                "MarkPrice",
					SLTriggerBy:                "LastPrice",
					TriggerDirection:           2,
					TriggerBy:                  "LastPrice",
					LastPriceOnCreated:         29105.5,
					ReduceOnly:                 true,
					CloseOnTrigger:             true,
					SMPType:                    "None",
					CreatedTime:                types.Time(time.UnixMilli(1682487465732)),
					UpdatedTime:                types.Time(time.UnixMilli(1682487465735)),
				},
			},
			NextPageCursor: "page_token%3D69406%26",
		}},
		{name: "option", req: &PreUpgradeOrderHistoryRequest{Category: cOption, BaseCoin: currency.BTC, OrderID: "8f7b23e1-7a5d-4c3f-9b1e-2d6c4a8f0e11", OrderLinkID: "option-order-1", Cursor: "69406"}, exp: &PreUpgradeOrderHistoryResponse{
			Category: "option",
			List: []PreUpgradeOrder{
				{
					OrderID:                    "8f7b23e1-7a5d-4c3f-9b1e-2d6c4a8f0e11",
					OrderLinkID:                "option-order-1",
					BlockTradeID:               "b5f1c2d3-1111-4c2b-8f3a-9c0d1e2f3a4b",
					Symbol:                     "BTC-28APR23-30000-C",
					Price:                      1250,
					Quantity:                   1.5,
					Side:                       "Buy",
					IsLeverage:                 "0",
					OrderStatus:                "PartiallyFilledCanceled",
					CancelType:                 "CancelByUser",
					RejectReason:               "EC_NoError",
					AveragePrice:               1240,
					LeavesQuantity:             0.5,
					LeavesValue:                625,
					CumulativeExecutedQuantity: 1,
					CumulativeExecutedValue:    1240,
					CumulativeExecutedFee:      0.4,
					TimeInForce:                "GTC",
					OrderType:                  "Limit",
					StopOrderType:              "UNKNOWN",
					OrderIV:                    0.5124,
					TPTriggerBy:                "UNKNOWN",
					SLTriggerBy:                "UNKNOWN",
					TriggerBy:                  "UNKNOWN",
					LastPriceOnCreated:         1245,
					PlaceType:                  "iv",
					SMPType:                    "CancelMaker",
					SMPGroup:                   7,
					SMPOrderID:                 "5f3e8a9b-2c1d-4e6f-8a7b-9c0d1e2f3a4b",
					CreatedTime:                types.Time(time.UnixMilli(1682409600000)),
					UpdatedTime:                types.Time(time.UnixMilli(1682413200000)),
				},
			},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetPreUpgradeOrderHistory(t.Context(), tc.req)
			require.NoError(t, err, "GetPreUpgradeOrderHistory must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetPreUpgradeOrderHistory should decode every field")
				return
			}
			assert.Equal(t, tc.req.Category, result.Category, "GetPreUpgradeOrderHistory should return the requested category")
		})
	}
}

func TestGetPreUpgradeTradeHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetPreUpgradeTradeHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetPreUpgradeTradeHistory must reject a nil request")
	_, err = e.GetPreUpgradeTradeHistory(t.Context(), &PreUpgradeTradeHistoryRequest{Symbol: "BTCUSDT"})
	require.ErrorIs(t, err, errCategoryNotSet, "GetPreUpgradeTradeHistory must reject an empty category")
	_, err = e.GetPreUpgradeTradeHistory(t.Context(), &PreUpgradeTradeHistoryRequest{Category: cLinear, StartTime: preUpgradeTestEnd, EndTime: preUpgradeTestStart})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetPreUpgradeTradeHistory must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name string
		req  *PreUpgradeTradeHistoryRequest
		exp  *PreUpgradeTradeHistoryResponse
	}{
		{name: "linear", req: &PreUpgradeTradeHistoryRequest{Category: cLinear, Symbol: "BTCUSDT", ExecutionType: "Trade", StartTime: preUpgradeTestStart, EndTime: preUpgradeTestEnd, Limit: 1}, exp: &PreUpgradeTradeHistoryResponse{
			Category: "linear",
			List: []PreUpgradeExecution{
				{
					Symbol:            "BTCUSDT",
					OrderID:           "1682553600-BTCUSDT-592334-Sell",
					OrderLinkID:       "linear-close-1",
					Side:              "Sell",
					OrderPrice:        28400,
					OrderQuantity:     0.02,
					LeavesQuantity:    0.009,
					OrderType:         "Limit",
					StopOrderType:     "UNKNOWN",
					ExecutionFee:      0.06247978,
					ExecutionID:       "11f1c4ed-ff20-4d73-acb7-96e43a917f25",
					ExecutionPrice:    28399.9,
					ExecutionQuantity: 0.011,
					ExecutionType:     "Trade",
					ExecutionValue:    312.3989,
					ExecutionTime:     types.Time(time.UnixMilli(1682553600000)),
					IsMaker:           true,
					FeeRate:           0.0002,
					MarkPrice:         28401.2,
					ClosedSize:        0.011,
				},
			},
			NextPageCursor: "page_token%3D96184191%26",
		}},
		{name: "option", req: &PreUpgradeTradeHistoryRequest{Category: cOption, BaseCoin: currency.BTC, OrderID: "8f7b23e1-7a5d-4c3f-9b1e-2d6c4a8f0e11", OrderLinkID: "option-order-1", Cursor: "96184191"}, exp: &PreUpgradeTradeHistoryResponse{
			Category: "option",
			List: []PreUpgradeExecution{
				{
					Symbol:            "BTC-28APR23-30000-C",
					OrderID:           "8f7b23e1-7a5d-4c3f-9b1e-2d6c4a8f0e11",
					OrderLinkID:       "option-order-1",
					Side:              "Buy",
					OrderPrice:        1250,
					OrderQuantity:     1.5,
					LeavesQuantity:    0.5,
					OrderType:         "Limit",
					ExecutionFee:      0.4,
					ExecutionID:       "c2f1a3b4-5d6e-4f70-8a9b-0c1d2e3f4a5b",
					ExecutionPrice:    1240,
					ExecutionQuantity: 1,
					ExecutionType:     "BlockTrade",
					ExecutionValue:    1240,
					ExecutionTime:     types.Time(time.UnixMilli(1682413200000)),
					FeeRate:           0.0003,
					TradeIV:           0.5118,
					MarkIV:            0.5096,
					MarkPrice:         1236.42,
					IndexPrice:        29331.05,
					UnderlyingPrice:   29340.1,
					BlockTradeID:      "b5f1c2d3-1111-4c2b-8f3a-9c0d1e2f3a4b",
				},
			},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetPreUpgradeTradeHistory(t.Context(), tc.req)
			require.NoError(t, err, "GetPreUpgradeTradeHistory must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetPreUpgradeTradeHistory should decode every field")
				return
			}
			assert.Equal(t, tc.req.Category, result.Category, "GetPreUpgradeTradeHistory should return the requested category")
		})
	}
}

func TestGetPreUpgradeClosedPNL(t *testing.T) {
	t.Parallel()
	_, err := e.GetPreUpgradeClosedPNL(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetPreUpgradeClosedPNL must reject a nil request")
	_, err = e.GetPreUpgradeClosedPNL(t.Context(), &PreUpgradeClosedPNLRequest{Symbol: "BTCUSDT"})
	require.ErrorIs(t, err, errCategoryNotSet, "GetPreUpgradeClosedPNL must reject an empty category")
	_, err = e.GetPreUpgradeClosedPNL(t.Context(), &PreUpgradeClosedPNLRequest{Category: cLinear})
	require.ErrorIs(t, err, errSymbolMissing, "GetPreUpgradeClosedPNL must reject an empty symbol")
	_, err = e.GetPreUpgradeClosedPNL(t.Context(), &PreUpgradeClosedPNLRequest{Category: cLinear, Symbol: "BTCUSDT", StartTime: preUpgradeTestEnd, EndTime: preUpgradeTestStart})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetPreUpgradeClosedPNL must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPreUpgradeClosedPNL(t.Context(), &PreUpgradeClosedPNLRequest{Category: cLinear, Symbol: "BTCUSDT", StartTime: preUpgradeTestStart, EndTime: preUpgradeTestEnd, Limit: 1, Cursor: "page_token=1"})
	require.NoError(t, err, "GetPreUpgradeClosedPNL must not error")
	if mockTests {
		exp := &PreUpgradeClosedPNLResponse{
			Category: "linear",
			List: []PreUpgradeClosedPNL{
				{
					Symbol:               "BTCUSDT",
					OrderID:              "67836246-460e-4c52-a009-af0c3e1d12bc",
					Side:                 "Sell",
					Quantity:             0.2,
					OrderPrice:           27203.4,
					OrderType:            "Market",
					ExecutionType:        "Trade",
					ClosedSize:           0.2,
					CumulativeEntryValue: 5588.88,
					AverageEntryPrice:    27944.4,
					CumulativeExitValue:  5726.4252,
					AverageExitPrice:     28632.13,
					ClosedPNL:            204.25510011,
					FillCount:            22,
					Leverage:             10,
					CreatedTime:          types.Time(time.UnixMilli(1682487465732)),
					UpdatedTime:          types.Time(time.UnixMilli(1682487465732)),
				},
			},
			NextPageCursor: "page_token%3D2",
		}
		assert.Equal(t, exp, result, "GetPreUpgradeClosedPNL should decode every field")
		return
	}
	assert.Equal(t, cLinear, result.Category, "GetPreUpgradeClosedPNL should return the requested category")
}

func TestGetPreUpgradeTransactionLog(t *testing.T) {
	t.Parallel()
	_, err := e.GetPreUpgradeTransactionLog(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetPreUpgradeTransactionLog must reject a nil request")
	_, err = e.GetPreUpgradeTransactionLog(t.Context(), &PreUpgradeTransactionLogRequest{})
	require.ErrorIs(t, err, errCategoryNotSet, "GetPreUpgradeTransactionLog must reject an empty category")
	_, err = e.GetPreUpgradeTransactionLog(t.Context(), &PreUpgradeTransactionLogRequest{Category: cLinear, StartTime: preUpgradeTestEnd, EndTime: preUpgradeTestStart})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetPreUpgradeTransactionLog must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name string
		req  *PreUpgradeTransactionLogRequest
		exp  *PreUpgradeTransactionLogResponse
	}{
		{name: "linear", req: &PreUpgradeTransactionLogRequest{Category: cLinear, BaseCoin: currency.BTC, StartTime: preUpgradeTestStart, EndTime: preUpgradeTestEnd, Limit: 2}, exp: &PreUpgradeTransactionLogResponse{
			List: []PreUpgradeTransactionLog{
				{
					Symbol:          "BTCPERP",
					Category:        "linear",
					Side:            "Buy",
					TransactionTime: types.Time(time.UnixMilli(1682496000000)),
					Type:            "TRADE",
					Quantity:        0.01,
					Size:            0.01,
					Currency:        currency.USDC,
					TradePrice:      28360,
					Fee:             0.17016,
					Change:          -0.17016,
					CashBalance:     1001.31404,
					FeeRate:         0.0006,
					BonusChange:     -0.01,
					TradeID:         "e4a1b2c3-d4e5-4f60-8a7b-9c0d1e2f3a40",
					OrderID:         "3b5c7d9e-1f2a-4b6c-8d0e-2f4a6b8c0d1e",
					OrderLinkID:     "usdc-perp-1",
				},
				{
					Symbol:          "BTCPERP",
					Category:        "linear",
					Side:            "Buy",
					TransactionTime: types.Time(time.UnixMilli(1682524800000)),
					Type:            "SETTLEMENT",
					Quantity:        0.01,
					Size:            0.01,
					Currency:        currency.USDC,
					TradePrice:      28360,
					Funding:         -0.02836,
					CashFlow:        -0.4,
					Change:          -0.02836,
					CashBalance:     1001.28568,
					FeeRate:         0.0001,
				},
			},
			NextPageCursor: "21%3A0%2C21%3A0",
		}},
		{name: "option", req: &PreUpgradeTransactionLogRequest{Category: cOption, Type: "DELIVERY", Limit: 1, Cursor: "21%3A0%2C21%3A0"}, exp: &PreUpgradeTransactionLogResponse{
			List: []PreUpgradeTransactionLog{
				{
					Symbol:          "ETH-14JUN23-1750-C",
					Category:        "option",
					Side:            "Buy",
					TransactionTime: types.Time(time.UnixMilli(1686729604507)),
					Type:            "DELIVERY",
					Quantity:        0.5,
					Currency:        currency.USDC,
					TradePrice:      1740.25036667,
					CashBalance:     1001.1438885,
				},
			},
			NextPageCursor: "21%3A0%2C21%3A0",
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetPreUpgradeTransactionLog(t.Context(), tc.req)
			require.NoError(t, err, "GetPreUpgradeTransactionLog must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetPreUpgradeTransactionLog should decode every field")
				return
			}
			assert.NotNil(t, result, "GetPreUpgradeTransactionLog should return a result")
		})
	}
}

func TestGetPreUpgradeOptionDeliveryRecord(t *testing.T) {
	t.Parallel()
	_, err := e.GetPreUpgradeOptionDeliveryRecord(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetPreUpgradeOptionDeliveryRecord must reject a nil request")
	_, err = e.GetPreUpgradeOptionDeliveryRecord(t.Context(), &PreUpgradeDeliveryRecordRequest{})
	require.ErrorIs(t, err, errCategoryNotSet, "GetPreUpgradeOptionDeliveryRecord must reject an empty category")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPreUpgradeOptionDeliveryRecord(t.Context(), &PreUpgradeDeliveryRecordRequest{Category: cOption, Symbol: "ETH-14JUN23-1750-C", ExpiryDate: time.Date(2023, 6, 14, 8, 0, 0, 0, time.UTC), Limit: 1, Cursor: "21"})
	require.NoError(t, err, "GetPreUpgradeOptionDeliveryRecord must not error")
	if mockTests {
		exp := &PreUpgradeDeliveryRecordResponse{
			Category: "option",
			List: []PreUpgradeDeliveryRecord{
				{
					DeliveryTime:        types.Time(time.UnixMilli(1686729604507)),
					Symbol:              "ETH-14JUN23-1750-C",
					Side:                "Buy",
					Position:            0.5,
					DeliveryPrice:       1740.25036667,
					Strike:              1750,
					Fee:                 0.0435,
					DeliveryRealisedPNL: 0.175,
				},
			},
			NextPageCursor: "21%3A0%2C21%3A0",
		}
		assert.Equal(t, exp, result, "GetPreUpgradeOptionDeliveryRecord should decode every field")
		return
	}
	assert.Equal(t, cOption, result.Category, "GetPreUpgradeOptionDeliveryRecord should return the requested category")
}

func TestGetPreUpgradeUSDCSessionSettlement(t *testing.T) {
	t.Parallel()
	_, err := e.GetPreUpgradeUSDCSessionSettlement(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetPreUpgradeUSDCSessionSettlement must reject a nil request")
	_, err = e.GetPreUpgradeUSDCSessionSettlement(t.Context(), &PreUpgradeSessionSettlementRequest{})
	require.ErrorIs(t, err, errCategoryNotSet, "GetPreUpgradeUSDCSessionSettlement must reject an empty category")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetPreUpgradeUSDCSessionSettlement(t.Context(), &PreUpgradeSessionSettlementRequest{Category: cLinear, Symbol: "ETHPERP", Limit: 1, Cursor: "25"})
	require.NoError(t, err, "GetPreUpgradeUSDCSessionSettlement must not error")
	if mockTests {
		exp := &PreUpgradeSessionSettlementResponse{
			Category: "linear",
			List: []PreUpgradeSettlement{
				{
					Symbol:              "ETHPERP",
					Side:                "Sell",
					Size:                -0.5,
					SessionAveragePrice: 1668.41,
					MarkPrice:           1668.44,
					RealisedPNL:         45.76,
					CreatedTime:         types.Time(time.UnixMilli(1686787200000)),
				},
			},
			NextPageCursor: "25%3A0%2C25%3A0",
		}
		assert.Equal(t, exp, result, "GetPreUpgradeUSDCSessionSettlement should decode every field")
		return
	}
	assert.Equal(t, cLinear, result.Category, "GetPreUpgradeUSDCSessionSettlement should return the requested category")
}
