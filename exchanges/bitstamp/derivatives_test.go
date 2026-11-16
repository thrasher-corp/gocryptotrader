package bitstamp

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/futures"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestGetMarginTiers(t *testing.T) {
	t.Parallel()
	tiers, err := e.GetMarginTiers(t.Context())
	require.NoError(t, err, "GetMarginTiers must not error")
	require.NotEmpty(t, tiers, "GetMarginTiers must return tiers")
	if mockTests {
		require.Len(t, tiers[0].Tiers, 7, "GetMarginTiers must return every tier of the market")
		exp := MarginTiersResponse{
			Market: "BTC/USD-PERP",
			Tiers: []MarginTier{
				{Tier: 1, SizeLimitHigh: 1000000, MaxLeverage: 10, InitialMarginRate: 0.1, MaintenanceMarginRate: 0.05, CloseOutMarginRate: 0.025},
				{
					Tier:                              2,
					SizeLimitLow:                      1000000,
					SizeLimitHigh:                     1500000,
					MaxLeverage:                       7,
					InitialMarginRate:                 0.1429,
					MaintenanceMarginRate:             0.0715,
					CloseOutMarginRate:                0.0358,
					InitialMarginPreviousLevelMax:     100000,
					MaintenanceMarginPreviousLevelMax: 50000,
					CloseOutMarginPreviousLevelMax:    25000,
				},
			},
		}
		assert.Equal(t, exp, MarginTiersResponse{Market: tiers[0].Market, Tiers: tiers[0].Tiers[:2]}, "GetMarginTiers should decode the market's first tiers")
	}
	for _, m := range tiers {
		assert.NotEmpty(t, m.Market, "Market should be set")
		require.NotEmpty(t, m.Tiers, "Tiers must not be empty")
		assert.Equal(t, uint64(1), m.Tiers[0].Tier, "first tier should be tier 1")
		assert.Positive(t, m.Tiers[0].MaxLeverage.Float64(), "MaxLeverage should be positive")
		assert.Greater(t, m.Tiers[0].SizeLimitHigh.Float64(), m.Tiers[0].SizeLimitLow.Float64(), "SizeLimitHigh should exceed SizeLimitLow")
	}
}

func TestGetMarketHours(t *testing.T) {
	t.Parallel()
	hours, err := e.GetMarketHours(t.Context())
	require.NoError(t, err, "GetMarketHours must not error")
	require.NotEmpty(t, hours, "GetMarketHours must return schedules")
	if mockTests {
		require.Len(t, hours, 2, "GetMarketHours must return every market's schedule")
		assert.Equal(t, goldMarketHours(t), hours[0], "GetMarketHours should decode the schedule")
	}
	for _, h := range hours {
		assert.NotEmpty(t, h.Market, "Market should be set")
		assert.NotEmpty(t, h.TradingHours, "TradingHours should be set")
		require.NotEmpty(t, h.ReferenceIndexPublishingHours.Schedule, "Schedule must not be empty")
		assert.True(t, h.ReferenceIndexPublishingHours.Schedule[0].End.After(h.ReferenceIndexPublishingHours.Schedule[0].Start), "schedule entries should end after they start")
	}
}

func TestGetMarketHoursForMarket(t *testing.T) {
	t.Parallel()
	_, err := e.GetMarketHoursForMarket(t.Context(), currency.EMPTYPAIR)
	assert.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetMarketHoursForMarket should error on an empty pair")
	gold, err := e.GetMarketHoursForMarket(t.Context(), currency.NewPair(currency.NewCode("GOLD"), currency.NewCode("USD-PERP")))
	require.NoError(t, err, "GetMarketHoursForMarket must not error")
	if mockTests {
		exp := goldMarketHours(t)
		assert.Equal(t, &exp, gold, "GetMarketHoursForMarket should decode the schedule")
		return
	}
	assert.Equal(t, "GOLD/USD-PERP", gold.Market, "Market should be correct")
	assert.NotZero(t, gold.ReferenceIndexPublishingHours.NextOpen, "NextOpen should be set")
}

