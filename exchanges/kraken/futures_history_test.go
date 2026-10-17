package kraken

import (
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

var (
	futuresHistoryTestSince  = time.Date(2026, 10, 9, 0, 0, 0, 0, time.UTC)
	futuresHistoryTestBefore = futuresHistoryTestSince.Add(time.Hour)
)

const (
	futuresHistoryTestAccount   = "4b2e9c1a-7f3d-4e8b-9a6c-2d1f5e8b7c30"
	futuresHistoryTestToken     = "MTc5MTUwNDAwMDQ1NS82MTcyNzE4NzY5MDE="
	futuresHistoryTestNextToken = "MTc5MTUwNDAwMDQ1NS82MTcyNzE4NzcwMDI="
)

func TestGetFuturesExecutionEvents(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesExecutionEvents(t.Context(), &FuturesHistoryExecutionEventsRequest{Since: futuresHistoryTestBefore, Before: futuresHistoryTestSince})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetFuturesExecutionEvents must reject a reversed window")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	order := FuturesHistoryOrder{
		UID:                   "a2efb3f4-c1c5-4ddb-ba1d-c02d4dde1a7f",
		AccountUID:            futuresHistoryTestAccount,
		Tradeable:             "PF_XBTUSD",
		PositionUID:           "c3f5a8d2-61b4-4e9f-8a7d-0b2c4e6f8a1d",
		Direction:             "Buy",
		Quantity:              0.0007,
		FilledQuantity:        0.0001,
		Timestamp:             types.Time(time.UnixMilli(1791504000450)),
		LimitPrice:            81689,
		OrderType:             "IoC",
		ClientOrderID:         "gct-history-1",
		LastUpdateTime:        types.Time(time.UnixMilli(1791504000455)),
		SpotData:              &FuturesHistorySpotOrderData{Margin: true, FeePreference: "Quote", QuantityPreference: "Base"},
		ParentOfGroupID:       "b8e1c2d3-4f5a-4b6c-8d7e-9f0a1b2c3d4e",
		MemberOfGroupID:       "c9f2d3e4-5a6b-4c7d-9e8f-0a1b2c3d4e5f",
		PartnerOrderID:        "partner-order-7781",
		RegulatoryExternalUID: "d0a3e4f5-6b7c-4d8e-af90-1b2c3d4e5f60",
	}
	oldTaker := order
	oldTaker.UID = "a2efb3f4-c1c5-4ddb-ba1d-c02d4dde1a80"
	oldTaker.Quantity = 0.0008
	oldTaker.FilledQuantity = 0
	oldTaker.Timestamp = types.Time(time.UnixMilli(1791504000440))
	oldTaker.ClientOrderID = "gct-history-0"
	oldTaker.LastUpdateTime = types.Time(time.UnixMilli(1791504000445))
	for _, tc := range []struct {
		name string
		req  *FuturesHistoryExecutionEventsRequest
		exp  *FuturesHistoryExecutionEventsResponse
	}{
		{
			name: "nil request",
			exp:  &FuturesHistoryExecutionEventsResponse{AccountUID: futuresHistoryTestAccount, Elements: []FuturesHistoryExecutionElement{}},
		},
		{
			name: "every parameter",
			req: &FuturesHistoryExecutionEventsRequest{
				Pair:              futuresTestPair,
				Since:             futuresHistoryTestSince,
				Before:            futuresHistoryTestBefore,
				Ascending:         true,
				ContinuationToken: futuresHistoryTestToken,
				Count:             1,
			},
			exp: &FuturesHistoryExecutionEventsResponse{
				AccountUID: futuresHistoryTestAccount,
				Length:     1,
				Elements: []FuturesHistoryExecutionElement{
					{
						UID:       "6f1c2b3a-9d8e-4f70-a1b2-c3d4e5f60718",
						Timestamp: types.Time(time.UnixMilli(1791504000456)),
						Event: FuturesHistoryExecutionEvent{
							Execution: FuturesHistoryExecutionDetails{
								Execution: FuturesHistoryExecution{
									UID:           "9cef6672-8ab8-4d34-8aed-0fd45a9c9851",
									Order:         order,
									Timestamp:     types.Time(time.UnixMilli(1791504000455)),
									Quantity:      0.0001,
									Price:         81689.5,
									MarkPrice:     81698.35759703598,
									ExecutionType: "taker",
									LimitFilled:   true,
									OldTakerOrder: &oldTaker,
									USDValue:      8.17,
									OrderData: &FuturesHistoryOrderData{
										Fee:          0.0040845,
										PositionSize: 0.3,
										FeeCalculation: FuturesHistoryFeeCalculation{
											PercentageFee:             0.05,
											UserFeeDiscountApplied:    0.01,
											MarketShareRebateCredited: 0.002,
										},
										RealisedPNL: 1.25,
									},
									PartnerTradeID: "partner-trade-5521",
									RegulatoryData: &FuturesHistoryRegulatoryData{
										Venue:        "PGSL",
										Counterparty: "984500E2F5A1BD49C863",
										ExternalUID:  "e1b4f5a6-7c8d-4e9f-b0a1-2c3d4e5f6071",
									},
								},
								TakerReducedQuantity: 0.0002,
							},
						},
					},
				},
				ContinuationToken: futuresHistoryTestNextToken,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetFuturesExecutionEvents(t.Context(), tc.req)
			require.NoError(t, err, "GetFuturesExecutionEvents must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetFuturesExecutionEvents should decode every field")
				return
			}
			assert.NotEmpty(t, result.AccountUID, "GetFuturesExecutionEvents should return the account")
		})
	}
}

