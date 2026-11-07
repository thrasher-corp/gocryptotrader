package bybit

import (
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

func TestGetPositionInfo(t *testing.T) {
	t.Parallel()
	_, err := e.GetPositionInfo(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetPositionInfo must reject a nil request")
	_, err = e.GetPositionInfo(t.Context(), &PositionInfoRequest{})
	require.ErrorIs(t, err, errCategoryNotSet, "GetPositionInfo must reject a request without a category")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	ensureTradablePairs(t)
	for _, tc := range []struct {
		name string
		req  *PositionInfoRequest
		// mockOnly marks requests with a placeholder cursor that Bybit could reject live
		mockOnly bool
		exp      *PositionInfoResponse
	}{
		{
			name: "linear symbol",
			req:  &PositionInfoRequest{Category: cLinear, Symbol: "BTCUSDT", Limit: 20},
			exp: &PositionInfoResponse{
				Category: "linear",
				List: []PositionInfo{
					{
						PositionIndex:                          1,
						RiskID:                                 1,
						RiskLimitValue:                         2000000,
						Symbol:                                 "BTCUSDT",
						Side:                                   "Buy",
						Size:                                   0.5,
						AveragePrice:                           61250.5,
						PositionValue:                          30625.25,
						AutoAddMargin:                          true,
						PositionStatus:                         "Normal",
						Leverage:                               10,
						BreakEvenPrice:                         61284.2,
						MarkPrice:                              61800,
						LiquidationPrice:                       55489.3,
						LiquidationPriceByMarkPrice:            55489.3,
						PositionInitialMargin:                  3078.88,
						PositionInitialMarginByMarkPrice:       3106.5,
						PositionMaintenanceMargin:              160.1,
						PositionMaintenanceMarginByMarkPrice:   161.5,
						TakeProfit:                             70000,
						StopLoss:                               58000,
						TrailingStop:                           500,
						NetDeltaRatio:                          0.25,
						UnrealisedPNL:                          274.75,
						CurrentRealisedPNL:                     -16.84,
						CumulativeRealisedPNL:                  1205.6,
						ADLRankIndicator:                       2,
						CreatedTime:                            types.Time(time.UnixMilli(1676538056258)),
						UpdatedTime:                            types.Time(time.Unix(1788300000, 0)),
						OpenTime:                               types.Time(time.Unix(1788200000, 0)),
						Sequence:                               5723621632,
						IsReduceOnly:                           true,
						MaintenanceMarginRateSystemUpdatedTime: types.Time(time.Unix(1788400000, 0)),
						LeverageSystemUpdatedTime:              types.Time(time.Unix(1788400000, 0)),
						TPSLMode:                               "Full",
					},
				},
			},
		},
		{
			name: "option symbol",
			req:  &PositionInfoRequest{Category: cOption, Symbol: optionsTradablePair.String(), BaseCoin: currency.BTC, Limit: 20},
			exp: &PositionInfoResponse{
				Category: "option",
				List: []PositionInfo{
					{
						Symbol:              "BTC-26NOV24-92000-C",
						Side:                "Sell",
						Size:                0.3,
						AveragePrice:        3500,
						PositionValue:       1048.56,
						PositionStatus:      "Normal",
						MarkPrice:           3495.2,
						SessionAveragePrice: 3500,
						Delta:               -0.15528,
						Gamma:               -0.00002511,
						Vega:                -28.15237,
						Theta:               43.70985,
						UnrealisedPNL:       1.44,
						CurrentRealisedPNL:  -0.6,
						CreatedTime:         types.Time(time.Unix(1731900003, 0)),
						UpdatedTime:         types.Time(time.Unix(1731900003, 0)),
						OpenTime:            types.Time(time.Unix(1731900003, 0)),
						Sequence:            4688002129,
						TPSLMode:            "Full",
					},
				},
			},
		},
		{
			name: "inverse symbol",
			req:  &PositionInfoRequest{Category: cInverse, Symbol: "BTCUSD"},
			exp: &PositionInfoResponse{
				Category: "inverse",
				List: []PositionInfo{
					{
						RiskID:                               1,
						RiskLimitValue:                       150,
						Symbol:                               "BTCUSD",
						Side:                                 "Sell",
						Size:                                 300,
						AveragePrice:                         27464.50441675,
						PositionValue:                        0.01092319,
						AutoAddMargin:                        true,
						PositionStatus:                       "Normal",
						Leverage:                             10,
						BreakEvenPrice:                       27420.73034991,
						MarkPrice:                            28224.5,
						PositionInitialMargin:                0.00010923,
						PositionInitialMarginByMarkPrice:     0.00010923,
						PositionMaintenanceMargin:            0.0000015,
						PositionMaintenanceMarginByMarkPrice: 0.0000015,
						UnrealisedPNL:                        -0.00029413,
						CurrentRealisedPNL:                   0.00013123,
						CumulativeRealisedPNL:                -0.00096902,
						ADLRankIndicator:                     2,
						CreatedTime:                          types.Time(time.UnixMilli(1676538056258)),
						UpdatedTime:                          types.Time(time.UnixMilli(1697673600012)),
						Sequence:                             5723621632,
						TPSLMode:                             "Full",
						BustPrice:                            999999,
						PositionBalance:                      0.00139186,
						TradeMode:                            1,
					},
				},
			},
		},
		{
			name: "linear open positions by settle coin",
			req:  &PositionInfoRequest{Category: cLinear, SettleCoin: currency.USDT, Limit: 2},
			exp: &PositionInfoResponse{
				Category:       "linear",
				NextPageCursor: "MOVEUSDT%2C1787198400021%2C0",
				List: []PositionInfo{
					{
						RiskID:                               1,
						RiskLimitValue:                       25000,
						Symbol:                               "BBUSDT",
						Side:                                 "Sell",
						Size:                                 883,
						AveragePrice:                         0.01105794,
						PositionValue:                        7.492255,
						PositionStatus:                       "Normal",
						Leverage:                             1,
						BreakEvenPrice:                       0.01105546,
						MarkPrice:                            0.008485,
						LiquidationPrice:                     0.20933581,
						PositionInitialMargin:                7.50299557,
						PositionInitialMarginByMarkPrice:     7.50299557,
						PositionMaintenanceMargin:            0.16058567,
						PositionMaintenanceMarginByMarkPrice: 0.16058567,
						UnrealisedPNL:                        2.271903,
						CurrentRealisedPNL:                   0.00318599,
						CumulativeRealisedPNL:                0.00318599,
						ADLRankIndicator:                     3,
						CreatedTime:                          types.Time(time.UnixMilli(1728875391204)),
						UpdatedTime:                          types.Time(time.UnixMilli(1787198400023)),
						OpenTime:                             types.Time(time.UnixMilli(1786983352599)),
						Sequence:                             169532619968,
						TPSLMode:                             "Full",
					},
					{
						RiskID:         1,
						RiskLimitValue: 2000000,
						Symbol:         "BTCPERP",
						PositionStatus: "Normal",
						Leverage:       3,
						MarkPrice:      61811.5,
						CreatedTime:    types.Time(time.UnixMilli(1698857343517)),
						UpdatedTime:    types.Time(time.UnixMilli(1698857343517)),
						Sequence:       -1,
						TPSLMode:       "Full",
					},
				},
			},
		},
		{
			name:     "next page of open positions",
			req:      &PositionInfoRequest{Category: cLinear, SettleCoin: currency.USDT, Limit: 2, Cursor: "MOVEUSDT%2C1787198400021%2C0"},
			mockOnly: true,
			exp: &PositionInfoResponse{
				Category: "linear",
				List: []PositionInfo{
					{
						RiskID:                               1,
						RiskLimitValue:                       25000,
						Symbol:                               "MOVEUSDT",
						Side:                                 "Buy",
						Size:                                 120,
						AveragePrice:                         0.0921,
						PositionValue:                        11.052,
						PositionStatus:                       "Normal",
						Leverage:                             2,
						BreakEvenPrice:                       0.09223,
						MarkPrice:                            0.0935,
						LiquidationPrice:                     0.0465,
						LiquidationPriceByMarkPrice:          0.0465,
						PositionInitialMargin:                5.53,
						PositionInitialMarginByMarkPrice:     5.61,
						PositionMaintenanceMargin:            0.11,
						PositionMaintenanceMarginByMarkPrice: 0.12,
						UnrealisedPNL:                        0.168,
						CurrentRealisedPNL:                   -0.0061,
						CumulativeRealisedPNL:                -0.0061,
						ADLRankIndicator:                     1,
						CreatedTime:                          types.Time(time.UnixMilli(1787198400021)),
						UpdatedTime:                          types.Time(time.UnixMilli(1787198400021)),
						OpenTime:                             types.Time(time.UnixMilli(1787198400021)),
						Sequence:                             169532619970,
						TPSLMode:                             "Full",
					},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tc.mockOnly && !mockTests {
				t.Skip("request carries a placeholder cursor")
			}
			result, err := e.GetPositionInfo(t.Context(), tc.req)
			require.NoError(t, err, "GetPositionInfo must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetPositionInfo should decode every field")
				return
			}
			assert.Equal(t, tc.req.Category, result.Category, "GetPositionInfo should return the requested category")
		})
	}
}

func TestSetLeverageLevel(t *testing.T) {
	t.Parallel()
	err := e.SetLeverageLevel(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "SetLeverageLevel must reject a nil request")
	err = e.SetLeverageLevel(t.Context(), &SetLeverageRequest{})
	require.ErrorIs(t, err, errCategoryNotSet, "SetLeverageLevel must reject a request without a category")
	err = e.SetLeverageLevel(t.Context(), &SetLeverageRequest{Category: cLinear})
	require.ErrorIs(t, err, errSymbolMissing, "SetLeverageLevel must reject a request without a symbol")
	err = e.SetLeverageLevel(t.Context(), &SetLeverageRequest{Category: cLinear, Symbol: "BTCUSDT", SellLeverage: 3})
	require.ErrorIs(t, err, errInvalidLeverage, "SetLeverageLevel must reject a request without a buy leverage")
	err = e.SetLeverageLevel(t.Context(), &SetLeverageRequest{Category: cLinear, Symbol: "BTCUSDT", BuyLeverage: 3})
	require.ErrorIs(t, err, errInvalidLeverage, "SetLeverageLevel must reject a request without a sell leverage")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	err = e.SetLeverageLevel(t.Context(), &SetLeverageRequest{Category: cLinear, Symbol: "BTCUSDT", BuyLeverage: 3, SellLeverage: 3})
	assert.NoError(t, err, "SetLeverageLevel should not error")
}

func TestGetFuturesLeverage(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesLeverage(t.Context(), "", "MNTUSDT")
	require.ErrorIs(t, err, errCategoryNotSet, "GetFuturesLeverage must reject a request without a category")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFuturesLeverage(t.Context(), cLinear, "MNTUSDT")
	require.NoError(t, err, "GetFuturesLeverage must not error")
	if mockTests {
		exp := &FuturesLeverageResponse{
			Category: "linear",
			List: []FuturesLeverage{
				{
					Symbol:        "MNTUSDT",
					Leverage:      10,
					Side:          "Sell",
					PositionIndex: 2,
				},
				{
					Symbol:        "MNTUSDT",
					Leverage:      12.5,
					Side:          "Buy",
					PositionIndex: 1,
				},
			},
		}
		assert.Equal(t, exp, result, "GetFuturesLeverage should decode every field")
		return
	}
	assert.Equal(t, cLinear, result.Category, "GetFuturesLeverage should return the requested category")
}

func TestSwitchPositionMode(t *testing.T) {
	t.Parallel()
	err := e.SwitchPositionMode(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "SwitchPositionMode must reject a nil request")
	err = e.SwitchPositionMode(t.Context(), &SwitchPositionModeRequest{})
	require.ErrorIs(t, err, errCategoryNotSet, "SwitchPositionMode must reject a request without a category")
	err = e.SwitchPositionMode(t.Context(), &SwitchPositionModeRequest{Category: cLinear})
	require.ErrorIs(t, err, errEitherSymbolOrCoinRequired, "SwitchPositionMode must reject a request without a symbol or coin")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, req := range []*SwitchPositionModeRequest{
		{Category: cLinear, Symbol: "BTCUSDT", Mode: 3},
		{Category: cLinear, Coin: currency.USDT},
	} {
		assert.NoErrorf(t, e.SwitchPositionMode(t.Context(), req), "SwitchPositionMode should not error for symbol %q and coin %q", req.Symbol, req.Coin)
	}
}

func TestSetTradingStop(t *testing.T) {
	t.Parallel()
	err := e.SetTradingStop(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "SetTradingStop must reject a nil request")
	err = e.SetTradingStop(t.Context(), &TradingStopRequest{})
	require.ErrorIs(t, err, errCategoryNotSet, "SetTradingStop must reject a request without a category")
	err = e.SetTradingStop(t.Context(), &TradingStopRequest{Category: cLinear})
	require.ErrorIs(t, err, errSymbolMissing, "SetTradingStop must reject a request without a symbol")
	err = e.SetTradingStop(t.Context(), &TradingStopRequest{Category: cLinear, Symbol: "BTCUSDT"})
	require.ErrorIs(t, err, errTakeProfitOrStopLossModeMissing, "SetTradingStop must reject a request without a TP/SL mode")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	takeProfit, stopLoss, cancel, trailingStop := 0.6, 0.2, 0.0, 500.0
	for _, tc := range []struct {
		name string
		req  *TradingStopRequest
	}{
		{
			name: "partial limit take profit and stop loss",
			req:  &TradingStopRequest{Category: cLinear, Symbol: "XRPUSDT", TPSLMode: "Partial", TakeProfit: &takeProfit, StopLoss: &stopLoss, TPTriggerBy: "MarkPrice", SLTriggerBy: "IndexPrice", TPOrderType: "Limit", SLOrderType: "Limit", TPSize: 50, SLSize: 50, TPLimitPrice: 0.57, SLLimitPrice: 0.21},
		},
		{
			name: "cancel the take profit and set a trailing stop",
			req:  &TradingStopRequest{Category: cLinear, Symbol: "BTCUSDT", TPSLMode: "Full", PositionIndex: 1, TakeProfit: &cancel, TrailingStop: &trailingStop, ActivePrice: 65000},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.NoError(t, e.SetTradingStop(t.Context(), tc.req), "SetTradingStop should not error")
		})
	}
}

func TestSetAutoAddMargin(t *testing.T) {
	t.Parallel()
	err := e.SetAutoAddMargin(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "SetAutoAddMargin must reject a nil request")
	err = e.SetAutoAddMargin(t.Context(), &SetAutoAddMarginRequest{})
	require.ErrorIs(t, err, errCategoryNotSet, "SetAutoAddMargin must reject a request without a category")
	err = e.SetAutoAddMargin(t.Context(), &SetAutoAddMarginRequest{Category: cLinear})
	require.ErrorIs(t, err, errSymbolMissing, "SetAutoAddMargin must reject a request without a symbol")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, req := range []*SetAutoAddMarginRequest{
		{Category: cLinear, Symbol: "BTCUSDT", PositionIndex: 2},
		{Category: cLinear, Symbol: "BTCUSDT", AutoAddMargin: true},
	} {
		assert.NoErrorf(t, e.SetAutoAddMargin(t.Context(), req), "SetAutoAddMargin should not error turning it %t", req.AutoAddMargin)
	}
}

func TestAddOrReduceMargin(t *testing.T) {
	t.Parallel()
	_, err := e.AddOrReduceMargin(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "AddOrReduceMargin must reject a nil request")
	_, err = e.AddOrReduceMargin(t.Context(), &AddOrReduceMarginRequest{})
	require.ErrorIs(t, err, errCategoryNotSet, "AddOrReduceMargin must reject a request without a category")
	_, err = e.AddOrReduceMargin(t.Context(), &AddOrReduceMarginRequest{Category: cInverse})
	require.ErrorIs(t, err, errSymbolMissing, "AddOrReduceMargin must reject a request without a symbol")
	_, err = e.AddOrReduceMargin(t.Context(), &AddOrReduceMarginRequest{Category: cInverse, Symbol: "ETHUSD"})
	require.ErrorIs(t, err, errMarginNotSet, "AddOrReduceMargin must reject a request without a margin")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tc := range []struct {
		name string
		req  *AddOrReduceMarginRequest
		exp  *AddOrReduceMarginResponse
	}{
		{
			name: "add",
			req:  &AddOrReduceMarginRequest{Category: cInverse, Symbol: "ETHUSD", Margin: 0.01},
			exp: &AddOrReduceMarginResponse{
				Category:                  "inverse",
				Symbol:                    "ETHUSD",
				RiskID:                    11,
				RiskLimitValue:            500,
				Size:                      200,
				AveragePrice:              1812.70004844,
				LiquidationPrice:          1550.8,
				BustPrice:                 1544.2,
				MarkPrice:                 1812.9,
				PositionValue:             0.11033265,
				Leverage:                  12,
				PositionStatus:            "Normal",
				PositionInitialMargin:     0.01926611,
				PositionMaintenanceMargin: 0.00055165,
				TakeProfit:                2000,
				StopLoss:                  1500,
				UnrealisedPNL:             0.00001217,
				CumulativeRealisedPNL:     -0.04618929,
				CreatedTime:               types.Time(time.UnixMilli(1672737740039)),
				UpdatedTime:               types.Time(time.UnixMilli(1684234363788)),
			},
		},
		{
			name: "reduce in hedge mode",
			req:  &AddOrReduceMarginRequest{Category: cInverse, Symbol: "ADAUSD", Margin: -10, PositionIndex: 2},
			exp: &AddOrReduceMarginResponse{
				Category:                  "inverse",
				Symbol:                    "ADAUSD",
				PositionIndex:             2,
				RiskID:                    1,
				RiskLimitValue:            200000,
				Size:                      1000,
				AveragePrice:              0.3567,
				LiquidationPrice:          0.4152,
				BustPrice:                 0.4171,
				MarkPrice:                 0.3561,
				PositionValue:             2803.41,
				Leverage:                  6.5,
				AutoAddMargin:             true,
				PositionStatus:            "Normal",
				PositionInitialMargin:     431.29,
				PositionMaintenanceMargin: 14.02,
				TakeProfit:                0.32,
				StopLoss:                  0.38,
				TrailingStop:              0.005,
				UnrealisedPNL:             4.72,
				CumulativeRealisedPNL:     -1.36,
				CreatedTime:               types.Time(time.Unix(1684230000, 0)),
				UpdatedTime:               types.Time(time.UnixMilli(1684234363790)),
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.AddOrReduceMargin(t.Context(), tc.req)
			require.NoError(t, err, "AddOrReduceMargin must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "AddOrReduceMargin should decode every field")
				return
			}
			assert.Equal(t, tc.req.Symbol, result.Symbol, "AddOrReduceMargin should return the position of the symbol")
		})
	}
}

func TestGetClosedPNL(t *testing.T) {
	t.Parallel()
	_, err := e.GetClosedPNL(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetClosedPNL must reject a nil request")
	_, err = e.GetClosedPNL(t.Context(), &ClosedPNLRequest{})
	require.ErrorIs(t, err, errCategoryNotSet, "GetClosedPNL must reject a request without a category")
	_, err = e.GetClosedPNL(t.Context(), &ClosedPNLRequest{Category: cLinear, StartTime: time.UnixMilli(1788739200000), EndTime: time.UnixMilli(1788220800000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetClosedPNL must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name string
		req  *ClosedPNLRequest
		// mockOnly marks requests with a placeholder cursor that Bybit could reject live
		mockOnly bool
		exp      *ClosedPNLResponse
	}{
		{
			name: "linear",
			req:  &ClosedPNLRequest{Category: cLinear},
			exp: &ClosedPNLResponse{
				Category:       "linear",
				NextPageCursor: "5a373bfe-188d-4913-9c81-d57ab5be8068%3A1672214887231423699",
				List: []ClosedPNL{
					{
						Symbol:               "ETHPERP",
						OrderID:              "5a373bfe-188d-4913-9c81-d57ab5be8068",
						Side:                 "Sell",
						Quantity:             3,
						OrderPrice:           1122.95,
						OrderType:            "Market",
						ExecutionType:        "Trade",
						ClosedSize:           3,
						CumulativeEntryValue: 3584.9255,
						AverageEntryPrice:    1194.97516667,
						CumulativeExitValue:  3541.795,
						AverageExitPrice:     1180.59833333,
						ClosedPNL:            -47.4065323,
						FillCount:            4,
						Leverage:             3,
						OpenFee:              2.1509553,
						CloseFee:             2.125077,
						CreatedTime:          types.Time(time.UnixMilli(1672214887231)),
						UpdatedTime:          types.Time(time.UnixMilli(1672214887236)),
					},
				},
			},
		},
		{
			name:     "linear symbol in a window",
			req:      &ClosedPNLRequest{Category: cLinear, Symbol: "BTCUSDT", StartTime: time.UnixMilli(1788220800000), EndTime: time.UnixMilli(1788739200000), Limit: 1, Cursor: "page2"},
			mockOnly: true,
			exp: &ClosedPNLResponse{
				Category: "linear",
				List: []ClosedPNL{
					{
						Symbol:               "BTCUSDT",
						OrderID:              "1dc1a2d2-7a44-4d1e-b6a6-8f6b0dfd1c49",
						Side:                 "Buy",
						Quantity:             0.1,
						OrderPrice:           60800,
						OrderType:            "Limit",
						ExecutionType:        "MovePosition",
						ClosedSize:           0.1,
						CumulativeEntryValue: 6200,
						AverageEntryPrice:    62000,
						CumulativeExitValue:  6080,
						AverageExitPrice:     60800,
						ClosedPNL:            120.5,
						FillCount:            1,
						Leverage:             10,
						OpenFee:              3.41,
						CreatedTime:          types.Time(time.Unix(1788400000, 0)),
						UpdatedTime:          types.Time(time.UnixMilli(1788400000005)),
					},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tc.mockOnly && !mockTests {
				t.Skip("request carries a placeholder cursor")
			}
			result, err := e.GetClosedPNL(t.Context(), tc.req)
			require.NoError(t, err, "GetClosedPNL must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetClosedPNL should decode every field")
				return
			}
			assert.Equal(t, tc.req.Category, result.Category, "GetClosedPNL should return the requested category")
		})
	}
}

func TestGetClosedOptionsPositions(t *testing.T) {
	t.Parallel()
	_, err := e.GetClosedOptionsPositions(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetClosedOptionsPositions must reject a nil request")
	_, err = e.GetClosedOptionsPositions(t.Context(), &ClosedOptionsPositionsRequest{})
	require.ErrorIs(t, err, errCategoryNotSet, "GetClosedOptionsPositions must reject a request without a category")
	_, err = e.GetClosedOptionsPositions(t.Context(), &ClosedOptionsPositionsRequest{Category: cOption, StartTime: time.UnixMilli(1749772800000), EndTime: time.UnixMilli(1749686400000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetClosedOptionsPositions must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name string
		req  *ClosedOptionsPositionsRequest
		// mockOnly marks requests with a placeholder cursor that Bybit could reject live
		mockOnly bool
		exp      *ClosedOptionsPositionsResponse
	}{
		{
			name: "latest",
			req:  &ClosedOptionsPositionsRequest{Category: cOption, Limit: 2},
			exp: &ClosedOptionsPositionsResponse{
				Category:       "option",
				NextPageCursor: "1749726002161%3A0%2C1749715220240%3A1",
				List: []ClosedOptionsPosition{
					{
						Symbol:            "BTC-12JUN25-104019-C-USDT",
						Side:              "Sell",
						TotalOpenFee:      0.94506647,
						DeliveryFee:       0.32184533,
						Quantity:          0.02,
						CloseTime:         types.Time(time.UnixMilli(1749726002161)),
						AverageExitPrice:  107281.77405,
						DeliveryPrice:     107281.77405031,
						OpenTime:          types.Time(time.UnixMilli(1749722990063)),
						AverageEntryPrice: 3371.5,
						TotalPNL:          0.90760719,
					},
					{
						Symbol:            "BTC-12JUN25-104000-C-USDT",
						Side:              "Buy",
						TotalOpenFee:      0.86379999,
						TotalCloseFee:     0.32287622,
						Quantity:          0.02,
						CloseTime:         types.Time(time.UnixMilli(1749715220240)),
						AverageExitPrice:  3566,
						OpenTime:          types.Time(time.UnixMilli(1749710568608)),
						AverageEntryPrice: 3946.5,
						TotalPNL:          -7.60858218,
					},
				},
			},
		},
		{
			name:     "symbol in a window",
			req:      &ClosedOptionsPositionsRequest{Category: cOption, Symbol: "BTC-12JUN25-104019-C-USDT", StartTime: time.UnixMilli(1749686400000), EndTime: time.UnixMilli(1749772800000), Limit: 1, Cursor: "page2"},
			mockOnly: true,
			exp: &ClosedOptionsPositionsResponse{
				Category: "option",
				List: []ClosedOptionsPosition{
					{
						Symbol:            "BTC-12JUN25-104019-C-USDT",
						Side:              "Sell",
						TotalOpenFee:      0.94506647,
						DeliveryFee:       0.32184533,
						Quantity:          0.02,
						CloseTime:         types.Time(time.UnixMilli(1749726002161)),
						AverageExitPrice:  107281.77405,
						DeliveryPrice:     107281.77405031,
						OpenTime:          types.Time(time.UnixMilli(1749722990063)),
						AverageEntryPrice: 3371.5,
						TotalPNL:          0.90760719,
					},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tc.mockOnly && !mockTests {
				t.Skip("request carries a placeholder cursor")
			}
			result, err := e.GetClosedOptionsPositions(t.Context(), tc.req)
			require.NoError(t, err, "GetClosedOptionsPositions must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetClosedOptionsPositions should decode every field")
				return
			}
			assert.Equal(t, cOption, result.Category, "GetClosedOptionsPositions should return options")
		})
	}
}

func TestMovePositions(t *testing.T) {
	t.Parallel()
	leg := MovePositionLeg{Category: cSpot, Symbol: "BTCUSDT", Price: 100, Side: "Sell", Quantity: 0.01}
	for _, tc := range []struct {
		name string
		req  *MovePositionsRequest
		err  error
	}{
		{name: "nil", err: common.ErrNilPointer},
		{name: "no from UID", req: &MovePositionsRequest{}, err: errFromUIDRequired},
		{name: "no to UID", req: &MovePositionsRequest{FromUID: 100307601}, err: errToUIDRequired},
		{name: "no legs", req: &MovePositionsRequest{FromUID: 100307601, ToUID: 592324}, err: errNoPositionsToMove},
		{name: "no category", req: &MovePositionsRequest{FromUID: 100307601, ToUID: 592324, List: []MovePositionLeg{{}}}, err: errCategoryNotSet},
		{name: "no symbol", req: &MovePositionsRequest{FromUID: 100307601, ToUID: 592324, List: []MovePositionLeg{{Category: cSpot}}}, err: errSymbolMissing},
		{name: "no price", req: &MovePositionsRequest{FromUID: 100307601, ToUID: 592324, List: []MovePositionLeg{{Category: cSpot, Symbol: "BTCUSDT"}}}, err: limits.ErrPriceBelowMin},
		{name: "no side", req: &MovePositionsRequest{FromUID: 100307601, ToUID: 592324, List: []MovePositionLeg{{Category: cSpot, Symbol: "BTCUSDT", Price: 100}}}, err: order.ErrSideIsInvalid},
		{name: "no quantity", req: &MovePositionsRequest{FromUID: 100307601, ToUID: 592324, List: []MovePositionLeg{leg, {Category: cSpot, Symbol: "BTCUSDT", Price: 100, Side: "Sell"}}}, err: limits.ErrAmountBelowMin},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.MovePositions(t.Context(), tc.req)
			assert.ErrorIs(t, err, tc.err, "MovePositions should return the expected error")
		})
	}

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tc := range []struct {
		name string
		req  *MovePositionsRequest
		exp  *MovePositionsResponse
	}{
		{
			name: "accepted",
			req:  &MovePositionsRequest{FromUID: 100307601, ToUID: 592324, List: []MovePositionLeg{leg}},
			exp: &MovePositionsResponse{
				BlockTradeID: "e9bb926c95f54cf1ba3e315a58b8597b",
				Status:       "Processing",
			},
		},
		{
			name: "rejected",
			req: &MovePositionsRequest{FromUID: 100307601, ToUID: 592324, List: []MovePositionLeg{
				{Category: cLinear, Symbol: "BTCUSDT", Price: 61800, Side: "Buy", Quantity: 0.5},
				{Category: cOption, Symbol: "BTC-26NOV24-92000-C", Price: 3495, Side: "Sell", Quantity: 0.3},
			}},
			exp: &MovePositionsResponse{
				BlockTradeID: "5f0b6b0a4d2a4c4a9a2b0b8e7d1f2a3c",
				Status:       "Rejected",
				RejectParty:  "Taker",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.MovePositions(t.Context(), tc.req)
			require.NoError(t, err, "MovePositions must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "MovePositions should decode every field")
				return
			}
			assert.NotEmpty(t, result.BlockTradeID, "MovePositions should return the block trade ID")
		})
	}
}

func TestGetMovePositionHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetMovePositionHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetMovePositionHistory must reject a nil request")
	_, err = e.GetMovePositionHistory(t.Context(), &MovePositionHistoryRequest{StartTime: time.UnixMilli(1788739200000), EndTime: time.UnixMilli(1788220800000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetMovePositionHistory must reject a start time after the end time")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name string
		req  *MovePositionHistoryRequest
		// mockOnly marks requests with a placeholder cursor that Bybit could reject live
		mockOnly bool
		exp      *MovePositionHistoryResponse
	}{
		{
			name: "filled",
			req:  &MovePositionHistoryRequest{Status: "Filled", Limit: 1},
			exp: &MovePositionHistoryResponse{
				List: []MovePositionRecord{
					{
						BlockTradeID: "1a82e5801af74b67b7ad71ba00a7391a",
						Category:     "option",
						OrderID:      "8e09c5b8-f651-4cec-968d-52764cac11ec",
						UserID:       592324,
						Symbol:       "BTC-14OCT23-27000-C",
						Side:         "Buy",
						Price:        6,
						Quantity:     0.99,
						ExecutionFee: 0.0001,
						Status:       "Filled",
						ExecutionID:  "677ad344-6bb4-4ace-baca-128fcffcaca7",
						CreatedAt:    types.Time(time.UnixMilli(1697186522865)),
						UpdatedAt:    types.Time(time.UnixMilli(1697186523289)),
					},
				},
				NextPageCursor: "page_token%3D1241742%26",
			},
		},
		{
			name:     "rejected block trade of a symbol in a window",
			req:      &MovePositionHistoryRequest{Category: cLinear, Symbol: "BTCUSDT", StartTime: time.UnixMilli(1788220800000), EndTime: time.UnixMilli(1788739200000), Status: "Rejected", BlockTradeID: "5f0b6b0a4d2a4c4a9a2b0b8e7d1f2a3c", Limit: 2, Cursor: "page2"},
			mockOnly: true,
			exp: &MovePositionHistoryResponse{
				List: []MovePositionRecord{
					{
						BlockTradeID:  "5f0b6b0a4d2a4c4a9a2b0b8e7d1f2a3c",
						Category:      "linear",
						UserID:        100307601,
						Symbol:        "BTCUSDT",
						Side:          "Buy",
						Price:         61800,
						Quantity:      0.5,
						Status:        "Rejected",
						ResultCode:    110007,
						ResultMessage: "Insufficient available balance",
						CreatedAt:     types.Time(time.Unix(1788300000, 0)),
						UpdatedAt:     types.Time(time.UnixMilli(1788300000120)),
						RejectParty:   "Taker",
					},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tc.mockOnly && !mockTests {
				t.Skip("request carries a placeholder cursor")
			}
			result, err := e.GetMovePositionHistory(t.Context(), tc.req)
			require.NoError(t, err, "GetMovePositionHistory must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetMovePositionHistory should decode every field")
				return
			}
			assert.NotNil(t, result, "GetMovePositionHistory should return a page")
		})
	}
}

func TestConfirmNewRiskLimit(t *testing.T) {
	t.Parallel()
	err := e.ConfirmNewRiskLimit(t.Context(), "", "BTCUSDT")
	require.ErrorIs(t, err, errCategoryNotSet, "ConfirmNewRiskLimit must reject a request without a category")
	err = e.ConfirmNewRiskLimit(t.Context(), cLinear, "")
	require.ErrorIs(t, err, errSymbolMissing, "ConfirmNewRiskLimit must reject a request without a symbol")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	assert.NoError(t, e.ConfirmNewRiskLimit(t.Context(), cLinear, "BTCUSDT"), "ConfirmNewRiskLimit should not error")
}