// goldMarketHours returns the mock GOLD/USD-PERP reference index publishing schedule
func goldMarketHours(t *testing.T) MarketHoursResponse {
	t.Helper()
	return MarketHoursResponse{
		Market:       "GOLD/USD-PERP",
		TradingHours: "24/7",
		ReferenceIndexPublishingHours: ReferenceIndexPublishingHours{
			IsReferenceIndexPublishing: true,
			CurrentOpen:                parseRFC3339(t, "2026-09-24T22:00:00+00:00"),
			CurrentClose:               parseRFC3339(t, "2026-09-25T21:00:00+00:00"),
			NextOpen:                   parseRFC3339(t, "2026-09-27T22:00:00+00:00"),
			NextClose:                  parseRFC3339(t, "2026-09-28T21:00:00+00:00"),
			Schedule: []ScheduleEntry{
				{Start: parseRFC3339(t, "2026-09-24T22:00:00+00:00"), End: parseRFC3339(t, "2026-09-25T21:00:00+00:00"), Type: ScheduleTypeOpenHours},
				{Start: parseRFC3339(t, "2026-09-25T21:00:00+00:00"), End: parseRFC3339(t, "2026-09-27T22:00:00+00:00"), Type: ScheduleTypeClosedHours},
				{Start: parseRFC3339(t, "2026-09-27T22:00:00+00:00"), End: parseRFC3339(t, "2026-09-28T21:00:00+00:00"), Type: ScheduleTypeOpenHours},
				{Start: parseRFC3339(t, "2026-09-28T21:00:00+00:00"), End: parseRFC3339(t, "2026-09-28T22:00:00+00:00"), Type: ScheduleTypeClosedHours},
			},
		},
	}
}

func TestGetOpenPositions(t *testing.T) {
	t.Parallel()
	skipIfLiveWithoutCredentials(t)
	positions, err := e.GetOpenPositions(t.Context(), currency.EMPTYPAIR)
	require.NoError(t, err, "GetOpenPositions must not error")
	if mockTests {
		assert.Len(t, positions, 3, "GetOpenPositions should return all positions")
	}
	positions, err = e.GetOpenPositions(t.Context(), perpetualPair)
	require.NoError(t, err, "GetOpenPositions must not error for a market")
	if mockTests {
		require.Len(t, positions, 2, "GetOpenPositions must return the market's positions")
		// The cumulative liquidation fees are null, which decodes as zero
		assert.Equal(t, testIsolatedPosition, positions[1], "GetOpenPositions should decode the position")
	}
}

func TestGetPositionStatus(t *testing.T) {
	t.Parallel()
	_, err := e.GetPositionStatus(t.Context(), "")
	assert.ErrorIs(t, err, errPositionIDRequired, "GetPositionStatus should error without a position ID")

	skipIfLiveWithoutCredentials(t)
	status, err := e.GetPositionStatus(t.Context(), "1234567890")
	if !mockTests {
		assert.ErrorIs(t, err, request.ErrAuthRequestFailed, "GetPositionStatus should error for an unknown position")
		return
	}
	require.NoError(t, err, "GetPositionStatus must not error")
	exp := &PositionStatusResponse{
		PositionResponse: testIsolatedPosition,
		TimeOpened:       types.Time(time.UnixMicro(1750068000000000)),
		Status:           PositionStatusOpen,
	}
	assert.Equal(t, exp, status, "GetPositionStatus should decode the open position")
}