func TestGetFuturesOrderEvents(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesOrderEvents(t.Context(), &FuturesHistoryOrderEventsRequest{Since: futuresHistoryTestBefore, Before: futuresHistoryTestSince})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetFuturesOrderEvents must reject a reversed window")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	placed := FuturesHistoryOrder{
		UID:                   "7a1b2c3d-4e5f-4a6b-9c7d-8e9f0a1b2c3d",
		AccountUID:            futuresHistoryTestAccount,
		Tradeable:             "PF_XBTUSD",
		PositionUID:           "c3f5a8d2-61b4-4e9f-8a7d-0b2c4e6f8a1d",
		Direction:             "Buy",
		Quantity:              0.5,
		Timestamp:             types.Time(time.UnixMilli(1791504010000)),
		LimitPrice:            81000,
		OrderType:             "Post",
		ClientOrderID:         "gct-order-1",
		LastUpdateTime:        types.Time(time.UnixMilli(1791504010001)),
		SpotData:              &FuturesHistorySpotOrderData{Margin: true, FeePreference: "Quote", QuantityPreference: "Base"},
		ParentOfGroupID:       "b8e1c2d3-4f5a-4b6c-8d7e-9f0a1b2c3d4e",
		MemberOfGroupID:       "c9f2d3e4-5a6b-4c7d-9e8f-0a1b2c3d4e5f",
		PartnerOrderID:        "partner-order-7781",
		RegulatoryExternalUID: "d0a3e4f5-6b7c-4d8e-af90-1b2c3d4e5f60",
	}
	edited := placed
	edited.Quantity = 0.4
	edited.LimitPrice = 81050
	edited.LastUpdateTime = types.Time(time.UnixMilli(1791504020000))
	rejected := FuturesHistoryOrder{
		UID:            "a2f0c1d2-3e4f-4a5b-8c6d-7e8f9a0b1c3e",
		AccountUID:     futuresHistoryTestAccount,
		Tradeable:      "PF_XBTUSD",
		Direction:      "Sell",
		Quantity:       0.75,
		Timestamp:      types.Time(time.UnixMilli(1791504090000)),
		LimitPrice:     81700,
		OrderType:      "Post",
		ClientOrderID:  "gct-history-3",
		ReduceOnly:     true,
		LastUpdateTime: types.Time(time.UnixMilli(1791504090001)),
	}
	resting := FuturesHistoryOrder{
		UID:            "a2f0c1d2-3e4f-4a5b-8c6d-7e8f9a0b1c2d",
		AccountUID:     futuresHistoryTestAccount,
		Tradeable:      "PF_XBTUSD",
		Direction:      "Sell",
		Quantity:       1.5,
		FilledQuantity: 0.25,
		Timestamp:      types.Time(time.UnixMilli(1791504100000)),
		LimitPrice:     81950,
		OrderType:      "Post",
		ClientOrderID:  "gct-history-2",
		ReduceOnly:     true,
		LastUpdateTime: types.Time(time.UnixMilli(1791504160000)),
	}
	attempted := resting
	attempted.Quantity = 5
	attempted.LastUpdateTime = types.Time(time.UnixMilli(1791504170000))
	for _, tc := range []struct {
		name string
		req  *FuturesHistoryOrderEventsRequest
		exp  *FuturesHistoryOrderEventsResponse
	}{
		{
			name: "nil request",
			exp: &FuturesHistoryOrderEventsResponse{
				AccountUID: futuresHistoryTestAccount,
				ServerTime: time.Date(2026, 10, 9, 1, 0, 0, 123000000, time.UTC),
				Elements:   []FuturesHistoryOrderElement{},
			},
		},
		{
			name: "filters",
			req: &FuturesHistoryOrderEventsRequest{
				Pair:   futuresTestPair,
				Since:  futuresHistoryTestSince,
				Before: futuresHistoryTestBefore,
				Count:  3,
				Opened: new(true),
				Closed: new(false),
			},
			exp: &FuturesHistoryOrderEventsResponse{
				AccountUID: futuresHistoryTestAccount,
				Length:     3,
				ServerTime: time.Date(2026, 10, 9, 1, 0, 1, 456000000, time.UTC),
				Elements: []FuturesHistoryOrderElement{
					{
						UID:       "11a2b3c4-d5e6-4f70-8192-a3b4c5d6e7f8",
						Timestamp: types.Time(time.UnixMilli(1791504010001)),
						Event:     FuturesHistoryOrderEvent{OrderPlaced: &FuturesHistoryOrderPlaced{Order: placed, Reason: "new_user_order"}},
					},
					{
						UID:       "22b3c4d5-e6f7-4a81-92a3-b4c5d6e7f809",
						Timestamp: types.Time(time.UnixMilli(1791504020000)),
						Event: FuturesHistoryOrderEvent{
							OrderUpdated: &FuturesHistoryOrderUpdated{OldOrder: placed, NewOrder: edited, Reason: "edited_by_user", ReducedQuantity: 0.1},
						},
					},
					{
						UID:       "33c4d5e6-f7a8-4b92-a3b4-c5d6e7f8091a",
						Timestamp: types.Time(time.UnixMilli(1791504030000)),
						Event: FuturesHistoryOrderEvent{
							OrderNotFound: &FuturesHistoryOrderNotFound{AccountUID: futuresHistoryTestAccount, OrderID: "Uuid(uuid=2ceb1d31-f619-457b-870c-fd4ddbb10d45)"},
						},
					},
				},
				ContinuationToken: futuresHistoryTestNextToken,
			},
		},
		{
			name: "continuation",
			req:  &FuturesHistoryOrderEventsRequest{Ascending: true, ContinuationToken: futuresHistoryTestToken, Count: 3},
			exp: &FuturesHistoryOrderEventsResponse{
				AccountUID: futuresHistoryTestAccount,
				Length:     3,
				ServerTime: time.Date(2026, 10, 9, 1, 0, 2, 789000000, time.UTC),
				Elements: []FuturesHistoryOrderElement{
					{
						UID:       "44d5e6f7-a8b9-4ca3-b4c5-d6e7f8091a2b",
						Timestamp: types.Time(time.UnixMilli(1791504090001)),
						Event: FuturesHistoryOrderEvent{
							OrderRejected: &FuturesHistoryOrderRejected{Order: rejected, OrderError: "post_would_execute", Reason: "new_user_order"},
						},
					},
					{
						UID:       "55e6f7a8-b9c0-4db4-c5d6-e7f8091a2b3c",
						Timestamp: types.Time(time.UnixMilli(1791504110000)),
						Event:     FuturesHistoryOrderEvent{OrderCancelled: &FuturesHistoryOrderCancelled{Order: edited, Reason: "cancelled_by_user"}},
					},
					{
						UID:       "66f7a8b9-c0d1-4ec5-d6e7-f8091a2b3c4d",
						Timestamp: types.Time(time.UnixMilli(1791504170000)),
						Event: FuturesHistoryOrderEvent{
							OrderEditRejected: &FuturesHistoryOrderEditRejected{OldOrder: resting, AttemptedOrder: attempted, OrderError: "insufficient_available_funds"},
						},
					},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetFuturesOrderEvents(t.Context(), tc.req)
			require.NoError(t, err, "GetFuturesOrderEvents must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetFuturesOrderEvents should decode every field")
				return
			}
			assert.NotEmpty(t, result.AccountUID, "GetFuturesOrderEvents should return the account")
		})
	}
}

