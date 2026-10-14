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
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestGetFuturesMarginMode(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesMarginMode(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "GetFuturesMarginMode must reject an empty symbol")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFuturesMarginMode(t.Context(), "XBTUSDTM")
	require.NoError(t, err, "GetFuturesMarginMode must not error")
	if mockTests {
		exp := &FuturesMarginModeResponse{
			Symbol:     "XBTUSDTM",
			MarginMode: "ISOLATED",
		}
		assert.Equal(t, exp, result, "GetFuturesMarginMode should decode every field")
		return
	}
	require.NotNil(t, result, "GetFuturesMarginMode must return a response")
	assert.NotEmpty(t, result.MarginMode, "GetFuturesMarginMode should return the margin mode")
}

func TestSwitchFuturesMarginMode(t *testing.T) {
	t.Parallel()
	_, err := e.SwitchFuturesMarginMode(t.Context(), "", "CROSS")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "SwitchFuturesMarginMode must reject an empty symbol")
	_, err = e.SwitchFuturesMarginMode(t.Context(), "XBTUSDTM", "")
	require.ErrorIs(t, err, errMarginModeRequired, "SwitchFuturesMarginMode must reject an empty margin mode")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.SwitchFuturesMarginMode(t.Context(), "XBTUSDTM", "CROSS")
	require.NoError(t, err, "SwitchFuturesMarginMode must not error")
	if mockTests {
		exp := &FuturesMarginModeResponse{
			Symbol:     "XBTUSDTM",
			MarginMode: "CROSS",
		}
		assert.Equal(t, exp, result, "SwitchFuturesMarginMode should decode every field")
		return
	}
	require.NotNil(t, result, "SwitchFuturesMarginMode must return a response")
	assert.Equal(t, "CROSS", result.MarginMode, "SwitchFuturesMarginMode should return the new margin mode")
}

func TestBatchSwitchFuturesMarginMode(t *testing.T) {
	t.Parallel()
	_, err := e.BatchSwitchFuturesMarginMode(t.Context(), "", []string{"XBTUSDTM"})
	require.ErrorIs(t, err, errMarginModeRequired, "BatchSwitchFuturesMarginMode must reject an empty margin mode")
	_, err = e.BatchSwitchFuturesMarginMode(t.Context(), "ISOLATED", nil)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "BatchSwitchFuturesMarginMode must reject an empty symbol list")
	_, err = e.BatchSwitchFuturesMarginMode(t.Context(), "ISOLATED", []string{"XBTUSDTM", ""})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "BatchSwitchFuturesMarginMode must reject an empty symbol")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.BatchSwitchFuturesMarginMode(t.Context(), "ISOLATED", []string{"XBTUSDTM", "ETHUSDTM"})
	require.NoError(t, err, "BatchSwitchFuturesMarginMode must not error")
	if mockTests {
		exp := &BatchSwitchFuturesMarginModeResponse{
			MarginMode: map[string]string{
				"ETHUSDTM": "ISOLATED",
				"XBTUSDTM": "CROSS",
			},
			Errors: []FuturesMarginModeError{
				{
					Code:    "50002",
					Message: "exist.order.or.position",
					Symbol:  "XBTUSDTM",
				},
			},
		}
		assert.Equal(t, exp, result, "BatchSwitchFuturesMarginMode should decode every field")
		return
	}
	require.NotNil(t, result, "BatchSwitchFuturesMarginMode must return a response")
	assert.Len(t, result.MarginMode, 2, "BatchSwitchFuturesMarginMode should return each symbol's margin mode")
}

func TestGetFuturesPositionMode(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFuturesPositionMode(t.Context())
	require.NoError(t, err, "GetFuturesPositionMode must not error")
	if mockTests {
		exp := &FuturesPositionModeResponse{
			PositionMode: 1,
		}
		assert.Equal(t, exp, result, "GetFuturesPositionMode should decode every field")
		return
	}
	assert.NotNil(t, result, "GetFuturesPositionMode should return a response")
}

func TestSwitchFuturesPositionMode(t *testing.T) {
	t.Parallel()
	_, err := e.SwitchFuturesPositionMode(t.Context(), "")
	require.ErrorIs(t, err, errPositionModeRequired, "SwitchFuturesPositionMode must reject an empty position mode")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.SwitchFuturesPositionMode(t.Context(), "1")
	require.NoError(t, err, "SwitchFuturesPositionMode must not error")
	if mockTests {
		exp := &FuturesPositionModeResponse{
			PositionMode: 1,
		}
		assert.Equal(t, exp, result, "SwitchFuturesPositionMode should decode every field")
		return
	}
	require.NotNil(t, result, "SwitchFuturesPositionMode must return a response")
	assert.Equal(t, types.Number(1), result.PositionMode, "SwitchFuturesPositionMode should return hedge mode")
}

