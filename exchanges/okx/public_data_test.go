package okx

import (
	"math"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	"github.com/thrasher-corp/gocryptotrader/types"
)

func TestGetInstruments(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *InstrumentsRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "missing instrument type", arg: &InstrumentsRequest{Underlying: mainPair.String()}, err: errInvalidInstrumentType},
		{name: "option without family or underlying", arg: &InstrumentsRequest{InstrumentType: instTypeOption}, err: errInstrumentFamilyOrUnderlyingRequired},
		{name: "events without series", arg: &InstrumentsRequest{InstrumentType: instTypeEvents}, err: errSeriesIDRequired},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.GetInstruments(t.Context(), tc.arg)
			assert.ErrorIs(t, err, tc.err, "GetInstruments should reject an invalid request")
		})
	}

	t.Run("futures by underlying", func(t *testing.T) {
		t.Parallel()
		resp, err := e.GetInstruments(contextGenerate(), &InstrumentsRequest{InstrumentType: instTypeFutures, Underlying: "SOL-USD"})
		require.NoError(t, err, "GetInstruments must not error")
		require.NotEmpty(t, resp, "GetInstruments must return live instruments for SOL-USD futures")
		for i := range resp {
			assert.Equal(t, instTypeFutures, resp[i].InstrumentType, "InstrumentType should be correct")
			assert.Equal(t, "SOL-USD", resp[i].Underlying, "Underlying should be correct")
			assert.True(t, resp[i].InstrumentID.IsPopulated(), "InstrumentID should be populated")
			assert.NotEmpty(t, resp[i].State, "State should not be empty")
		}
		if mockTests {
			exp := InstrumentResponse{
				InstrumentType:            "FUTURES",
				InstrumentID:              currency.NewPairWithDelimiter("SOL", "USD_UM_XPERP-310404", "-"),
				Underlying:                "SOL-USD",
				TradingFeeGroupID:         "7",
				InstrumentFamily:          "SOL-USD_UM_XPERP",
				Category:                  "1",
				SettlementCurrency:        currency.USD,
				ContractValue:             0.01,
				ContractMultiplier:        1,
				ContractValueCurrency:     currency.SOL,
				ListTime:                  types.Time(time.UnixMilli(1775017500279)),
				ExpiryTime:                types.Time(time.UnixMilli(1933056000000)),
				MaxLeverage:               50,
				TickSize:                  0.01,
				LotSize:                   1,
				MinimumOrderSize:          1,
				ContractType:              "linear",
				Alias:                     "this_five_years",
				State:                     "live",
				RuleType:                  "xperp",
				MaxLimitSize:              100000000,
				MaxMarketSize:             100000,
				MaxLimitAmountUSD:         20000000,
				MaxTWAPSize:               100000000.0000000000000000,
				MaxIcebergSize:            100000000.0000000000000000,
				MaxTriggerSize:            100000000.0000000000000000,
				MaxStopSize:               100000,
				TradableQuoteCurrencies:   []currency.Code{},
				InstrumentIDCode:          255653,
				InstrumentCategory:        "1",
				InitialPriceLimitPercent:  0.02,
				FloatingPriceLimitPercent: 0.005,
				MaximumPriceLimitPercent:  0.01,
				RPIMinimumLevel:           2,
				RPIMinimumPriceBand:       20,
				PositionLimitAmount:       42000000,
				PositionLimitPercent:      25,
				UpcomingChanges:           []InstrumentUpcomingChange{},
			}
			assert.Equal(t, exp, resp[0], "GetInstruments should decode every field")
		}
	})
	t.Run("spot", func(t *testing.T) {
		t.Parallel()
		resp, err := e.GetInstruments(contextGenerate(), &InstrumentsRequest{InstrumentType: instTypeSpot})
		require.NoError(t, err, "GetInstruments must not error")
		require.NotEmpty(t, resp, "GetInstruments must return spot instruments")
		if mockTests {
			exp := InstrumentResponse{
				InstrumentType:            "SPOT",
				InstrumentID:              currency.NewPairWithDelimiter("BTC", "USDT", "-"),
				TradingFeeGroupID:         "12",
				Category:                  "1",
				BaseCurrency:              currency.BTC,
				QuoteCurrency:             currency.USDT,
				ListTime:                  types.Time(time.UnixMilli(1611907686000)),
				OpenType:                  "fix_price",
				MaxLeverage:               10,
				TickSize:                  0.1,
				LotSize:                   0.00000001,
				MinimumOrderSize:          0.00001,
				State:                     "live",
				RuleType:                  "normal",
				MaxLimitSize:              9999999999,
				MaxMarketSize:             1000000,
				MaxLimitAmountUSD:         20000000,
				MaxMarketAmountUSD:        1000000,
				MaxTWAPSize:               9999999999.0000000000000000,
				MaxIcebergSize:            9999999999.0000000000000000,
				MaxTriggerSize:            9999999999.0000000000000000,
				MaxStopSize:               1000000,
				TradableQuoteCurrencies:   []currency.Code{currency.USDT},
				InstrumentIDCode:          3,
				InstrumentCategory:        "1",
				FloatingPriceLimitPercent: 0.005,
				MaximumPriceLimitPercent:  0.01,
				RPIMinimumLevel:           3,
				RPIMinimumPriceBand:       20,
				UpcomingChanges:           []InstrumentUpcomingChange{},
			}
			assert.Equal(t, exp, resp[0], "GetInstruments should decode every field")
		}
	})
	t.Run("swap by underlying", func(t *testing.T) {
		t.Parallel()
		resp, err := e.GetInstruments(contextGenerate(), &InstrumentsRequest{InstrumentType: instTypeSwap, Underlying: mainPair.String()})
		require.NoError(t, err, "GetInstruments must not error")
		require.NotEmpty(t, resp, "GetInstruments must return swap instruments")
		if mockTests {
			exp := InstrumentResponse{
				InstrumentType:              "SWAP",
				InstrumentID:                currency.NewPairWithDelimiter("BTC", "USDT-SWAP", "-"),
				Underlying:                  "BTC-USDT",
				TradingFeeGroupID:           "4",
				InstrumentFamily:            "BTC-USDT",
				Category:                    "1",
				SettlementCurrency:          currency.USDT,
				ContractValue:               0.01,
				ContractMultiplier:          1,
				ContractValueCurrency:       currency.BTC,
				ListTime:                    types.Time(time.UnixMilli(1573557408000)),
				ContinuousTradingSwitchTime: types.Time(time.UnixMilli(1611916860000)),
				OpenType:                    "call_auction",
				MaxLeverage:                 100,
				TickSize:                    0.1,
				LotSize:                     0.01,
				MinimumOrderSize:            0.01,
				ContractType:                "linear",
				State:                       "live",
				RuleType:                    "normal",
				MaxLimitSize:                100000000,
				MaxMarketSize:               35000,
				MaxLimitAmountUSD:           20000000,
				MaxTWAPSize:                 100000000.0000000000000000,
				MaxIcebergSize:              100000000.0000000000000000,
				MaxTriggerSize:              100000000.0000000000000000,
				MaxStopSize:                 35000,
				TradableQuoteCurrencies:     []currency.Code{},
				InstrumentIDCode:            10459,
				InstrumentCategory:          "1",
				InitialPriceLimitPercent:    0.02,
				FloatingPriceLimitPercent:   0.005,
				MaximumPriceLimitPercent:    0.01,
				RPIMinimumLevel:             3,
				RPIMinimumPriceBand:         20,
				PositionLimitAmount:         250000,
				PositionLimitPercent:        30,
				UpcomingChanges:             []InstrumentUpcomingChange{},
			}
			assert.Equal(t, exp, resp[0], "GetInstruments should decode every field")
		}
	})
	t.Run("event contracts by series", func(t *testing.T) {
		t.Parallel()
		resp, err := e.GetInstruments(contextGenerate(), &InstrumentsRequest{InstrumentType: instTypeEvents, SeriesID: "BTC-ABOVE-DAILY"})
		require.NoError(t, err, "GetInstruments must not error")
		require.NotEmpty(t, resp, "GetInstruments must return event contracts")
		if mockTests {
			exp := InstrumentResponse{
				InstrumentType:          "EVENTS",
				SeriesID:                "BTC-ABOVE-DAILY",
				InstrumentID:            currency.NewPairWithDelimiter("BTC", "ABOVE-DAILY-261123-1600-56000", "-"),
				Underlying:              "BTC-USDT",
				TradingFeeGroupID:       "1",
				Category:                "1",
				SettlementCurrency:      currency.USDT,
				ContractValue:           1,
				ContractMultiplier:      1,
				StrikePrice:             56000,
				ListTime:                types.Time(time.UnixMilli(1794815943566)),
				ExpiryTime:              types.Time(time.UnixMilli(1795420800000)),
				TickSize:                0.001,
				LotSize:                 0.1,
				MinimumOrderSize:        0.01,
				State:                   "live",
				RuleType:                "normal",
				MaxLimitSize:            20000000,
				TradableQuoteCurrencies: []currency.Code{},
				InstrumentIDCode:        566263,
				Frequency:               "DAILY",
				SettlementMethod:        "PRICE_ABOVE",
				UpcomingChanges:         []InstrumentUpcomingChange{},
			}
			assert.Equal(t, exp, resp[0], "GetInstruments should decode every field")
		}
	})
}

func TestGetEventContractSeries(t *testing.T) {
	t.Parallel()
	_, err := e.GetEventContractSeries(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetEventContractSeries must reject a nil request")

	result, err := e.GetEventContractSeries(contextGenerate(), &EventContractSeriesRequest{SeriesID: "BTC-ABOVE-DAILY"})
	require.NoError(t, err, "GetEventContractSeries must not error")
	if mockTests {
		exp := []EventContractSeriesResponse{
			{
				SeriesID:  "BTC-ABOVE-DAILY",
				Frequency: "daily",
				Title:     "BTC Price above Daily",
				Category:  "Crypto",
				Settlement: EventContractSettlement{
					Method:     "price_above",
					SourceName: "okx_index",
					Underlying: "BTC-USDT",
				},
			},
		}
		assert.Equal(t, exp, result, "GetEventContractSeries should decode every field")
	} else {
		assert.NotEmpty(t, result, "GetEventContractSeries should return the series")
	}
}

func TestGetEventContractEvents(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *EventContractEventsRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "missing series", arg: &EventContractEventsRequest{Limit: 2}, err: errSeriesIDRequired},
		{name: "start after end", arg: &EventContractEventsRequest{SeriesID: "BTC-UPDOWN-15MIN", Before: time.UnixMilli(1795405500000), After: time.UnixMilli(1795404600000)}, err: common.ErrStartAfterEnd},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.GetEventContractEvents(t.Context(), tc.arg)
			assert.ErrorIs(t, err, tc.err, "GetEventContractEvents should reject an invalid request")
		})
	}

	result, err := e.GetEventContractEvents(contextGenerate(), &EventContractEventsRequest{SeriesID: "BTC-UPDOWN-15MIN", State: "expired", Limit: 2})
	require.NoError(t, err, "GetEventContractEvents must not error")
	if mockTests {
		exp := []EventContractEventResponse{
			{
				SeriesID:   "BTC-UPDOWN-15MIN",
				EventID:    "BTC-UPDOWN-15MIN-261123-1145-1200",
				FixTime:    types.Time(time.UnixMilli(1795405501031)),
				ExpiryTime: types.Time(time.UnixMilli(1795406400000)),
				State:      "expired",
			},
			{
				SeriesID:   "BTC-UPDOWN-15MIN",
				EventID:    "BTC-UPDOWN-15MIN-261123-1130-1145",
				FixTime:    types.Time(time.UnixMilli(1795404601032)),
				ExpiryTime: types.Time(time.UnixMilli(1795405500000)),
				State:      "expired",
			},
		}
		assert.Equal(t, exp, result, "GetEventContractEvents should decode every field")
	} else {
		assert.NotEmpty(t, result, "GetEventContractEvents should return events")
	}
}