func TestGetFuturesTriggerEvents(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesTriggerEvents(t.Context(), &FuturesHistoryTriggerEventsRequest{Since: futuresHistoryTestBefore, Before: futuresHistoryTestSince})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetFuturesTriggerEvents must reject a reversed window")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	stop := FuturesHistoryTrigger{
		UID:            "5d2e8f1a-3b4c-4d5e-9f6a-7b8c9d0e1f2a",
		AccountID:      4815162342,
		AccountUID:     futuresHistoryTestAccount,
		Tradeable:      "PF_XBTUSD",
		Direction:      "Sell",
		Quantity:       0.5,
		Timestamp:      types.Time(time.UnixMilli(1791504200000)),
		LimitPrice:     80000,
		OrderType:      "Limit",
		ClientOrderID:  "gct-trigger-1",
		ReduceOnly:     true,
		LastUpdateTime: types.Time(time.UnixMilli(1791504260000)),
		TriggerOptions: FuturesHistoryTriggerOptions{
			TriggerPrice:        80500,
			TriggerSignal:       "MarkPrice",
			TriggerSide:         "Below",
			TrailingStopOptions: FuturesHistoryTrailingStopOptions{MaximumDeviation: 1.5, Unit: "Percent"},
			LimitPriceOffset:    FuturesHistoryPriceOffset{PriceOffset: -50, Unit: "QuoteCurrency"},
		},
	}
	moved := stop
	moved.TriggerOptions.TriggerPrice = 80400
	moved.LastUpdateTime = types.Time(time.UnixMilli(1791504270000))
	takeProfit := FuturesHistoryTrigger{
		UID:            "6e3f9a2b-4c5d-4e6f-8a7b-9c0d1e2f3a4b",
		AccountID:      4815162342,
		AccountUID:     futuresHistoryTestAccount,
		Tradeable:      "PF_XBTUSD",
		Direction:      "Buy",
		Quantity:       0.25,
		Timestamp:      types.Time(time.UnixMilli(1791504300000)),
		LimitPrice:     83000,
		OrderType:      "Market",
		ClientOrderID:  "gct-trigger-2",
		LastUpdateTime: types.Time(time.UnixMilli(1791504330000)),
		TriggerOptions: FuturesHistoryTriggerOptions{
			TriggerPrice:        82900,
			TriggerSignal:       "LastPrice",
			TriggerSide:         "Above",
			TrailingStopOptions: FuturesHistoryTrailingStopOptions{MaximumDeviation: 250, Unit: "QuoteCurrency"},
			LimitPriceOffset:    FuturesHistoryPriceOffset{PriceOffset: 0.2, Unit: "Percent"},
		},
	}
	attempted := takeProfit
	attempted.TriggerOptions.TriggerPrice = 70000
	attempted.LastUpdateTime = types.Time(time.UnixMilli(1791504340000))
	for _, tc := range []struct {
		name string
		req  *FuturesHistoryTriggerEventsRequest
		exp  *FuturesHistoryTriggerEventsResponse
	}{
		{
			name: "nil request",
			exp: &FuturesHistoryTriggerEventsResponse{
				AccountUID: futuresHistoryTestAccount,
				ServerTime: time.Date(2026, 10, 9, 1, 0, 3, 12000000, time.UTC),
				Elements:   []FuturesHistoryTriggerElement{},
			},
		},
		{
			name: "filters",
			req: &FuturesHistoryTriggerEventsRequest{
				Pair:   futuresTestPair,
				Since:  futuresHistoryTestSince,
				Before: futuresHistoryTestBefore,
				Count:  3,
				Opened: new(false),
				Closed: new(true),
			},
			exp: &FuturesHistoryTriggerEventsResponse{
				AccountUID: futuresHistoryTestAccount,
				Length:     3,
				ServerTime: time.Date(2026, 10, 9, 1, 0, 4, 345000000, time.UTC),
				Elements: []FuturesHistoryTriggerElement{
					{
						UID:       "77a8b9c0-d1e2-4fd6-e7f8-091a2b3c4d5e",
						Timestamp: types.Time(time.UnixMilli(1791504260000)),
						Event:     FuturesHistoryTriggerEvent{OrderTriggerPlaced: &FuturesHistoryTriggerPlaced{Order: stop, Reason: "new_user_order"}},
					},
					{
						UID:       "88b9c0d1-e2f3-40e7-f809-1a2b3c4d5e6f",
						Timestamp: types.Time(time.UnixMilli(1791504270000)),
						Event: FuturesHistoryTriggerEvent{
							OrderTriggerUpdated: &FuturesHistoryTriggerUpdated{OldOrderTrigger: stop, NewOrderTrigger: moved, Reason: "edited_by_user"},
						},
					},
					{
						UID:       "99c0d1e2-f3a4-41f8-091a-2b3c4d5e6f70",
						Timestamp: types.Time(time.UnixMilli(1791504280000)),
						Event:     FuturesHistoryTriggerEvent{OrderTriggerActivated: &FuturesHistoryTriggerActivated{Order: moved}},
					},
				},
				ContinuationToken: futuresHistoryTestNextToken,
			},
		},
		{
			name: "continuation",
			req:  &FuturesHistoryTriggerEventsRequest{Ascending: true, ContinuationToken: futuresHistoryTestToken, Count: 2},
			exp: &FuturesHistoryTriggerEventsResponse{
				AccountUID: futuresHistoryTestAccount,
				Length:     2,
				ServerTime: time.Date(2026, 10, 9, 1, 0, 5, 678000000, time.UTC),
				Elements: []FuturesHistoryTriggerElement{
					{
						UID:       "aad1e2f3-a4b5-4209-1a2b-3c4d5e6f7081",
						Timestamp: types.Time(time.UnixMilli(1791504340000)),
						Event: FuturesHistoryTriggerEvent{
							OrderTriggerEditRejected: &FuturesHistoryTriggerEditRejected{
								AttemptedOrderTrigger: attempted,
								OldOrderTrigger:       takeProfit,
								Reason:                "edited_by_user",
								OrderError:            "invalid_trigger_price",
							},
						},
					},
					{
						UID:       "bbe2f3a4-b5c6-431a-2b3c-4d5e6f708192",
						Timestamp: types.Time(time.UnixMilli(1791504350000)),
						Event:     FuturesHistoryTriggerEvent{OrderTriggerCancelled: &FuturesHistoryTriggerCancelled{Order: takeProfit, Reason: "cancelled_by_user"}},
					},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetFuturesTriggerEvents(t.Context(), tc.req)
			require.NoError(t, err, "GetFuturesTriggerEvents must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetFuturesTriggerEvents should decode every field")
				return
			}
			assert.NotEmpty(t, result.AccountUID, "GetFuturesTriggerEvents should return the account")
		})
	}
}