func TestGetPositionHistory(t *testing.T) { //nolint:tparallel // Cases run serially through the shared exchange
	t.Parallel()
	_, err := e.GetPositionHistory(t.Context(), nil)
	assert.ErrorIs(t, err, common.ErrNilPointer, "GetPositionHistory should error on a nil request")
	_, err = e.GetPositionHistory(t.Context(), &PositionHistoryRequest{Limit: 1001})
	assert.ErrorIs(t, err, errInvalidLimit, "GetPositionHistory should error on an invalid limit")

	skipIfLiveWithoutCredentials(t)
	for _, tc := range []struct {
		name string
		req  *PositionHistoryRequest
	}{
		{name: "descending with a limit", req: &PositionHistoryRequest{Limit: 10, Sort: SortDescending}},
		{name: "market since a position ID", req: &PositionHistoryRequest{Pair: perpetualPair, SinceID: "1234567870"}},
	} {
		// Serial and in order, as every case goes through the same exchange
		t.Run(tc.name, func(t *testing.T) {
			history, err := e.GetPositionHistory(t.Context(), tc.req)
			require.NoError(t, err, "GetPositionHistory must not error")
			if mockTests {
				require.Len(t, history, 1, "GetPositionHistory must return the positions")
				exp := PositionHistoryResponse{
					ID:                       "1234567880",
					Market:                   "BTC/USD-PERP",
					MarketType:               MarketTypePerpetual,
					MarginMode:               MarginModeIsolated,
					PNLCurrency:              currency.USD,
					EntryPrice:               80000,
					PNLPercentage:            5,
					PNLRealised:              40,
					PNLSettled:               40,
					Leverage:                 3,
					PNL:                      40,
					CumulativePricePNL:       42,
					CumulativeTradingFees:    1.6,
					CumulativeFunding:        -0.4,
					AmountDelta:              0.01,
					TimeOpened:               types.Time(time.UnixMicro(1750068000000000)),
					TimeClosed:               types.Time(time.UnixMicro(1750154400000000)),
					Status:                   PositionStatusSettled,
					ExitPrice:                84000,
					SettlementPrice:          84000,
					CumulativeSocialisedLoss: 0,
				}
				assert.Equal(t, exp, history[0], "GetPositionHistory should return the position")
			}
		})
	}
}

func TestClosePositions(t *testing.T) {
	t.Parallel()
	t.Run("without positions", func(t *testing.T) {
		t.Parallel()
		// The API answers HTTP 200 with response code 400.043 when there is no position to close
		ex := newResponseTestExchange(t, http.StatusOK, `{"response_code":"400.043","response_explanation":"No positions present."}`)
		closed, err := ex.ClosePosition(t.Context(), "1234567890")
		assert.ErrorIs(t, err, futures.ErrNoPositionsFound, "ClosePosition should report that there is no position to close")
		assert.Nil(t, closed, "ClosePosition should not return a closed position")
		closedAll, err := ex.ClosePositions(t.Context(), &ClosePositionsRequest{Pair: perpetualPair})
		assert.ErrorIs(t, err, futures.ErrNoPositionsFound, "ClosePositions should report that there are no positions to close")
		assert.Nil(t, closedAll, "ClosePositions should not return closed positions")
	})
	_, err := e.ClosePositions(t.Context(), nil)
	assert.ErrorIs(t, err, common.ErrNilPointer, "ClosePositions should error on a nil request")

	skipIfLiveWithoutCredentials(t, canManipulateRealOrders)
	resp, err := e.ClosePositions(t.Context(), &ClosePositionsRequest{Pair: perpetualPair, MarginMode: MarginModeIsolated})
	require.NoError(t, err, "ClosePositions must not error")
	if mockTests {
		closed := testClosedPosition
		closed.ID = "1234567880"
		exp := &ClosePositionsResponse{Closed: []ClosedPositionResponse{closed}, Failed: []ClosedPositionResponse{}}
		assert.Equal(t, exp, resp, "ClosePositions should decode the closed positions")
	}
}

func TestClosePosition(t *testing.T) {
	t.Parallel()
	t.Run("without positions", func(t *testing.T) {
		t.Parallel()
		// The API answers HTTP 200 with response code 400.043 when there is no position to close
		ex := newResponseTestExchange(t, http.StatusOK, `{"response_code":"400.043","response_explanation":"No positions present."}`)
		closed, err := ex.ClosePosition(t.Context(), "1234567890")
		assert.ErrorIs(t, err, futures.ErrNoPositionsFound, "ClosePosition should report that there is no position to close")
		assert.Nil(t, closed, "ClosePosition should not return a closed position")
	})
	_, err := e.ClosePosition(t.Context(), "")
	assert.ErrorIs(t, err, errPositionIDRequired, "ClosePosition should error without a position ID")

	skipIfLiveWithoutCredentials(t, canManipulateRealOrders)
	resp, err := e.ClosePosition(t.Context(), "1234567890")
	if !mockTests {
		assert.ErrorIs(t, err, request.ErrAuthRequestFailed, "ClosePosition should error for an unknown position")
		return
	}
	require.NoError(t, err, "ClosePosition must not error")
	assert.Equal(t, &testClosedPosition, resp, "ClosePosition should decode the closed position")
}