func TestGetMaximumOpenPositionSize(t *testing.T) {
	t.Parallel()
	_, err := e.GetMaximumOpenPositionSize(t.Context(), "", 1, 1)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "GetMaximumOpenPositionSize must reject an empty symbol")
	_, err = e.GetMaximumOpenPositionSize(t.Context(), "XBTUSDTM", 0, 1)
	require.ErrorIs(t, err, limits.ErrPriceBelowMin, "GetMaximumOpenPositionSize must reject a zero price")
	_, err = e.GetMaximumOpenPositionSize(t.Context(), "XBTUSDTM", 1, 0)
	require.ErrorIs(t, err, errInvalidLeverage, "GetMaximumOpenPositionSize must reject a zero leverage")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetMaximumOpenPositionSize(t.Context(), "XBTUSDTM", 1, 1)
	require.NoError(t, err, "GetMaximumOpenPositionSize must not error")
	if mockTests {
		exp := &FuturesMaxOpenSizeResponse{
			Symbol:          "XBTUSDTM",
			MaxBuyOpenSize:  1000000,
			MaxSellOpenSize: 999000,
		}
		assert.Equal(t, exp, result, "GetMaximumOpenPositionSize should decode every field")
		return
	}
	require.NotNil(t, result, "GetMaximumOpenPositionSize must return a response")
	assert.Equal(t, "XBTUSDTM", result.Symbol, "GetMaximumOpenPositionSize should return the requested contract")
}