func TestGetCorporateActions(t *testing.T) {
	t.Parallel()
	_, err := e.GetCorporateActions(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetCorporateActions must reject a nil request")
	_, err = e.GetCorporateActions(t.Context(), &CorporateActionsRequest{Ticker: "JNJ", InstrumentID: "JNJ-USDT-SWAP"})
	require.ErrorIs(t, err, errTickerAndInstrumentIDExclusive, "GetCorporateActions must reject a ticker with an instrument ID")

	result, err := e.GetCorporateActions(contextGenerate(), &CorporateActionsRequest{Limit: 2})
	require.NoError(t, err, "GetCorporateActions must not error")
	require.NotNil(t, result, "GetCorporateActions must return a page")
	if mockTests {
		exp := &CorporateActionsResponse{
			DataStatus: "SUCCESS",
			DataAsOf:   time.Date(2026, time.November, 23, 4, 5, 59, 99000000, time.UTC),
			NextAfter:  "djF8MTc5NTQ4MjAwMDAwMHwzY2UzYjFlYjdhYjJlNDFiODIzNDFkMzNhYWY5MDY4NXxlMTk5OTU1YWUxOTkyNzFm",
			Events: []CorporateActionEvent{
				{
					EventID:            "072cb53bdb65f2a8210382c076e18281",
					EventVersion:       1,
					Underlying:         "JNJ",
					BloombergGlobalID:  "BBG000BMHYD1",
					Exchange:           "XNYS",
					ActionType:         "CASH_DIVIDEND",
					Class:              "DIVIDEND",
					DividendSize:       "SMALL",
					CashDividend:       "USD 1.34 / share",
					Status:             "SCHEDULED",
					PlannedCutoverTime: time.Date(2026, time.November, 24, 1, 0, 0, 0, time.UTC),
					Instruments: []CorporateActionInstrument{
						{
							InstrumentID: "JNJ-USDT-SWAP",
							State:        "live",
						},
						{
							InstrumentID: "XJNJ-USDT",
							State:        "live",
						},
					},
				},
				{
					EventID:            "3ce3b1eb7ab2e41b82341d33aaf90685",
					EventVersion:       1,
					Underlying:         "COST",
					BloombergGlobalID:  "BBG000F6H8W8",
					Exchange:           "XNGS",
					ActionType:         "CASH_DIVIDEND",
					Class:              "DIVIDEND",
					DividendSize:       "SMALL",
					CashDividend:       "USD 30 / share",
					Status:             "SCHEDULED",
					PlannedCutoverTime: time.Date(2026, time.November, 24, 1, 0, 0, 0, time.UTC),
					Instruments: []CorporateActionInstrument{
						{
							InstrumentID: "COST-USDT-SWAP",
							State:        "live",
						},
						{
							InstrumentID: "XCOST-USDT",
							State:        "live",
						},
					},
				},
			},
		}
		assert.Equal(t, exp, result, "GetCorporateActions should decode every field")
	} else {
		assert.NotEmpty(t, result.DataStatus, "GetCorporateActions should return the data status")
	}
}

func TestGetEventContractMarkets(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *EventContractMarketsRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "missing series", arg: &EventContractMarketsRequest{Limit: 2}, err: errSeriesIDRequired},
		{name: "start after end", arg: &EventContractMarketsRequest{SeriesID: "BTC-UPDOWN-15MIN", Before: time.UnixMilli(1795405500000), After: time.UnixMilli(1795404600000)}, err: common.ErrStartAfterEnd},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.GetEventContractMarkets(t.Context(), tc.arg)
			assert.ErrorIs(t, err, tc.err, "GetEventContractMarkets should reject an invalid request")
		})
	}

	t.Run("expired up or down", func(t *testing.T) {
		t.Parallel()
		result, err := e.GetEventContractMarkets(contextGenerate(), &EventContractMarketsRequest{SeriesID: "BTC-UPDOWN-15MIN", State: "expired", Limit: 2})
		require.NoError(t, err, "GetEventContractMarkets must not error")
		if mockTests {
			exp := []EventContractMarketResponse{
				{
					SeriesID:     "BTC-UPDOWN-15MIN",
					EventID:      "BTC-UPDOWN-15MIN-261123-1145-1200",
					InstrumentID: "BTC-UPDOWN-15MIN-261123-1145-1200",
					ListTime:     types.Time(time.UnixMilli(1795401840000)),
					FixTime:      types.Time(time.UnixMilli(1795405501031)),
					ExpiryTime:   types.Time(time.UnixMilli(1795406400000)),
					State:        "expired",
					Outcome:      "1",
					FloorStrike:  61295.6,
					SettleValue:  61295.747,
				},
				{
					SeriesID:     "BTC-UPDOWN-15MIN",
					EventID:      "BTC-UPDOWN-15MIN-261123-1130-1145",
					InstrumentID: "BTC-UPDOWN-15MIN-261123-1130-1145",
					ListTime:     types.Time(time.UnixMilli(1795400940000)),
					FixTime:      types.Time(time.UnixMilli(1795404601032)),
					ExpiryTime:   types.Time(time.UnixMilli(1795405500000)),
					State:        "expired",
					Outcome:      "2",
					FloorStrike:  61444.2,
					SettleValue:  61315.185,
				},
			}
			assert.Equal(t, exp, result, "GetEventContractMarkets should decode every field")
		} else {
			assert.NotEmpty(t, result, "GetEventContractMarkets should return markets")
		}
	})
	t.Run("between bracket without an upper bound", func(t *testing.T) {
		t.Parallel()
		arg := &EventContractMarketsRequest{SeriesID: "BTC-BETWEEN-DAILY", EventID: "BTC-BETWEEN-DAILY-261129-1600", InstrumentID: "BTC-BETWEEN-DAILY-261129-1600-66000-INF"}
		if !mockTests {
			arg = &EventContractMarketsRequest{SeriesID: "BTC-BETWEEN-DAILY", Limit: 2}
		}
		result, err := e.GetEventContractMarkets(contextGenerate(), arg)
		require.NoError(t, err, "GetEventContractMarkets must not error")
		if mockTests {
			exp := []EventContractMarketResponse{
				{
					SeriesID:     "BTC-BETWEEN-DAILY",
					EventID:      "BTC-BETWEEN-DAILY-261129-1600",
					InstrumentID: "BTC-BETWEEN-DAILY-261129-1600-66000-INF",
					ListTime:     types.Time(time.UnixMilli(1795247974000)),
					ExpiryTime:   types.Time(time.UnixMilli(1795939200000)),
					State:        "live",
					Outcome:      "0",
					FloorStrike:  66000,
					CapStrike:    types.Number(math.Inf(1)),
				},
			}
			assert.Equal(t, exp, result, "GetEventContractMarkets should decode every field")
		} else {
			assert.NotEmpty(t, result, "GetEventContractMarkets should return markets")
		}
	})
	t.Run("expired hit", func(t *testing.T) {
		t.Parallel()
		result, err := e.GetEventContractMarkets(contextGenerate(), &EventContractMarketsRequest{SeriesID: "BTC-HIT-DAILY", State: "expired", Limit: 2})
		require.NoError(t, err, "GetEventContractMarkets must not error")
		if mockTests {
			exp := []EventContractMarketResponse{
				{
					SeriesID:     "BTC-HIT-DAILY",
					EventID:      "BTC-HIT-DAILY-261123-1600",
					InstrumentID: "BTC-HIT-DAILY-261123-1600-60000",
					ListTime:     types.Time(time.UnixMilli(1795334378000)),
					ExpiryTime:   types.Time(time.UnixMilli(1795420800000)),
					State:        "expired",
					Outcome:      "1",
					FloorStrike:  60000,
					SettleValue:  60007.8,
					HitDirection: "up",
				},
				{
					SeriesID:     "BTC-HIT-DAILY",
					EventID:      "BTC-HIT-DAILY-261123-1600",
					InstrumentID: "BTC-HIT-DAILY-261123-1600-61000",
					ListTime:     types.Time(time.UnixMilli(1795334378000)),
					ExpiryTime:   types.Time(time.UnixMilli(1795420800000)),
					State:        "expired",
					Outcome:      "1",
					FloorStrike:  61000,
					SettleValue:  61100.3,
					HitDirection: "up",
				},
			}
			assert.Equal(t, exp, result, "GetEventContractMarkets should decode every field")
		} else {
			assert.NotEmpty(t, result, "GetEventContractMarkets should return markets")
		}
	})
}

func TestGetEstimatedDeliveryPrice(t *testing.T) {
	t.Parallel()
	_, err := e.GetEstimatedDeliveryPrice(t.Context(), "")
	require.ErrorIs(t, err, errMissingInstrumentID, "GetEstimatedDeliveryPrice must require an instrument ID")

	testexch.UpdatePairsOnce(t, e)
	p, err := e.GetAvailablePairs(asset.Futures)
	require.NoError(t, err, "GetAvailablePairs must not error")
	require.NotEmpty(t, p, "GetAvailablePairs must not return empty pairs")

	result, err := e.GetEstimatedDeliveryPrice(contextGenerate(), p[0].String())
	require.NoError(t, err, "GetEstimatedDeliveryPrice must not error")
	require.NotNil(t, result, "GetEstimatedDeliveryPrice must return a price")
	if mockTests {
		exp := &EstimatedDeliveryPriceResponse{
			InstrumentType: "FUTURES",
			InstrumentID:   "BTC-USD-261127",
			Timestamp:      types.Time(time.UnixMilli(1795406764303)),
		}
		assert.Equal(t, exp, result, "GetEstimatedDeliveryPrice should decode every field")
	} else {
		assert.Equal(t, p[0].String(), result.InstrumentID, "GetEstimatedDeliveryPrice should return the instrument's price")
	}
}