func TestGetFuturesPositionEvents(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesPositionEvents(t.Context(), &FuturesHistoryPositionEventsRequest{Since: futuresHistoryTestBefore, Before: futuresHistoryTestSince})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetFuturesPositionEvents must reject a reversed window")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name string
		req  *FuturesHistoryPositionEventsRequest
		exp  *FuturesHistoryPositionEventsResponse
	}{
		{
			name: "nil request",
			exp: &FuturesHistoryPositionEventsResponse{
				AccountUID: futuresHistoryTestAccount,
				Length:     1,
				ServerTime: time.Date(2026, 10, 9, 1, 0, 6, 901000000, time.UTC),
				Elements: []FuturesHistoryPositionElement{
					{
						UID:       "ccf3a4b5-c6d7-442b-3c4d-5e6f708192a3",
						Timestamp: types.Time(time.UnixMilli(1790323200000)),
						Event: FuturesHistoryPositionEvent{
							PositionUpdate: FuturesHistoryPositionUpdate{
								AccountUID:           futuresHistoryTestAccount,
								Tradeable:            "FF_XBTUSD_260925",
								OldPosition:          2,
								OldAverageEntryPrice: 82500,
								PositionChange:       "close",
								SettlementPrice:      84012.5,
								Timestamp:            types.Time(time.UnixMilli(1790323200000)),
								UpdateReason:         "settlement",
							},
						},
					},
				},
			},
		},
		{
			name: "every parameter",
			req: &FuturesHistoryPositionEventsRequest{
				Pair:               futuresTestPair,
				Since:              futuresHistoryTestSince,
				Before:             futuresHistoryTestBefore,
				Ascending:          true,
				ContinuationToken:  futuresHistoryTestToken,
				Count:              2,
				Opened:             true,
				Closed:             true,
				Increased:          true,
				Decreased:          true,
				Reversed:           true,
				NoChange:           true,
				Trades:             true,
				FundingRealisation: true,
				Settlement:         true,
			},
			exp: &FuturesHistoryPositionEventsResponse{
				AccountUID: futuresHistoryTestAccount,
				Length:     2,
				ServerTime: time.Date(2026, 10, 9, 1, 0, 7, 234000000, time.UTC),
				Elements: []FuturesHistoryPositionElement{
					{
						UID:       "dda4b5c6-d7e8-453c-4d5e-6f708192a3b4",
						Timestamp: types.Time(time.UnixMilli(1791504000460)),
						Event: FuturesHistoryPositionEvent{
							PositionUpdate: FuturesHistoryPositionUpdate{
								AccountUID:           futuresHistoryTestAccount,
								Tradeable:            "PF_XBTUSD",
								OldPosition:          0.5,
								OldAverageEntryPrice: 81000,
								NewPosition:          0.3,
								NewAverageEntryPrice: 81000.5,
								FillTime:             types.Time(time.UnixMilli(1791504000455)),
								Fee:                  0.0123,
								FeeCurrency:          currency.USD,
								RealisedPNL:          5.25,
								PositionChange:       "decrease",
								ExecutionUID:         "9cef6672-8ab8-4d34-8aed-0fd45a9c9851",
								ExecutionPrice:       81689,
								ExecutionSize:        0.2,
								TradeType:            "userExecution",
								Timestamp:            types.Time(time.UnixMilli(1791504000460)),
								UpdateReason:         "trade",
							},
						},
					},
					{
						UID:       "eeb5c6d7-e8f9-464d-5e6f-708192a3b4c5",
						Timestamp: types.Time(time.UnixMilli(1791507600001)),
						Event: FuturesHistoryPositionEvent{
							PositionUpdate: FuturesHistoryPositionUpdate{
								AccountUID:             futuresHistoryTestAccount,
								Tradeable:              "PF_XBTUSD",
								OldPosition:            0.3,
								OldAverageEntryPrice:   81000.5,
								NewPosition:            0.3,
								NewAverageEntryPrice:   81000.5,
								PositionChange:         "noChange",
								FundingRealisationTime: types.Time(time.UnixMilli(1791507600000)),
								RealisedFunding:        -0.42,
								Timestamp:              types.Time(time.UnixMilli(1791507600001)),
								UpdateReason:           "fundingRealisation",
							},
						},
					},
				},
				ContinuationToken: futuresHistoryTestNextToken,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetFuturesPositionEvents(t.Context(), tc.req)
			require.NoError(t, err, "GetFuturesPositionEvents must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetFuturesPositionEvents should decode every field")
				return
			}
			assert.NotEmpty(t, result.AccountUID, "GetFuturesPositionEvents should return the account")
		})
	}
}

func TestGetFuturesAccountLog(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesAccountLog(t.Context(), &FuturesHistoryAccountLogRequest{Since: futuresHistoryTestBefore, Before: futuresHistoryTestSince})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetFuturesAccountLog must reject a reversed window")
	_, err = e.GetFuturesAccountLog(t.Context(), &FuturesHistoryAccountLogRequest{FromID: 10, ToID: 9})
	require.ErrorIs(t, err, errInvalidAccountLogRange, "GetFuturesAccountLog must reject a reversed ID range")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name string
		req  *FuturesHistoryAccountLogRequest
		exp  *FuturesHistoryAccountLogResponse
	}{
		{
			name: "nil request",
			exp:  &FuturesHistoryAccountLogResponse{AccountUID: futuresHistoryTestAccount, Logs: []FuturesHistoryAccountLogEntry{}},
		},
		{
			name: "every parameter",
			req: &FuturesHistoryAccountLogRequest{
				Since:             futuresHistoryTestSince,
				Before:            futuresHistoryTestBefore,
				FromID:            1,
				ToID:              5000,
				Ascending:         true,
				EntryTypes:        []string{"futures liquidation", "conversion"},
				Count:             2,
				ConversionDetails: true,
			},
			exp: &FuturesHistoryAccountLogResponse{
				AccountUID: futuresHistoryTestAccount,
				Logs: []FuturesHistoryAccountLogEntry{
					{
						Asset:                "pf_xbtusd",
						BookingUID:           "f1c6d7e8-f9a0-475e-6f70-8192a3b4c5d6",
						Collateral:           currency.NewCode("usd"),
						Contract:             "pf_xbtusd",
						Date:                 time.Date(2026, 10, 9, 0, 0, 0, 455000000, time.UTC),
						ExecutionID:          "9cef6672-8ab8-4d34-8aed-0fd45a9c9851",
						Fee:                  0.0040845,
						FundingRate:          1.64955714332e-7,
						ID:                   4242,
						EntryType:            "futures liquidation",
						MarginAccount:        "flex",
						MarkPrice:            81698.35759703598,
						NewAverageEntryPrice: 81344.5,
						NewBalance:           0.3,
						OldAverageEntryPrice: 81000.25,
						OldBalance:           0.5,
						RealisedFunding:      -1.31965e-6,
						RealisedPNL:          -12.5,
						TradePrice:           81689.5,
						LiquidationFee:       1.75,
					},
					{
						Asset:                      "eth",
						BookingUID:                 "02d7e8f9-a0b1-486f-7081-92a3b4c5d6e7",
						Collateral:                 currency.NewCode("eth"),
						Date:                       time.Date(2026, 10, 9, 0, 30, 0, 0, time.UTC),
						ExecutionID:                "0f1e2d3c-4b5a-4968-8776-655443322110",
						ID:                         4243,
						EntryType:                  "conversion",
						MarginAccount:              "flex",
						NewBalance:                 1.4985,
						OldBalance:                 0.25,
						ConversionSpreadPercentage: 0.25,
						ExchangeRate:               2450.12,
						ConversionFee:              0.05,
						ExchangeRateFrom:           currency.NewCode("eth"),
					},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetFuturesAccountLog(t.Context(), tc.req)
			require.NoError(t, err, "GetFuturesAccountLog must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetFuturesAccountLog should decode every field")
				return
			}
			assert.NotEmpty(t, result.AccountUID, "GetFuturesAccountLog should return the account")
		})
	}
}