func TestGetFuturesPositionDetails(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesPositionDetails(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "GetFuturesPositionDetails must reject an empty symbol")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	// Hedge mode isolated margin positions and an inverse cross margin position between them populate every field
	for _, tc := range []struct {
		symbol string
		exp    []FuturesPositionDetail
	}{
		{symbol: "XBTUSDTM", exp: []FuturesPositionDetail{
			{
				ID:                           "500000000001478576",
				Symbol:                       "XBTUSDTM",
				MaintenanceMarginRequirement: 0.004,
				DeleveragePercentage:         0.09,
				OpeningTimestamp:             types.Time(time.UnixMilli(1759395308117)),
				CurrentTimestamp:             types.Time(time.UnixMilli(1759395824264)),
				CurrentQuantity:              1,
				CurrentCost:                  118.6371,
				CurrentCommission:            0.07118226,
				UnrealisedCost:               118.6371,
				RealisedGrossCost:            -0.0215,
				RealisedCost:                 0.04968226,
				IsOpen:                       true,
				MarkPrice:                    118604.2,
				MarkValue:                    118.6042,
				PositionCost:                 118.6371,
				PositionInitialMargin:        19.7728500039,
				PositionMargin:               20.2728500039,
				PositionMaintenanceMargin:    0.54557932,
				RealisedGrossPNL:             0.0215,
				RealisedPNL:                  -0.04968226,
				UnrealisedPNL:                -0.0329,
				UnrealisedPNLPercent:         -0.0003,
				UnrealisedROEPercent:         -0.0017,
				AverageEntryPrice:            118637.1,
				LiquidationPrice:             99321.13,
				BankruptPrice:                98864.25,
				SettleCurrency:               currency.USDT,
				MaintainMargin:               0.004,
				MarginMode:                   "ISOLATED",
				PositionSide:                 "LONG",
				Leverage:                     6.01,
				CumulativeTradeFee:           0.07118226,
				CumulativeFundingFee:         -0.0103,
				CumulativeTax:                0.0071,
				AutoDeposit:                  true,
				RiskLimit:                    500000,
				RealLeverage:                 6.01,
				PositionCross:                0.5,
				PositionCrossMargin:          0.5,
				PositionProfit:               0.0103,
				PositionCommission:           0.0712,
				PositionCommissionCommon:     0.0711,
				PositionCommissionFunding:    0.0001,
				PositionLoss:                 0.0021,
				PositionFunding:              -0.0012,
				MaintenanceMargin:            20.2399500039,
				RiskLimitLevel:               2,
				WithdrawPNL:                  0.01,
				AggregateRate:                0.0046,
			},
			{
				ID:                           "500000000001478577",
				Symbol:                       "XBTUSDTM",
				MaintenanceMarginRequirement: 0.004,
				DeleveragePercentage:         0.31,
				OpeningTimestamp:             types.Time(time.UnixMilli(1759395400117)),
				CurrentTimestamp:             types.Time(time.UnixMilli(1759395824264)),
				CurrentQuantity:              -2,
				CurrentCost:                  -237.2,
				CurrentCommission:            0.14232,
				UnrealisedCost:               -237.2,
				RealisedCost:                 0.14232,
				IsOpen:                       true,
				MarkPrice:                    118604.2,
				MarkValue:                    -237.2084,
				PositionCost:                 -237.2,
				PositionInitialMargin:        47.44,
				PositionMargin:               47.44,
				PositionMaintenanceMargin:    1.091,
				RealisedPNL:                  -0.14232,
				UnrealisedPNL:                -0.0084,
				UnrealisedROEPercent:         -0.0002,
				AverageEntryPrice:            118600,
				LiquidationPrice:             141700.12,
				BankruptPrice:                142320,
				SettleCurrency:               currency.USDT,
				MaintainMargin:               0.004,
				MarginMode:                   "ISOLATED",
				PositionSide:                 "SHORT",
				Leverage:                     5.02,
				CumulativeTradeFee:           0.14232,
				RiskLimit:                    500000,
				RealLeverage:                 5.02,
				PositionCommission:           0.1426,
				PositionCommissionCommon:     0.1424,
				MaintenanceMargin:            47.4316,
				RiskLimitLevel:               2,
				AggregateRate:                0.0046,
			},
		}},
		{symbol: "XBTUSDM", exp: []FuturesPositionDetail{
			{
				ID:                           "500000000001533743",
				Symbol:                       "XBTUSDM",
				MaintenanceMarginRequirement: 0.0040000133,
				CrossMode:                    true,
				DeleveragePercentage:         0.31,
				OpeningTimestamp:             types.Time(time.UnixMilli(1759401696659)),
				CurrentTimestamp:             types.Time(time.UnixMilli(1759401710595)),
				CurrentQuantity:              100,
				CurrentCost:                  0.00084226,
				CurrentCommission:            0.00000051,
				UnrealisedCost:               0.00084226,
				RealisedCost:                 0.00000051,
				IsOpen:                       true,
				MarkPrice:                    118728.55,
				MarkValue:                    0.00084225,
				PositionCost:                 0.00084226,
				PositionInitialMargin:        0.00084226,
				PositionMargin:               0.00084225,
				PositionMaintenanceMargin:    0.00000337,
				RealisedPNL:                  -0.00000051,
				UnrealisedPNL:                -0.00000001,
				UnrealisedPNLPercent:         -0.0001,
				UnrealisedROEPercent:         -0.0001,
				AverageEntryPrice:            118727.9,
				LiquidationPrice:             0.01,
				BankruptPrice:                0.01,
				SettleCurrency:               currency.NewCode("XBT"),
				IsInverse:                    true,
				MaintainMargin:               0.0040000133,
				MarginMode:                   "CROSS",
				PositionSide:                 "LONG",
				Leverage:                     1,
				CumulativeTradeFee:           0.00000051,
				InitialMarginRate:            1,
				AMR:                          3.0383568235,
			},
		}},
	} {
		t.Run(tc.symbol, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetFuturesPositionDetails(t.Context(), tc.symbol)
			require.NoError(t, err, "GetFuturesPositionDetails must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetFuturesPositionDetails should decode every field")
				return
			}
			for i := range result {
				assert.Equalf(t, tc.symbol, result[i].Symbol, "GetFuturesPositionDetails should return positions of the requested contract, position %d", i)
			}
		})
	}
}

