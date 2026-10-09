package kraken

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestGetFuturesInstruments(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *FuturesInstrumentsRequest
		exp  *FuturesInstrumentsResponse
	}{
		{
			name: "every futures market",
			exp: &FuturesInstrumentsResponse{
				Instruments: []FuturesInstrument{
					{
						MakerProtectionMilliseconds: 20,
						Category:                    "Pre-IPO",
						ContractSize:                1,
						ContractValueTradePrecision: 2,
						Description:                 "Anthropic Pre-IPO Perpetual",
						FundingRateCoefficient:      8,
						ImpactMidSize:               0.48,
						MarginSchedules:             map[string]FuturesMarginSchedule{},
						RetailMarginLevels:          []FuturesMarginLevel{{InitialMargin: 0.15, MaintenanceMargin: 0.075}},
						MarginLevels: []FuturesMarginLevel{
							{InitialMargin: 0.1, MaintenanceMargin: 0.05},
							{NonContractUnits: 50000, InitialMargin: 0.2, MaintenanceMargin: 0.1},
						},
						MaxPositionSize:        1000,
						MaxOpenInterestUSD:     10000000,
						MaxOpenInterestShare:   0.25,
						MaxRelativeFundingRate: 7e-06,
						MinRelativeFundingRate: 6.25e-06,
						OpeningDate:            time.Date(2026, 6, 15, 11, 21, 55, 0, time.UTC),
						FeeScheduleUID:         "873d5b70-3446-45cd-9cdf-956a025605c2",
						Symbol:                 "PF_ANTHROPICXUSD",
						Pair:                   "ANTHROPICx:USD",
						Base:                   currency.NewCode("ANTHROPICx"),
						Quote:                  currency.USD,
						TickSize:               0.01,
						Tradeable:              true,
						Type:                   "flexible_futures",
						PlatformsPermitted:     []string{"payward_digital_solutions"},
						CountriesBanned:        []string{},
						RebateLevels:           map[string]types.Number{"0.5": 0.002, "2.5": 0.005},
					},
					{
						MakerProtectionMilliseconds: 20,
						ContractSize:                1,
						FundingRateCoefficient:      8,
						ImpactMidSize:               1000,
						ISIN:                        "GB00J62YGL67",
						MarginSchedules: map[string]FuturesMarginSchedule{
							"europa": {
								Retail: []FuturesMarginLevel{{InitialMargin: 0.1, MaintenanceMargin: 0.05}},
								Professional: []FuturesMarginLevel{
									{InitialMargin: 0.08, MaintenanceMargin: 0.04},
									{Contracts: 10000000, InitialMargin: 0.2, MaintenanceMargin: 0.1},
								},
							},
							"dlt": {
								Retail: []FuturesMarginLevel{{InitialMargin: 0.45, MaintenanceMargin: 0.225}},
								Professional: []FuturesMarginLevel{
									{InitialMargin: 0.03, MaintenanceMargin: 0.015},
									{Contracts: 500000, InitialMargin: 0.06, MaintenanceMargin: 0.03},
								},
							},
						},
						RetailMarginLevels: []FuturesMarginLevel{{InitialMargin: 0.5, MaintenanceMargin: 0.25}},
						MarginLevels: []FuturesMarginLevel{
							{InitialMargin: 0.02, MaintenanceMargin: 0.01},
							{Contracts: 500000, InitialMargin: 0.04, MaintenanceMargin: 0.02},
						},
						MaxPositionSize:        75000000,
						MaxRelativeFundingRate: 0.005,
						OpeningDate:            time.Date(2018, 8, 31, 0, 0, 0, 0, time.UTC),
						FeeScheduleUID:         "a6cbc326-9477-4a6c-911a-d4cb3ed7481e",
						Symbol:                 "PI_XBTUSD",
						Pair:                   "BTC:USD",
						Base:                   currency.BTC,
						Quote:                  currency.USD,
						TickSize:               0.5,
						Tradeable:              true,
						Type:                   "futures_inverse",
						Underlying:             "rr_xbtusd",
						MTF:                    true,
						PlatformsPermitted:     []string{"dlt", "europa", "mtf"},
						CountriesBanned:        []string{},
						RebateLevels:           map[string]types.Number{},
					},
					{
						Category:                    "Forex",
						ContractSize:                1,
						ContractValueTradePrecision: -1,
						FundingRateCoefficient:      8,
						ImpactMidSize:               3500,
						MarginSchedules:             map[string]FuturesMarginSchedule{},
						RetailMarginLevels:          []FuturesMarginLevel{{InitialMargin: 0.03, MaintenanceMargin: 0.015}},
						MarginLevels: []FuturesMarginLevel{
							{InitialMargin: 0.02, MaintenanceMargin: 0.01},
							{NonContractUnits: 500000, InitialMargin: 0.04, MaintenanceMargin: 0.02},
						},
						MaxPositionSize:        5000000,
						MaxRelativeFundingRate: 0.001,
						OpeningDate:            time.Date(2025, 4, 17, 12, 59, 47, 0, time.UTC),
						PostOnly:               true,
						FeeScheduleUID:         "97ae0cc8-2964-43ea-91e9-d8e720b02b19",
						Symbol:                 "PF_EURUSD",
						Pair:                   "EUR:USD",
						Base:                   currency.EUR,
						Quote:                  currency.USD,
						TickSize:               0.00001,
						Tradeable:              true,
						Type:                   "flexible_futures",
						TraditionalFinance:     true,
						PlatformsPermitted:     []string{"payward_digital_solutions"},
						CountriesBanned:        []string{"GB"},
						RebateLevels:           map[string]types.Number{},
					},
				},
				ServerTime: time.Date(2026, 10, 9, 0, 34, 13, 523000000, time.UTC),
			},
		},
		{
			name: "options",
			req:  &FuturesInstrumentsRequest{ContractTypes: []string{"options"}},
			exp: &FuturesInstrumentsResponse{
				Instruments: []FuturesInstrument{
					{
						Category:                    "Layer 1",
						ContractSize:                1,
						ContractValueTradePrecision: 2,
						LastTradingTime:             time.Date(2026, 12, 25, 8, 0, 0, 0, time.UTC),
						MaxPositionSize:             25,
						OpeningDate:                 time.Date(2026, 8, 21, 8, 0, 9, 0, time.UTC),
						FeeScheduleUID:              "97ae0cc8-2964-43ea-91e9-d8e720b02b19",
						Symbol:                      "OF_XBTUSD_261225_120000_C",
						Pair:                        "BTC:USD",
						Base:                        currency.BTC,
						Quote:                       currency.USD,
						TickSize:                    1,
						Tradeable:                   true,
						Type:                        "options",
						UnderlyingFuture:            "FF_XBTUSD_261225",
						PlatformsPermitted:          []string{"payward_digital_solutions"},
						CountriesBanned:             []string{},
						OptionType:                  "call",
						StrikePrice:                 120000,
						RebateLevels:                map[string]types.Number{},
					},
				},
				ServerTime: time.Date(2026, 10, 9, 0, 35, 1, 170000000, time.UTC),
			},
		},
		{
			name: "expired inverse futures",
			req:  &FuturesInstrumentsRequest{ContractTypes: []string{"futures_inverse"}, Expired: true},
			exp: &FuturesInstrumentsResponse{
				Instruments: []FuturesInstrument{
					{
						Category:        "Layer 1",
						ContractSize:    1,
						ImpactMidSize:   1000,
						ISIN:            "GB00BTQJRH93",
						LastTradingTime: time.Date(2026, 8, 28, 15, 0, 0, 0, time.UTC),
						MarginSchedules: map[string]FuturesMarginSchedule{
							"europa": {
								Retail:       []FuturesMarginLevel{{InitialMargin: 0.1, MaintenanceMargin: 0.05}},
								Professional: []FuturesMarginLevel{{InitialMargin: 0.08, MaintenanceMargin: 0.04}},
							},
						},
						RetailMarginLevels: []FuturesMarginLevel{{InitialMargin: 0.5, MaintenanceMargin: 0.25}},
						MarginLevels: []FuturesMarginLevel{
							{InitialMargin: 0.02, MaintenanceMargin: 0.01},
							{Contracts: 500000, InitialMargin: 0.04, MaintenanceMargin: 0.02},
						},
						MaxPositionSize:    40000000,
						OpeningDate:        time.Date(2026, 7, 31, 15, 0, 46, 0, time.UTC),
						FeeScheduleUID:     "a6cbc326-9477-4a6c-911a-d4cb3ed7481e",
						Symbol:             "FI_XBTUSD_260828",
						Pair:               "BTC:USD",
						Base:               currency.BTC,
						Quote:              currency.USD,
						TickSize:           0.5,
						Tradeable:          true,
						Type:               "futures_inverse",
						Underlying:         "rr_xbtusd",
						MTF:                true,
						PlatformsPermitted: []string{"europa", "mtf"},
						CountriesBanned:    []string{},
						IsExpired:          true,
						RebateLevels:       map[string]types.Number{},
					},
				},
				ServerTime: time.Date(2026, 10, 9, 0, 35, 7, 660000000, time.UTC),
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetFuturesInstruments(t.Context(), tc.req)
			require.NoError(t, err, "GetFuturesInstruments must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetFuturesInstruments should decode every field")
				return
			}
			assert.NotEmpty(t, result.Instruments, "GetFuturesInstruments should return instruments")
		})
	}
}