func TestGetDeliveryHistory(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *DeliveryHistoryRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "missing instrument type", arg: &DeliveryHistoryRequest{Limit: 3}, err: errInvalidInstrumentType},
		{name: "missing instrument family", arg: &DeliveryHistoryRequest{InstrumentType: instTypeFutures, Limit: 3}, err: errInstrumentFamilyRequired},
		{name: "limit above maximum", arg: &DeliveryHistoryRequest{InstrumentType: instTypeFutures, InstrumentFamily: futuresPair.String(), Limit: 345}, err: errLimitValueExceedsMaxOf100},
		{name: "start after end", arg: &DeliveryHistoryRequest{InstrumentType: instTypeFutures, InstrumentFamily: futuresPair.String(), Before: time.UnixMilli(1793347200000), After: time.UnixMilli(1790323200000)}, err: common.ErrStartAfterEnd},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.GetDeliveryHistory(t.Context(), tc.arg)
			assert.ErrorIs(t, err, tc.err, "GetDeliveryHistory should reject an invalid request")
		})
	}

	result, err := e.GetDeliveryHistory(contextGenerate(), &DeliveryHistoryRequest{InstrumentType: instTypeFutures, InstrumentFamily: futuresPair.String(), Limit: 3})
	require.NoError(t, err, "GetDeliveryHistory must not error")
	if mockTests {
		exp := []DeliveryHistoryResponse{
			{
				Timestamp: types.Time(time.UnixMilli(1793347200000)),
				Details: []DeliveryHistoryDetail{
					{
						InstrumentID: "BTC-USD-261030",
						Price:        78121.9922175920711791,
						Type:         "delivery",
					},
				},
			},
			{
				Timestamp: types.Time(time.UnixMilli(1790323200000)),
				Details: []DeliveryHistoryDetail{
					{
						InstrumentID: "BTC-USD-260925",
						Price:        83933.3435920476243962,
						Type:         "delivery",
					},
				},
			},
			{
				Timestamp: types.Time(time.UnixMilli(1788508800000)),
				Details: []DeliveryHistoryDetail{
					{
						InstrumentID: "BTC-USD-260904",
						Price:        80758.0793433037220285,
						Type:         "delivery",
					},
				},
			},
		}
		assert.Equal(t, exp, result, "GetDeliveryHistory should decode every field")
	} else {
		assert.NotEmpty(t, result, "GetDeliveryHistory should return deliveries")
	}
}

func TestGetEstimatedSettlementPrice(t *testing.T) {
	t.Parallel()
	_, err := e.GetEstimatedSettlementPrice(t.Context(), "")
	require.ErrorIs(t, err, errMissingInstrumentID, "GetEstimatedSettlementPrice must require an instrument ID")

	testexch.UpdatePairsOnce(t, e)
	p, err := e.GetAvailablePairs(asset.Futures)
	require.NoError(t, err, "GetAvailablePairs must not error")
	require.NotEmpty(t, p, "GetAvailablePairs must not return empty pairs")

	result, err := e.GetEstimatedSettlementPrice(contextGenerate(), p[0].String())
	require.NoError(t, err, "GetEstimatedSettlementPrice must not error")
	require.NotNil(t, result, "GetEstimatedSettlementPrice must return a price")
	if mockTests {
		exp := &EstimatedSettlementPriceResponse{
			InstrumentID:       "BTC-USD-261127",
			NextSettlementTime: types.Time(time.UnixMilli(1795420800000)),
			Timestamp:          types.Time(time.UnixMilli(1795406765143)),
		}
		assert.Equal(t, exp, result, "GetEstimatedSettlementPrice should decode every field")
	} else {
		assert.Equal(t, p[0].String(), result.InstrumentID, "GetEstimatedSettlementPrice should return the instrument's price")
	}
}

func TestGetFuturesSettlementHistory(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *FuturesSettlementHistoryRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "missing instrument family", arg: &FuturesSettlementHistoryRequest{Limit: 2}, err: errInstrumentFamilyRequired},
		{name: "start after end", arg: &FuturesSettlementHistoryRequest{InstrumentFamily: futuresPair.String(), Before: time.UnixMilli(1795334400000), After: time.UnixMilli(1795248000000)}, err: common.ErrStartAfterEnd},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.GetFuturesSettlementHistory(t.Context(), tc.arg)
			assert.ErrorIs(t, err, tc.err, "GetFuturesSettlementHistory should reject an invalid request")
		})
	}

	result, err := e.GetFuturesSettlementHistory(contextGenerate(), &FuturesSettlementHistoryRequest{InstrumentFamily: futuresPair.String(), Limit: 2})
	require.NoError(t, err, "GetFuturesSettlementHistory must not error")
	if mockTests {
		exp := []FuturesSettlementHistoryResponse{
			{
				Timestamp: types.Time(time.UnixMilli(1795334400000)),
				Details: []FuturesSettlementDetail{
					{
						InstrumentID:    "BTC-USD-270924",
						SettlementPrice: 61685.9231828872214133,
					},
				},
			},
			{
				Timestamp: types.Time(time.UnixMilli(1795248000000)),
				Details: []FuturesSettlementDetail{
					{
						InstrumentID:    "BTC-USD-271231",
						SettlementPrice: 61725.0601702508960573,
					},
					{
						InstrumentID:    "BTC-USD-261225",
						SettlementPrice: 58207.0560147849462365,
					},
				},
			},
		}
		assert.Equal(t, exp, result, "GetFuturesSettlementHistory should decode every field")
	} else {
		assert.NotEmpty(t, result, "GetFuturesSettlementHistory should return settlements")
	}
}

func TestGetFundingRate(t *testing.T) {
	t.Parallel()
	_, err := e.GetFundingRate(t.Context(), "")
	require.ErrorIs(t, err, errMissingInstrumentID, "GetFundingRate must require an instrument ID")

	t.Run("one instrument", func(t *testing.T) {
		t.Parallel()
		result, err := e.GetFundingRate(contextGenerate(), perpetualSwapPair.String())
		require.NoError(t, err, "GetFundingRate must not error")
		if mockTests {
			exp := []FundingRateResponse{
				{
					InstrumentType:        "SWAP",
					InstrumentID:          "BTC-USDT-SWAP",
					FundingRateMethod:     "current_period",
					FormulaType:           "withRate",
					FundingRate:           0.0000275120739438,
					FundingTime:           types.Time(time.UnixMilli(1795420800000)),
					NextFundingTime:       types.Time(time.UnixMilli(1795449600000)),
					PreviousFundingTime:   types.Time(time.UnixMilli(1795392000000)),
					MinFundingRate:        -0.00375,
					MaxFundingRate:        0.00375,
					InterestRate:          0.0001000000000000,
					ImpactValue:           20000.0000000000000000,
					SettlementState:       "settled",
					SettlementFundingRate: 0.0000151578247325,
					Premium:               -0.0003085004105178,
					Timestamp:             types.Time(time.UnixMilli(1795406714475)),
				},
			}
			assert.Equal(t, exp, result, "GetFundingRate should decode every field")
		} else {
			require.Len(t, result, 1, "GetFundingRate must return the instrument's funding rate")
			assert.Equal(t, perpetualSwapPair.String(), result[0].InstrumentID, "GetFundingRate should return the instrument's funding rate")
		}
	})
	t.Run("every instrument", func(t *testing.T) {
		t.Parallel()
		result, err := e.GetFundingRate(contextGenerate(), "ANY")
		require.NoError(t, err, "GetFundingRate must not error")
		if mockTests {
			exp := []FundingRateResponse{
				{
					InstrumentType:        "SWAP",
					InstrumentID:          "CHIP-USDT-SWAP",
					FundingRateMethod:     "current_period",
					FormulaType:           "withRate",
					FundingRate:           -0.0000157021991588,
					FundingTime:           types.Time(time.UnixMilli(1795420800000)),
					NextFundingTime:       types.Time(time.UnixMilli(1795435200000)),
					PreviousFundingTime:   types.Time(time.UnixMilli(1795406400000)),
					MinFundingRate:        -0.01,
					MaxFundingRate:        0.01,
					InterestRate:          0.0001000000000000,
					ImpactValue:           4000.0000000000000000,
					SettlementState:       "settled",
					SettlementFundingRate: -0.0000067744919258,
					Premium:               -0.0000009999990007,
					Timestamp:             types.Time(time.UnixMilli(1795406713372)),
				},
				{
					InstrumentType:        "SWAP",
					InstrumentID:          "OPG-USDT-SWAP",
					FundingRateMethod:     "current_period",
					FormulaType:           "withRate",
					FundingRate:           0.0000500000000000,
					FundingTime:           types.Time(time.UnixMilli(1795420800000)),
					NextFundingTime:       types.Time(time.UnixMilli(1795435200000)),
					PreviousFundingTime:   types.Time(time.UnixMilli(1795406400000)),
					MinFundingRate:        -0.01,
					MaxFundingRate:        0.01,
					InterestRate:          0.0001000000000000,
					ImpactValue:           2000.0000000000000000,
					SettlementState:       "settled",
					SettlementFundingRate: 0.0000500000000000,
					Timestamp:             types.Time(time.UnixMilli(1795406713472)),
				},
			}
			assert.Equal(t, exp, result, "GetFundingRate should decode every field")
		} else {
			assert.Greater(t, len(result), 1, "GetFundingRate should return every instrument's funding rate")
		}
	})
}

func TestGetFundingRateHistory(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *FundingRateHistoryRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "missing instrument ID", arg: &FundingRateHistoryRequest{Limit: 2}, err: errMissingInstrumentID},
		{name: "start after end", arg: &FundingRateHistoryRequest{InstrumentID: perpetualSwapPair.String(), Before: time.UnixMilli(1795392000000), After: time.UnixMilli(1795363200000)}, err: common.ErrStartAfterEnd},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.GetFundingRateHistory(t.Context(), tc.arg)
			assert.ErrorIs(t, err, tc.err, "GetFundingRateHistory should reject an invalid request")
		})
	}

	result, err := e.GetFundingRateHistory(contextGenerate(), &FundingRateHistoryRequest{InstrumentID: perpetualSwapPair.String(), Limit: 2})
	require.NoError(t, err, "GetFundingRateHistory must not error")
	if mockTests {
		exp := []FundingRateResponse{
			{
				InstrumentType:    "SWAP",
				InstrumentID:      "BTC-USDT-SWAP",
				FundingRateMethod: "current_period",
				FormulaType:       "withRate",
				FundingRate:       0.0000151578247325,
				RealisedRate:      new(types.Number(0.0000151578247325)),
				FundingTime:       types.Time(time.UnixMilli(1795392000000)),
			},
			{
				InstrumentType:    "SWAP",
				InstrumentID:      "BTC-USDT-SWAP",
				FundingRateMethod: "current_period",
				FormulaType:       "withRate",
				FundingRate:       0.0000227750962068,
				RealisedRate:      new(types.Number(0.0000227750962068)),
				FundingTime:       types.Time(time.UnixMilli(1795363200000)),
			},
		}
		assert.Equal(t, exp, result, "GetFundingRateHistory should decode every field")
	} else {
		assert.NotEmpty(t, result, "GetFundingRateHistory should return funding rates")
	}
}