func TestGetFuturesPositionList(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name           string
		settleCurrency currency.Code
		exp            []FuturesPosition
	}{
		{name: "all", exp: []FuturesPosition{
			{
				ID:                           "500000000001478576",
				Symbol:                       "XBTUSDTM",
				DeleveragePercentage:         0.05,
				OpeningTimestamp:             types.Time(time.UnixMilli(1759458359537)),
				CurrentTimestamp:             types.Time(time.UnixMilli(1759458415086)),
				CurrentQuantity:              1,
				CurrentCost:                  120.0164,
				CurrentCommission:            0.07200984,
				UnrealisedCost:               120.0164,
				RealisedGrossCost:            -0.0215,
				RealisedCost:                 0.05050984,
				IsOpen:                       true,
				MarkPrice:                    120009.57,
				MarkValue:                    120.00957,
				PositionCost:                 120.0164,
				PositionInitialMargin:        20.0027333373,
				PositionMargin:               20.5027333373,
				RealisedGrossPNL:             0.0215,
				RealisedPNL:                  -0.05050984,
				UnrealisedPNL:                -0.00683,
				UnrealisedPNLPercent:         -0.0001,
				UnrealisedROEPercent:         -0.0003,
				AverageEntryPrice:            120016.4,
				LiquidationPrice:             100475.86,
				BankruptPrice:                100013.67,
				SettleCurrency:               currency.USDT,
				MarginMode:                   "ISOLATED",
				PositionSide:                 "LONG",
				Leverage:                     6,
				AutoDeposit:                  true,
				MaintenanceMarginRequirement: 0.004,
				RiskLimit:                    500000,
				RealLeverage:                 6,
				PositionCross:                0.5,
				PositionCrossMargin:          0.5,
				PositionCommission:           0.0721,
				PositionCommissionCommon:     0.072,
				PositionLoss:                 0.0021,
				PositionFunding:              -0.0012,
				PositionMaintenanceMargin:    0.552044022,
				MaintenanceMargin:            20.4959033373,
				MaintainMargin:               0.004,
				DealCommission:               -0.07200984,
				FundingFee:                   0.0103,
				Tax:                          0.0072,
				WithdrawPNL:                  0.01,
			},
			{
				ID:                           "500000000001533743",
				Symbol:                       "XBTUSDM",
				CrossMode:                    true,
				DeleveragePercentage:         0.31,
				OpeningTimestamp:             types.Time(time.UnixMilli(1759401696659)),
				CurrentTimestamp:             types.Time(time.UnixMilli(1759401710595)),
				CurrentQuantity:              100,
				CurrentCost:                  0.00084226,
				CurrentCommission:            0.00000051,
				UnrealisedCost:               0.00084226,
				RealisedCost:                 0.00000051,
				IsOpen:                       true,
				MarkPrice:                    118728.55,
				MarkValue:                    0.00084225,
				PositionCost:                 0.00084226,
				PositionInitialMargin:        0.00084226,
				PositionMargin:               0.00084225,
				RealisedPNL:                  -0.00000051,
				UnrealisedPNL:                -0.00000001,
				UnrealisedPNLPercent:         -0.0001,
				UnrealisedROEPercent:         -0.0001,
				AverageEntryPrice:            118727.9,
				LiquidationPrice:             0.01,
				BankruptPrice:                0.01,
				SettleCurrency:               currency.NewCode("XBT"),
				IsInverse:                    true,
				MarginMode:                   "CROSS",
				PositionSide:                 "LONG",
				Leverage:                     1,
				MaintenanceMarginRequirement: 0.0040000133,
				PositionMaintenanceMargin:    0.00000337,
				MaintainMargin:               0.0040000133,
				DealCommission:               -0.00000051,
			},
		}},
		{name: "USDT", settleCurrency: currency.USDT, exp: []FuturesPosition{
			{
				ID:                           "500000000001478576",
				Symbol:                       "XBTUSDTM",
				DeleveragePercentage:         0.05,
				OpeningTimestamp:             types.Time(time.UnixMilli(1759458359537)),
				CurrentTimestamp:             types.Time(time.UnixMilli(1759458415086)),
				CurrentQuantity:              1,
				CurrentCost:                  120.0164,
				CurrentCommission:            0.07200984,
				UnrealisedCost:               120.0164,
				RealisedGrossCost:            -0.0215,
				RealisedCost:                 0.05050984,
				IsOpen:                       true,
				MarkPrice:                    120009.57,
				MarkValue:                    120.00957,
				PositionCost:                 120.0164,
				PositionInitialMargin:        20.0027333373,
				PositionMargin:               20.5027333373,
				RealisedGrossPNL:             0.0215,
				RealisedPNL:                  -0.05050984,
				UnrealisedPNL:                -0.00683,
				UnrealisedPNLPercent:         -0.0001,
				UnrealisedROEPercent:         -0.0003,
				AverageEntryPrice:            120016.4,
				LiquidationPrice:             100475.86,
				BankruptPrice:                100013.67,
				SettleCurrency:               currency.USDT,
				MarginMode:                   "ISOLATED",
				PositionSide:                 "LONG",
				Leverage:                     6,
				AutoDeposit:                  true,
				MaintenanceMarginRequirement: 0.004,
				RiskLimit:                    500000,
				RealLeverage:                 6,
				PositionCross:                0.5,
				PositionCrossMargin:          0.5,
				PositionCommission:           0.0721,
				PositionCommissionCommon:     0.072,
				PositionLoss:                 0.0021,
				PositionFunding:              -0.0012,
				PositionMaintenanceMargin:    0.552044022,
				MaintenanceMargin:            20.4959033373,
				MaintainMargin:               0.004,
				DealCommission:               -0.07200984,
				FundingFee:                   0.0103,
				Tax:                          0.0072,
				WithdrawPNL:                  0.01,
			},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetFuturesPositionList(t.Context(), tc.settleCurrency)
			require.NoError(t, err, "GetFuturesPositionList must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetFuturesPositionList should decode every field")
				return
			}
			if tc.settleCurrency.IsEmpty() {
				return
			}
			for i := range result {
				assert.Truef(t, tc.settleCurrency.Equal(result[i].SettleCurrency), "GetFuturesPositionList should return positions settled in %s, position %d", tc.settleCurrency, i)
			}
		})
	}
}

func TestGetPositionHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetPositionHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetPositionHistory must reject a nil request")
	_, err = e.GetPositionHistory(t.Context(), &GetPositionHistoryRequest{Symbol: "XBTUSDTM", From: time.Now(), To: time.Now().Add(-time.Hour * 5)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetPositionHistory must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	_, err = e.GetPositionHistory(cancelledContext(t), &GetPositionHistoryRequest{})
	require.ErrorIs(t, err, context.Canceled, "GetPositionHistory must return the request error")
	result, err := e.GetPositionHistory(t.Context(), &GetPositionHistoryRequest{
		Symbol: "XBTUSDTM", From: time.UnixMilli(1735516800000), To: time.UnixMilli(1735603200000), Limit: 10, PageID: 1,
	})
	require.NoError(t, err, "GetPositionHistory must not error")
	if mockTests {
		exp := &FuturesPositionHistoryResponse{
			CurrentPage: 1,
			PageSize:    10,
			TotalNumber: 2,
			TotalPage:   1,
			Items: []FuturesPositionHistoryItem{
				{
					CloseID:              "500000000036305465",
					UserID:               "633559791e1cbc0001f319bc",
					Symbol:               "XBTUSDTM",
					SettleCurrency:       currency.USDT,
					Leverage:             1,
					Type:                 "CLOSE_LONG",
					PNL:                  0.51214413,
					RealisedGrossCost:    -0.5837,
					TradeFee:             0.03766066,
					FundingFee:           -0.03389521,
					OpenTime:             types.Time(time.UnixMilli(1735549162120)),
					CloseTime:            types.Time(time.UnixMilli(1735589352069)),
					OpenPrice:            93859.8,
					ClosePrice:           94443.5,
					MarginMode:           "CROSS",
					RealisedGrossCostNew: -0.5837,
					Side:                 "LONG",
					PositionSide:         "BOTH",
				},
				{
					CloseID:              "500000000036305466",
					UserID:               "633559791e1cbc0001f319bc",
					Symbol:               "XBTUSDTM",
					SettleCurrency:       currency.USDT,
					Leverage:             10,
					Type:                 "LIQUID_SHORT",
					PNL:                  -9.5034,
					RealisedGrossCost:    9.4378,
					WithdrawPNL:          0.25,
					TradeFee:             0.0566,
					FundingFee:           0.009,
					OpenTime:             types.Time(time.Unix(1735520000, 0)),
					CloseTime:            types.Time(time.Unix(1735560000, 0)),
					OpenPrice:            93500,
					ClosePrice:           94443.8,
					MarginMode:           "ISOLATED",
					RealisedGrossCostNew: 9.4378,
					Tax:                  0.0012,
					ROE:                  -1.0164,
					LiquidAmount:         10,
					LiquidPrice:          102843.6,
					Side:                 "SHORT",
					PositionSide:         "BOTH",
				},
			},
		}
		assert.Equal(t, exp, result, "GetPositionHistory should decode every field")
		return
	}
	assert.NotNil(t, result, "GetPositionHistory should return a page")
}

func TestGetMaxWithdrawMargin(t *testing.T) {
	t.Parallel()
	_, err := e.GetMaxWithdrawMargin(t.Context(), "", "BOTH")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "GetMaxWithdrawMargin must reject an empty symbol")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	_, err = e.GetMaxWithdrawMargin(cancelledContext(t), "XBTUSDTM", "")
	require.ErrorIs(t, err, context.Canceled, "GetMaxWithdrawMargin must return the request error")
	result, err := e.GetMaxWithdrawMargin(t.Context(), "XBTUSDTM", "BOTH")
	require.NoError(t, err, "GetMaxWithdrawMargin must not error")
	if mockTests {
		assert.Equal(t, 21.1135719252, result, "GetMaxWithdrawMargin should return the margin the endpoint sent")
		return
	}
	assert.GreaterOrEqual(t, result, 0.0, "GetMaxWithdrawMargin should not return a negative margin")
}

