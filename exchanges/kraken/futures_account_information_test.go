package kraken

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// futuresTradingTestOptionPair is a call option, whose symbol Kraken documents no example of
var futuresTradingTestOptionPair = currency.NewPairWithDelimiter("OP", "XBTUSD_261225_C_90000", currency.UnderscoreDelimiter)

func TestGetFuturesAccounts(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFuturesAccounts(t.Context())
	require.NoError(t, err, "GetFuturesAccounts must not error")
	if mockTests {
		exp := &FuturesAccountsResponse{
			Accounts: FuturesAccounts{
				Cash: FuturesCashAccount{
					Type:     "cashAccount",
					Balances: map[string]types.Number{"xbt": 141.31756797, "xrp": 52465.1254},
				},
				Flex: FuturesFlexAccount{
					Type: "multiCollateralMarginAccount",
					Currencies: map[string]FuturesFlexCurrency{
						"XBT": {Quantity: 0.1185308247, Value: 9699.62, Collateral: 9481.38, Available: 0.1135308247},
						"USD": {Quantity: 5000, Value: 5000.01, Collateral: 4999.99, Available: 4875.5},
					},
					InitialMargin:           81.84,
					InitialMarginWithOrders: 122.76,
					MaintenanceMargin:       40.92,
					BalanceValue:            14699.63,
					PortfolioValue:          14712.06,
					CollateralValue:         14481.37,
					UnrealisedPNL:           12.43,
					UnrealisedFunding:       -0.0073,
					TotalUnrealised:         12.4227,
					TotalUnrealisedAsMargin: 11.8,
					AvailableMargin:         14411.33,
					MarginEquity:            14493.17,
					PortfolioMarginBreakdown: &FuturesPortfolioMarginBreakdown{
						TotalCrossAssetNettedMarketRisk:          61.2,
						TotalMarketRisk:                          73.5,
						TotalScenarioPNLs:                        []float64{-73.5, -36.75, 36.75, 73.4},
						TotalAbsoluteOptionPositionDeltaNotional: 4091.9,
						NetPortfolioDelta:                        0.05,
						TotalPremium:                             125.4,
						IsBuyOnly:                                true,
						FuturesMaintenanceMargin:                 40.91,
					},
					UnrealisedPNLInterestRate: 0.0001,
				},
				MarginAccounts: map[string]FuturesMarginAccount{
					"fi_xbtusd": {
						Type:     "marginAccount",
						Currency: currency.XBT.Lower(),
						Balances: map[string]types.Number{"FI_XBTUSD_261225": 50000, "FI_XBTUSD_261030": -15000, "xbt": 2.01451296},
						Auxiliary: FuturesMarginAuxiliary{
							USD:            164882.71,
							PortfolioValue: 2.01528804,
							UnrealisedPNL:  0.00077508,
							AvailableFunds: 1.91341962,
							Funding:        0.00002179,
						},
						MarginRequirements: FuturesMarginRequirements{
							InitialMargin:        0.10186842,
							MaintenanceMargin:    0.05093421,
							LiquidationThreshold: 0.03820066,
							TerminationThreshold: 0.02546711,
						},
						TriggerEstimates: FuturesMarginRequirements{
							InitialMargin:        31100,
							MaintenanceMargin:    30000,
							LiquidationThreshold: 28900,
							TerminationThreshold: 28300,
						},
					},
				},
			},
			ServerTime: futuresTradingTestTime(8, 30, 0, 250),
		}
		assert.Equal(t, exp, result, "GetFuturesAccounts should decode every field")
		return
	}
	assert.Equal(t, "cashAccount", result.Accounts.Cash.Type, "GetFuturesAccounts should return the cash account")
}

func TestFuturesAccountsUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var accounts FuturesAccounts
	require.NoError(t, accounts.UnmarshalJSON([]byte(`{"cash":{"type":"cashAccount","balances":{}},"flex":{"type":"multiCollateralMarginAccount"}}`)), "UnmarshalJSON must not error without margin accounts")
	exp := FuturesAccounts{
		Cash:           FuturesCashAccount{Type: "cashAccount", Balances: map[string]types.Number{}},
		Flex:           FuturesFlexAccount{Type: "multiCollateralMarginAccount"},
		MarginAccounts: map[string]FuturesMarginAccount{},
	}
	assert.Equal(t, exp, accounts, "UnmarshalJSON should decode the cash and flex accounts")
	assert.Error(t, accounts.UnmarshalJSON([]byte(`[]`)), "UnmarshalJSON should reject accounts that are not an object")
	for _, name := range []string{"cash", "flex", "fi_xbtusd"} {
		err := accounts.UnmarshalJSON([]byte(`{"` + name + `":{"type":1}}`))
		assert.ErrorContainsf(t, err, "error decoding "+name+" account", "UnmarshalJSON should name the %s account it cannot decode", name)
	}
}