func TestGetOpenInterestData(t *testing.T) {
	t.Parallel()
	_, err := e.GetOpenInterestData(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetOpenInterestData must reject a nil request")
	_, err = e.GetOpenInterestData(t.Context(), &OpenInterestRequest{})
	require.ErrorIs(t, err, errInvalidInstrumentType, "GetOpenInterestData must require an instrument type")
	_, err = e.GetOpenInterestData(t.Context(), &OpenInterestRequest{InstrumentType: instTypeOption})
	require.ErrorIs(t, err, errInstrumentFamilyRequired, "GetOpenInterestData must require an option instrument family")

	testexch.UpdatePairsOnce(t, e)
	p, err := e.GetAvailablePairs(asset.Options)
	require.NoError(t, err, "GetAvailablePairs must not error")
	require.NotEmpty(t, p, "GetAvailablePairs must not return empty pairs")

	instrumentID := p[0].String()
	// instFamily is the documented open-interest filter; an option
	// instrument resolves to its family, e.g. BTC-USD_UM for
	// BTC-USD_UM-260928-79000-C.
	family, err := e.instrumentFamilyFromInstID(instTypeOption, instrumentID)
	require.NoError(t, err, "instrumentFamilyFromInstID must not error")

	result, err := e.GetOpenInterestData(contextGenerate(), &OpenInterestRequest{InstrumentType: instTypeOption, InstrumentFamily: family, InstrumentID: instrumentID})
	require.NoError(t, err, "GetOpenInterestData must not error")
	if mockTests {
		exp := []OpenInterestResponse{
			{
				InstrumentType: "OPTION",
				InstrumentID:   "SOL-USD_UM-261123-54-C",
				Timestamp:      types.Time(time.UnixMilli(1795406812300)),
			},
		}
		assert.Equal(t, exp, result, "GetOpenInterestData should decode every field")
	} else {
		assert.NotEmpty(t, result, "GetOpenInterestData should return the instrument's open interest")
	}
}

func TestGetLimitPrice(t *testing.T) {
	t.Parallel()
	_, err := e.GetLimitPrice(t.Context(), "")
	require.ErrorIs(t, err, errMissingInstrumentID, "GetLimitPrice must require an instrument ID")

	result, err := e.GetLimitPrice(contextGenerate(), perpetualSwapPair.String())
	require.NoError(t, err, "GetLimitPrice must not error")
	require.NotNil(t, result, "GetLimitPrice must return limits")
	if mockTests {
		exp := &LimitPriceResponse{
			InstrumentType: "SWAP",
			InstrumentID:   "BTC-USDT-SWAP",
			BuyLimit:       61491.8,
			SellLimit:      60879.6,
			Timestamp:      types.Time(time.UnixMilli(1795406818499)),
			Enabled:        true,
		}
		assert.Equal(t, exp, result, "GetLimitPrice should decode every field")
	} else {
		assert.Equal(t, perpetualSwapPair.String(), result.InstrumentID, "GetLimitPrice should return the instrument's limits")
	}
}

func TestGetOptionMarketData(t *testing.T) {
	t.Parallel()
	_, err := e.GetOptionMarketData(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetOptionMarketData must reject a nil request")
	_, err = e.GetOptionMarketData(t.Context(), &OptionMarketDataRequest{})
	require.ErrorIs(t, err, errInstrumentFamilyRequired, "GetOptionMarketData must require an instrument family")

	t.Run("every expiry", func(t *testing.T) {
		t.Parallel()
		result, err := e.GetOptionMarketData(contextGenerate(), &OptionMarketDataRequest{InstrumentFamily: "BTC-USD"})
		require.NoError(t, err, "GetOptionMarketData must not error")
		if mockTests {
			exp := []OptionMarketDataResponse{
				{
					InstrumentType:       "OPTION",
					InstrumentID:         "BTC-USD-261125-63000-C",
					Underlying:           "BTC-USD",
					Delta:                0.25195559929273303,
					Gamma:                6.7239483300038385,
					Vega:                 0.0002490362691627403,
					Theta:                -0.0033503791288932525,
					DeltaBlackScholes:    0.25879024026881686,
					GammaBlackScholes:    0.00011823809913330682,
					VegaBlackScholes:     15.223512423037562,
					ThetaBlackScholes:    -205.98740329486088,
					Leverage:             146.3134645256806,
					MarkVolatility:       0.5817060192671589,
					BidVolatility:        0.5688565185546874,
					AskVolatility:        0.5883877294921874,
					AtTheMoneyVolatility: 0.6018645032253356,
					ForwardPrice:         61135.723661352621895,
					BuyAPR:               1.09710983,
					SellAPR:              1.18150289,
					Distance:             0.03049406,
					Timestamp:            types.Time(time.UnixMilli(1795406809633)),
				},
				{
					InstrumentType:       "OPTION",
					InstrumentID:         "BTC-USD-261125-63000-P",
					Underlying:           "BTC-USD",
					Delta:                -0.778469354971949,
					Gamma:                8.7847982385332,
					Vega:                 0.0002490362691627403,
					Theta:                -0.0033503791288932525,
					DeltaBlackScholes:    -0.7412097597311831,
					GammaBlackScholes:    0.00011823809913330682,
					VegaBlackScholes:     15.223512423037562,
					ThetaBlackScholes:    -201.1626422544119,
					Leverage:             26.838724187371078,
					MarkVolatility:       0.5817060192671589,
					BidVolatility:        0.5505460083007812,
					AskVolatility:        0.5908291308593749,
					AtTheMoneyVolatility: 0.6018645032253356,
					ForwardPrice:         61135.723661352621895,
					Distance:             0.03049406,
					Timestamp:            types.Time(time.UnixMilli(1795406809633)),
				},
			}
			assert.Equal(t, exp, result, "GetOptionMarketData should decode every field")
		} else {
			assert.NotEmpty(t, result, "GetOptionMarketData should return market data")
		}
	})
	t.Run("one expiry", func(t *testing.T) {
		t.Parallel()
		expiry := time.Date(2026, time.November, 27, 8, 0, 0, 0, time.UTC)
		if !mockTests {
			instruments, err := e.GetInstruments(contextGenerate(), &InstrumentsRequest{InstrumentType: instTypeOption, InstrumentFamily: "BTC-USD"})
			require.NoError(t, err, "GetInstruments must not error")
			expiry = nextListedExpiry(instruments, time.Now())
			require.NotZero(t, expiry, "GetInstruments must return a later expiry whose instruments are all live and listed")
		}
		result, err := e.GetOptionMarketData(contextGenerate(), &OptionMarketDataRequest{InstrumentFamily: "BTC-USD", ExpiryTime: expiry})
		require.NoError(t, err, "GetOptionMarketData must not error")
		if mockTests {
			exp := []OptionMarketDataResponse{
				{
					InstrumentType:       "OPTION",
					InstrumentID:         "BTC-USD-261127-67000-C",
					Underlying:           "BTC-USD",
					Delta:                0.06053753405052682,
					Gamma:                1.9780059873455549,
					Vega:                 0.00013062012342175133,
					Theta:                -0.000856364382043697,
					DeltaBlackScholes:    0.06206962309178092,
					GammaBlackScholes:    0.00003433472785907126,
					VegaBlackScholes:     7.985565741487505,
					ThetaBlackScholes:    -52.614621360065236,
					Leverage:             652.7035786258936,
					MarkVolatility:       0.5457280272846157,
					BidVolatility:        0.5420011035156249,
					AskVolatility:        0.551766708984375,
					AtTheMoneyVolatility: 0.5482844270371727,
					ForwardPrice:         61144.28447732440451,
					BuyAPR:               0.13153153,
					SellAPR:              0.1403003,
					Distance:             0.09576881,
					Timestamp:            types.Time(time.UnixMilli(1795406807595)),
				},
				{
					InstrumentType:       "OPTION",
					InstrumentID:         "BTC-USD-261127-67000-P",
					Underlying:           "BTC-USD",
					Delta:                -1.0350612846713882,
					Gamma:                4.169203624789385,
					Vega:                 0.00013062012342175133,
					Theta:                -0.0008563316321950913,
					DeltaBlackScholes:    -0.9379289511447764,
					GammaBlackScholes:    0.00003433472785907126,
					VegaBlackScholes:     7.985565741487505,
					ThetaBlackScholes:    -47.90537419119797,
					Leverage:             10.295232943477316,
					MarkVolatility:       0.5457280272846157,
					BidVolatility:        0.429696640625,
					AskVolatility:        0.5712979199218748,
					AtTheMoneyVolatility: 0.5482844270371727,
					ForwardPrice:         61144.28447732440451,
					Distance:             0.09576881,
					Timestamp:            types.Time(time.UnixMilli(1795406807595)),
				},
			}
			assert.Equal(t, exp, result, "GetOptionMarketData should decode every field")
		} else {
			assert.NotEmpty(t, result, "GetOptionMarketData should return the expiry's market data")
		}
	})
}

func TestGetDiscountRateAndInterestFreeQuota(t *testing.T) {
	t.Parallel()
	_, err := e.GetDiscountRateAndInterestFreeQuota(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetDiscountRateAndInterestFreeQuota must reject a nil request")

	result, err := e.GetDiscountRateAndInterestFreeQuota(contextGenerate(), &DiscountRateRequest{})
	require.NoError(t, err, "GetDiscountRateAndInterestFreeQuota must not error")
	if mockTests {
		exp := []DiscountRateResponse{
			{
				Currency: currency.NewCode("ANTHROPIC"),
				Details:  []DiscountRateDetail{},
			},
			{
				Currency:          currency.BTC,
				InterestFreeQuota: 0.23,
				MinDiscountRate:   0.8,
				Details: []DiscountRateDetail{
					{
						DiscountRate:           0.98,
						MaxAmount:              20,
						Tier:                   "1",
						LiquidationPenaltyRate: 0.02,
						DiscountCurrencyEquity: 19.6,
					},
					{
						DiscountRate:           0.9775,
						MaxAmount:              25,
						MinAmount:              20,
						Tier:                   "2",
						LiquidationPenaltyRate: 0.0225,
						DiscountCurrencyEquity: 24.4875,
					},
				},
			},
		}
		assert.Equal(t, exp, result, "GetDiscountRateAndInterestFreeQuota should decode every field")
	} else {
		assert.NotEmpty(t, result, "GetDiscountRateAndInterestFreeQuota should return discount rates")
	}
}

func TestGetSystemTime(t *testing.T) {
	t.Parallel()
	t.Run("missing server time", func(t *testing.T) {
		t.Parallel()
		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeOKXData(t, w, nil)
		}))
		_, err := ex.GetSystemTime(t.Context())
		assert.ErrorIs(t, err, common.ErrNoResponse, "GetSystemTime should reject a reply without a server time")
	})

	result, err := e.GetSystemTime(contextGenerate())
	require.NoError(t, err, "GetSystemTime must not error")
	if mockTests {
		assert.Equal(t, time.UnixMilli(1795406820204), result, "GetSystemTime should decode the server time")
	} else {
		assert.WithinDuration(t, time.Now(), result, time.Minute, "GetSystemTime should return the current time")
	}
}