func TestGetFuturesCrossMarginLeverage(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesCrossMarginLeverage(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "GetFuturesCrossMarginLeverage must reject an empty symbol")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFuturesCrossMarginLeverage(t.Context(), "XBTUSDTM")
	require.NoError(t, err, "GetFuturesCrossMarginLeverage must not error")
	if mockTests {
		exp := &FuturesCrossMarginLeverageResponse{
			Symbol:   "XBTUSDTM",
			Leverage: 3,
		}
		assert.Equal(t, exp, result, "GetFuturesCrossMarginLeverage should decode every field")
		return
	}
	require.NotNil(t, result, "GetFuturesCrossMarginLeverage must return a response")
	assert.Positive(t, result.Leverage.Float64(), "GetFuturesCrossMarginLeverage should return the leverage")
}

func TestModifyFuturesCrossMarginLeverage(t *testing.T) {
	t.Parallel()
	_, err := e.ModifyFuturesCrossMarginLeverage(t.Context(), "", 10)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "ModifyFuturesCrossMarginLeverage must reject an empty symbol")
	_, err = e.ModifyFuturesCrossMarginLeverage(t.Context(), "XBTUSDTM", 0)
	require.ErrorIs(t, err, errInvalidLeverage, "ModifyFuturesCrossMarginLeverage must reject a zero leverage")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.ModifyFuturesCrossMarginLeverage(t.Context(), "XBTUSDTM", 10)
	require.NoError(t, err, "ModifyFuturesCrossMarginLeverage must not error")
	assert.True(t, result, "ModifyFuturesCrossMarginLeverage should report the change accepted")
}

func TestAddMargin(t *testing.T) {
	t.Parallel()
	_, err := e.AddMargin(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "AddMargin must reject a nil request")
	_, err = e.AddMargin(t.Context(), &AddMarginRequest{Margin: 1, BusinessNumber: "6200c9b83aecfb000152dasfdee"})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "AddMargin must reject an empty symbol")
	_, err = e.AddMargin(t.Context(), &AddMarginRequest{Symbol: "XBTUSDTM", BusinessNumber: "6200c9b83aecfb000152dasfdee"})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "AddMargin must reject a zero margin")
	_, err = e.AddMargin(t.Context(), &AddMarginRequest{Symbol: "XBTUSDTM", Margin: 1})
	require.ErrorIs(t, err, errBusinessNumberRequired, "AddMargin must reject an empty business number")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	_, err = e.AddMargin(cancelledContext(t), &AddMarginRequest{Symbol: "XBTUSDTM", Margin: 1, BusinessNumber: "6200c9b83aecfb000152dasfdee"})
	require.ErrorIs(t, err, context.Canceled, "AddMargin must return the request error")
	result, err := e.AddMargin(t.Context(), &AddMarginRequest{Symbol: "XBTUSDTM", Margin: 1, BusinessNumber: "6200c9b83aecfb000152dasfdee", PositionSide: "BOTH"})
	require.NoError(t, err, "AddMargin must not error")
	if mockTests {
		// An isolated margin position is never in cross mode, so CrossMode stays false
		exp := &AddMarginResponse{
			ID:                           "6200c9b83aecfb000152ddcd",
			Symbol:                       "XBTUSDTM",
			AutoDeposit:                  true,
			MaintenanceMarginRequirement: 0.005,
			RiskLimit:                    500000,
			RealLeverage:                 18.72,
			DeleveragePercentage:         0.66,
			OpeningTimestamp:             types.Time(time.UnixMilli(1646287090131)),
			CurrentTimestamp:             types.Time(time.UnixMilli(1646295055021)),
			CurrentQuantity:              1,
			CurrentCost:                  43.388,
			CurrentCommission:            0.0260328,
			UnrealisedCost:               43.388,
			RealisedGrossCost:            -0.0123,
			RealisedCost:                 0.0137328,
			IsOpen:                       true,
			MarkPrice:                    43536.65,
			MarkValue:                    43.53665,
			PositionCost:                 43.388,
			PositionCross:                1.000024985,
			PositionInitialMargin:        2.1694,
			PositionCommission:           0.02733446,
			PositionLoss:                 0.0021,
			PositionMargin:               3.19675944,
			PositionMaintenanceMargin:    0.24861326,
			MaintenanceMargin:            3.34540944,
			RealisedGrossPNL:             0.0123,
			RealisedPNL:                  -0.0137328,
			UnrealisedPNL:                0.14865,
			UnrealisedPNLPercent:         0.0034,
			UnrealisedROEPercent:         0.0685,
			AverageEntryPrice:            43388,
			LiquidationPrice:             41440,
			BankruptPrice:                41218,
			UserID:                       100000001,
			SettleCurrency:               currency.USDT,
		}
		assert.Equal(t, exp, result, "AddMargin should decode every field")
		return
	}
	require.NotNil(t, result, "AddMargin must return the position")
	assert.Equal(t, "XBTUSDTM", result.Symbol, "AddMargin should return the requested position")
}