func TestGetFuturesOpenPositions(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFuturesOpenPositions(t.Context())
	require.NoError(t, err, "GetFuturesOpenPositions must not error")
	if mockTests {
		exp := &FuturesOpenPositionsResponse{
			OpenPositions: []FuturesOpenPosition{
				{
					Symbol:            "PI_XBTUSD",
					Side:              "short",
					Size:              10000,
					AverageEntryPrice: 81802.5,
					UnrealisedPNL:     -0.00104,
					UnrealisedFunding: 1.045432180096817e-05,
				},
				{
					Symbol:               "PF_XBTUSD",
					Side:                 "long",
					Size:                 0.0003,
					AverageEntryPrice:    81496,
					UnrealisedPNL:        0.1023,
					UnrealisedFunding:    -0.0073428045972263895,
					PNLCurrency:          currency.BTC,
					MaximumFixedLeverage: 10,
				},
				{
					Symbol:            "OP_XBTUSD_261225_C_90000",
					Side:              "long",
					Size:              0.1,
					AverageEntryPrice: 1250.5,
					UnrealisedPNL:     12.34,
					PNLCurrency:       currency.USD,
					Greeks: &FuturesPositionGreeks{
						ImpliedVolatility: 0.4812,
						Delta:             0.3125,
						Gamma:             0.0000412,
						Vega:              152.37,
						Theta:             -45.21,
						Rho:               61.08,
					},
				},
			},
			ServerTime: futuresTradingTestTime(8, 31, 0, 375),
		}
		assert.Equal(t, exp, result, "GetFuturesOpenPositions should decode every field")
		return
	}
	assert.NotZero(t, result.ServerTime, "GetFuturesOpenPositions should return the server time")
}

func TestGetFuturesUnwindQueue(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetFuturesUnwindQueue(t.Context())
	require.NoError(t, err, "GetFuturesUnwindQueue must not error")
	if mockTests {
		exp := &FuturesUnwindQueueResponse{
			Queue: []FuturesUnwindQueuePosition{
				{Symbol: "PF_XBTUSD", Percentile: 20},
				{Symbol: "FI_XBTUSD_261225", Percentile: 100},
				{Symbol: "PF_ETHUSD", Percentile: 80},
			},
			ServerTime: futuresTradingTestTime(8, 32, 0, 20),
		}
		assert.Equal(t, exp, result, "GetFuturesUnwindQueue should decode every field")
		return
	}
	assert.NotZero(t, result.ServerTime, "GetFuturesUnwindQueue should return the server time")
}

func TestGetFuturesPortfolioMarginParameters(t *testing.T) {
	t.Parallel()
	if !mockTests {
		t.Skip("Kraken documents portfolio margining as available only in its pre-production environments")
	}
	result, err := e.GetFuturesPortfolioMarginParameters(t.Context())
	require.NoError(t, err, "GetFuturesPortfolioMarginParameters must not error")
	exp := &FuturesPortfolioMarginParametersResponse{
		CrossAssetNettingFactor:             0.5,
		ExtremePriceShockMultiplier:         3,
		VolatilityShockMultiplicationFactor: 0.75,
		VolatilityShockExponentFactor:       0.33,
		OptionExpiryTimeShockHours:          24,
		OptionsInitialMarginFactor:          1.3,
		TotalOptionOrdersInInitialMargin:    50,
		PriceShockLevels:                    []float64{-0.15, -0.075, 0.075, 0.15},
		OptionsUserLimits: FuturesOptionsUserLimits{
			MaximumNetPositionDelta: 25,
			LimitsPerBaseCurrency: map[string]FuturesOptionsBaseCurrencyLimits{
				"BTC": {MaximumTotalPositionSize: 100, MaximumTotalOpenOrdersSize: 50},
				"ETH": {MaximumTotalPositionSize: 1500, MaximumTotalOpenOrdersSize: 750},
			},
		},
		ServerTime: futuresTradingTestTime(8, 33, 0, 9),
	}
	assert.Equal(t, exp, result, "GetFuturesPortfolioMarginParameters should decode every field")
}

func TestSimulateFuturesPortfolio(t *testing.T) {
	t.Parallel()
	_, err := e.SimulateFuturesPortfolio(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrEmptyParams, "SimulateFuturesPortfolio must reject an empty portfolio")
	_, err = e.SimulateFuturesPortfolio(t.Context(), make([]FuturesSimulatedPosition, 501))
	require.ErrorIs(t, err, errFuturesTooManyEntries, "SimulateFuturesPortfolio must reject more than 500 positions")
	_, err = e.SimulateFuturesPortfolio(t.Context(), []FuturesSimulatedPosition{{Size: 1, EntryPrice: 81500}})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "SimulateFuturesPortfolio must reject a position without an instrument")

	if !mockTests {
		t.Skip("Kraken documents portfolio margining as available only in its pre-production environments")
	}
	result, err := e.SimulateFuturesPortfolio(t.Context(), []FuturesSimulatedPosition{
		{Instrument: futuresTestPair, Size: 0.5, EntryPrice: 81500},
		{Instrument: futuresTradingTestOptionPair, Size: -2, EntryPrice: 1250.5},
	})
	require.NoError(t, err, "SimulateFuturesPortfolio must not error")
	exp := &FuturesSimulatePortfolioResponse{
		MaintenanceMargin: 2041.3,
		InitialMargin:     4082.6,
		PNL:               -152.75,
		PortfolioMarginBreakdown: FuturesPortfolioMarginBreakdown{
			TotalCrossAssetNettedMarketRisk:          1530.2,
			TotalMarketRisk:                          1836.24,
			TotalScenarioPNLs:                        []float64{-1836.24, -918.12, 918.12, 1836.2},
			TotalAbsoluteOptionPositionDeltaNotional: 102297.5,
			NetPortfolioDelta:                        -0.125,
			TotalPremium:                             2501,
			IsBuyOnly:                                true,
			FuturesMaintenanceMargin:                 203.75,
		},
		Greeks: map[string]FuturesPositionGreeks{
			"OP_XBTUSD_261225_C_90000": {ImpliedVolatility: 0.4813, Delta: -0.625, Gamma: -0.0000824, Vega: -304.74, Theta: 90.42, Rho: -122.16},
		},
		ServerTime: futuresTradingTestTime(8, 34, 0, 442),
	}
	assert.Equal(t, exp, result, "SimulateFuturesPortfolio should decode every field")
}