func TestGetMarkPrice(t *testing.T) {
	t.Parallel()
	_, err := e.GetMarkPrice(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetMarkPrice must reject a nil request")
	_, err = e.GetMarkPrice(t.Context(), &MarkPriceRequest{InstrumentID: mainPair.String()})
	require.ErrorIs(t, err, errInvalidInstrumentType, "GetMarkPrice must require an instrument type")

	result, err := e.GetMarkPrice(contextGenerate(), &MarkPriceRequest{InstrumentType: instTypeMargin})
	require.NoError(t, err, "GetMarkPrice must not error")
	if mockTests {
		exp := []MarkPriceResponse{
			{
				InstrumentType: "MARGIN",
				InstrumentID:   "ADA-USDT",
				MarkPrice:      0.1444,
				Timestamp:      types.Time(time.UnixMilli(1795406779898)),
			},
			{
				InstrumentType: "MARGIN",
				InstrumentID:   "BTC-USDT",
				MarkPrice:      61227.2,
				Timestamp:      types.Time(time.UnixMilli(1795406779898)),
			},
		}
		assert.Equal(t, exp, result, "GetMarkPrice should decode every field")
	} else {
		assert.NotEmpty(t, result, "GetMarkPrice should return mark prices")
	}
}

func TestGetPositionTiers(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *PositionTiersRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "missing instrument type", arg: &PositionTiersRequest{TradeMode: TradeModeCross, Currency: currency.ETH}, err: errInvalidInstrumentType},
		{name: "missing trade mode", arg: &PositionTiersRequest{InstrumentType: instTypeFutures, InstrumentFamilies: []string{futuresPair.String()}, Currency: currency.ETH}, err: errInvalidTradeMode},
		{name: "derivative without instrument family", arg: &PositionTiersRequest{InstrumentType: instTypeFutures, TradeMode: TradeModeCross}, err: errInstrumentFamilyRequired},
		{name: "margin without instrument ID or currency", arg: &PositionTiersRequest{InstrumentType: instTypeMargin, TradeMode: TradeModeCross}, err: errEitherInstIDOrCcyIsRequired},
		{name: "too many instrument families", arg: &PositionTiersRequest{InstrumentType: instTypeSwap, TradeMode: TradeModeCross, InstrumentFamilies: []string{"BTC-USDT", "ETH-USDT", "SOL-USDT", "XRP-USDT", "DOGE-USDT", "LTC-USDT"}}, err: errExceedLimit},
		{name: "too many instrument IDs", arg: &PositionTiersRequest{InstrumentType: instTypeMargin, TradeMode: TradeModeCross, InstrumentIDs: []string{"BTC-USDT", "ETH-USDT", "SOL-USDT", "XRP-USDT", "DOGE-USDT", "LTC-USDT"}}, err: errExceedLimit},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.GetPositionTiers(t.Context(), tc.arg)
			assert.ErrorIs(t, err, tc.err, "GetPositionTiers should reject an invalid request")
		})
	}

	t.Run("futures", func(t *testing.T) {
		t.Parallel()
		result, err := e.GetPositionTiers(contextGenerate(), &PositionTiersRequest{InstrumentType: instTypeFutures, TradeMode: TradeModeCross, InstrumentFamilies: []string{futuresPair.String()}, Currency: currency.ETH})
		require.NoError(t, err, "GetPositionTiers must not error")
		if mockTests {
			exp := []PositionTierResponse{
				{
					Underlying:                   "BTC-USD",
					InstrumentFamily:             "BTC-USD",
					Tier:                         "1",
					MaxSize:                      12000,
					MaintenanceMarginRequirement: 0.0065,
					InitialMarginRequirement:     0.05,
					MaxLeverage:                  20,
				},
				{
					Underlying:                   "BTC-USD",
					InstrumentFamily:             "BTC-USD",
					Tier:                         "2",
					MinSize:                      12000.1,
					MaxSize:                      15000,
					MaintenanceMarginRequirement: 0.01,
					InitialMarginRequirement:     0.0666,
					MaxLeverage:                  15,
				},
			}
			assert.Equal(t, exp, result, "GetPositionTiers should decode every field")
		} else {
			assert.NotEmpty(t, result, "GetPositionTiers should return tiers")
		}
	})
	t.Run("margin", func(t *testing.T) {
		t.Parallel()
		result, err := e.GetPositionTiers(contextGenerate(), &PositionTiersRequest{InstrumentType: instTypeMargin, TradeMode: TradeModeCross, InstrumentIDs: []string{mainPair.String()}, Tier: 1})
		require.NoError(t, err, "GetPositionTiers must not error")
		if mockTests {
			exp := []PositionTierResponse{
				{
					InstrumentID:                 "BTC-USDT",
					Tier:                         "1",
					MaxSize:                      20,
					MaintenanceMarginRequirement: 0.02,
					InitialMarginRequirement:     0.1,
					MaxLeverage:                  10,
					QuoteMaxLoan:                 1600000,
					BaseMaxLoan:                  20,
				},
			}
			assert.Equal(t, exp, result, "GetPositionTiers should decode every field")
		} else {
			assert.NotEmpty(t, result, "GetPositionTiers should return tiers")
		}
	})
}

func TestGetInterestRateAndLoanQuota(t *testing.T) {
	t.Parallel()
	result, err := e.GetInterestRateAndLoanQuota(contextGenerate())
	require.NoError(t, err, "GetInterestRateAndLoanQuota must not error")
	require.NotNil(t, result, "GetInterestRateAndLoanQuota must return quotas")
	if mockTests {
		exp := &InterestRateLoanQuotaResponse{
			Basic: []InterestRateLoanQuotaBasic{
				{
					Currency:     currency.USDT,
					InterestRate: 0.000096,
					LoanQuota:    5000000,
				},
				{
					Currency:     currency.BTC,
					InterestRate: 0.00001392,
					LoanQuota:    175,
				},
			},
			VIP: []InterestRateLoanQuotaLevel{
				{
					UserLevel:            "VIP 1",
					LoanQuotaCoefficient: 5,
				},
				{
					UserLevel:            "VIP 2",
					LoanQuotaCoefficient: 6,
				},
			},
			Regular: []InterestRateLoanQuotaLevel{
				{
					UserLevel:            "Lv1",
					LoanQuotaCoefficient: 1,
				},
				{
					UserLevel:            "Lv2",
					LoanQuotaCoefficient: 2,
				},
			},
			ConfiguredCurrencyList: []InterestRateLoanQuotaCurrency{
				{
					Currency:     currency.USDT,
					InterestRate: 0.000096,
				},
				{
					Currency:     currency.USDC,
					InterestRate: 0.00010968,
				},
			},
			Configurations: []InterestRateLoanQuotaConfig{
				{
					Currency:     currency.USDT,
					StrategyType: "1",
					UserLevel:    "Lv1",
				},
				{
					Currency:     currency.USDT,
					StrategyType: "1",
					UserLevel:    "Lv2",
				},
			},
		}
		assert.Equal(t, exp, result, "GetInterestRateAndLoanQuota should decode every field")
	} else {
		assert.NotEmpty(t, result.Basic, "GetInterestRateAndLoanQuota should return basic rates")
	}
}

func TestGetPublicUnderlyings(t *testing.T) {
	t.Parallel()
	_, err := e.GetPublicUnderlyings(t.Context(), "")
	require.ErrorIs(t, err, errInvalidInstrumentType, "GetPublicUnderlyings must require an instrument type")

	result, err := e.GetPublicUnderlyings(contextGenerate(), instTypeFutures)
	require.NoError(t, err, "GetPublicUnderlyings must not error")
	if mockTests {
		assert.Equal(t, []string{"ENA-USD", "PENG-USD", "USAR-USD"}, result, "GetPublicUnderlyings should decode every underlying")
	} else {
		assert.NotEmpty(t, result, "GetPublicUnderlyings should return underlyings")
	}
}

func TestGetInsuranceFundInformation(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *InsuranceFundRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "empty request", arg: &InsuranceFundRequest{}, err: errInvalidInstrumentType},
		{name: "missing instrument type", arg: &InsuranceFundRequest{Limit: 2}, err: errInvalidInstrumentType},
		{name: "derivative without instrument family", arg: &InsuranceFundRequest{InstrumentType: instTypeSwap, Limit: 2}, err: errInstrumentFamilyRequired},
		{name: "margin without currency", arg: &InsuranceFundRequest{InstrumentType: instTypeMargin, Limit: 2}, err: currency.ErrCurrencyCodeEmpty},
		{name: "start after end", arg: &InsuranceFundRequest{InstrumentType: instTypeSwap, InstrumentFamily: mainPair.String(), Before: time.UnixMilli(1795404858000), After: time.UnixMilli(1795404798000)}, err: common.ErrStartAfterEnd},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.GetInsuranceFundInformation(t.Context(), tc.arg)
			assert.ErrorIs(t, err, tc.err, "GetInsuranceFundInformation should reject an invalid request")
		})
	}

	for _, tc := range []struct {
		name string
		arg  *InsuranceFundRequest
		exp  *InsuranceFundResponse
	}{
		{name: "swap", arg: &InsuranceFundRequest{InstrumentType: instTypeSwap, InstrumentFamily: mainPair.String(), Limit: 2}, exp: &InsuranceFundResponse{
			Total:            2613229569.8279,
			InstrumentFamily: "BTC-USDT",
			InstrumentType:   "SWAP",
			Details: []InsuranceFundDetail{
				{
					Balance:   102250125.9065,
					Currency:  currency.USDT,
					Type:      "regular_update",
					Timestamp: types.Time(time.UnixMilli(1795406718000)),
				},
				{
					Balance:   102250125.9065,
					Currency:  currency.USDT,
					Type:      "regular_update",
					Timestamp: types.Time(time.UnixMilli(1795406658000)),
				},
			},
		}},
		// The underlying BTC-USD spans several futures families, each answered separately, so name one family
		{name: "futures", arg: &InsuranceFundRequest{InstrumentType: instTypeFutures, InstrumentFamily: futuresPair.String(), Limit: 2}, exp: &InsuranceFundResponse{
			Total:            2613229569.8279,
			InstrumentFamily: "BTC-USD",
			InstrumentType:   "FUTURES",
			Details: []InsuranceFundDetail{
				{
					Balance:   460.4855,
					Currency:  currency.BTC,
					Type:      "regular_update",
					Timestamp: types.Time(time.UnixMilli(1795406716000)),
				},
				{
					Balance:   460.4855,
					Currency:  currency.BTC,
					Type:      "regular_update",
					Timestamp: types.Time(time.UnixMilli(1795406655000)),
				},
			},
		}},
		{name: "margin", arg: &InsuranceFundRequest{InstrumentType: instTypeMargin, Currency: currency.BTC, Limit: 2}, exp: &InsuranceFundResponse{
			Total:          2613229569.8279,
			InstrumentType: "MARGIN",
			Details: []InsuranceFundDetail{
				{
					Balance:   46.3607,
					Currency:  currency.BTC,
					Type:      "regular_update",
					Timestamp: types.Time(time.UnixMilli(1795406725000)),
				},
				{
					Balance:   46.3607,
					Currency:  currency.BTC,
					Type:      "regular_update",
					Timestamp: types.Time(time.UnixMilli(1795406665000)),
				},
			},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			r, err := e.GetInsuranceFundInformation(contextGenerate(), tc.arg)
			require.NoError(t, err, "GetInsuranceFundInformation must not error")
			require.NotNil(t, r, "GetInsuranceFundInformation must return the fund")
			if mockTests {
				assert.Equal(t, tc.exp, r, "GetInsuranceFundInformation should decode every field")
				return
			}
			assert.Positive(t, r.Total, "Total should be positive")
			assert.NotEmpty(t, r.Details, "Details should not be empty")
			for _, d := range r.Details {
				assert.Positive(t, d.Balance, "Balance should be positive")
				assert.NotEmpty(t, d.Type, "Type should not be empty")
				assert.Positive(t, d.Timestamp, "Timestamp should be positive")
			}
		})
	}
}