func TestRemoveMarginManually(t *testing.T) {
	t.Parallel()
	_, err := e.RemoveMarginManually(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "RemoveMarginManually must reject a nil request")
	_, err = e.RemoveMarginManually(t.Context(), &RemoveMarginManuallyRequest{WithdrawAmount: 1})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "RemoveMarginManually must reject an empty symbol")
	_, err = e.RemoveMarginManually(t.Context(), &RemoveMarginManuallyRequest{Symbol: "ADAUSDTM", WithdrawAmount: -1})
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "RemoveMarginManually must reject a negative amount")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	_, err = e.RemoveMarginManually(cancelledContext(t), &RemoveMarginManuallyRequest{Symbol: "ADAUSDTM", WithdrawAmount: 0.1})
	require.ErrorIs(t, err, context.Canceled, "RemoveMarginManually must return the request error")
	result, err := e.RemoveMarginManually(t.Context(), &RemoveMarginManuallyRequest{Symbol: "ADAUSDTM", WithdrawAmount: 0.1, PositionSide: "BOTH"})
	require.NoError(t, err, "RemoveMarginManually must not error")
	assert.Equal(t, 0.1, result, "RemoveMarginManually should return the amount withdrawn")
}

func TestGetFuturesCrossMarginRiskLimit(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesCrossMarginRiskLimit(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFuturesCrossMarginRiskLimit must reject a nil request")
	_, err = e.GetFuturesCrossMarginRiskLimit(t.Context(), &GetFuturesCrossMarginRiskLimitRequest{})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "GetFuturesCrossMarginRiskLimit must reject an empty symbol list")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	_, err = e.GetFuturesCrossMarginRiskLimit(cancelledContext(t), &GetFuturesCrossMarginRiskLimitRequest{Symbols: []string{"XBTUSDTM"}})
	require.ErrorIs(t, err, context.Canceled, "GetFuturesCrossMarginRiskLimit must return the request error")
	result, err := e.GetFuturesCrossMarginRiskLimit(t.Context(), &GetFuturesCrossMarginRiskLimitRequest{Symbols: []string{"XBTUSDTM", "ETHUSDTM"}, TotalMargin: 10000, Leverage: 20})
	require.NoError(t, err, "GetFuturesCrossMarginRiskLimit must not error")
	if mockTests {
		exp := []FuturesCrossMarginRiskLimit{
			{
				Symbol:                "XBTUSDTM",
				MaxOpenSize:           1960,
				MaxOpenValue:          199943.52,
				TotalMargin:           10000,
				Price:                 102012,
				Leverage:              20,
				MaintenanceMarginRate: 0.00416136,
				InitialMarginRate:     0.05,
				Currency:              currency.USDT,
			},
			{
				Symbol:                "ETHUSDTM",
				MaxOpenSize:           7823,
				MaxOpenValue:          199987.172,
				TotalMargin:           10000,
				Price:                 2556.4,
				Leverage:              20,
				MaintenanceMarginRate: 0.0054623236,
				InitialMarginRate:     0.05,
				Currency:              currency.USDT,
			},
		}
		assert.Equal(t, exp, result, "GetFuturesCrossMarginRiskLimit should decode every field")
		return
	}
	assert.Len(t, result, 2, "GetFuturesCrossMarginRiskLimit should return a limit per contract")
}

func TestGetFuturesCrossMarginRequirement(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesCrossMarginRequirement(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFuturesCrossMarginRequirement must reject a nil request")
	_, err = e.GetFuturesCrossMarginRequirement(t.Context(), &GetFuturesCrossMarginRequirementRequest{PositionValue: 1000})
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "GetFuturesCrossMarginRequirement must reject an empty symbol list")
	_, err = e.GetFuturesCrossMarginRequirement(t.Context(), &GetFuturesCrossMarginRequirementRequest{Symbols: []string{"XBTUSDTM"}})
	require.ErrorIs(t, err, errPositionValueRequired, "GetFuturesCrossMarginRequirement must reject a zero position value")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	_, err = e.GetFuturesCrossMarginRequirement(cancelledContext(t), &GetFuturesCrossMarginRequirementRequest{Symbols: []string{"XBTUSDTM"}, PositionValue: 1000})
	require.ErrorIs(t, err, context.Canceled, "GetFuturesCrossMarginRequirement must return the request error")
	result, err := e.GetFuturesCrossMarginRequirement(t.Context(), &GetFuturesCrossMarginRequirementRequest{Symbols: []string{"XBTUSDTM", "ETHUSDTM"}, PositionValue: 1000, Leverage: 10})
	require.NoError(t, err, "GetFuturesCrossMarginRequirement must not error")
	if mockTests {
		exp := []FuturesCrossMarginRequirement{
			{
				Symbol:                "XBTUSDTM",
				Size:                  9,
				PositionValue:         1000,
				Price:                 105016,
				InitialMarginRate:     0.1,
				MaintenanceMarginRate: 0.00400013,
			},
			{
				Symbol:                "ETHUSDTM",
				Size:                  39,
				PositionValue:         1000,
				Price:                 2556.4,
				InitialMarginRate:     0.1,
				MaintenanceMarginRate: 0.00500021,
			},
		}
		assert.Equal(t, exp, result, "GetFuturesCrossMarginRequirement should decode every field")
		return
	}
	assert.Len(t, result, 2, "GetFuturesCrossMarginRequirement should return the requirement per contract")
}