func TestGetPositionSettlementTransactions(t *testing.T) { //nolint:tparallel // Cases run serially through the shared exchange
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *SettlementTransactionsRequest
		err  error
	}{
		{name: "nil request", req: nil, err: common.ErrNilPointer},
		{name: "limit above the maximum", req: &SettlementTransactionsRequest{Limit: 1001}, err: errInvalidLimit},
		{name: "start after end", req: &SettlementTransactionsRequest{Since: time.Unix(2, 0), Until: time.Unix(1, 0)}, err: common.ErrStartAfterEnd},
	} {
		// Serial and in order, as the requests below go through the same exchange
		t.Run(tc.name, func(t *testing.T) {
			_, err := e.GetPositionSettlementTransactions(t.Context(), tc.req)
			assert.ErrorIs(t, err, tc.err, "GetPositionSettlementTransactions should return the correct error")
		})
	}

	skipIfLiveWithoutCredentials(t)
	for _, tc := range []struct {
		name string
		req  *SettlementTransactionsRequest
	}{
		{name: "with a limit", req: &SettlementTransactionsRequest{Limit: 10}},
		{name: "by transaction ID", req: &SettlementTransactionsRequest{TransactionID: "123123"}},
	} {
		// Serial and in order, as every case goes through the same exchange
		t.Run(tc.name, func(t *testing.T) {
			txs, err := e.GetPositionSettlementTransactions(t.Context(), tc.req)
			if tc.req.TransactionID != "" && !mockTests {
				assert.ErrorIs(t, err, request.ErrAuthRequestFailed, "GetPositionSettlementTransactions should error for an unknown transaction")
				return
			}
			require.NoError(t, err, "GetPositionSettlementTransactions must not error")
			if mockTests {
				require.Len(t, txs, 1, "GetPositionSettlementTransactions must return the transactions")
				exp := SettlementTransactionResponse{
					TransactionID:            "123123",
					PositionID:               "1234567890",
					SettlementTime:           types.Time(time.Unix(1750089600, 0)),
					SettlementType:           "PERIODIC",
					SettlementPrice:          84000,
					Market:                   "BTC/USD-PERP",
					MarketType:               MarketTypePerpetual,
					PNLCurrency:              currency.USD,
					PNLSettled:               12.12,
					PNLComponentPrice:        12.5,
					PNLComponentFees:         -0.34,
					PNLComponentFunding:      -0.04,
					MarginMode:               MarginModeIsolated,
					Size:                     0.01,
					StrikePrice:              84000,
					FeesComponentTrading:     0.34,
					FeesComponentLiquidation: 0,
				}
				assert.Equal(t, exp, txs[0], "GetPositionSettlementTransactions should return the transaction")
			}
		})
	}
}

func TestGetDerivativesTradeHistory(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *DerivativesTradeHistoryRequest
		err  error
	}{
		{name: "nil request", req: nil, err: common.ErrNilPointer},
		{name: "without a pair or order ID", req: &DerivativesTradeHistoryRequest{}, err: errPairOrOrderIDRequired},
		{name: "limit above the maximum", req: &DerivativesTradeHistoryRequest{Pair: perpetualPair, Limit: 1001}, err: errInvalidLimit},
		{name: "start after end", req: &DerivativesTradeHistoryRequest{Pair: perpetualPair, Since: time.Unix(2, 0), Until: time.Unix(1, 0)}, err: common.ErrStartAfterEnd},
	} {
		// Serial and in order, as the request below goes through the same exchange
		t.Run(tc.name, func(t *testing.T) {
			_, err := e.GetDerivativesTradeHistory(t.Context(), tc.req)
			assert.ErrorIs(t, err, tc.err, "GetDerivativesTradeHistory should return the correct error")
		})
	}
	t.Run("valid request", func(t *testing.T) {
		t.Parallel()
		skipIfLiveWithoutCredentials(t)
		trades, err := e.GetDerivativesTradeHistory(t.Context(), &DerivativesTradeHistoryRequest{OrderID: 1234123412341235, Limit: 10, Sort: SortAscending})
		require.NoError(t, err, "GetDerivativesTradeHistory must not error")
		if mockTests {
			require.Len(t, trades, 2, "GetDerivativesTradeHistory must return the trades")
			exp := DerivativesTradeResponse{
				TradeID:     1234123412341300,
				OrderID:     1234123412341235,
				PositionID:  "1234567890",
				DateTime:    types.DateTime(time.Date(2025, 6, 16, 10, 1, 0, 849000000, time.UTC)),
				Fee:         0.1344,
				FeeCurrency: currency.USD,
				Market:      "BTC/USD-PERP",
				MarginMode:  MarginModeIsolated,
				Leverage:    3,
				Side:        "SELL",
				Type:        "TRADE",
				Price:       84000,
				Amount:      0.004,
				TradeUTI:    "4851007IRIW87EC5H6978f171f98ee8973c09fecb910a08d173f",
			}
			assert.Equal(t, exp, trades[0], "GetDerivativesTradeHistory should return the trade")
		}
	})
}