func TestCurrencyUnitConvert(t *testing.T) {
	t.Parallel()
	_, err := e.CurrencyUnitConvert(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CurrencyUnitConvert must reject a nil request")
	_, err = e.CurrencyUnitConvert(t.Context(), &UnitConvertRequest{ConvertType: "1", Size: 1, Price: 3500})
	require.ErrorIs(t, err, errMissingInstrumentID, "CurrencyUnitConvert must require an instrument ID")
	_, err = e.CurrencyUnitConvert(t.Context(), &UnitConvertRequest{ConvertType: "1", InstrumentID: perpetualSwapPair.String(), Price: 3500})
	require.ErrorIs(t, err, errMissingQuantity, "CurrencyUnitConvert must require a size")

	result, err := e.CurrencyUnitConvert(contextGenerate(), &UnitConvertRequest{ConvertType: "1", InstrumentID: perpetualSwapPair.String(), Size: 1, Price: 3500, Unit: "coin", OperationType: "close"})
	require.NoError(t, err, "CurrencyUnitConvert must not error")
	require.NotNil(t, result, "CurrencyUnitConvert must return a conversion")
	if mockTests {
		exp := &UnitConvertResponse{
			ConvertType:  "1",
			InstrumentID: "BTC-USDT-SWAP",
			Price:        3500,
			Size:         100,
			Unit:         "coin",
		}
		assert.Equal(t, exp, result, "CurrencyUnitConvert should decode every field")
	} else {
		assert.Positive(t, result.Size.Float64(), "CurrencyUnitConvert should return a converted size")
	}
}

func TestGetInstrumentTickBands(t *testing.T) {
	t.Parallel()
	_, err := e.GetInstrumentTickBands(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetInstrumentTickBands must reject a nil request")
	_, err = e.GetInstrumentTickBands(t.Context(), &InstrumentTickBandsRequest{})
	require.ErrorIs(t, err, errInvalidInstrumentType, "GetInstrumentTickBands must require an instrument type")

	for _, tc := range []struct {
		name string
		arg  *InstrumentTickBandsRequest
		exp  []InstrumentTickBandResponse
	}{
		{name: "options", arg: &InstrumentTickBandsRequest{InstrumentType: instTypeOption}, exp: []InstrumentTickBandResponse{
			{
				InstrumentType:   "OPTION",
				InstrumentFamily: "BTC-USD_UM",
				TickBands: []TickBand{
					{
						MaxPrice: 5,
						TickSize: 5,
					},
					{
						MinPrice: 5,
						MaxPrice: 300000,
						TickSize: 5,
					},
				},
			},
			{
				InstrumentType:   "OPTION",
				InstrumentFamily: "BTC-USD",
				TickBands: []TickBand{
					{
						MaxPrice: 0.005,
						TickSize: 0.0001,
					},
					{
						MinPrice: 0.005,
						MaxPrice: 10000000,
						TickSize: 0.0005,
					},
				},
			},
		}},
		{name: "event contracts", arg: &InstrumentTickBandsRequest{InstrumentType: instTypeEvents}, exp: []InstrumentTickBandResponse{
			{
				InstrumentType: "EVENTS",
				SeriesID:       "BTC-UPDOWN-15MIN",
				TickBands: []TickBand{
					{
						MaxPrice: 0.04,
						TickSize: 0.001,
					},
					{
						MinPrice: 0.04,
						MaxPrice: 0.96,
						TickSize: 0.01,
					},
				},
			},
			{
				InstrumentType: "EVENTS",
				SeriesID:       "BTC-ABOVE-DAILY",
				TickBands: []TickBand{
					{
						MaxPrice: 0.04,
						TickSize: 0.001,
					},
					{
						MinPrice: 0.04,
						MaxPrice: 0.96,
						TickSize: 0.01,
					},
				},
			},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetInstrumentTickBands(contextGenerate(), tc.arg)
			require.NoError(t, err, "GetInstrumentTickBands must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetInstrumentTickBands should decode every field")
			} else {
				assert.NotEmpty(t, result, "GetInstrumentTickBands should return tick bands")
			}
		})
	}
}

func TestGetPremiumHistory(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *PremiumHistoryRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "missing instrument ID", arg: &PremiumHistoryRequest{Limit: 10}, err: errMissingInstrumentID},
		{name: "start after end", arg: &PremiumHistoryRequest{InstrumentID: perpetualSwapPair.String(), Before: time.UnixMilli(1795404861000), After: time.UnixMilli(1795404803000)}, err: common.ErrStartAfterEnd},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.GetPremiumHistory(t.Context(), tc.arg)
			assert.ErrorIs(t, err, tc.err, "GetPremiumHistory should reject an invalid request")
		})
	}

	result, err := e.GetPremiumHistory(contextGenerate(), &PremiumHistoryRequest{InstrumentID: perpetualSwapPair.String(), Limit: 10})
	require.NoError(t, err, "GetPremiumHistory must not error")
	if mockTests {
		exp := []PremiumHistoryResponse{
			{
				InstrumentID: "BTC-USDT-SWAP",
				Premium:      -0.0004295399349646,
				Timestamp:    types.Time(time.UnixMilli(1795406778000)),
			},
			{
				InstrumentID: "BTC-USDT-SWAP",
				Premium:      -0.0003085004105178,
				Timestamp:    types.Time(time.UnixMilli(1795406714000)),
			},
		}
		assert.Equal(t, exp, result, "GetPremiumHistory should decode every field")
	} else {
		assert.NotEmpty(t, result, "GetPremiumHistory should return premiums")
	}
}

func TestGetIndexTickers(t *testing.T) {
	t.Parallel()
	_, err := e.GetIndexTickers(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetIndexTickers must reject a nil request")
	_, err = e.GetIndexTickers(t.Context(), &IndexTickersRequest{})
	require.ErrorIs(t, err, errEitherInstIDOrCcyIsRequired, "GetIndexTickers must require a quote currency or index")

	result, err := e.GetIndexTickers(contextGenerate(), &IndexTickersRequest{QuoteCurrency: currency.USDT})
	require.NoError(t, err, "GetIndexTickers must not error")
	if mockTests {
		exp := []IndexTickerResponse{
			{
				InstrumentID:       "BLEND-USDT",
				IndexPrice:         0.04668,
				HighestPrice24Hour: 0.0496,
				LowestPrice24Hour:  0.0462,
				OpenPrice24Hour:    0.04922,
				OpenPriceUTC0:      0.04703,
				OpenPriceUTC8:      0.04794,
				Timestamp:          types.Time(time.UnixMilli(1795406737112)),
			},
			{
				InstrumentID:       "XPOETPROTECT-USDT",
				IndexPrice:         3.66,
				HighestPrice24Hour: 3.67,
				LowestPrice24Hour:  3.54,
				OpenPrice24Hour:    3.54,
				OpenPriceUTC0:      3.54,
				OpenPriceUTC8:      3.54,
				Timestamp:          types.Time(time.UnixMilli(1795406737103)),
			},
		}
		assert.Equal(t, exp, result, "GetIndexTickers should decode every field")
	} else {
		assert.NotEmpty(t, result, "GetIndexTickers should return index tickers")
	}
}

func TestGetIndexCandlesticks(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *PriceCandlesticksRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "missing index", arg: &PriceCandlesticksRequest{Bar: kline.OneHour, Limit: 2}, err: errMissingInstrumentID},
		{name: "start after end", arg: &PriceCandlesticksRequest{InstrumentID: "BTC-USD", Before: time.UnixMilli(1795402800000), After: time.UnixMilli(1795392000000)}, err: common.ErrStartAfterEnd},
		{name: "unsupported bar", arg: &PriceCandlesticksRequest{InstrumentID: "BTC-USD", Bar: kline.TenMin}, err: kline.ErrUnsupportedInterval},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.GetIndexCandlesticks(t.Context(), tc.arg)
			assert.ErrorIs(t, err, tc.err, "GetIndexCandlesticks should reject an invalid request")
		})
	}

	result, err := e.GetIndexCandlesticks(contextGenerate(), &PriceCandlesticksRequest{InstrumentID: "BTC-USD", Bar: kline.OneHour, Limit: 2})
	require.NoError(t, err, "GetIndexCandlesticks must not error")
	if mockTests {
		exp := []PriceCandlestickResponse{
			{
				OpenTime:     types.Time(time.UnixMilli(1795406400000)),
				OpenPrice:    61192.5,
				HighestPrice: 61228.7,
				LowestPrice:  61113.5,
				ClosePrice:   61162.4,
			},
			{
				OpenTime:     types.Time(time.UnixMilli(1795402800000)),
				OpenPrice:    60518.1,
				HighestPrice: 61857.5,
				LowestPrice:  60506.9,
				ClosePrice:   61192.5,
				Confirmed:    true,
			},
		}
		assert.Equal(t, exp, result, "GetIndexCandlesticks should decode every field")
	} else {
		assert.NotEmpty(t, result, "GetIndexCandlesticks should return candlesticks")
	}
}

func TestGetIndexCandlesticksHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetIndexCandlesticksHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetIndexCandlesticksHistory must reject a nil request")
	_, err = e.GetIndexCandlesticksHistory(t.Context(), &PriceCandlesticksRequest{Bar: kline.TenMin, Limit: 10})
	require.ErrorIs(t, err, errMissingInstrumentID, "GetIndexCandlesticksHistory must require an index")

	for _, tc := range []struct {
		index string
		exp   []PriceCandlestickResponse
	}{
		{index: "BTC-USD", exp: []PriceCandlestickResponse{
			{
				OpenTime:     types.Time(time.UnixMilli(1795406700000)),
				OpenPrice:    61185.6,
				HighestPrice: 61188,
				LowestPrice:  61162.1,
				ClosePrice:   61162.4,
			},
			{
				OpenTime:     types.Time(time.UnixMilli(1795406400000)),
				OpenPrice:    61192.5,
				HighestPrice: 61228.7,
				LowestPrice:  61113.5,
				ClosePrice:   61185.6,
				Confirmed:    true,
			},
		}},
		{index: mainPair.String(), exp: []PriceCandlestickResponse{
			{
				OpenTime:     types.Time(time.UnixMilli(1795406700000)),
				OpenPrice:    61281,
				HighestPrice: 61281,
				LowestPrice:  61255.3,
				ClosePrice:   61255.7,
			},
			{
				OpenTime:     types.Time(time.UnixMilli(1795406400000)),
				OpenPrice:    61286.8,
				HighestPrice: 61320.4,
				LowestPrice:  61201.2,
				ClosePrice:   61281,
				Confirmed:    true,
			},
		}},
	} {
		t.Run(tc.index, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetIndexCandlesticksHistory(contextGenerate(), &PriceCandlesticksRequest{InstrumentID: tc.index, Bar: kline.FiveMin, Limit: 10})
			require.NoError(t, err, "GetIndexCandlesticksHistory must not error")
			require.NotEmpty(t, result, "GetIndexCandlesticksHistory must return candlesticks")
			if mockTests {
				assert.Equal(t, tc.exp, result[:2], "GetIndexCandlesticksHistory should decode every field")
			}
		})
	}
}