func TestCheckSymbols(t *testing.T) {
	t.Parallel()
	assert.ErrorIs(t, checkSymbols(nil), currency.ErrSymbolStringEmpty, "checkSymbols should reject an empty list")
	assert.ErrorIs(t, checkSymbols([]string{"XBTUSDTM", ""}), currency.ErrSymbolStringEmpty, "checkSymbols should reject an empty symbol")
	assert.NoError(t, checkSymbols([]string{"XBTUSDTM", "ETHUSDTM"}), "checkSymbols should accept a list of symbols")
}

func TestGetFuturesRiskLimitLevel(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesRiskLimitLevel(t.Context(), "")
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "GetFuturesRiskLimitLevel must reject an empty symbol")

	result, err := e.GetFuturesRiskLimitLevel(t.Context(), "XBTUSDTM")
	require.NoError(t, err, "GetFuturesRiskLimitLevel must not error")
	if mockTests {
		exp := []FuturesRiskLimitLevel{
			{
				Symbol:         "XBTUSDTM",
				Level:          1,
				MaxRiskLimit:   250000,
				MaxLeverage:    125,
				InitialMargin:  0.008,
				MaintainMargin: 0.004,
			},
			{
				Symbol:         "XBTUSDTM",
				Level:          2,
				MaxRiskLimit:   600000,
				MinRiskLimit:   250000,
				MaxLeverage:    100,
				InitialMargin:  0.01,
				MaintainMargin: 0.005,
			},
			{
				Symbol:         "XBTUSDTM",
				Level:          3,
				MaxRiskLimit:   2500000,
				MinRiskLimit:   600000,
				MaxLeverage:    75,
				InitialMargin:  0.014,
				MaintainMargin: 0.007,
			},
		}
		assert.Equal(t, exp, result, "GetFuturesRiskLimitLevel should decode every field")
		return
	}
	require.NotEmpty(t, result, "GetFuturesRiskLimitLevel must return risk limit levels")
	for i := range result {
		assert.Equalf(t, "XBTUSDTM", result[i].Symbol, "GetFuturesRiskLimitLevel should return levels of the requested contract, level %d", i)
		assert.Positivef(t, result[i].Level, "GetFuturesRiskLimitLevel should return a level number, level %d", i)
		assert.Positivef(t, result[i].MaxRiskLimit, "GetFuturesRiskLimitLevel should return an upper limit, level %d", i)
		assert.Positivef(t, result[i].MaxLeverage, "GetFuturesRiskLimitLevel should return a maximum leverage, level %d", i)
		assert.Positivef(t, result[i].InitialMargin, "GetFuturesRiskLimitLevel should return an initial margin rate, level %d", i)
		assert.Positivef(t, result[i].MaintainMargin, "GetFuturesRiskLimitLevel should return a maintenance margin rate, level %d", i)
	}
}

func TestUpdateFuturesRiskLimitLevel(t *testing.T) {
	t.Parallel()
	_, err := e.UpdateFuturesRiskLimitLevel(t.Context(), "", 2)
	require.ErrorIs(t, err, currency.ErrSymbolStringEmpty, "UpdateFuturesRiskLimitLevel must reject an empty symbol")
	_, err = e.UpdateFuturesRiskLimitLevel(t.Context(), "XBTUSDTM", 0)
	require.ErrorIs(t, err, errRiskLimitLevelRequired, "UpdateFuturesRiskLimitLevel must reject a zero level")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.UpdateFuturesRiskLimitLevel(t.Context(), "XBTUSDTM", 2)
	require.NoError(t, err, "UpdateFuturesRiskLimitLevel must not error")
	assert.True(t, result, "UpdateFuturesRiskLimitLevel should report the request accepted")
}