func TestGetMarginInfo(t *testing.T) {
	t.Parallel()
	skipIfLiveWithoutCredentials(t)
	info, err := e.GetMarginInfo(t.Context())
	require.NoError(t, err, "GetMarginInfo must not error")
	if mockTests {
		// The implied leverage is null, which decodes as zero
		exp := &MarginInfoResponse{
			AccountMargin:          20000,
			AccountMarginAvailable: 18000,
			AccountMarginReserved:  2000,
			AccountMarginCurrency:  currency.USD,
			Assets: []AssetMarginInfo{
				{Asset: currency.USD, TotalAmount: 15000, Available: 13500, Reserved: 1500, MarginAvailable: 13500},
				{Asset: currency.BTC, TotalAmount: 0.1, Available: 0.09, Reserved: 0.01, MarginAvailable: 7560},
			},
			InitialMarginRatio:     10,
			MaintenanceMarginRatio: 5,
		}
		assert.Equal(t, exp, info, "GetMarginInfo should decode the margin information")
	}
}

func TestGetEstimatedOrderImpact(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *OrderImpactRequest
		err  error
	}{
		{name: "nil request", req: nil, err: common.ErrNilPointer},
		{name: "without a pair", req: &OrderImpactRequest{}, err: currency.ErrCurrencyPairEmpty},
		{name: "without an order type", req: &OrderImpactRequest{Pair: perpetualPair}, err: errOrderTypeRequired},
		{name: "without an amount", req: &OrderImpactRequest{Pair: perpetualPair, OrderType: OrderSubtypeMarket}, err: order.ErrAmountIsInvalid},
		{name: "without a side", req: &OrderImpactRequest{Pair: perpetualPair, OrderType: OrderSubtypeMarket, Amount: 1}, err: order.ErrSideIsInvalid},
		{name: "without a margin mode", req: &OrderImpactRequest{Pair: perpetualPair, OrderType: OrderSubtypeMarket, Amount: 1, Side: order.Sell}, err: errMarginModeRequired},
		{name: "without a leverage", req: &OrderImpactRequest{Pair: perpetualPair, OrderType: OrderSubtypeMarket, Amount: 1, Side: order.Sell, MarginMode: MarginModeCross}, err: errLeverageRequired},
		{name: "negative price", req: &OrderImpactRequest{Pair: perpetualPair, OrderType: OrderSubtypeLimit, Amount: 1, Side: order.Sell, MarginMode: MarginModeCross, Leverage: 3, Price: -1}, err: errInvalidPrice},
		{name: "negative collateral", req: &OrderImpactRequest{Pair: perpetualPair, OrderType: OrderSubtypeMarket, Amount: 1, Side: order.Sell, MarginMode: MarginModeCross, Leverage: 3, AdditionalCollateral: map[currency.Code]float64{currency.USD: -1}}, err: errInvalidCollateral},
	} {
		// Serial and in order, as the request below goes through the same exchange
		t.Run(tc.name, func(t *testing.T) {
			_, err := e.GetEstimatedOrderImpact(t.Context(), tc.req)
			assert.ErrorIs(t, err, tc.err, "GetEstimatedOrderImpact should return the correct error")
		})
	}
	t.Run("valid request", func(t *testing.T) {
		t.Parallel()
		skipIfLiveWithoutCredentials(t)
		resp, err := e.GetEstimatedOrderImpact(t.Context(), &OrderImpactRequest{Pair: perpetualPair, OrderType: OrderSubtypeMarket, Amount: 0.01, Side: order.Sell, MarginMode: MarginModeCross, Leverage: 3, ReduceOnly: true})
		require.NoError(t, err, "GetEstimatedOrderImpact must not error")
		if mockTests {
			exp := &OrderImpactResponse{
				MarginCurrency:             currency.USD,
				AdditionalRequiredMargin:   100,
				EstimatedLiquidationPrices: map[string]types.Number{"BTC/USD-PERP": 90000, "ETH/USD-PERP": 900},
				EstimatedFee:               0.61,
				IsOrderPlaceable:           OrderPlaceableYes,
				MarginTier:                 OrderImpactMarginTier{Breakdown: []MarginTierExposure{{Tier: 1, Exposure: 1000, Leverage: 5}}},
			}
			assert.Equal(t, exp, resp, "GetEstimatedOrderImpact should decode the impact")
		}
	})
}