func TestGetFuturesTradingInstruments(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	for _, tc := range []struct {
		name string
		req  *FuturesTradingInstrumentsRequest
		exp  *FuturesTradingInstrumentsResponse
	}{
		{
			name: "every futures market",
			exp: &FuturesTradingInstrumentsResponse{
				Instruments: []FuturesTradingInstrument{
					{
						FundingRateCoefficient: 8,
						ImpactMidSize:          0.48,
						MaxPositionSize:        1000,
						MaxOpenInterestUSD:     10000000,
						MaxOpenInterestShare:   0.25,
						OpeningDate:            time.Date(2026, 6, 15, 11, 21, 55, 0, time.UTC),
						MarginLevels: []FuturesMarginLevel{
							{InitialMargin: 0.1, MaintenanceMargin: 0.05},
							{NonContractUnits: 50000, InitialMargin: 0.2, MaintenanceMargin: 0.1},
						},
						MaxRelativeFundingRate:      7e-06,
						MinRelativeFundingRate:      6.25e-06,
						Symbol:                      "PF_ANTHROPICXUSD",
						Pair:                        "ANTHROPICx:USD",
						Base:                        currency.NewCode("ANTHROPICx"),
						Quote:                       currency.USD,
						TickSize:                    0.01,
						Type:                        "flexible_futures",
						ContractValueTradePrecision: 2,
						Description:                 "Anthropic Pre-IPO Perpetual",
						MakerProtectionMilliseconds: 20,
						FeeScheduleUID:              "873d5b70-3446-45cd-9cdf-956a025605c2",
						RebateLevels:                map[string]types.Number{"0.5": 0.002, "2.5": 0.005},
						Restricted:                  true,
					},
					{
						LastTradingTime: time.Date(2026, 12, 25, 16, 0, 0, 0, time.UTC),
						ImpactMidSize:   1000,
						MaxPositionSize: 40000000,
						OpeningDate:     time.Date(2026, 5, 29, 15, 0, 0, 0, time.UTC),
						MarginLevels: []FuturesMarginLevel{
							{InitialMargin: 0.02, MaintenanceMargin: 0.01},
							{Contracts: 500000, InitialMargin: 0.04, MaintenanceMargin: 0.02},
						},
						Symbol:         "FI_XBTUSD_261225",
						Pair:           "BTC:USD",
						Base:           currency.BTC,
						Quote:          currency.USD,
						TickSize:       0.5,
						Type:           "futures_inverse",
						Underlying:     "rr_xbtusd",
						ISIN:           "GB00BTQJRM47",
						PostOnly:       true,
						FeeScheduleUID: "a6cbc326-9477-4a6c-911a-d4cb3ed7481e",
						RebateLevels:   map[string]types.Number{},
						MTF:            true,
					},
				},
				ServerTime: time.Date(2026, 10, 9, 0, 36, 20, 50000000, time.UTC),
			},
		},
		{
			name: "contract types",
			req:  &FuturesTradingInstrumentsRequest{ContractTypes: []string{"options", "flexible_futures"}},
			exp: &FuturesTradingInstrumentsResponse{
				Instruments: []FuturesTradingInstrument{
					{
						LastTradingTime:             time.Date(2026, 10, 2, 8, 0, 0, 0, time.UTC),
						ImpactMidSize:               0.5,
						MaxPositionSize:             25,
						OpeningDate:                 time.Date(2026, 9, 25, 8, 0, 11, 0, time.UTC),
						Symbol:                      "OF_XBTUSD_261002_84000_P",
						Pair:                        "BTC:USD",
						Base:                        currency.BTC,
						Quote:                       currency.USD,
						TickSize:                    5,
						Type:                        "options",
						ContractValueTradePrecision: 2,
						FeeScheduleUID:              "97ae0cc8-2964-43ea-91e9-d8e720b02b19",
						OptionType:                  "put",
						StrikePrice:                 84000,
						UnderlyingFuture:            "FF_XBTUSD_261002",
						RebateLevels:                map[string]types.Number{},
						IsExpired:                   true,
					},
					{
						FundingRateCoefficient:      8,
						ImpactMidSize:               3500,
						MaxPositionSize:             5000000,
						OpeningDate:                 time.Date(2025, 4, 17, 12, 59, 47, 0, time.UTC),
						MarginLevels:                []FuturesMarginLevel{{InitialMargin: 0.02, MaintenanceMargin: 0.01}},
						MaxRelativeFundingRate:      0.001,
						Symbol:                      "PF_EURUSD",
						Pair:                        "EUR:USD",
						Base:                        currency.EUR,
						Quote:                       currency.USD,
						TickSize:                    0.00001,
						Type:                        "flexible_futures",
						ContractValueTradePrecision: -1,
						FeeScheduleUID:              "97ae0cc8-2964-43ea-91e9-d8e720b02b19",
						RebateLevels:                map[string]types.Number{},
						TraditionalFinance:          true,
					},
				},
				ServerTime: time.Date(2026, 10, 9, 0, 36, 22, 810000000, time.UTC),
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetFuturesTradingInstruments(t.Context(), tc.req)
			require.NoError(t, err, "GetFuturesTradingInstruments must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetFuturesTradingInstruments should decode every field")
				return
			}
			assert.NotEmpty(t, result.Instruments, "GetFuturesTradingInstruments should return instruments")
		})
	}
}