func TestGetMarkPriceCandlesticks(t *testing.T) {
	t.Parallel()
	_, err := e.GetMarkPriceCandlesticks(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetMarkPriceCandlesticks must reject a nil request")
	_, err = e.GetMarkPriceCandlesticks(t.Context(), &PriceCandlesticksRequest{Bar: kline.FiveMin, Limit: 10})
	require.ErrorIs(t, err, errMissingInstrumentID, "GetMarkPriceCandlesticks must require an instrument ID")

	result, err := e.GetMarkPriceCandlesticks(contextGenerate(), &PriceCandlesticksRequest{InstrumentID: mainPair.String(), Bar: kline.FiveMin, Limit: 10})
	require.NoError(t, err, "GetMarkPriceCandlesticks must not error")
	require.NotEmpty(t, result, "GetMarkPriceCandlesticks must return candlesticks")
	if mockTests {
		exp := []PriceCandlestickResponse{
			{
				OpenTime:     types.Time(time.UnixMilli(1795406700000)),
				OpenPrice:    61281.4,
				HighestPrice: 61281.4,
				LowestPrice:  61243.3,
				ClosePrice:   61243.3,
			},
			{
				OpenTime:     types.Time(time.UnixMilli(1795406400000)),
				OpenPrice:    61287.6,
				HighestPrice: 61321.2,
				LowestPrice:  61202,
				ClosePrice:   61281.4,
				Confirmed:    true,
			},
		}
		assert.Equal(t, exp, result[:2], "GetMarkPriceCandlesticks should decode every field")
	}
}

func TestGetMarkPriceCandlesticksHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetMarkPriceCandlesticksHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetMarkPriceCandlesticksHistory must reject a nil request")
	_, err = e.GetMarkPriceCandlesticksHistory(t.Context(), &PriceCandlesticksRequest{Bar: kline.FiveMin, Limit: 10})
	require.ErrorIs(t, err, errMissingInstrumentID, "GetMarkPriceCandlesticksHistory must require an instrument ID")

	result, err := e.GetMarkPriceCandlesticksHistory(contextGenerate(), &PriceCandlesticksRequest{InstrumentID: perpetualSwapPair.String(), Bar: kline.FiveMin, Limit: 10})
	require.NoError(t, err, "GetMarkPriceCandlesticksHistory must not error")
	require.NotEmpty(t, result, "GetMarkPriceCandlesticksHistory must return candlesticks")
	if mockTests {
		exp := []PriceCandlestickResponse{
			{
				OpenTime:     types.Time(time.UnixMilli(1795406700000)),
				OpenPrice:    61257.8,
				HighestPrice: 61257.8,
				LowestPrice:  61225.8,
				ClosePrice:   61229.2,
			},
			{
				OpenTime:     types.Time(time.UnixMilli(1795406400000)),
				OpenPrice:    61260.8,
				HighestPrice: 61296.8,
				LowestPrice:  61172.3,
				ClosePrice:   61257.8,
				Confirmed:    true,
			},
		}
		assert.Equal(t, exp, result[:2], "GetMarkPriceCandlesticksHistory should decode every field")
	}
}

func TestGetPriceCandlesticks(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name string
		arg  *PriceCandlesticksRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "missing instrument ID", arg: &PriceCandlesticksRequest{Bar: kline.FiveMin}, err: errMissingInstrumentID},
		{name: "unsupported bar", arg: &PriceCandlesticksRequest{InstrumentID: mainPair.String(), Bar: kline.TenMin}, err: kline.ErrUnsupportedInterval},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.getPriceCandlesticks(t.Context(), "market/mark-price-candles", getMarkPriceCandlesticksEPL, tc.arg)
			assert.ErrorIs(t, err, tc.err, "getPriceCandlesticks should reject the request before sending it")
		})
	}

	t.Run("sends the query to the caller's path", func(t *testing.T) {
		t.Parallel()
		ex := newQueryPinExchange(t, "/market/history-index-candles",
			url.Values{"instId": {"BTC-USD"}, "bar": {"1Dutc"}, "after": {"1795402800000"}, "before": {"1795392000000"}, "limit": {"2"}},
			[][]string{{"1795399200000", "61281.4", "61321.2", "61202", "61243.3", "1"}, {"1795395600000", "61287.6", "61300", "61243.3", "61281.4", "0"}})
		result, err := ex.getPriceCandlesticks(t.Context(), "market/history-index-candles", getIndexCandlesticksHistoryEPL, &PriceCandlesticksRequest{
			InstrumentID: "BTC-USD", Bar: kline.OneDay, After: time.UnixMilli(1795402800000), Before: time.UnixMilli(1795392000000), Limit: 2,
		})
		require.NoError(t, err, "getPriceCandlesticks must not error")
		exp := []PriceCandlestickResponse{
			{OpenTime: types.Time(time.UnixMilli(1795399200000)), OpenPrice: 61281.4, HighestPrice: 61321.2, LowestPrice: 61202, ClosePrice: 61243.3, Confirmed: true},
			{OpenTime: types.Time(time.UnixMilli(1795395600000)), OpenPrice: 61287.6, HighestPrice: 61300, LowestPrice: 61243.3, ClosePrice: 61281.4},
		}
		assert.Equal(t, exp, result, "getPriceCandlesticks should decode every candlestick")
	})

	t.Run("mark price candlesticks", func(t *testing.T) {
		t.Parallel()
		result, err := e.getPriceCandlesticks(contextGenerate(), "market/mark-price-candles", getMarkPriceCandlesticksEPL, &PriceCandlesticksRequest{InstrumentID: mainPair.String(), Bar: kline.FiveMin, Limit: 10})
		require.NoError(t, err, "getPriceCandlesticks must not error")
		require.NotEmpty(t, result, "getPriceCandlesticks must return candlesticks")
		if mockTests {
			exp := PriceCandlestickResponse{OpenTime: types.Time(time.UnixMilli(1795406700000)), OpenPrice: 61281.4, HighestPrice: 61281.4, LowestPrice: 61243.3, ClosePrice: 61243.3}
			assert.Equal(t, exp, result[0], "getPriceCandlesticks should decode the latest candlestick")
		}
	})
}

func TestPriceCandlestickResponseUnmarshalJSON(t *testing.T) {
	t.Parallel()
	t.Run("completed", func(t *testing.T) {
		t.Parallel()
		data := []byte(`[ [ "1597026383085", "3.721", "3.743", "3.677", "3.708", "1" ], [ "1597026383085", "3.731", "3.799", "3.494", "3.72", "1" ]]`)
		var resp []PriceCandlestickResponse
		require.NoError(t, json.Unmarshal(data, &resp), "Unmarshal must not error")
		exp := []PriceCandlestickResponse{
			{OpenTime: types.Time(time.UnixMilli(1597026383085)), OpenPrice: 3.721, HighestPrice: 3.743, LowestPrice: 3.677, ClosePrice: 3.708, Confirmed: true},
			{OpenTime: types.Time(time.UnixMilli(1597026383085)), OpenPrice: 3.731, HighestPrice: 3.799, LowestPrice: 3.494, ClosePrice: 3.72, Confirmed: true},
		}
		assert.Equal(t, exp, resp, "Unmarshal should decode each completed candlestick")
	})
	t.Run("uncompleted", func(t *testing.T) {
		t.Parallel()
		var resp PriceCandlestickResponse
		require.NoError(t, json.Unmarshal([]byte(`["1597026383085","3.731","3.799","3.494","3.72","0"]`), &resp), "Unmarshal must not error")
		exp := PriceCandlestickResponse{OpenTime: types.Time(time.UnixMilli(1597026383085)), OpenPrice: 3.731, HighestPrice: 3.799, LowestPrice: 3.494, ClosePrice: 3.72}
		assert.Equal(t, exp, resp, "Unmarshal should decode an uncompleted candlestick")
	})
}

func TestGetExchangeRate(t *testing.T) {
	t.Parallel()
	result, err := e.GetExchangeRate(contextGenerate())
	require.NoError(t, err, "GetExchangeRate must not error")
	require.NotNil(t, result, "GetExchangeRate must return a rate")
	if mockTests {
		assert.Equal(t, &ExchangeRateResponse{
			USDCNY: 6.664,
		}, result, "GetExchangeRate should decode every field")
	} else {
		assert.Positive(t, result.USDCNY.Float64(), "GetExchangeRate should return a rate")
	}
}

func TestGetIndexComponents(t *testing.T) {
	t.Parallel()
	_, err := e.GetIndexComponents(t.Context(), "")
	require.ErrorIs(t, err, errIndexRequired, "GetIndexComponents must require an index")

	t.Run("missing components", func(t *testing.T) {
		t.Parallel()
		ex := newMockExchange(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			writeOKXData(t, w, nil)
		}))
		_, err := ex.GetIndexComponents(t.Context(), mainPair.String())
		assert.ErrorIs(t, err, errIndexComponentNotFound, "GetIndexComponents should reject a reply without components")
	})

	result, err := e.GetIndexComponents(contextGenerate(), mainPair.String())
	require.NoError(t, err, "GetIndexComponents must not error")
	require.NotNil(t, result, "GetIndexComponents must return components")
	if mockTests {
		exp := &IndexComponentsResponse{
			Index:     "BTC-USDT",
			LastPrice: 61256.7,
			Timestamp: types.Time(time.UnixMilli(1795406738370)),
			Components: []IndexComponent{
				{
					ExchangeName:    "Coinbase",
					Symbol:          "BTC/USD",
					SymbolPairPrice: 61162.5,
					Weight:          0.1875,
					ConvertedPrice:  61252.5412356163560433,
				},
				{
					ExchangeName:    "Bitget",
					Symbol:          "BTC/USDT",
					SymbolPairPrice: 61258.01,
					Weight:          0.125,
					ConvertedPrice:  61258.01,
				},
				{
					ExchangeName:    "Binance",
					Symbol:          "BTC/USDT",
					SymbolPairPrice: 61256.01,
					Weight:          0.25,
					ConvertedPrice:  61256.01,
				},
			},
		}
		assert.Equal(t, exp, result, "GetIndexComponents should decode every field")
	} else {
		assert.NotEmpty(t, result.Index, "Index should not be empty")
		assert.NotEmpty(t, result.Components, "Components should not be empty")
	}
}