func TestGetCollateralChangeImpact(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *CollateralChangeImpactRequest
		err  error
	}{
		{name: "nil request", req: nil, err: common.ErrNilPointer},
		{name: "without a margin mode", req: &CollateralChangeImpactRequest{}, err: errMarginModeRequired},
		{name: "without collateral", req: &CollateralChangeImpactRequest{MarginMode: MarginModeCross}, err: errCollateralRequired},
		{name: "target collateral with collateral deltas", req: &CollateralChangeImpactRequest{MarginMode: MarginModeCross, TargetCollateral: map[currency.Code]float64{currency.USD: 1}, CollateralDeltas: map[currency.Code]float64{currency.USD: 1}}, err: errCollateralRequired},
	} {
		// Serial and in order, as the request below goes through the same exchange
		t.Run(tc.name, func(t *testing.T) {
			_, err := e.GetCollateralChangeImpact(t.Context(), tc.req)
			assert.ErrorIs(t, err, tc.err, "GetCollateralChangeImpact should return the correct error")
		})
	}
	t.Run("valid request", func(t *testing.T) {
		t.Parallel()
		skipIfLiveWithoutCredentials(t)
		resp, err := e.GetCollateralChangeImpact(t.Context(), &CollateralChangeImpactRequest{MarginMode: MarginModeCross, CollateralDeltas: map[currency.Code]float64{currency.BTC: -0.01, currency.USD: 120}})
		require.NoError(t, err, "GetCollateralChangeImpact must not error")
		if mockTests {
			exp := &CollateralChangeImpactResponse{
				MarginCurrency:                  currency.USD,
				EstimatedInitialMarginRatio:     50,
				EstimatedMaintenanceMarginRatio: 25,
				EstimatedLiquidationPrices:      map[string]types.Number{"1234567890": 9000},
				TotalEstimatedMargin:            1200.61,
			}
			assert.Equal(t, exp, resp, "GetCollateralChangeImpact should return the impact")
		}
	})
}

func TestGetCollateralCurrencies(t *testing.T) {
	t.Parallel()
	skipIfLiveWithoutCredentials(t)
	currencies, err := e.GetCollateralCurrencies(t.Context())
	require.NoError(t, err, "GetCollateralCurrencies must not error")
	if mockTests {
		assert.Equal(t, []CollateralCurrencyResponse{{Currency: currency.USD}, {Currency: currency.BTC, Haircut: 0.1}}, currencies, "GetCollateralCurrencies should return the currencies")
	}
}

func TestAdjustPositionCollateral(t *testing.T) {
	t.Parallel()
	assert.ErrorIs(t, e.AdjustPositionCollateral(t.Context(), nil), common.ErrNilPointer, "AdjustPositionCollateral should error on a nil request")
	assert.ErrorIs(t, e.AdjustPositionCollateral(t.Context(), &AdjustPositionCollateralRequest{NewAmount: 1}), errPositionIDRequired, "AdjustPositionCollateral should error without a position ID")
	assert.ErrorIs(t, e.AdjustPositionCollateral(t.Context(), &AdjustPositionCollateralRequest{PositionID: "1234567890"}), order.ErrAmountIsInvalid, "AdjustPositionCollateral should error without an amount")

	skipIfLiveWithoutCredentials(t, canManipulateRealOrders)
	err := e.AdjustPositionCollateral(t.Context(), &AdjustPositionCollateralRequest{PositionID: "1234567890", NewAmount: 12.5})
	if !mockTests {
		assert.ErrorIs(t, err, request.ErrAuthRequestFailed, "AdjustPositionCollateral should error for an unknown position")
		return
	}
	assert.NoError(t, err, "AdjustPositionCollateral should not error")
}