func TestGetFuturesAccountLogCSV(t *testing.T) {
	t.Parallel()
	const file = "date,uid,info\n2026-10-09T00:30:00Z,02d7e8f9-a0b1-486f-7081-92a3b4c5d6e7,conversion\n"
	ex := newHTTPTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method, "Method should be GET")
		assert.Equal(t, "/api/history/v3/accountlogcsv", r.URL.Path, "path should be the account log CSV's")
		assert.Equal(t, "test-key", r.Header.Get("APIKey"), "APIKey should be the key")
		if r.URL.RawQuery != "conversion_details=true" {
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"status":"unauthorized","reason":"You are not authorized to access this endpoint."}`))
			return
		}
		w.Header().Set("Content-Type", "text/csv")
		_, _ = w.Write([]byte(file))
	})
	result, err := ex.GetFuturesAccountLogCSV(t.Context(), &FuturesHistoryAccountLogCSVRequest{ConversionDetails: true})
	require.NoError(t, err, "GetFuturesAccountLogCSV must not error")
	assert.Equal(t, file, string(result), "GetFuturesAccountLogCSV should return the file as sent")

	_, err = ex.GetFuturesAccountLogCSV(t.Context(), nil)
	assert.ErrorIs(t, err, errAPIResponse, "GetFuturesAccountLogCSV should return the history API's error")
	assert.ErrorIs(t, err, request.ErrBadStatus, "GetFuturesAccountLogCSV should keep the status error")

	if mockTests {
		return
	}
	sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	result, err = e.GetFuturesAccountLogCSV(t.Context(), nil)
	require.NoError(t, err, "GetFuturesAccountLogCSV must not error")
	assert.NotEmpty(t, result, "GetFuturesAccountLogCSV should return a file")
}

func TestGetFuturesPublicExecutionEvents(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesPublicExecutionEvents(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFuturesPublicExecutionEvents must reject a nil request")
	_, err = e.GetFuturesPublicExecutionEvents(t.Context(), &FuturesHistoryMarketEventsRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetFuturesPublicExecutionEvents must reject an empty pair")
	_, err = e.GetFuturesPublicExecutionEvents(t.Context(), &FuturesHistoryMarketEventsRequest{Pair: futuresTestPair, Since: futuresHistoryTestBefore, Before: futuresHistoryTestSince})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetFuturesPublicExecutionEvents must reject a reversed window")

	result, err := e.GetFuturesPublicExecutionEvents(t.Context(), &FuturesHistoryMarketEventsRequest{
		Pair:      futuresTestPair,
		Since:     futuresHistoryTestSince,
		Before:    futuresHistoryTestBefore,
		Ascending: true,
		Count:     2,
	})
	require.NoError(t, err, "GetFuturesPublicExecutionEvents must not error")
	if mockTests {
		taker := FuturesHistoryPublicOrder{
			UID:            "a2efb3f4-c1c5-4ddb-ba1d-c02d4dde1a7f",
			Tradeable:      "PF_XBTUSD",
			Direction:      "Buy",
			Quantity:       0.0007,
			Timestamp:      types.Time(time.UnixMilli(1791504000452)),
			LimitPrice:     81689,
			OrderType:      "IoC",
			LastUpdateTime: types.Time(time.UnixMilli(1791504000455)),
		}
		oldTaker := taker
		oldTaker.LastUpdateTime = types.Time(time.UnixMilli(1791504000453))
		partlyFilledTaker := taker
		partlyFilledTaker.Quantity = 0.0006
		exp := &FuturesHistoryPublicExecutionEventsResponse{
			Length: 2,
			Elements: []FuturesHistoryPublicExecutionElement{
				{
					UID:       "250a7f18-0ea6-48ce-b77b-46240b84026b",
					Timestamp: types.Time(time.UnixMilli(1791504000455)),
					Event: FuturesHistoryPublicExecutionEvent{
						Execution: FuturesHistoryPublicExecutionDetails{
							Execution: FuturesHistoryPublicExecution{
								UID: "9cef6672-8ab8-4d34-8aed-0fd45a9c9851",
								MakerOrder: FuturesHistoryPublicOrder{
									UID:            "a2efb395-36a7-47cd-9c16-314d7e602c8a",
									Tradeable:      "PF_XBTUSD",
									Direction:      "Sell",
									Quantity:       0.0001,
									Timestamp:      types.Time(time.UnixMilli(1791503937839)),
									LimitPrice:     81689,
									OrderType:      "Post",
									LastUpdateTime: types.Time(time.UnixMilli(1791504000455)),
								},
								TakerOrder:  taker,
								Timestamp:   types.Time(time.UnixMilli(1791504000455)),
								Quantity:    0.0001,
								Price:       81689,
								MarkPrice:   81698.35759703598,
								LimitFilled: true,
								USDValue:    8.17,
							},
							TakerReducedQuantity: 0.0003,
						},
					},
				},
				{
					UID:       "f5da9c7e-380e-4a49-95d1-4515d254dcdd",
					Timestamp: types.Time(time.UnixMilli(1791504000455)),
					Event: FuturesHistoryPublicExecutionEvent{
						Execution: FuturesHistoryPublicExecutionDetails{
							Execution: FuturesHistoryPublicExecution{
								UID: "c75bf7bf-c56e-4434-b1f8-809dfdae866b",
								MakerOrder: FuturesHistoryPublicOrder{
									UID:            "a2efb39a-6482-4592-a9b8-c0331b255cd2",
									Tradeable:      "PF_XBTUSD",
									Direction:      "Sell",
									Quantity:       0.0002,
									Timestamp:      types.Time(time.UnixMilli(1791503941233)),
									LimitPrice:     81689,
									OrderType:      "Post",
									LastUpdateTime: types.Time(time.UnixMilli(1791504000455)),
								},
								TakerOrder:    partlyFilledTaker,
								Timestamp:     types.Time(time.UnixMilli(1791504000455)),
								Quantity:      0.0002,
								Price:         81689,
								MarkPrice:     81698.35759703598,
								LimitFilled:   true,
								OldTakerOrder: &oldTaker,
								USDValue:      16.34,
							},
						},
					},
				},
			},
			ContinuationToken: "MTc5MTUwNDAwMDQ1NS82MTcyNzE4Nzc0Njk=",
		}
		assert.Equal(t, exp, result, "GetFuturesPublicExecutionEvents should decode every field")
	} else {
		assert.Len(t, result.Elements, 2, "GetFuturesPublicExecutionEvents should return the requested number of trades")
	}

	require.NotEmpty(t, result.ContinuationToken, "GetFuturesPublicExecutionEvents must return a continuation token")
	result, err = e.GetFuturesPublicExecutionEvents(t.Context(), &FuturesHistoryMarketEventsRequest{
		Pair:              futuresTestPair,
		Ascending:         true,
		ContinuationToken: result.ContinuationToken,
		Count:             2,
	})
	require.NoError(t, err, "GetFuturesPublicExecutionEvents must not error continuing the listing")
	if !mockTests {
		assert.Len(t, result.Elements, 2, "GetFuturesPublicExecutionEvents should continue the listing")
		return
	}
	taker := FuturesHistoryPublicOrder{
		UID:            "a2efb3f4-c1c5-4ddb-ba1d-c02d4dde1a7f",
		Tradeable:      "PF_XBTUSD",
		Direction:      "Buy",
		Quantity:       0.0004,
		Timestamp:      types.Time(time.UnixMilli(1791504000455)),
		LimitPrice:     81689,
		OrderType:      "IoC",
		LastUpdateTime: types.Time(time.UnixMilli(1791504000455)),
	}
	nextTaker := taker
	nextTaker.Quantity = 0.0003
	exp := &FuturesHistoryPublicExecutionEventsResponse{
		Length: 2,
		Elements: []FuturesHistoryPublicExecutionElement{
			{
				UID:       "0e92029a-f9f0-4a75-8887-4c152af49fbd",
				Timestamp: types.Time(time.UnixMilli(1791504000455)),
				Event: FuturesHistoryPublicExecutionEvent{
					Execution: FuturesHistoryPublicExecutionDetails{
						Execution: FuturesHistoryPublicExecution{
							UID: "e713e9a5-45fe-4668-9631-3b6e0dad20bc",
							MakerOrder: FuturesHistoryPublicOrder{
								UID:            "a2efb3a9-c9e0-4051-876a-a29fa329449f",
								Tradeable:      "PF_XBTUSD",
								Direction:      "Sell",
								Quantity:       0.0001,
								Timestamp:      types.Time(time.UnixMilli(1791503951323)),
								LimitPrice:     81689,
								OrderType:      "Post",
								LastUpdateTime: types.Time(time.UnixMilli(1791503951323)),
							},
							TakerOrder:  taker,
							Timestamp:   types.Time(time.UnixMilli(1791504000455)),
							Quantity:    0.0001,
							Price:       81689,
							MarkPrice:   81698.35759703598,
							LimitFilled: true,
							USDValue:    8.17,
						},
					},
				},
			},
			{
				UID:       "75fa7466-383d-48b5-8392-9aff793bba7a",
				Timestamp: types.Time(time.UnixMilli(1791504000455)),
				Event: FuturesHistoryPublicExecutionEvent{
					Execution: FuturesHistoryPublicExecutionDetails{
						Execution: FuturesHistoryPublicExecution{
							UID: "34328430-c402-427f-a339-37c452655b90",
							MakerOrder: FuturesHistoryPublicOrder{
								UID:            "a2efb3b9-9955-4218-a21c-c4b6dab41fb4",
								Tradeable:      "PF_XBTUSD",
								Direction:      "Sell",
								Quantity:       0.0001,
								Timestamp:      types.Time(time.UnixMilli(1791503961685)),
								LimitPrice:     81689,
								OrderType:      "Post",
								LastUpdateTime: types.Time(time.UnixMilli(1791503961685)),
							},
							TakerOrder:  nextTaker,
							Timestamp:   types.Time(time.UnixMilli(1791504000455)),
							Quantity:    0.0001,
							Price:       81689,
							MarkPrice:   81698.35759703598,
							LimitFilled: true,
							USDValue:    8.17,
						},
					},
				},
			},
		},
		ContinuationToken: "MTc5MTUwNDAwMDQ1NS82MTcyNzE4Nzc0NzE=",
	}
	assert.Equal(t, exp, result, "GetFuturesPublicExecutionEvents should decode a continued listing")
}

func TestGetFuturesPublicOrderEvents(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesPublicOrderEvents(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFuturesPublicOrderEvents must reject a nil request")
	_, err = e.GetFuturesPublicOrderEvents(t.Context(), &FuturesHistoryMarketEventsRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetFuturesPublicOrderEvents must reject an empty pair")
	_, err = e.GetFuturesPublicOrderEvents(t.Context(), &FuturesHistoryMarketEventsRequest{Pair: futuresTestPair, Since: futuresHistoryTestBefore, Before: futuresHistoryTestSince})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetFuturesPublicOrderEvents must reject a reversed window")

	result, err := e.GetFuturesPublicOrderEvents(t.Context(), &FuturesHistoryMarketEventsRequest{
		Pair:      futuresTestPair,
		Since:     futuresHistoryTestSince,
		Before:    futuresHistoryTestBefore,
		Ascending: true,
		Count:     2,
	})
	require.NoError(t, err, "GetFuturesPublicOrderEvents must not error")
	if mockTests {
		edited := FuturesHistoryPublicOrder{
			UID:            "a2efc022-1b1a-41cc-992f-34d3f6504ede",
			Tradeable:      "PF_XBTUSD",
			Direction:      "Buy",
			Quantity:       0.0626,
			Timestamp:      types.Time(time.UnixMilli(1791506043440)),
			LimitPrice:     81761,
			OrderType:      "Post",
			LastUpdateTime: types.Time(time.UnixMilli(1791506051092)),
		}
		repriced := edited
		repriced.LimitPrice = 81759
		repriced.LastUpdateTime = types.Time(time.UnixMilli(1791506051169))
		exp := &FuturesHistoryPublicOrderEventsResponse{
			Length: 2,
			Elements: []FuturesHistoryPublicOrderElement{
				{
					UID:       "de77d0fd-f96b-48af-8621-4c9984640ed3",
					Timestamp: types.Time(time.UnixMilli(1791504000004)),
					Event: FuturesHistoryPublicOrderEvent{
						OrderPlaced: &FuturesHistoryPublicOrderPlaced{
							Order: FuturesHistoryPublicOrder{
								UID:            "a2efb3f4-11ce-418c-94f9-c84128968616",
								Tradeable:      "PF_XBTUSD",
								Direction:      "Buy",
								Quantity:       0.0001,
								Timestamp:      types.Time(time.UnixMilli(1791504000004)),
								LimitPrice:     81684,
								OrderType:      "Post",
								LastUpdateTime: types.Time(time.UnixMilli(1791504000004)),
							},
							Reason: "new_user_order",
						},
					},
				},
				{
					UID:       "419afb50-4938-465c-a288-6893604810e3",
					Timestamp: types.Time(time.UnixMilli(1791506051169)),
					Event: FuturesHistoryPublicOrderEvent{
						OrderUpdated: &FuturesHistoryPublicOrderUpdated{OldOrder: edited, NewOrder: repriced, Reason: "edited_by user", ReducedQuantity: 0.0012},
					},
				},
			},
			ContinuationToken: "MTc5MTUwNDAwMDA1MC82MTcyNzE4NzE3ODU=",
		}
		assert.Equal(t, exp, result, "GetFuturesPublicOrderEvents should decode every field")
	} else {
		assert.Len(t, result.Elements, 2, "GetFuturesPublicOrderEvents should return the requested number of events")
	}

	require.NotEmpty(t, result.ContinuationToken, "GetFuturesPublicOrderEvents must return a continuation token")
	result, err = e.GetFuturesPublicOrderEvents(t.Context(), &FuturesHistoryMarketEventsRequest{
		Pair:              futuresTestPair,
		Ascending:         true,
		ContinuationToken: result.ContinuationToken,
		Count:             2,
	})
	require.NoError(t, err, "GetFuturesPublicOrderEvents must not error continuing the listing")
	if !mockTests {
		assert.Len(t, result.Elements, 2, "GetFuturesPublicOrderEvents should continue the listing")
		return
	}
	exp := &FuturesHistoryPublicOrderEventsResponse{
		Length: 2,
		Elements: []FuturesHistoryPublicOrderElement{
			{
				UID:       "e71001e6-529b-403c-832b-611314a24a0b",
				Timestamp: types.Time(time.UnixMilli(1791504000050)),
				Event: FuturesHistoryPublicOrderEvent{
					OrderCancelled: &FuturesHistoryPublicOrderCancelled{
						Order: FuturesHistoryPublicOrder{
							UID:            "a2efb22d-a547-45ba-bef9-e585fb71c8a1",
							Tradeable:      "PF_XBTUSD",
							Direction:      "Sell",
							Quantity:       0.0004,
							Timestamp:      types.Time(time.UnixMilli(1791503702193)),
							LimitPrice:     81705,
							OrderType:      "Limit",
							ReduceOnly:     true,
							LastUpdateTime: types.Time(time.UnixMilli(1791503702193)),
						},
						Reason: "cancelled_by_user",
					},
				},
			},
			{
				UID:       "9229b7e9-4554-44a6-a979-6f6d2792abf9",
				Timestamp: types.Time(time.UnixMilli(1791506051079)),
				Event: FuturesHistoryPublicOrderEvent{
					OrderRejected: &FuturesHistoryPublicOrderRejected{
						Order: FuturesHistoryPublicOrder{
							UID:            "a2efc02d-c2f0-4a8a-8158-135d45d0cf8e",
							Tradeable:      "PF_XBTUSD",
							Direction:      "Buy",
							Quantity:       0.0223,
							Timestamp:      types.Time(time.UnixMilli(1791506051079)),
							LimitPrice:     81784,
							OrderType:      "Post",
							LastUpdateTime: types.Time(time.UnixMilli(1791506051079)),
						},
						OrderError: "post_would_execute",
						Reason:     "new_user_order",
					},
				},
			},
		},
		ContinuationToken: "MTc5MTUwNDAwMDA1MC82MTcyNzE4NzE3ODc=",
	}
	assert.Equal(t, exp, result, "GetFuturesPublicOrderEvents should decode a continued listing")
}

func TestGetFuturesPublicMarkPriceEvents(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesPublicMarkPriceEvents(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFuturesPublicMarkPriceEvents must reject a nil request")
	_, err = e.GetFuturesPublicMarkPriceEvents(t.Context(), &FuturesHistoryMarketEventsRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetFuturesPublicMarkPriceEvents must reject an empty pair")
	_, err = e.GetFuturesPublicMarkPriceEvents(t.Context(), &FuturesHistoryMarketEventsRequest{Pair: futuresTestPair, Since: futuresHistoryTestBefore, Before: futuresHistoryTestSince})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetFuturesPublicMarkPriceEvents must reject a reversed window")

	result, err := e.GetFuturesPublicMarkPriceEvents(t.Context(), &FuturesHistoryMarketEventsRequest{
		Pair:      futuresTestPair,
		Since:     futuresHistoryTestSince,
		Before:    futuresHistoryTestBefore,
		Ascending: true,
		Count:     2,
	})
	require.NoError(t, err, "GetFuturesPublicMarkPriceEvents must not error")
	if mockTests {
		exp := &FuturesHistoryPublicMarkPriceEventsResponse{
			Length: 2,
			Elements: []FuturesHistoryPublicMarkPriceElement{
				{
					UID:       "f3e8f9a0-b1c2-497a-8192-a3b4c5d6e7f8",
					Timestamp: types.Time(time.UnixMilli(1791504000001)),
					Event:     FuturesHistoryPublicMarkPriceEvent{MarkPriceChanged: FuturesHistoryMarkPriceChanged{Price: 81698.35759703598}},
				},
				{
					Timestamp: types.Time(time.UnixMilli(1791504001001)),
					Event:     FuturesHistoryPublicMarkPriceEvent{MarkPriceChanged: FuturesHistoryMarkPriceChanged{Price: 81698.59355851753}},
				},
			},
			ContinuationToken: "MTc5MTUwNDAwMjAwMS82MTcyNzE4OTMyMzM=",
		}
		assert.Equal(t, exp, result, "GetFuturesPublicMarkPriceEvents should decode every field")
	} else {
		require.Len(t, result.Elements, 2, "GetFuturesPublicMarkPriceEvents must return the requested number of prices")
		assert.Positive(t, result.Elements[0].Event.MarkPriceChanged.Price.Float64(), "GetFuturesPublicMarkPriceEvents should return a mark price")
	}

	require.NotEmpty(t, result.ContinuationToken, "GetFuturesPublicMarkPriceEvents must return a continuation token")
	result, err = e.GetFuturesPublicMarkPriceEvents(t.Context(), &FuturesHistoryMarketEventsRequest{
		Pair:              futuresTestPair,
		Ascending:         true,
		ContinuationToken: result.ContinuationToken,
		Count:             2,
	})
	require.NoError(t, err, "GetFuturesPublicMarkPriceEvents must not error continuing the listing")
	if !mockTests {
		assert.Len(t, result.Elements, 2, "GetFuturesPublicMarkPriceEvents should continue the listing")
		return
	}
	exp := &FuturesHistoryPublicMarkPriceEventsResponse{
		Length: 2,
		Elements: []FuturesHistoryPublicMarkPriceElement{
			{
				Timestamp: types.Time(time.UnixMilli(1791504002001)),
				Event:     FuturesHistoryPublicMarkPriceEvent{MarkPriceChanged: FuturesHistoryMarkPriceChanged{Price: 81699.28590958091}},
			},
			{
				Timestamp: types.Time(time.UnixMilli(1791504003001)),
				Event:     FuturesHistoryPublicMarkPriceEvent{MarkPriceChanged: FuturesHistoryMarkPriceChanged{Price: 81698.92939928537}},
			},
		},
		ContinuationToken: "MTc5MTUwNDAwNDAwMS82MTcyNzE5MDg1MTA=",
	}
	assert.Equal(t, exp, result, "GetFuturesPublicMarkPriceEvents should decode a continued listing")
}

func TestFuturesHistoryExecutionEventKeyCase(t *testing.T) {
	t.Parallel()
	for _, key := range []string{"execution", "Execution"} {
		var event FuturesHistoryExecutionEvent
		err := json.Unmarshal([]byte(`{"`+key+`":{"execution":{"uid":"9cef6672-8ab8-4d34-8aed-0fd45a9c9851"},"takerReducedQuantity":""}}`), &event)
		require.NoErrorf(t, err, "Unmarshal must not error for key %s", key)
		assert.Equalf(t, "9cef6672-8ab8-4d34-8aed-0fd45a9c9851", event.Execution.Execution.UID, "Unmarshal should decode the execution under key %s", key)
	}
}

func TestFuturesHistoryPagingParams(t *testing.T) {
	t.Parallel()
	since, before := time.UnixMilli(1791504000123), time.UnixMilli(1791590400456)
	_, err := futuresHistoryPagingParams(before, since, false, "", 0)
	assert.ErrorIs(t, err, common.ErrStartAfterEnd, "futuresHistoryPagingParams should reject a range that ends before it starts")
	params, err := futuresHistoryPagingParams(time.Time{}, time.Time{}, false, "", 0)
	require.NoError(t, err, "futuresHistoryPagingParams must not error")
	assert.Empty(t, params, "futuresHistoryPagingParams should set nothing unasked")
	params, err = futuresHistoryPagingParams(since, before, true, "MTc5MTUwNDAwMDQ1NS82MTcyNzE4Nzc0Njk=", 100)
	require.NoError(t, err, "futuresHistoryPagingParams must not error")
	exp := url.Values{"since": {"1791504000123"}, "before": {"1791590400456"}, "sort": {"asc"}, "continuation_token": {"MTc5MTUwNDAwMDQ1NS82MTcyNzE4Nzc0Njk="}, "count": {"100"}}
	assert.Equal(t, exp, params, "futuresHistoryPagingParams should set every paging parameter in milliseconds")
}

func TestFuturesHistoryAccountParams(t *testing.T) {
	t.Parallel()
	since, before := time.UnixMilli(1791504000123), time.UnixMilli(1791590400456)
	_, err := e.futuresHistoryAccountParams(futuresTestPair, before, since, false, "", 0)
	assert.ErrorIs(t, err, common.ErrStartAfterEnd, "futuresHistoryAccountParams should reject a range that ends before it starts")
	for _, pair := range partialTestPairs {
		_, err = e.futuresHistoryAccountParams(pair, time.Time{}, time.Time{}, false, "", 0)
		assert.ErrorIsf(t, err, currency.ErrCurrencyCodeEmpty, "futuresHistoryAccountParams should reject %q, which lacks a base or a quote", pair)
	}
	params, err := e.futuresHistoryAccountParams(futuresTestPair, time.Time{}, time.Time{}, false, "", 0)
	require.NoError(t, err, "futuresHistoryAccountParams must not error")
	assert.Equal(t, url.Values{"tradeable": {"PF_XBTUSD"}}, params, "futuresHistoryAccountParams should filter by the pair's market")
	params, err = e.futuresHistoryAccountParams(currency.EMPTYPAIR, since, time.Time{}, false, "", 25)
	require.NoError(t, err, "futuresHistoryAccountParams must not error without a pair")
	assert.Equal(t, url.Values{"since": {"1791504000123"}, "count": {"25"}}, params, "futuresHistoryAccountParams should not filter without a pair")
}

func TestFuturesHistoryMarketParams(t *testing.T) {
	t.Parallel()
	since, before := time.UnixMilli(1791504000123), time.UnixMilli(1791590400456)
	_, _, err := e.futuresHistoryMarketParams(nil)
	assert.ErrorIs(t, err, common.ErrNilPointer, "futuresHistoryMarketParams should reject a nil request")
	_, _, err = e.futuresHistoryMarketParams(&FuturesHistoryMarketEventsRequest{})
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "futuresHistoryMarketParams should reject a request without a pair")
	for _, pair := range partialTestPairs {
		_, _, err = e.futuresHistoryMarketParams(&FuturesHistoryMarketEventsRequest{Pair: pair})
		assert.ErrorIsf(t, err, currency.ErrCurrencyPairEmpty, "futuresHistoryMarketParams should reject %q, which lacks a base or a quote", pair)
	}
	_, _, err = e.futuresHistoryMarketParams(&FuturesHistoryMarketEventsRequest{Pair: futuresTestPair, Since: before, Before: since})
	assert.ErrorIs(t, err, common.ErrStartAfterEnd, "futuresHistoryMarketParams should reject a range that ends before it starts")
	symbol, params, err := e.futuresHistoryMarketParams(&FuturesHistoryMarketEventsRequest{Pair: futuresTestPair, Ascending: true, Count: 10})
	require.NoError(t, err, "futuresHistoryMarketParams must not error")
	assert.Equal(t, "PF_XBTUSD", symbol, "futuresHistoryMarketParams should return the market symbol")
	assert.Equal(t, url.Values{"sort": {"asc"}, "count": {"10"}}, params, "futuresHistoryMarketParams should return the paging parameters")
}