func TestGetEconomicCalendarData(t *testing.T) {
	t.Parallel()
	_, err := e.GetEconomicCalendarData(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetEconomicCalendarData must reject a nil request")
	_, err = e.GetEconomicCalendarData(t.Context(), &EconomicCalendarRequest{Before: time.UnixMilli(1700121600000), After: time.UnixMilli(1700035200000)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetEconomicCalendarData must reject a start after its end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetEconomicCalendarData(contextGenerate(), &EconomicCalendarRequest{Before: time.UnixMilli(1700035200000)})
	require.NoError(t, err, "GetEconomicCalendarData must not error")
	if mockTests {
		exp := []EconomicCalendarEventResponse{
			{
				CalendarID:           "330631",
				Date:                 types.Time(time.UnixMilli(1700121600000)),
				Region:               "Slovakia",
				Category:             "Harmonised Inflation Rate YoY",
				Event:                "Harmonised Inflation Rate YoY",
				ReferenceDate:        types.Time(time.UnixMilli(1698710400000)),
				Actual:               "7.8%",
				Previous:             "9%",
				Forecast:             "7.8%",
				DateSpan:             "0",
				Importance:           "1",
				UpdateTime:           types.Time(time.UnixMilli(1700121605007)),
				PreviousInitialValue: "8.9%",
				Currency:             currency.EUR,
				Unit:                 "%",
			},
		}
		assert.Equal(t, exp, result, "GetEconomicCalendarData should decode every field")
	} else {
		assert.NotNil(t, result, "GetEconomicCalendarData should return the calendar")
	}
}

func TestGetHistoricalMarketData(t *testing.T) {
	t.Parallel()
	begin, end := time.UnixMilli(1794758400000), time.UnixMilli(1794844800000)
	tooMany := []string{"1", "2", "3", "4", "5", "6", "7", "8", "9", "10", "11"}
	for _, tc := range []struct {
		name string
		arg  *HistoricalMarketDataRequest
		err  error
	}{
		{name: "nil request", err: common.ErrNilPointer},
		{name: "missing module", arg: &HistoricalMarketDataRequest{InstrumentType: instTypeSpot, InstrumentIDs: []string{mainPair.String()}, DateAggregationType: "daily", Begin: begin, End: end}, err: errModuleRequired},
		{name: "missing instrument type", arg: &HistoricalMarketDataRequest{Module: "1", InstrumentIDs: []string{mainPair.String()}, DateAggregationType: "daily", Begin: begin, End: end}, err: errInvalidInstrumentType},
		{name: "spot without instrument IDs", arg: &HistoricalMarketDataRequest{Module: "1", InstrumentType: instTypeSpot, DateAggregationType: "daily", Begin: begin, End: end}, err: errMissingInstrumentID},
		{name: "derivative without instrument families", arg: &HistoricalMarketDataRequest{Module: "1", InstrumentType: instTypeSwap, DateAggregationType: "daily", Begin: begin, End: end}, err: errInstrumentFamilyRequired},
		{name: "too many instrument IDs", arg: &HistoricalMarketDataRequest{Module: "1", InstrumentType: instTypeSpot, InstrumentIDs: tooMany, DateAggregationType: "daily", Begin: begin, End: end}, err: errExceedLimit},
		{name: "too many instrument families", arg: &HistoricalMarketDataRequest{Module: "1", InstrumentType: instTypeSwap, InstrumentFamilies: tooMany, DateAggregationType: "daily", Begin: begin, End: end}, err: errExceedLimit},
		{name: "missing date aggregation type", arg: &HistoricalMarketDataRequest{Module: "1", InstrumentType: instTypeSpot, InstrumentIDs: []string{mainPair.String()}, Begin: begin, End: end}, err: errDateAggregationTypeRequired},
		{name: "missing begin", arg: &HistoricalMarketDataRequest{Module: "1", InstrumentType: instTypeSpot, InstrumentIDs: []string{mainPair.String()}, DateAggregationType: "daily", End: end}, err: common.ErrDateUnset},
		{name: "missing end", arg: &HistoricalMarketDataRequest{Module: "1", InstrumentType: instTypeSpot, InstrumentIDs: []string{mainPair.String()}, DateAggregationType: "daily", Begin: begin}, err: common.ErrDateUnset},
		{name: "begin after end", arg: &HistoricalMarketDataRequest{Module: "1", InstrumentType: instTypeSpot, InstrumentIDs: []string{mainPair.String()}, DateAggregationType: "daily", Begin: end, End: begin}, err: common.ErrStartAfterEnd},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, err := e.GetHistoricalMarketData(t.Context(), tc.arg)
			assert.ErrorIs(t, err, tc.err, "GetHistoricalMarketData should reject an invalid request")
		})
	}

	if !mockTests {
		end = time.Now().Add(-48 * time.Hour)
		begin = end.Add(-24 * time.Hour)
	}
	result, err := e.GetHistoricalMarketData(contextGenerate(), &HistoricalMarketDataRequest{Module: "1", InstrumentType: instTypeSpot, InstrumentIDs: []string{mainPair.String()}, DateAggregationType: "daily", Begin: begin, End: end})
	require.NoError(t, err, "GetHistoricalMarketData must not error")
	if mockTests {
		exp := []HistoricalMarketDataResponse{
			{
				Timestamp:           types.Time(time.UnixMilli(1795406780813)),
				TotalSizeMB:         8.70,
				DateAggregationType: "daily",
				Details: []HistoricalMarketDataDetail{
					{
						InstrumentID:   "BTC-USDT",
						InstrumentType: "SPOT",
						DateRangeStart: types.Time(time.UnixMilli(1794758400000)),
						DateRangeEnd:   types.Time(time.UnixMilli(1794844800000)),
						GroupSizeMB:    8.70,
						GroupDetails: []HistoricalMarketDataFile{
							{
								Filename: "BTC-USDT-trades-2026-11-17.zip",
								DataTime: types.Time(time.UnixMilli(1794844800000)),
								SizeMB:   4.16,
								URL:      "https://static.okx.com/cdn/okex/traderecords/trades/daily/20261117/BTC-USDT-trades-2026-11-17.zip?v=1001",
							},
							{
								Filename: "BTC-USDT-trades-2026-11-16.zip",
								DataTime: types.Time(time.UnixMilli(1794758400000)),
								SizeMB:   4.54,
								URL:      "https://static.okx.com/cdn/okex/traderecords/trades/daily/20261116/BTC-USDT-trades-2026-11-16.zip?v=1001",
							},
						},
					},
				},
			},
		}
		assert.Equal(t, exp, result, "GetHistoricalMarketData should decode every field")
	} else {
		assert.NotEmpty(t, result, "GetHistoricalMarketData should return the data files")
	}
}

func TestHistoricalMarketDataFileUnmarshalJSON(t *testing.T) {
	t.Parallel()
	recordedTime, earlierTime := types.Time(time.UnixMilli(1794758400000)), types.Time(time.UnixMilli(1794672000000))
	for _, tc := range []struct {
		name     string
		previous types.Time
		payload  string
		exp      HistoricalMarketDataFile
	}{
		{
			name:    "recorded dateTs",
			payload: `{"dateTs":"1794758400000","filename":"BTC-USDT-trades-2026-11-16.zip","sizeMB":"4.54","url":"https://static.okx.com/BTC-USDT-trades-2026-11-16.zip"}`,
			exp:     HistoricalMarketDataFile{Filename: "BTC-USDT-trades-2026-11-16.zip", DataTime: recordedTime, SizeMB: 4.54, URL: "https://static.okx.com/BTC-USDT-trades-2026-11-16.zip"},
		},
		{
			name:    "documented dataTs",
			payload: `{"dataTs":"1794758400000","filename":"BTC-USDT-trades-2026-11-16.zip","sizeMB":"4.54","url":"https://static.okx.com/BTC-USDT-trades-2026-11-16.zip"}`,
			exp:     HistoricalMarketDataFile{Filename: "BTC-USDT-trades-2026-11-16.zip", DataTime: recordedTime, SizeMB: 4.54, URL: "https://static.okx.com/BTC-USDT-trades-2026-11-16.zip"},
		},
		{name: "recorded spelling first", payload: `{"dataTs":"1794672000000","dateTs":"1794758400000"}`, exp: HistoricalMarketDataFile{DataTime: recordedTime}},
		{name: "empty recorded spelling", payload: `{"dateTs":"","dataTs":"1794672000000"}`, exp: HistoricalMarketDataFile{DataTime: earlierTime}},
		{name: "documented spelling replaces an earlier time", previous: recordedTime, payload: `{"dataTs":"1794672000000"}`, exp: HistoricalMarketDataFile{DataTime: earlierTime}},
		{name: "neither spelling", previous: recordedTime, payload: `{"filename":"BTC-USDT-trades-2026-11-16.zip"}`, exp: HistoricalMarketDataFile{Filename: "BTC-USDT-trades-2026-11-16.zip", DataTime: recordedTime}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			file := HistoricalMarketDataFile{DataTime: tc.previous}
			require.NoError(t, json.Unmarshal([]byte(tc.payload), &file), "Unmarshal must not error")
			assert.Equal(t, tc.exp, file, "Unmarshal should decode the file and its timestamp")
		})
	}
	t.Run("invalid documented timestamp", func(t *testing.T) {
		t.Parallel()
		var file HistoricalMarketDataFile
		assert.Error(t, json.Unmarshal([]byte(`{"dataTs":"yesterday"}`), &file), "Unmarshal should reject an invalid timestamp")
	})
}

func TestGetMMInstrumentTypes(t *testing.T) {
	t.Parallel()
	_, err := e.GetMMInstrumentTypes(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetMMInstrumentTypes must reject a nil request")

	result, err := e.GetMMInstrumentTypes(contextGenerate(), &MMInstrumentTypesRequest{InstrumentType: instTypeSpot, InstrumentID: mainPair.String()})
	require.NoError(t, err, "GetMMInstrumentTypes must not error")
	if mockTests {
		exp := []MMInstrumentTypeResponse{
			{
				InstrumentID:   "BTC-USDT",
				InstrumentType: "SPOT",
				PairType:       "Type A",
			},
		}
		assert.Equal(t, exp, result, "GetMMInstrumentTypes should decode every field")
	} else {
		assert.NotEmpty(t, result, "GetMMInstrumentTypes should return the classification")
	}
}

func TestGetDeltaHedgeCurrencies(t *testing.T) {
	t.Parallel()
	_, err := e.GetDeltaHedgeCurrencies(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetDeltaHedgeCurrencies must reject a nil request")

	result, err := e.GetDeltaHedgeCurrencies(contextGenerate(), &DeltaHedgeCurrenciesRequest{Currency: currency.ETH})
	require.NoError(t, err, "GetDeltaHedgeCurrencies must not error")
	if mockTests {
		exp := []DeltaHedgeCurrencyResponse{
			{
				Currency:        currency.ETH,
				HedgeCurrencies: []currency.Code{currency.NewCode("BETH")},
			},
		}
		assert.Equal(t, exp, result, "GetDeltaHedgeCurrencies should decode every field")
	} else {
		assert.NotEmpty(t, result, "GetDeltaHedgeCurrencies should return the hedge currencies")
	}
}

func TestSetTimeWindowParams(t *testing.T) {
	t.Parallel()
	older, newer := time.UnixMilli(1795392000000), time.UnixMilli(1795402800000)
	for _, tc := range []struct {
		name          string
		before, after time.Time
		exp           url.Values
	}{
		{name: "no bounds", exp: url.Values{"instId": {"BTC-USDT"}}},
		{name: "before bound", before: older, exp: url.Values{"instId": {"BTC-USDT"}, "before": {"1795392000000"}}},
		{name: "after bound", after: newer, exp: url.Values{"instId": {"BTC-USDT"}, "after": {"1795402800000"}}},
		{name: "both bounds", before: older, after: newer, exp: url.Values{"instId": {"BTC-USDT"}, "before": {"1795392000000"}, "after": {"1795402800000"}}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			params := url.Values{"instId": {"BTC-USDT"}}
			setTimeWindowParams(params, tc.before, tc.after)
			assert.Equal(t, tc.exp, params, "setTimeWindowParams should add each set bound in milliseconds and keep the other parameters")
		})
	}
}