func TestGetLeverageSettings(t *testing.T) {
	t.Parallel()
	skipIfLiveWithoutCredentials(t)
	_, err := e.GetLeverageSettings(t.Context(), nil)
	assert.ErrorIs(t, err, common.ErrNilPointer, "GetLeverageSettings should error on a nil request")
	settings, err := e.GetLeverageSettings(t.Context(), &LeverageSettingsRequest{})
	require.NoError(t, err, "GetLeverageSettings must not error")
	if mockTests {
		exp := []LeverageSettingResponse{
			{MarginMode: MarginModeCross, Market: "BTC/USD-PERP", LeverageCurrent: 3, LeverageMax: 10},
			{MarginMode: MarginModeIsolated, Market: "BTC/USD-PERP", LeverageCurrent: 5, LeverageMax: 10},
		}
		assert.Equal(t, exp, settings, "GetLeverageSettings should decode every setting")
	}
	settings, err = e.GetLeverageSettings(t.Context(), &LeverageSettingsRequest{MarginMode: MarginModeCross, Pair: perpetualPair})
	require.NoError(t, err, "GetLeverageSettings must not error with filters")
	if mockTests {
		assert.Equal(t, []LeverageSettingResponse{{MarginMode: MarginModeCross, Market: "BTC/USD-PERP", LeverageCurrent: 3, LeverageMax: 10}}, settings, "GetLeverageSettings should return the filtered settings")
	}
}

func TestUpdateLeverageSetting(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *LeverageSettingRequest
		err  error
	}{
		{name: "nil request", req: nil, err: common.ErrNilPointer},
		{name: "without a pair", req: &LeverageSettingRequest{}, err: currency.ErrCurrencyPairEmpty},
		{name: "without a margin mode", req: &LeverageSettingRequest{Pair: perpetualPair}, err: errMarginModeRequired},
		{name: "leverage below the minimum", req: &LeverageSettingRequest{Pair: perpetualPair, MarginMode: MarginModeIsolated, Leverage: 0.5}, err: errInvalidLeverage},
		{name: "leverage above the maximum", req: &LeverageSettingRequest{Pair: perpetualPair, MarginMode: MarginModeIsolated, Leverage: 26}, err: errInvalidLeverage},
	} {
		// Serial and in order, as the requests below go through the same exchange
		t.Run(tc.name, func(t *testing.T) {
			_, err := e.UpdateLeverageSetting(t.Context(), tc.req)
			assert.ErrorIs(t, err, tc.err, "UpdateLeverageSetting should return the correct error")
		})
	}
	t.Run("valid request", func(t *testing.T) {
		t.Parallel()
		skipIfLiveWithoutCredentials(t, canManipulateRealOrders)
		resp, err := e.UpdateLeverageSetting(t.Context(), &LeverageSettingRequest{Pair: perpetualPair, MarginMode: MarginModeIsolated, Leverage: 5})
		require.NoError(t, err, "UpdateLeverageSetting must not error")
		if !mockTests {
			return
		}
		exp := &LeverageSettingResponse{MarginMode: MarginModeIsolated, Market: "BTC/USD-PERP", LeverageCurrent: 5, LeverageMax: 10}
		assert.Equal(t, exp, resp, "UpdateLeverageSetting should return the setting")
		// The leverage is optional, so the margin mode can be changed without it
		resp, err = e.UpdateLeverageSetting(t.Context(), &LeverageSettingRequest{Pair: perpetualPair, MarginMode: MarginModeIsolated})
		require.NoError(t, err, "UpdateLeverageSetting must not error without a leverage")
		assert.Equal(t, exp, resp, "UpdateLeverageSetting should return the setting without a leverage")
	})
}