func TestGetFuturesInstrumentStatusList(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		req  *FuturesInstrumentStatusListRequest
		exp  *FuturesInstrumentStatusListResponse
	}{
		{
			name: "every futures market",
			exp: &FuturesInstrumentStatusListResponse{
				InstrumentStatus: []FuturesInstrumentStatus{
					{Symbol: "PF_XBTUSD", ExperiencingDislocation: true, PriceDislocationDirection: "ABOVE_UPPER_BOUND", ExperiencingExtremeVolatility: true, ExtremeVolatilityInitialMarginMultiplier: 2},
					{Symbol: "PI_XBTUSD", ExperiencingDislocation: true, PriceDislocationDirection: "BELOW_LOWER_BOUND", ExtremeVolatilityInitialMarginMultiplier: 1},
				},
				ServerTime: time.Date(2026, 10, 9, 0, 35, 15, 98000000, time.UTC),
			},
		},
		{
			name: "contract types",
			req:  &FuturesInstrumentStatusListRequest{ContractTypes: []string{"futures_inverse"}},
			exp: &FuturesInstrumentStatusListResponse{
				InstrumentStatus: []FuturesInstrumentStatus{{Symbol: "PI_XBTUSD", ExtremeVolatilityInitialMarginMultiplier: 1}},
				ServerTime:       time.Date(2026, 10, 9, 0, 35, 16, 612000000, time.UTC),
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetFuturesInstrumentStatusList(t.Context(), tc.req)
			require.NoError(t, err, "GetFuturesInstrumentStatusList must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetFuturesInstrumentStatusList should decode every field")
				return
			}
			assert.NotEmpty(t, result.InstrumentStatus, "GetFuturesInstrumentStatusList should return statuses")
		})
	}
}

func TestGetFuturesInstrumentStatus(t *testing.T) {
	t.Parallel()
	_, err := e.GetFuturesInstrumentStatus(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetFuturesInstrumentStatus must reject an empty pair")

	result, err := e.GetFuturesInstrumentStatus(t.Context(), futuresTestPair)
	require.NoError(t, err, "GetFuturesInstrumentStatus must not error")
	if mockTests {
		exp := &FuturesInstrumentStatusResponse{
			Symbol:                                   "PF_XBTUSD",
			ExperiencingDislocation:                  true,
			PriceDislocationDirection:                "BELOW_LOWER_BOUND",
			ExperiencingExtremeVolatility:            true,
			ExtremeVolatilityInitialMarginMultiplier: 3,
			ServerTime:                               time.Date(2026, 10, 9, 0, 34, 17, 169000000, time.UTC),
		}
		assert.Equal(t, exp, result, "GetFuturesInstrumentStatus should decode every field")
		return
	}
	assert.Equal(t, "PF_XBTUSD", result.Symbol, "GetFuturesInstrumentStatus should return the requested market's status")
}
