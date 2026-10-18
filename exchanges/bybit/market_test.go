package bybit

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/types"
)

// marketTestStart and marketTestEnd bound the fixed window the market tests query, so live and mock results agree
var (
	marketTestStart = time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	marketTestEnd   = time.Date(2026, 10, 1, 1, 0, 0, 0, time.UTC)
)

// optionExpiry returns the expiry date in an option symbol such as BTC-6NOV26-80000-C-USDT
func optionExpiry(tb testing.TB, p currency.Pair) time.Time {
	tb.Helper()
	expiry, err := time.Parse("2Jan06", strings.Split(p.Quote.String(), currency.DashDelimiter)[0])
	require.NoErrorf(tb, err, "option pair %s must carry an expiry date", p)
	return expiry
}

func TestGetBybitServerTime(t *testing.T) {
	t.Parallel()
	result, err := e.GetBybitServerTime(t.Context())
	require.NoError(t, err, "GetBybitServerTime must not error")
	if mockTests {
		exp := &ServerTimeResponse{
			TimeSecond: types.Time(time.Unix(1791534103, 0)),
			TimeNano:   types.Time(time.Unix(0, 1791534103963033914)),
		}
		assert.Equal(t, exp, result, "GetBybitServerTime should decode every field")
		return
	}
	assert.WithinDuration(t, time.Now(), result.TimeNano.Time(), time.Minute, "GetBybitServerTime should return the current time")
}

func TestGetKlines(t *testing.T) {
	t.Parallel()
	_, err := e.GetKlines(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetKlines must reject a nil request")
	_, err = e.GetKlines(t.Context(), &KlineRequest{Interval: kline.OneHour})
	require.ErrorIs(t, err, errSymbolMissing, "GetKlines must reject an empty symbol")
	_, err = e.GetKlines(t.Context(), &KlineRequest{Symbol: "BTCUSDT"})
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval, "GetKlines must reject an unsupported interval")
	_, err = e.GetKlines(t.Context(), &KlineRequest{Symbol: "BTCUSDT", Interval: kline.OneHour, Start: marketTestEnd, End: marketTestStart})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetKlines must reject a start after the end")

	for _, tc := range []struct {
		category, symbol string
		exp              *KlineResponse
	}{
		{category: cSpot, symbol: "BTCUSDT", exp: &KlineResponse{
			Category: "spot",
			Symbol:   "BTCUSDT",
			List: []KlineItem{
				{
					StartTime:  types.Time(time.UnixMilli(1790816400000)),
					OpenPrice:  83509.1,
					HighPrice:  83691,
					LowPrice:   83471.8,
					ClosePrice: 83514.3,
					Volume:     268.805241,
					Turnover:   22464396.7397634,
				},
			},
		}},
		{category: cLinear, symbol: "BTCUSDT", exp: &KlineResponse{
			Category: "linear",
			Symbol:   "BTCUSDT",
			List: []KlineItem{
				{
					StartTime:  types.Time(time.UnixMilli(1790816400000)),
					OpenPrice:  83468.7,
					HighPrice:  83656.1,
					LowPrice:   83438.7,
					ClosePrice: 83469.7,
					Volume:     786.118,
					Turnover:   65671255.4582,
				},
			},
		}},
		{category: cLinear, symbol: "BTCPERP", exp: &KlineResponse{
			Category: "linear",
			Symbol:   "BTCPERP",
			List: []KlineItem{
				{
					StartTime:  types.Time(time.UnixMilli(1790816400000)),
					OpenPrice:  83419.2,
					HighPrice:  83604,
					LowPrice:   83376.9,
					ClosePrice: 83425.1,
					Volume:     11.281,
					Turnover:   941566.5783,
				},
			},
		}},
		{category: cInverse, symbol: "BTCUSD", exp: &KlineResponse{
			Category: "inverse",
			Symbol:   "BTCUSD",
			List: []KlineItem{
				{
					StartTime:  types.Time(time.UnixMilli(1790816400000)),
					OpenPrice:  83425.2,
					HighPrice:  83600,
					LowPrice:   83385.9,
					ClosePrice: 83413.8,
					Volume:     2245520,
					Turnover:   26.895959,
				},
			},
		}},
	} {
		t.Run(tc.category+" "+tc.symbol, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetKlines(t.Context(), &KlineRequest{Category: tc.category, Symbol: tc.symbol, Interval: kline.OneHour, Start: marketTestStart, End: marketTestEnd, Limit: 1})
			require.NoError(t, err, "GetKlines must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetKlines should decode every field")
				return
			}
			assert.NotEmpty(t, result.List, "GetKlines should return candles")
		})
	}
}

func TestGetMarkPriceKline(t *testing.T) {
	t.Parallel()
	_, err := e.GetMarkPriceKline(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetMarkPriceKline must reject a nil request")

	for _, tc := range []struct {
		category, symbol string
		exp              *PriceKlineResponse
	}{
		{category: cLinear, symbol: "BTCUSDT", exp: &PriceKlineResponse{
			Category: "linear",
			Symbol:   "BTCUSDT",
			List: []PriceKlineItem{
				{
					StartTime:  types.Time(time.UnixMilli(1790816400000)),
					OpenPrice:  83470.19,
					HighPrice:  83651.6,
					LowPrice:   83438.7,
					ClosePrice: 83470.41,
				},
			},
		}},
		{category: cInverse, symbol: "BTCUSD", exp: &PriceKlineResponse{
			Category: "inverse",
			Symbol:   "BTCUSD",
			List: []PriceKlineItem{
				{
					StartTime:  types.Time(time.UnixMilli(1790816400000)),
					OpenPrice:  83425.2,
					HighPrice:  83602.99,
					LowPrice:   83385.9,
					ClosePrice: 83418.23,
				},
			},
		}},
	} {
		t.Run(tc.category+" "+tc.symbol, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetMarkPriceKline(t.Context(), &KlineRequest{Category: tc.category, Symbol: tc.symbol, Interval: kline.OneHour, Start: marketTestStart, End: marketTestEnd, Limit: 1})
			require.NoError(t, err, "GetMarkPriceKline must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetMarkPriceKline should decode every field")
				return
			}
			assert.NotEmpty(t, result.List, "GetMarkPriceKline should return candles")
		})
	}
}

func TestGetIndexPriceKline(t *testing.T) {
	t.Parallel()
	_, err := e.GetIndexPriceKline(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetIndexPriceKline must reject a nil request")

	for _, tc := range []struct {
		category, symbol string
		exp              *PriceKlineResponse
	}{
		{category: cLinear, symbol: "BTCUSDT", exp: &PriceKlineResponse{
			Category: "linear",
			Symbol:   "BTCUSDT",
			List: []PriceKlineItem{
				{
					StartTime:  types.Time(time.UnixMilli(1790816400000)),
					OpenPrice:  83510.02,
					HighPrice:  83691.44,
					LowPrice:   83481.76,
					ClosePrice: 83516.03,
				},
			},
		}},
		{category: cInverse, symbol: "BTCUSD", exp: &PriceKlineResponse{
			Category: "inverse",
			Symbol:   "BTCUSD",
			List: []PriceKlineItem{
				{
					StartTime:  types.Time(time.UnixMilli(1790816400000)),
					OpenPrice:  83452.63,
					HighPrice:  83639.26,
					LowPrice:   83424.58,
					ClosePrice: 83459.89,
				},
			},
		}},
	} {
		t.Run(tc.category+" "+tc.symbol, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetIndexPriceKline(t.Context(), &KlineRequest{Category: tc.category, Symbol: tc.symbol, Interval: kline.OneHour, Start: marketTestStart, End: marketTestEnd, Limit: 1})
			require.NoError(t, err, "GetIndexPriceKline must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetIndexPriceKline should decode every field")
				return
			}
			assert.NotEmpty(t, result.List, "GetIndexPriceKline should return candles")
		})
	}
}

func TestGetPremiumIndexPriceKline(t *testing.T) {
	t.Parallel()
	_, err := e.GetPremiumIndexPriceKline(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetPremiumIndexPriceKline must reject a nil request")

	result, err := e.GetPremiumIndexPriceKline(t.Context(), &KlineRequest{Category: cLinear, Symbol: "BTCUSDT", Interval: kline.OneHour, Start: marketTestStart, End: marketTestEnd, Limit: 1})
	require.NoError(t, err, "GetPremiumIndexPriceKline must not error")
	if mockTests {
		exp := &PriceKlineResponse{
			Category: "linear",
			Symbol:   "BTCUSDT",
			List: []PriceKlineItem{
				{
					StartTime:  types.Time(time.UnixMilli(1790816400000)),
					OpenPrice:  -0.00042629,
					HighPrice:  -0.00048729,
					LowPrice:   -0.00060889,
					ClosePrice: -0.00048729,
				},
			},
		}
		assert.Equal(t, exp, result, "GetPremiumIndexPriceKline should decode every field")
		return
	}
	assert.NotEmpty(t, result.List, "GetPremiumIndexPriceKline should return candles")
}

func TestKlineParams(t *testing.T) {
	t.Parallel()
	_, err := klineParams(nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "klineParams must reject a nil request")
	_, err = klineParams(&KlineRequest{Interval: kline.OneHour})
	require.ErrorIs(t, err, errSymbolMissing, "klineParams must reject an empty symbol")
	_, err = klineParams(&KlineRequest{Symbol: "BTCUSDT", Interval: kline.TenMin})
	require.ErrorIs(t, err, kline.ErrUnsupportedInterval, "klineParams must reject an interval Bybit does not offer")
	_, err = klineParams(&KlineRequest{Symbol: "BTCUSDT", Interval: kline.OneHour, Start: marketTestEnd, End: marketTestStart})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "klineParams must reject a start after the end")

	for _, tc := range []struct {
		name string
		req  *KlineRequest
		exp  url.Values
	}{
		{
			// Bybit takes linear when the category is left out
			name: "symbol and interval",
			req:  &KlineRequest{Symbol: "BTCUSDT", Interval: kline.OneDay},
			exp:  url.Values{"symbol": {"BTCUSDT"}, "interval": {"D"}},
		},
		{
			name: "start only",
			req:  &KlineRequest{Symbol: "BTCPERP", Interval: kline.OneMonth, Start: marketTestStart},
			exp:  url.Values{"symbol": {"BTCPERP"}, "interval": {"M"}, "start": {"1790812800000"}},
		},
		{
			// Bybit answers a window that starts and ends at one time with the candle that starts then
			name: "start equals end",
			req:  &KlineRequest{Symbol: "BTCUSDT", Interval: kline.OneHour, Start: marketTestStart, End: marketTestStart},
			exp:  url.Values{"symbol": {"BTCUSDT"}, "interval": {"60"}, "start": {"1790812800000"}, "end": {"1790812800000"}},
		},
		{
			name: "every parameter",
			req:  &KlineRequest{Category: cInverse, Symbol: "BTCUSD", Interval: kline.OneHour, Start: marketTestStart, End: marketTestEnd, Limit: 1000},
			exp: url.Values{
				"category": {"inverse"},
				"symbol":   {"BTCUSD"},
				"interval": {"60"},
				"start":    {"1790812800000"},
				"end":      {"1790816400000"},
				"limit":    {"1000"},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			params, err := klineParams(tc.req)
			require.NoError(t, err, "klineParams must not error")
			assert.Equal(t, tc.exp, params, "klineParams should send the set parameters, with times in milliseconds")
		})
	}
}

func TestGetInstrumentInfo(t *testing.T) {
	t.Parallel()
	_, err := e.GetInstrumentInfo(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetInstrumentInfo must reject a nil request")
	_, err = e.GetInstrumentInfo(t.Context(), &InstrumentInfoRequest{})
	require.ErrorIs(t, err, errCategoryNotSet, "GetInstrumentInfo must reject an empty category")

	for _, tc := range []struct {
		name string
		req  *InstrumentInfoRequest
		exp  *InstrumentsInfoResponse
	}{
		{name: "spot", req: &InstrumentInfoRequest{Category: cSpot}, exp: &InstrumentsInfoResponse{
			Category: "spot",
			List: []InstrumentInfo{
				{
					Symbol:           "BTCUSDT",
					SymbolID:         9,
					Status:           "Trading",
					BaseCoin:         currency.BTC,
					QuoteCoin:        currency.USDT,
					XStockMultiplier: 1,
					MarginTrading:    "utaOnly",
					PriceFilter: PriceFilter{
						TickSize: 0.1,
					},
					LotSizeFilter: LotSizeFilter{
						MaxOrderQuantity:          230,
						MinOrderQuantity:          0.000001,
						BasePrecision:             0.000001,
						QuotePrecision:            0.0000001,
						MinOrderAmount:            5,
						MaxOrderAmount:            8000000,
						MaxLimitOrderQuantity:     230,
						MaxMarketOrderQuantity:    120,
						PostOnlyMaxLimitOrderSize: 1150,
					},
					RiskParameters: RiskParameters{
						PriceLimitRatioX: 0.005,
						PriceLimitRatioY: 0.01,
					},
				},
				{
					Symbol:           "NVDAXUSDT",
					SymbolID:         996,
					Status:           "Trading",
					BaseCoin:         currency.NewCode("NVDAX"),
					QuoteCoin:        currency.USDT,
					SymbolType:       "xstocks",
					XStockMultiplier: 1.001701196801074,
					MarginTrading:    "utaOnly",
					PriceFilter: PriceFilter{
						TickSize: 0.01,
					},
					LotSizeFilter: LotSizeFilter{
						MaxOrderQuantity:          1806,
						MinOrderQuantity:          0.001,
						BasePrecision:             0.001,
						QuotePrecision:            0.00001,
						MinOrderAmount:            5,
						MaxOrderAmount:            60000,
						MaxLimitOrderQuantity:     1806,
						MaxMarketOrderQuantity:    903,
						PostOnlyMaxLimitOrderSize: 9030,
					},
					RiskParameters: RiskParameters{
						PriceLimitRatioX: 0.015,
						PriceLimitRatioY: 0.03,
					},
				},
				{
					Symbol:              "FTTUSDT",
					SymbolID:            67,
					Status:              "Trading",
					BaseCoin:            currency.NewCode("FTT"),
					QuoteCoin:           currency.USDT,
					Innovation:          true,
					XStockMultiplier:    1,
					MarginTrading:       "none",
					SpecialTreatmentTag: true,
					PriceFilter: PriceFilter{
						TickSize: 0.0001,
					},
					LotSizeFilter: LotSizeFilter{
						MaxOrderQuantity:          29000,
						MinOrderQuantity:          0.001,
						BasePrecision:             0.001,
						QuotePrecision:            0.0000001,
						MinOrderAmount:            5,
						MaxOrderAmount:            13000,
						MaxLimitOrderQuantity:     29000,
						MaxMarketOrderQuantity:    5800,
						PostOnlyMaxLimitOrderSize: 145000,
					},
					RiskParameters: RiskParameters{
						PriceLimitRatioX: 0.1,
						PriceLimitRatioY: 0.2,
					},
				},
			},
		}},
		{name: "linear", req: &InstrumentInfoRequest{Category: cLinear}, exp: &InstrumentsInfoResponse{
			Category: "linear",
			List: []InstrumentInfo{
				{
					Symbol:       "BTCUSDT",
					SymbolID:     5,
					ContractType: "LinearPerpetual",
					Status:       "Trading",
					BaseCoin:     currency.BTC,
					QuoteCoin:    currency.USDT,
					SettleCoin:   currency.USDT,
					LaunchTime:   types.Time(time.UnixMilli(1584230400000)),
					PriceScale:   2,
					LeverageFilter: LeverageFilter{
						MinLeverage:  1,
						MaxLeverage:  150,
						LeverageStep: 0.01,
					},
					PriceFilter: PriceFilter{
						MinPrice: 0.1,
						MaxPrice: 1999999.8,
						TickSize: 0.1,
					},
					LotSizeFilter: LotSizeFilter{
						MaxOrderQuantity:         1500,
						MinOrderQuantity:         0.001,
						QuantityStep:             0.001,
						PostOnlyMaxOrderQuantity: 1500,
						MaxMktOrderQuantity:      150,
						MinNotionalValue:         5,
					},
					UnifiedMarginTrade: true,
					FundingInterval:    480,
					CopyTrading:        "both",
					UpperFundingRate:   0.00333,
					LowerFundingRate:   -0.00333,
					DisplayName:        "BTCUSDT",
					FullName:           "Bitcoin",
					Tags:               []string{},
					RiskParameters: RiskParameters{
						PriceLimitRatioX: 0.01,
						PriceLimitRatioY: 0.02,
					},
				},
				{
					Symbol:       "BTCUSDT-24SEP27",
					SymbolID:     1515,
					ContractType: "LinearFutures",
					Status:       "Trading",
					BaseCoin:     currency.BTC,
					QuoteCoin:    currency.USDT,
					SettleCoin:   currency.USDT,
					LaunchTime:   types.Time(time.UnixMilli(1791532800000)),
					DeliveryTime: types.Time(time.UnixMilli(1821772800000)),
					PriceScale:   1,
					LeverageFilter: LeverageFilter{
						MinLeverage:  1,
						MaxLeverage:  50,
						LeverageStep: 0.01,
					},
					PriceFilter: PriceFilter{
						MinPrice: 0.1,
						MaxPrice: 1999999.8,
						TickSize: 0.1,
					},
					LotSizeFilter: LotSizeFilter{
						MaxOrderQuantity:         10,
						MinOrderQuantity:         0.001,
						QuantityStep:             0.001,
						PostOnlyMaxOrderQuantity: 10,
						MaxMktOrderQuantity:      10,
						MinNotionalValue:         5,
					},
					UnifiedMarginTrade: true,
					CopyTrading:        "none",
					DisplayName:        "BTCUSDT-24SEP27",
					FullName:           "Bitcoin",
					Tags:               []string{},
					RiskParameters: RiskParameters{
						PriceLimitRatioX: 0.05,
						PriceLimitRatioY: 0.1,
					},
				},
				{
					Symbol:       "10000NEXUSDT",
					SymbolID:     2677,
					ContractType: "LinearPerpetual",
					Status:       "Trading",
					BaseCoin:     currency.NewCode("10000NEX"),
					QuoteCoin:    currency.USDT,
					SettleCoin:   currency.USDT,
					LaunchTime:   types.Time(time.UnixMilli(1779350242000)),
					PriceScale:   5,
					LeverageFilter: LeverageFilter{
						MinLeverage:  1,
						MaxLeverage:  10,
						LeverageStep: 0.01,
					},
					PriceFilter: PriceFilter{
						MinPrice: 0.00001,
						MaxPrice: 199.99998,
						TickSize: 0.00001,
					},
					LotSizeFilter: LotSizeFilter{
						MaxOrderQuantity:         3600000,
						MinOrderQuantity:         10,
						QuantityStep:             10,
						PostOnlyMaxOrderQuantity: 3600000,
						MaxMktOrderQuantity:      1800000,
						MinNotionalValue:         5,
					},
					UnifiedMarginTrade: true,
					FundingInterval:    240,
					CopyTrading:        "none",
					UpperFundingRate:   0.025,
					LowerFundingRate:   -0.025,
					DisplayName:        "10000NEXUSDT",
					Tags: []string{
						"ST",
					},
					ForbidUnrealisedPNLWithdrawal: true,
					RiskParameters: RiskParameters{
						PriceLimitRatioX: 0.1,
						PriceLimitRatioY: 0.2,
					},
				},
			},
			NextPageCursor: "first%3D0GUSDT%26last%3DMNTUSDT-27NOV26",
		}},
		{name: "option", req: &InstrumentInfoRequest{Category: cOption}, exp: &InstrumentsInfoResponse{
			Category: "option",
			List: []InstrumentInfo{
				{
					Symbol:          "BTC-24SEP27-170000-C-USDT",
					SymbolID:        408164,
					OptionsType:     "Call",
					Status:          "Trading",
					BaseCoin:        currency.BTC,
					QuoteCoin:       currency.USDT,
					SettleCoin:      currency.USDT,
					LaunchTime:      types.Time(time.UnixMilli(1791446400000)),
					DeliveryTime:    types.Time(time.UnixMilli(1821772800000)),
					DeliveryFeeRate: 0.00015,
					PriceFilter: PriceFilter{
						MinPrice: 5,
						MaxPrice: 1110000,
						TickSize: 5,
					},
					LotSizeFilter: LotSizeFilter{
						MaxOrderQuantity: 500,
						MinOrderQuantity: 0.01,
						QuantityStep:     0.01,
					},
					DisplayName: "BTCUSDT-24SEP27-170000-C",
				},
				{
					Symbol:          "BTC-24SEP27-170000-P-USDT",
					SymbolID:        408165,
					OptionsType:     "Put",
					Status:          "Trading",
					BaseCoin:        currency.BTC,
					QuoteCoin:       currency.USDT,
					SettleCoin:      currency.USDT,
					LaunchTime:      types.Time(time.UnixMilli(1791446400000)),
					DeliveryTime:    types.Time(time.UnixMilli(1821772800000)),
					DeliveryFeeRate: 0.00015,
					PriceFilter: PriceFilter{
						MinPrice: 5,
						MaxPrice: 1110000,
						TickSize: 5,
					},
					LotSizeFilter: LotSizeFilter{
						MaxOrderQuantity: 500,
						MinOrderQuantity: 0.01,
						QuantityStep:     0.01,
					},
					DisplayName: "BTCUSDT-24SEP27-170000-P",
				},
			},
			NextPageCursor: "0%2C500",
		}},
		{name: "linear pre-market", req: &InstrumentInfoRequest{Category: cLinear, Status: "PreLaunch", Limit: 1}, exp: &InstrumentsInfoResponse{
			Category: "linear",
			List: []InstrumentInfo{
				{
					Symbol:       "ANTHROPICUSDT",
					SymbolID:     2755,
					ContractType: "LinearPerpetual",
					Status:       "PreLaunch",
					BaseCoin:     currency.NewCode("ANTHROPIC"),
					QuoteCoin:    currency.USDT,
					SettleCoin:   currency.USDT,
					SymbolType:   "stock",
					LaunchTime:   types.Time(time.UnixMilli(1783934824000)),
					PriceScale:   2,
					LeverageFilter: LeverageFilter{
						MinLeverage:  1,
						MaxLeverage:  25,
						LeverageStep: 0.01,
					},
					PriceFilter: PriceFilter{
						MinPrice: 0.01,
						MaxPrice: 199999.98,
						TickSize: 0.01,
					},
					LotSizeFilter: LotSizeFilter{
						MaxOrderQuantity:         124,
						MinOrderQuantity:         0.001,
						QuantityStep:             0.001,
						PostOnlyMaxOrderQuantity: 124,
						MaxMktOrderQuantity:      31,
						MinNotionalValue:         5,
					},
					UnifiedMarginTrade: true,
					FundingInterval:    480,
					CopyTrading:        "none",
					UpperFundingRate:   0.02,
					LowerFundingRate:   -0.02,
					DisplayName:        "ANTHROPICUSDT",
					FullName:           "Anthropic",
					MarketRegion:       "US",
					UnderlyingTicker:   "ANTHROPIC",
					Tags:               []string{},
					RiskParameters: RiskParameters{
						PriceLimitRatioX: 0.03,
						PriceLimitRatioY: 0.05,
					},
					IsPreListing: true,
					PreListingInfo: &PreListingInfo{
						CurrentAuctionPhase: "ContinuousTrading",
						// Bybit sends 946684800000 as every time of a contract that skips the call auction
						Phases: []PreListingPhase{
							{Phase: "CallAuction"},
							{Phase: "CallAuctionNoCancel"},
							{Phase: "CrossMatching"},
							{Phase: "ContinuousTrading"},
						},
						AuctionFeeInfo: AuctionFeeInfo{
							AuctionFeeRate: 0.0001,
							TakerFeeRate:   0.00055,
							MakerFeeRate:   0.0002,
						},
						SkipCallAuction: true,
					},
				},
			},
			NextPageCursor: "first%3DANTHROPICUSDT%26last%3DANTHROPICUSDT",
		}},
		{name: "linear pre-market with a call auction", req: &InstrumentInfoRequest{Category: cLinear, Status: "PreLaunch", Symbol: "BPUSDT"}, exp: &InstrumentsInfoResponse{
			Category: "linear",
			List: []InstrumentInfo{
				{
					Symbol:       "BPUSDT",
					SymbolID:     2593,
					ContractType: "LinearPerpetual",
					Status:       "PreLaunch",
					BaseCoin:     currency.NewCode("BP"),
					QuoteCoin:    currency.USDT,
					SettleCoin:   currency.USDT,
					LaunchTime:   types.Time(time.UnixMilli(1773639914000)),
					PriceScale:   5,
					LeverageFilter: LeverageFilter{
						MinLeverage:  1,
						MaxLeverage:  5,
						LeverageStep: 0.01,
					},
					PriceFilter: PriceFilter{
						MinPrice: 0.00001,
						MaxPrice: 199.99998,
						TickSize: 0.00001,
					},
					LotSizeFilter: LotSizeFilter{
						MaxOrderQuantity:         250000,
						MinOrderQuantity:         1,
						QuantityStep:             1,
						PostOnlyMaxOrderQuantity: 250000,
						MaxMktOrderQuantity:      50000,
						MinNotionalValue:         5,
					},
					UnifiedMarginTrade: true,
					FundingInterval:    240,
					CopyTrading:        "none",
					UpperFundingRate:   0.025,
					LowerFundingRate:   -0.025,
					DisplayName:        "BPUSDT",
					FullName:           "Backpack",
					Tags:               []string{"ST"},
					RiskParameters: RiskParameters{
						PriceLimitRatioX: 0.15,
						PriceLimitRatioY: 0.3,
					},
					IsPreListing: true,
					PreListingInfo: &PreListingInfo{
						CurrentAuctionPhase: "ContinuousTrading",
						Phases: []PreListingPhase{
							{
								Phase:     "CallAuction",
								StartTime: PreListingPhaseTime(time.UnixMilli(1773655200000)),
								EndTime:   PreListingPhaseTime(time.UnixMilli(1773656400000)),
							},
							{
								Phase:     "CallAuctionNoCancel",
								StartTime: PreListingPhaseTime(time.UnixMilli(1773656400000)),
								EndTime:   PreListingPhaseTime(time.UnixMilli(1773656700000)),
							},
							{
								Phase:     "CrossMatching",
								StartTime: PreListingPhaseTime(time.UnixMilli(1773656700000)),
								EndTime:   PreListingPhaseTime(time.UnixMilli(1773657000000)),
							},
							{
								Phase:     "ContinuousTrading",
								StartTime: PreListingPhaseTime(time.UnixMilli(1773657000000)),
							},
						},
						AuctionFeeInfo: AuctionFeeInfo{
							TakerFeeRate: 0.00055,
							MakerFeeRate: 0.0002,
						},
					},
				},
			},
		}},
		{name: "linear symbol", req: &InstrumentInfoRequest{Category: cLinear, Symbol: "BTCUSDT"}, exp: &InstrumentsInfoResponse{
			Category: "linear",
			List: []InstrumentInfo{
				{
					Symbol:       "BTCUSDT",
					SymbolID:     5,
					ContractType: "LinearPerpetual",
					Status:       "Trading",
					BaseCoin:     currency.BTC,
					QuoteCoin:    currency.USDT,
					SettleCoin:   currency.USDT,
					LaunchTime:   types.Time(time.UnixMilli(1584230400000)),
					PriceScale:   2,
					LeverageFilter: LeverageFilter{
						MinLeverage:  1,
						MaxLeverage:  150,
						LeverageStep: 0.01,
					},
					PriceFilter: PriceFilter{
						MinPrice: 0.1,
						MaxPrice: 1999999.8,
						TickSize: 0.1,
					},
					LotSizeFilter: LotSizeFilter{
						MaxOrderQuantity:         1500,
						MinOrderQuantity:         0.001,
						QuantityStep:             0.001,
						PostOnlyMaxOrderQuantity: 1500,
						MaxMktOrderQuantity:      150,
						MinNotionalValue:         5,
					},
					UnifiedMarginTrade: true,
					FundingInterval:    480,
					CopyTrading:        "both",
					UpperFundingRate:   0.00333,
					LowerFundingRate:   -0.00333,
					DisplayName:        "BTCUSDT",
					FullName:           "Bitcoin",
					Tags:               []string{},
					RiskParameters: RiskParameters{
						PriceLimitRatioX: 0.01,
						PriceLimitRatioY: 0.02,
					},
				},
			},
		}},
		{name: "spot symbol type", req: &InstrumentInfoRequest{Category: cSpot, SymbolType: "xstocks"}, exp: &InstrumentsInfoResponse{
			Category: "spot",
			List: []InstrumentInfo{
				{
					Symbol:           "NVDAXUSDT",
					SymbolID:         996,
					Status:           "Trading",
					BaseCoin:         currency.NewCode("NVDAX"),
					QuoteCoin:        currency.USDT,
					SymbolType:       "xstocks",
					XStockMultiplier: 1.001701196801074,
					MarginTrading:    "utaOnly",
					PriceFilter: PriceFilter{
						TickSize: 0.01,
					},
					LotSizeFilter: LotSizeFilter{
						MaxOrderQuantity:          1806,
						MinOrderQuantity:          0.001,
						BasePrecision:             0.001,
						QuotePrecision:            0.00001,
						MinOrderAmount:            5,
						MaxOrderAmount:            60000,
						MaxLimitOrderQuantity:     1806,
						MaxMarketOrderQuantity:    903,
						PostOnlyMaxLimitOrderSize: 9030,
					},
					RiskParameters: RiskParameters{
						PriceLimitRatioX: 0.015,
						PriceLimitRatioY: 0.03,
					},
				},
				{
					Symbol:           "AAPLXUSDT",
					SymbolID:         1002,
					Status:           "Trading",
					BaseCoin:         currency.NewCode("AAPLX"),
					QuoteCoin:        currency.USDT,
					SymbolType:       "xstocks",
					XStockMultiplier: 1.0032690125398187,
					MarginTrading:    "utaOnly",
					PriceFilter: PriceFilter{
						TickSize: 0.01,
					},
					LotSizeFilter: LotSizeFilter{
						MaxOrderQuantity:          270,
						MinOrderQuantity:          0.001,
						BasePrecision:             0.001,
						QuotePrecision:            0.00001,
						MinOrderAmount:            5,
						MaxOrderAmount:            65000,
						MaxLimitOrderQuantity:     270,
						MaxMarketOrderQuantity:    270,
						PostOnlyMaxLimitOrderSize: 1350,
					},
					RiskParameters: RiskParameters{
						PriceLimitRatioX: 0.015,
						PriceLimitRatioY: 0.03,
					},
				},
			},
		}},
		{name: "option page", req: &InstrumentInfoRequest{Category: cOption, BaseCoin: "BTC", Limit: 1, Cursor: "0%2C1"}, exp: &InstrumentsInfoResponse{
			Category: "option",
			List: []InstrumentInfo{
				{
					Symbol:          "BTC-24SEP27-170000-C-USDT",
					SymbolID:        408164,
					OptionsType:     "Call",
					Status:          "Trading",
					BaseCoin:        currency.BTC,
					QuoteCoin:       currency.USDT,
					SettleCoin:      currency.USDT,
					LaunchTime:      types.Time(time.UnixMilli(1791446400000)),
					DeliveryTime:    types.Time(time.UnixMilli(1821772800000)),
					DeliveryFeeRate: 0.00015,
					PriceFilter: PriceFilter{
						MinPrice: 5,
						MaxPrice: 1110000,
						TickSize: 5,
					},
					LotSizeFilter: LotSizeFilter{
						MaxOrderQuantity: 500,
						MinOrderQuantity: 0.01,
						QuantityStep:     0.01,
					},
					DisplayName: "BTCUSDT-24SEP27-170000-C",
				},
			},
			NextPageCursor: "1%2C2",
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetInstrumentInfo(t.Context(), tc.req)
			require.NoError(t, err, "GetInstrumentInfo must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetInstrumentInfo should decode every field")
				return
			}
			// Pre-market contracts come and go, so only the category is certain
			assert.Equal(t, tc.req.Category, result.Category, "GetInstrumentInfo should return the requested category")
		})
	}
}

func TestGetOrderBook(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderBook(t.Context(), "", "BTCUSDT", 2)
	require.ErrorIs(t, err, errCategoryNotSet, "GetOrderBook must reject an empty category")
	_, err = e.GetOrderBook(t.Context(), cSpot, "", 2)
	require.ErrorIs(t, err, errSymbolMissing, "GetOrderBook must reject an empty symbol")

	ensureTradablePairs(t)
	for _, tc := range []struct {
		category, symbol string
		exp              *OrderbookResponse
	}{
		{category: cSpot, symbol: "BTCUSDT", exp: &OrderbookResponse{
			Symbol: "BTCUSDT",
			Bids: []orderbook.Level{
				{
					Amount: 0.577019,
					Price:  82557.4,
				},
				{
					Amount: 0.001016,
					Price:  82557.3,
				},
			},
			Asks: []orderbook.Level{
				{
					Amount: 1.025382,
					Price:  82557.5,
				},
				{
					Amount: 0.000091,
					Price:  82557.8,
				},
			},
			Timestamp:         types.Time(time.UnixMilli(1791535211752)),
			UpdateID:          58415899,
			Sequence:          115179128763,
			CreationTimestamp: types.Time(time.UnixMilli(1791535211749)),
		}},
		{category: cLinear, symbol: "BTCUSDT", exp: &OrderbookResponse{
			Symbol: "BTCUSDT",
			Bids: []orderbook.Level{
				{
					Amount: 1.002,
					Price:  82525.9,
				},
				{
					Amount: 0.001,
					Price:  82525.8,
				},
			},
			Asks: []orderbook.Level{
				{
					Amount: 15.005,
					Price:  82526,
				},
				{
					Amount: 3.443,
					Price:  82526.1,
				},
			},
			Timestamp:         types.Time(time.UnixMilli(1791535211802)),
			UpdateID:          33732693,
			Sequence:          822945250628,
			CreationTimestamp: types.Time(time.UnixMilli(1791535211801)),
		}},
		{category: cInverse, symbol: "BTCUSD", exp: &OrderbookResponse{
			Symbol: "BTCUSD",
			Bids: []orderbook.Level{
				{
					Amount: 13763,
					Price:  82461.9,
				},
				{
					Amount: 6084,
					Price:  82460.3,
				},
			},
			Asks: []orderbook.Level{
				{
					Amount: 111881,
					Price:  82462,
				},
				{
					Amount: 5336,
					Price:  82462.4,
				},
			},
			Timestamp:         types.Time(time.UnixMilli(1791535212005)),
			UpdateID:          33335330,
			Sequence:          118955109837,
			CreationTimestamp: types.Time(time.UnixMilli(1791535212003)),
		}},
		{category: cOption, symbol: optionsTradablePair.String(), exp: &OrderbookResponse{
			Symbol: "BTC-26NOV24-92000-C",
			Bids: []orderbook.Level{
				{
					Amount: 18.55,
					Price:  83310,
				},
				{
					Amount: 0.96,
					Price:  5,
				},
			},
			Asks: []orderbook.Level{
				{
					Amount: 18.55,
					Price:  85565,
				},
			},
			Timestamp:         types.Time(time.UnixMilli(1791534197254)),
			UpdateID:          77632995738,
			Sequence:          77632995738,
			CreationTimestamp: types.Time(time.UnixMilli(1791534197254)),
		}},
	} {
		t.Run(tc.category, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetOrderBook(t.Context(), tc.category, tc.symbol, 2)
			require.NoError(t, err, "GetOrderBook must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetOrderBook should decode every field")
				return
			}
			assert.Equal(t, tc.symbol, result.Symbol, "GetOrderBook should return the requested symbol")
		})
	}
	t.Run("unlisted option", func(t *testing.T) {
		t.Parallel()
		_, err := e.GetOrderBook(t.Context(), cOption, "BTC-1JAN20-1000-C", 2)
		assert.ErrorIs(t, err, errOrderbookNotFound, "GetOrderBook should report an option Bybit does not list")
	})
	t.Run("unlisted symbol", func(t *testing.T) {
		t.Parallel()
		_, err := e.GetOrderBook(t.Context(), cLinear, "NOTAREALSYMBOLUSDT", 2)
		assert.ErrorIs(t, err, errAPIResponse, "GetOrderBook should return Bybit's error for a symbol it does not list")
	})
}

func TestGetRPIOrderBook(t *testing.T) {
	t.Parallel()
	_, err := e.GetRPIOrderBook(t.Context(), cSpot, "", 2)
	require.ErrorIs(t, err, errSymbolMissing, "GetRPIOrderBook must reject an empty symbol")
	_, err = e.GetRPIOrderBook(t.Context(), cSpot, "BTCUSDT", 0)
	require.ErrorIs(t, err, errLimitRequired, "GetRPIOrderBook must reject a zero limit")

	for _, tc := range []struct {
		category string
		exp      *RPIOrderbookResponse
	}{
		{category: cSpot, exp: &RPIOrderbookResponse{
			Symbol: "BTCUSDT",
			Bids: []RPIOrderbookLevel{
				{
					Price:      82557.4,
					NonRPISize: 0.577019,
					RPISize:    0.524532,
				},
				{
					Price:      82557.3,
					NonRPISize: 0.001016,
				},
			},
			Asks: []RPIOrderbookLevel{
				{
					Price:      82557.5,
					NonRPISize: 1.025382,
					RPISize:    0.29,
				},
				{
					Price:      82557.8,
					NonRPISize: 0.000091,
				},
			},
			Timestamp:         types.Time(time.UnixMilli(1791535212315)),
			UpdateID:          114976782,
			Sequence:          115179128923,
			CreationTimestamp: types.Time(time.UnixMilli(1791535212311)),
		}},
		{category: cLinear, exp: &RPIOrderbookResponse{
			Symbol: "BTCUSDT",
			Bids: []RPIOrderbookLevel{
				{
					Price:      82525.9,
					NonRPISize: 1.028,
				},
				{
					Price:      82525.8,
					NonRPISize: 0.001,
				},
			},
			Asks: []RPIOrderbookLevel{
				{
					Price:      82526,
					NonRPISize: 15.005,
				},
				{
					Price:      82526.1,
					NonRPISize: 3.443,
				},
			},
			Timestamp:         types.Time(time.UnixMilli(1791535212529)),
			UpdateID:          62720942,
			Sequence:          822945252818,
			CreationTimestamp: types.Time(time.UnixMilli(1791535212528)),
		}},
	} {
		t.Run(tc.category, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetRPIOrderBook(t.Context(), tc.category, "BTCUSDT", 2)
			require.NoError(t, err, "GetRPIOrderBook must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetRPIOrderBook should decode every field")
				return
			}
			assert.NotEmpty(t, result.Bids, "GetRPIOrderBook should return bids")
		})
	}
}

func TestGetFullDepthOrderBook(t *testing.T) {
	t.Parallel()
	_, err := e.GetFullDepthOrderBook(t.Context(), "", "BTCUSD")
	require.ErrorIs(t, err, errCategoryNotSet, "GetFullDepthOrderBook must reject an empty category")
	_, err = e.GetFullDepthOrderBook(t.Context(), cInverse, "")
	require.ErrorIs(t, err, errSymbolMissing, "GetFullDepthOrderBook must reject an empty symbol")

	result, err := e.GetFullDepthOrderBook(t.Context(), cInverse, "BTCUSD")
	require.NoError(t, err, "GetFullDepthOrderBook must not error")
	if mockTests {
		exp := &OrderbookResponse{
			Symbol: "BTCUSD",
			Bids: []orderbook.Level{
				{
					Amount: 13763,
					Price:  82449.3,
				},
				{
					Amount: 5696,
					Price:  82445.6,
				},
			},
			Asks: []orderbook.Level{
				{
					Amount: 119381,
					Price:  82449.4,
				},
				{
					Amount: 3674,
					Price:  82449.5,
				},
			},
			Timestamp:         types.Time(time.UnixMilli(1791535219718)),
			UpdateID:          19477715,
			Sequence:          118955113896,
			CreationTimestamp: types.Time(time.UnixMilli(1791535219706)),
		}
		assert.Equal(t, exp, result, "GetFullDepthOrderBook should decode every field")
		return
	}
	assert.NotEmpty(t, result.Bids, "GetFullDepthOrderBook should return bids")
}

func TestGetTickers(t *testing.T) {
	t.Parallel()
	_, err := e.GetTickers(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetTickers must reject a nil request")
	_, err = e.GetTickers(t.Context(), &TickersRequest{})
	require.ErrorIs(t, err, errCategoryNotSet, "GetTickers must reject an empty category")
	_, err = e.GetTickers(t.Context(), &TickersRequest{Category: cOption})
	require.ErrorIs(t, err, errBaseCoinOrSymbolRequired, "GetTickers must require a base coin or symbol for option")

	ensureTradablePairs(t)
	for _, tc := range []struct {
		name string
		req  *TickersRequest
		exp  *TickersResponse
	}{
		{name: "linear perpetual", req: &TickersRequest{Category: cLinear, Symbol: "BTCUSDT"}, exp: &TickersResponse{
			Category: "linear",
			List: []Ticker{
				{
					Symbol:                  "BTCUSDT",
					LastPrice:               82511.7,
					IndexPrice:              82551.02,
					MarkPrice:               82517.08,
					PreviousPrice24Hour:     82792.6,
					Price24HourPercent:      -0.003392,
					HighPrice24Hour:         83143.2,
					LowPrice24Hour:          80350.1,
					PreviousPrice1Hour:      82512.8,
					OpenInterest:            57998.471,
					OpenInterestValue:       4785864471.38,
					SingleOpenInterest:      28999.236,
					SingleOpenInterestValue: 2392932276.95,
					Turnover24Hour:          6869522385.2914,
					Volume24Hour:            83987.124,
					FundingRate:             0.0000355,
					NextFundingTime:         types.Time(time.UnixMilli(1791561600000)),
					Bid1Price:               82511.6,
					Bid1Size:                1.361,
					Ask1Price:               82511.7,
					Ask1Size:                12.171,
					FundingIntervalHour:     8,
					FundingCap:              0.00333,
				},
			},
		}},
		{name: "linear dated", req: &TickersRequest{Category: cLinear, Symbol: "BTCUSDT-24SEP27"}, exp: &TickersResponse{
			Category: "linear",
			List: []Ticker{
				{
					Symbol:                 "BTCUSDT-24SEP27",
					LastPrice:              86294.3,
					IndexPrice:             82550,
					MarkPrice:              86938.8,
					PreviousPrice24Hour:    86294.3,
					HighPrice24Hour:        86294.3,
					LowPrice24Hour:         86294.3,
					PreviousPrice1Hour:     86294.3,
					PredictedDeliveryPrice: 82560.1,
					BasisRate:              0.05320615,
					Basis:                  -82550.03,
					DeliveryFeeRate:        0.0005,
					DeliveryTime:           types.Time(time.UnixMilli(1821772800000)),
					Bid1Price:              86681,
					Bid1Size:               6,
					Ask1Price:              87203.4,
					Ask1Size:               5.964,
					BasisRateYear:          0.05546862,
				},
			},
		}},
		{name: "linear pre-market", req: &TickersRequest{Category: cLinear, Symbol: "ANTHROPICUSDT"}, exp: &TickersResponse{
			Category: "linear",
			List: []Ticker{
				{
					Symbol:                  "ANTHROPICUSDT",
					LastPrice:               2072.8,
					IndexPrice:              2071.42,
					MarkPrice:               2071.42,
					PreviousPrice24Hour:     2104.02,
					Price24HourPercent:      -0.014838,
					HighPrice24Hour:         2115.78,
					LowPrice24Hour:          2027.93,
					PreviousPrice1Hour:      2071.1,
					OpenInterest:            1238.098,
					OpenInterestValue:       2564620.96,
					SingleOpenInterest:      619.049,
					SingleOpenInterestValue: 1282310.48,
					Turnover24Hour:          883317.4938,
					Volume24Hour:            426.206,
					NextFundingTime:         types.Time(time.UnixMilli(1791561600000)),
					Bid1Price:               2067.9,
					Bid1Size:                3.595,
					Ask1Price:               2070.1,
					Ask1Size:                0.776,
					PreOpenPrice:            2070.5,
					PreQuantity:             12.5,
					CurrentPreListingPhase:  "CallAuction",
					FundingIntervalHour:     8,
					FundingCap:              0.02,
				},
			},
		}},
		{name: "spot", req: &TickersRequest{Category: cSpot, Symbol: "BTCUSDT"}, exp: &TickersResponse{
			Category: "spot",
			List: []Ticker{
				{
					Symbol:              "BTCUSDT",
					LastPrice:           82546.3,
					PreviousPrice24Hour: 82841.1,
					Price24HourPercent:  -0.0036,
					HighPrice24Hour:     83174,
					LowPrice24Hour:      80393.9,
					Turnover24Hour:      1121422992.0894127,
					Volume24Hour:        13681.851295,
					Bid1Price:           82546.3,
					Bid1Size:            0.260988,
					Ask1Price:           82546.4,
					Ask1Size:            1.498582,
					USDIndexPrice:       82479.492606,
				},
			},
		}},
		{name: "inverse", req: &TickersRequest{Category: cInverse, Symbol: "BTCUSD"}, exp: &TickersResponse{
			Category: "inverse",
			List: []Ticker{
				{
					Symbol:                  "BTCUSD",
					LastPrice:               82465.8,
					IndexPrice:              82482.55,
					MarkPrice:               82465.8,
					PreviousPrice24Hour:     82753.4,
					Price24HourPercent:      -0.003475,
					HighPrice24Hour:         83092.3,
					LowPrice24Hour:          80302.9,
					PreviousPrice1Hour:      82457.5,
					OpenInterest:            516566600,
					OpenInterestValue:       6264.01,
					SingleOpenInterest:      258283300,
					SingleOpenInterestValue: 3132.01,
					Turnover24Hour:          4585.9163,
					Volume24Hour:            375154141,
					FundingRate:             0.0001,
					NextFundingTime:         types.Time(time.UnixMilli(1791561600000)),
					Bid1Price:               82449.3,
					Bid1Size:                13763,
					Ask1Price:               82449.4,
					Ask1Size:                119381,
					FundingIntervalHour:     8,
					FundingCap:              0.005,
				},
			},
		}},
		{name: "option symbol", req: &TickersRequest{Category: cOption, Symbol: optionsTradablePair.String()}, exp: &TickersResponse{
			Category: "option",
			List: []Ticker{
				{
					Symbol:                 "BTC-26NOV24-92000-C",
					LastPrice:              3070,
					IndexPrice:             82551.02154501,
					MarkPrice:              3246.59506332,
					HighPrice24Hour:        3200,
					LowPrice24Hour:         3070,
					OpenInterest:           0.2,
					Turnover24Hour:         16974.40642989,
					Volume24Hour:           0.2,
					PredictedDeliveryPrice: 92150.25,
					Bid1Price:              2155,
					Bid1Size:               18.55,
					Bid1ImpliedVolatility:  0.3756,
					Ask1Price:              4130,
					Ask1Size:               18.55,
					Ask1ImpliedVolatility:  0.4818,
					MarkImpliedVolatility:  0.4368,
					UnderlyingPrice:        86945.1,
					TotalVolume:            1,
					TotalTurnover:          16975,
					Delta:                  -0.13980126,
					Gamma:                  0.00000598,
					Vega:                   189.31094574,
					Theta:                  -11.81194295,
					Change24Hour:           -0.040625,
				},
			},
		}},
		{name: "option expiry", req: &TickersRequest{Category: cOption, BaseCoin: optionsTradablePair.Base, ExpiryDate: optionExpiry(t, optionsTradablePair)}, exp: &TickersResponse{
			Category: "option",
			List: []Ticker{
				{
					Symbol:                "BTC-26NOV24-120000-C",
					IndexPrice:            82551.02154501,
					MarkPrice:             4320.60000156,
					Bid1Price:             3095,
					Bid1Size:              18.55,
					Bid1ImpliedVolatility: 0.3444,
					Ask1Price:             5535,
					Ask1Size:              18.55,
					Ask1ImpliedVolatility: 0.4338,
					MarkImpliedVolatility: 0.3911,
					UnderlyingPrice:       86945.1,
					Delta:                 0.25786296,
					Gamma:                 0.0000097,
					Vega:                  274.97638796,
					Theta:                 -15.36367729,
				},
				{
					Symbol:                "BTC-26NOV24-60000-P",
					LastPrice:             3070,
					IndexPrice:            82551.02154501,
					MarkPrice:             3246.59506332,
					HighPrice24Hour:       3200,
					LowPrice24Hour:        3070,
					OpenInterest:          0.2,
					Turnover24Hour:        16974.40642989,
					Volume24Hour:          0.2,
					Bid1Price:             2155,
					Bid1Size:              18.55,
					Bid1ImpliedVolatility: 0.3756,
					Ask1Price:             4130,
					Ask1Size:              18.55,
					Ask1ImpliedVolatility: 0.4818,
					MarkImpliedVolatility: 0.4368,
					UnderlyingPrice:       86945.1,
					TotalVolume:           1,
					TotalTurnover:         16975,
					Delta:                 -0.13980126,
					Gamma:                 0.00000598,
					Vega:                  189.31094574,
					Theta:                 -11.81194295,
					Change24Hour:          -0.040625,
				},
			},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetTickers(t.Context(), tc.req)
			require.NoError(t, err, "GetTickers must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetTickers should decode every field")
				return
			}
			assert.NotEmpty(t, result.List, "GetTickers should return tickers")
		})
	}
}

func TestGetFundingRateHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetFundingRateHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetFundingRateHistory must reject a nil request")
	_, err = e.GetFundingRateHistory(t.Context(), &FundingRateHistoryRequest{Symbol: "BTCUSDT"})
	require.ErrorIs(t, err, errCategoryNotSet, "GetFundingRateHistory must reject an empty category")
	_, err = e.GetFundingRateHistory(t.Context(), &FundingRateHistoryRequest{Category: cLinear})
	require.ErrorIs(t, err, errSymbolMissing, "GetFundingRateHistory must reject an empty symbol")
	_, err = e.GetFundingRateHistory(t.Context(), &FundingRateHistoryRequest{Category: cLinear, Symbol: "BTCUSDT", StartTime: marketTestStart})
	require.ErrorIs(t, err, common.ErrDateUnset, "GetFundingRateHistory must reject a start time without an end time")
	_, err = e.GetFundingRateHistory(t.Context(), &FundingRateHistoryRequest{Category: cLinear, Symbol: "BTCUSDT", StartTime: marketTestEnd, EndTime: marketTestStart})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetFundingRateHistory must reject a start after the end")

	for _, tc := range []struct {
		category, symbol string
		exp              *FundingRateHistoryResponse
	}{
		{category: cLinear, symbol: "BTCUSDT", exp: &FundingRateHistoryResponse{
			Category: "linear",
			List: []FundingRate{
				{
					Symbol:               "BTCUSDT",
					FundingRate:          0.00004127,
					FundingRateTimestamp: types.Time(time.UnixMilli(1790841600000)),
				},
				{
					Symbol:               "BTCUSDT",
					FundingRate:          0.00004076,
					FundingRateTimestamp: types.Time(time.UnixMilli(1790812800000)),
				},
			},
		}},
		{category: cInverse, symbol: "BTCUSD", exp: &FundingRateHistoryResponse{
			Category: "inverse",
			List: []FundingRate{
				{
					Symbol:               "BTCUSD",
					FundingRate:          0.00005626,
					FundingRateTimestamp: types.Time(time.UnixMilli(1790841600000)),
				},
				{
					Symbol:               "BTCUSD",
					FundingRate:          0.0001,
					FundingRateTimestamp: types.Time(time.UnixMilli(1790812800000)),
				},
			},
		}},
	} {
		t.Run(tc.category+" "+tc.symbol, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetFundingRateHistory(t.Context(), &FundingRateHistoryRequest{Category: tc.category, Symbol: tc.symbol, StartTime: marketTestStart, EndTime: marketTestStart.Add(8 * time.Hour), Limit: 2})
			require.NoError(t, err, "GetFundingRateHistory must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetFundingRateHistory should decode every field")
				return
			}
			assert.NotEmpty(t, result.List, "GetFundingRateHistory should return funding rates")
		})
	}
}

func TestGetPublicTradingHistory(t *testing.T) {
	t.Parallel()
	_, err := e.GetPublicTradingHistory(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetPublicTradingHistory must reject a nil request")
	_, err = e.GetPublicTradingHistory(t.Context(), &PublicTradingHistoryRequest{Symbol: "BTCUSDT"})
	require.ErrorIs(t, err, errCategoryNotSet, "GetPublicTradingHistory must reject an empty category")
	for _, category := range []string{cSpot, cLinear, cInverse} {
		_, err = e.GetPublicTradingHistory(t.Context(), &PublicTradingHistoryRequest{Category: category})
		require.ErrorIsf(t, err, errSymbolMissing, "GetPublicTradingHistory must require a symbol for %s", category)
	}

	for _, tc := range []struct {
		name string
		req  *PublicTradingHistoryRequest
		exp  *PublicTradingHistoryResponse
	}{
		{name: "spot", req: &PublicTradingHistoryRequest{Category: cSpot, Symbol: "BTCUSDT", Limit: 2}, exp: &PublicTradingHistoryResponse{
			Category: "spot",
			List: []PublicTrade{
				{
					ExecutionID: "2290000001228767766",
					Symbol:      "BTCUSDT",
					Price:       82546.3,
					Size:        0.000149,
					Side:        "Sell",
					Time:        types.Time(time.UnixMilli(1791535218199)),
					IsRPITrade:  true,
					Sequence:    115179134376,
				},
				{
					ExecutionID: "2290000001228767765",
					Symbol:      "BTCUSDT",
					Price:       82546.4,
					Size:        0.00006,
					Side:        "Buy",
					Time:        types.Time(time.UnixMilli(1791535217526)),
					Sequence:    115179133736,
				},
			},
		}},
		{name: "option", req: &PublicTradingHistoryRequest{Category: cOption, BaseCoin: currency.BTC, OptionType: "Put", Limit: 2}, exp: &PublicTradingHistoryResponse{
			Category: "option",
			List: []PublicTrade{
				{
					ExecutionID:           "7cac3c2d-5dde-5add-8aba-19882e17ce01",
					Symbol:                "BTC-12OCT26-79000-P-USDT",
					Price:                 95,
					Size:                  0.05,
					Side:                  "Sell",
					Time:                  types.Time(time.UnixMilli(1791535216611)),
					MarkPrice:             84.88789941,
					IndexPrice:            82557.99319349,
					MarkImpliedVolatility: 0.3439,
					ImpliedVolatility:     0.3533,
					Sequence:              77633960561,
				},
				{
					ExecutionID:           "11870e72-8edb-5f1e-8b40-20681a618d86",
					Symbol:                "BTC-10OCT26-78000-P-USDT",
					Price:                 10,
					Size:                  0.01,
					Side:                  "Sell",
					Time:                  types.Time(time.UnixMilli(1791535214665)),
					IsBlockTrade:          true,
					MarkPrice:             7.88463464,
					IndexPrice:            82557.97861947,
					MarkImpliedVolatility: 0.4849,
					ImpliedVolatility:     0.5006,
					Sequence:              77633958983,
				},
			},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetPublicTradingHistory(t.Context(), tc.req)
			require.NoError(t, err, "GetPublicTradingHistory must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetPublicTradingHistory should decode every field")
				return
			}
			assert.NotEmpty(t, result.List, "GetPublicTradingHistory should return trades")
		})
	}
}

func TestGetOpenInterestData(t *testing.T) {
	t.Parallel()
	_, err := e.GetOpenInterestData(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetOpenInterestData must reject a nil request")
	_, err = e.GetOpenInterestData(t.Context(), &OpenInterestRequest{Symbol: "BTCUSDT", IntervalTime: "1h"})
	require.ErrorIs(t, err, errCategoryNotSet, "GetOpenInterestData must reject an empty category")
	_, err = e.GetOpenInterestData(t.Context(), &OpenInterestRequest{Category: cLinear, IntervalTime: "1h"})
	require.ErrorIs(t, err, errSymbolMissing, "GetOpenInterestData must reject an empty symbol")
	_, err = e.GetOpenInterestData(t.Context(), &OpenInterestRequest{Category: cLinear, Symbol: "BTCUSDT"})
	require.ErrorIs(t, err, errIntervalTimeRequired, "GetOpenInterestData must reject an empty interval time")
	_, err = e.GetOpenInterestData(t.Context(), &OpenInterestRequest{Category: cLinear, Symbol: "BTCUSDT", IntervalTime: "1h", StartTime: marketTestEnd, EndTime: marketTestStart})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetOpenInterestData must reject a start after the end")

	for _, tc := range []struct {
		name, cursor string
		exp          *OpenInterestResponse
	}{
		{name: "first page", exp: &OpenInterestResponse{
			Category: "linear",
			Symbol:   "BTCUSDT",
			List: []OpenInterest{
				{
					OpenInterest:       54558.935,
					SingleOpenInterest: 27279.468,
					Timestamp:          types.Time(time.UnixMilli(1790816400000)),
				},
			},
			NextPageCursor: "lastid%3D24858%26lasttime%3D1790816400",
		}},
		{name: "next page", cursor: "lastid%3D24858%26lasttime%3D1790816400", exp: &OpenInterestResponse{
			Category: "linear",
			Symbol:   "BTCUSDT",
			List: []OpenInterest{
				{
					OpenInterest:       54488.313,
					SingleOpenInterest: 27244.157,
					Timestamp:          types.Time(time.UnixMilli(1790812800000)),
				},
			},
			NextPageCursor: "lastid%3D8793%26lasttime%3D1790812800",
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetOpenInterestData(t.Context(), &OpenInterestRequest{Category: cLinear, Symbol: "BTCUSDT", IntervalTime: "1h", StartTime: marketTestStart, EndTime: marketTestEnd, Limit: 1, Cursor: tc.cursor})
			require.NoError(t, err, "GetOpenInterestData must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetOpenInterestData should decode every field")
				return
			}
			assert.NotEmpty(t, result.List, "GetOpenInterestData should return open interest")
		})
	}
}

func TestGetHistoricalVolatility(t *testing.T) {
	t.Parallel()
	_, err := e.GetHistoricalVolatility(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetHistoricalVolatility must reject a nil request")
	_, err = e.GetHistoricalVolatility(t.Context(), &HistoricalVolatilityRequest{})
	require.ErrorIs(t, err, errCategoryNotSet, "GetHistoricalVolatility must reject an empty category")
	_, err = e.GetHistoricalVolatility(t.Context(), &HistoricalVolatilityRequest{Category: cOption, StartTime: marketTestStart})
	require.ErrorIs(t, err, common.ErrDateUnset, "GetHistoricalVolatility must reject a start time without an end time")
	_, err = e.GetHistoricalVolatility(t.Context(), &HistoricalVolatilityRequest{Category: cOption, StartTime: marketTestEnd, EndTime: marketTestStart})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetHistoricalVolatility must reject a start after the end")

	result, err := e.GetHistoricalVolatility(t.Context(), &HistoricalVolatilityRequest{Category: cOption, BaseCoin: currency.BTC, QuoteCoin: currency.USDT, Period: 7, StartTime: marketTestStart, EndTime: marketTestEnd})
	require.NoError(t, err, "GetHistoricalVolatility must not error")
	if mockTests {
		exp := []HistoricalVolatility{
			{
				Period: 7,
				Value:  0.3200234,
				Time:   types.Time(time.UnixMilli(1790812800000)),
			},
			{
				Period: 7,
				Value:  0.31963075,
				Time:   types.Time(time.UnixMilli(1790816400000)),
			},
		}
		assert.Equal(t, exp, result, "GetHistoricalVolatility should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetHistoricalVolatility should return volatility")
}

func TestGetInsurance(t *testing.T) {
	t.Parallel()
	result, err := e.GetInsurance(t.Context(), currency.BTC)
	require.NoError(t, err, "GetInsurance must not error")
	if mockTests {
		exp := &InsuranceResponse{
			UpdatedTime: types.Time(time.UnixMilli(1791534180000)),
			List: []InsurancePool{
				{
					Coin:    currency.BTC,
					Symbols: "BTCUSD",
					Balance: 5090.29091969,
					Value:   419908355.7,
				},
				{
					Coin:    currency.BTC,
					Symbols: "BTCUSDZ26,BTCUSDH27",
					Balance: 12.67272878,
					Value:   989960.76,
				},
			},
		}
		assert.Equal(t, exp, result, "GetInsurance should decode every field")
		return
	}
	assert.NotEmpty(t, result.List, "GetInsurance should return insurance pools")
}

func TestGetRiskLimit(t *testing.T) {
	t.Parallel()
	_, err := e.GetRiskLimit(t.Context(), "", "BTCUSDT", "")
	require.ErrorIs(t, err, errCategoryNotSet, "GetRiskLimit must reject an empty category")

	for _, tc := range []struct {
		name, symbol, cursor string
		exp                  *RiskLimitTiersResponse
	}{
		{name: "symbol", symbol: "BTCUSDT", exp: &RiskLimitTiersResponse{
			Category: "linear",
			List: []RiskLimitTier{
				{
					ID:                1,
					Symbol:            "BTCUSDT",
					RiskLimitValue:    300000,
					MaintenanceMargin: 0.0033,
					InitialMargin:     0.0066,
					IsLowestRisk:      true,
					MaxLeverage:       150,
				},
				{
					ID:                         2,
					Symbol:                     "BTCUSDT",
					RiskLimitValue:             2000000,
					MaintenanceMargin:          0.005,
					InitialMargin:              0.01,
					MaxLeverage:                100,
					MaintenanceMarginDeduction: 510,
				},
			},
		}},
		{name: "first page", exp: &RiskLimitTiersResponse{
			Category: "linear",
			List: []RiskLimitTier{
				{
					ID:                1,
					Symbol:            "0GUSDT",
					RiskLimitValue:    10000,
					MaintenanceMargin: 0.015,
					InitialMargin:     0.02,
					IsLowestRisk:      true,
					MaxLeverage:       50,
				},
				{
					ID:                         2,
					Symbol:                     "0GUSDT",
					RiskLimitValue:             25000,
					MaintenanceMargin:          0.02,
					InitialMargin:              0.04,
					MaxLeverage:                25,
					MaintenanceMarginDeduction: 50,
				},
			},
			NextPageCursor: "first%3D0GUSDT%26last%3D1000PEPEUSDT",
		}},
		{name: "next page", cursor: "first%3D0GUSDT%26last%3D1000PEPEUSDT", exp: &RiskLimitTiersResponse{
			Category: "linear",
			List: []RiskLimitTier{
				{
					ID:                1,
					Symbol:            "1000RATSUSDT",
					RiskLimitValue:    5000,
					MaintenanceMargin: 0.025,
					InitialMargin:     0.05,
					IsLowestRisk:      true,
					MaxLeverage:       20,
				},
				{
					ID:                         2,
					Symbol:                     "1000RATSUSDT",
					RiskLimitValue:             9000,
					MaintenanceMargin:          0.0263,
					InitialMargin:              0.0526,
					MaxLeverage:                19,
					MaintenanceMarginDeduction: 6.5,
				},
			},
			NextPageCursor: "first%3D1000RATSUSDT%26last%3DACEUSDT",
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetRiskLimit(t.Context(), cLinear, tc.symbol, tc.cursor)
			require.NoError(t, err, "GetRiskLimit must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetRiskLimit should decode every field")
				return
			}
			assert.NotEmpty(t, result.List, "GetRiskLimit should return risk limits")
		})
	}
}

func TestGetDeliveryPrice(t *testing.T) {
	t.Parallel()
	_, err := e.GetDeliveryPrice(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetDeliveryPrice must reject a nil request")
	_, err = e.GetDeliveryPrice(t.Context(), &DeliveryPriceRequest{})
	require.ErrorIs(t, err, errCategoryNotSet, "GetDeliveryPrice must reject an empty category")

	for _, tc := range []struct {
		name string
		req  *DeliveryPriceRequest
		exp  *DeliveryPriceResponse
	}{
		{name: "linear", req: &DeliveryPriceRequest{Category: cLinear, Limit: 2}, exp: &DeliveryPriceResponse{
			Category: "linear",
			List: []DeliveryPrice{
				{
					Symbol:        "ETH-26DEC25",
					DeliveryPrice: 2965.72,
					DeliveryTime:  types.Time(time.UnixMilli(1766736000000)),
				},
				{
					Symbol:        "BTC-26DEC25",
					DeliveryPrice: 88778.62,
					DeliveryTime:  types.Time(time.UnixMilli(1766736000000)),
				},
			},
			NextPageCursor: "lastid%3D1766736000000000000%26lasttime%3D0",
		}},
		{name: "option", req: &DeliveryPriceRequest{Category: cOption, BaseCoin: currency.BTC, SettleCoin: currency.USDT, Limit: 2}, exp: &DeliveryPriceResponse{
			Category: "option",
			List: []DeliveryPrice{
				{
					Symbol:        "BTC-9OCT26-80500-P-USDT",
					DeliveryPrice: 82561.3253306,
					DeliveryTime:  types.Time(time.UnixMilli(1791532800000)),
				},
				{
					Symbol:        "BTC-9OCT26-80500-C-USDT",
					DeliveryPrice: 82561.3253306,
					DeliveryTime:  types.Time(time.UnixMilli(1791532800000)),
				},
			},
			NextPageCursor: "0%2C2",
		}},
		{name: "linear next page", req: &DeliveryPriceRequest{Category: cLinear, Limit: 2, Cursor: "lastid%3D1766736000000000000%26lasttime%3D0"}, exp: &DeliveryPriceResponse{
			Category: "linear",
			List: []DeliveryPrice{
				{
					Symbol:        "BTC-26SEP25",
					DeliveryPrice: 109145.57,
					DeliveryTime:  types.Time(time.UnixMilli(1758873600000)),
				},
				{
					Symbol:        "ETH-26SEP25",
					DeliveryPrice: 3920.83,
					DeliveryTime:  types.Time(time.UnixMilli(1758873600000)),
				},
			},
			NextPageCursor: "lastid%3D1758873600000000000%26lasttime%3D0",
		}},
		{name: "linear symbol", req: &DeliveryPriceRequest{Category: cLinear, Symbol: "BTC-26SEP25"}, exp: &DeliveryPriceResponse{
			Category: "linear",
			List: []DeliveryPrice{
				{
					Symbol:        "BTC-26SEP25",
					DeliveryPrice: 109145.57,
					DeliveryTime:  types.Time(time.UnixMilli(1758873600000)),
				},
			},
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetDeliveryPrice(t.Context(), tc.req)
			require.NoError(t, err, "GetDeliveryPrice must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetDeliveryPrice should decode every field")
				return
			}
			// Options without a symbol list only the symbols delivering at the time
			assert.Equal(t, tc.req.Category, result.Category, "GetDeliveryPrice should return the requested category")
		})
	}
}

func TestGetNewDeliveryPrice(t *testing.T) {
	t.Parallel()
	_, err := e.GetNewDeliveryPrice(t.Context(), "", currency.BTC, currency.EMPTYCODE)
	require.ErrorIs(t, err, errCategoryNotSet, "GetNewDeliveryPrice must reject an empty category")
	_, err = e.GetNewDeliveryPrice(t.Context(), cOption, currency.EMPTYCODE, currency.EMPTYCODE)
	require.ErrorIs(t, err, currency.ErrCurrencyCodeEmpty, "GetNewDeliveryPrice must reject an empty base coin")

	result, err := e.GetNewDeliveryPrice(t.Context(), cOption, currency.BTC, currency.USDT)
	require.NoError(t, err, "GetNewDeliveryPrice must not error")
	if mockTests {
		exp := &NewDeliveryPriceResponse{
			Category: "option",
			List: []NewDeliveryPrice{
				{
					DeliveryPrice: 82561.3253306,
					DeliveryTime:  types.Time(time.UnixMilli(1791504000000)),
				},
				{
					DeliveryPrice: 83095.56619776,
					DeliveryTime:  types.Time(time.UnixMilli(1791417600000)),
				},
			},
		}
		assert.Equal(t, exp, result, "GetNewDeliveryPrice should decode every field")
		return
	}
	assert.NotEmpty(t, result.List, "GetNewDeliveryPrice should return delivery prices")
}

func TestGetLongShortRatio(t *testing.T) {
	t.Parallel()
	_, err := e.GetLongShortRatio(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetLongShortRatio must reject a nil request")
	_, err = e.GetLongShortRatio(t.Context(), &LongShortRatioRequest{Symbol: "BTCUSDT", Period: "1h"})
	require.ErrorIs(t, err, errCategoryNotSet, "GetLongShortRatio must reject an empty category")
	_, err = e.GetLongShortRatio(t.Context(), &LongShortRatioRequest{Category: cLinear, Period: "1h"})
	require.ErrorIs(t, err, errSymbolMissing, "GetLongShortRatio must reject an empty symbol")
	_, err = e.GetLongShortRatio(t.Context(), &LongShortRatioRequest{Category: cLinear, Symbol: "BTCUSDT"})
	require.ErrorIs(t, err, errPeriodRequired, "GetLongShortRatio must reject an empty period")
	_, err = e.GetLongShortRatio(t.Context(), &LongShortRatioRequest{Category: cLinear, Symbol: "BTCUSDT", Period: "1h", StartTime: marketTestEnd, EndTime: marketTestStart})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetLongShortRatio must reject a start after the end")

	for _, tc := range []struct {
		name, cursor string
		exp          *LongShortRatioResponse
	}{
		{name: "first page", exp: &LongShortRatioResponse{
			List: []LongShortRatio{
				{
					Symbol:    "BTCUSDT",
					BuyRatio:  0.5846,
					SellRatio: 0.4154,
					Timestamp: types.Time(time.UnixMilli(1790816400000)),
				},
			},
			NextPageCursor: "lastid%3D0%26lasttime%3D1790816400",
		}},
		{name: "next page", cursor: "lastid%3D0%26lasttime%3D1790816400", exp: &LongShortRatioResponse{
			List: []LongShortRatio{
				{
					Symbol:    "BTCUSDT",
					BuyRatio:  0.5855,
					SellRatio: 0.4145,
					Timestamp: types.Time(time.UnixMilli(1790812800000)),
				},
			},
			NextPageCursor: "lastid%3D0%26lasttime%3D1790812800",
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetLongShortRatio(t.Context(), &LongShortRatioRequest{Category: cLinear, Symbol: "BTCUSDT", Period: "1h", StartTime: marketTestStart, EndTime: marketTestEnd, Limit: 1, Cursor: tc.cursor})
			require.NoError(t, err, "GetLongShortRatio must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "GetLongShortRatio should decode every field")
				return
			}
			assert.NotEmpty(t, result.List, "GetLongShortRatio should return ratios")
		})
	}
}

func TestGetIndexPriceComponents(t *testing.T) {
	t.Parallel()
	_, err := e.GetIndexPriceComponents(t.Context(), "")
	require.ErrorIs(t, err, errIndexNameRequired, "GetIndexPriceComponents must reject an empty index name")

	result, err := e.GetIndexPriceComponents(t.Context(), "BTCUSDT")
	require.NoError(t, err, "GetIndexPriceComponents must not error")
	if mockTests {
		exp := &IndexPriceComponentsResponse{
			IndexName:  "BTCUSDT",
			LastPrice:  82565.81,
			UpdateTime: types.Time(time.UnixMilli(1791534307540)),
			Components: []IndexPriceComponent{
				{
					Exchange:        "Bybit",
					SpotPair:        "BTCUSDT",
					EquivalentPrice: 82560.8,
					Multiplier:      1,
					Price:           82560.8,
					Weight:          0.2161,
				},
				{
					Exchange:        "Binance",
					SpotPair:        "BTCUSDT",
					EquivalentPrice: 82567.38,
					Multiplier:      1,
					Price:           82567.38,
					Weight:          0.3288,
				},
			},
		}
		assert.Equal(t, exp, result, "GetIndexPriceComponents should decode every field")
		return
	}
	assert.NotEmpty(t, result.Components, "GetIndexPriceComponents should return components")
}

func TestGetOrderPriceLimit(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderPriceLimit(t.Context(), cLinear, "")
	require.ErrorIs(t, err, errSymbolMissing, "GetOrderPriceLimit must reject an empty symbol")

	result, err := e.GetOrderPriceLimit(t.Context(), cLinear, "BTCUSDT")
	require.NoError(t, err, "GetOrderPriceLimit must not error")
	if mockTests {
		exp := &OrderPriceLimitResponse{
			Symbol:    "BTCUSDT",
			BuyLimit:  83352.3,
			SellLimit: 81701.7,
			Timestamp: types.Time(time.UnixMilli(1791534307183)),
		}
		assert.Equal(t, exp, result, "GetOrderPriceLimit should decode every field")
		return
	}
	assert.Positive(t, result.BuyLimit.Float64(), "GetOrderPriceLimit should return a buy limit")
}

func TestGetADLAlert(t *testing.T) {
	t.Parallel()
	result, err := e.GetADLAlert(t.Context(), "BTCUSDT")
	require.NoError(t, err, "GetADLAlert must not error")
	if mockTests {
		exp := &ADLAlertResponse{
			UpdatedTime: types.Time(time.UnixMilli(1791534240000)),
			List: []ADLAlert{
				{
					Coin:                currency.USDT,
					Symbol:              "BTCUSDT",
					Balance:             386878598.28325313,
					MaxBalance:          386900000.5,
					InsurancePNLRatio:   -1,
					PNLRatio:            -0.000123,
					ADLTriggerThreshold: 20000,
					ADLStopRatio:        -0.95,
				},
			},
		}
		assert.Equal(t, exp, result, "GetADLAlert should decode every field")
		return
	}
	assert.NotEmpty(t, result.List, "GetADLAlert should return alerts")
}

func TestGetFeeGroupStructure(t *testing.T) {
	t.Parallel()
	_, err := e.GetFeeGroupStructure(t.Context(), "", 1)
	require.ErrorIs(t, err, errProductTypeRequired, "GetFeeGroupStructure must reject an empty product type")

	result, err := e.GetFeeGroupStructure(t.Context(), "contract", 1)
	require.NoError(t, err, "GetFeeGroupStructure must not error")
	if mockTests {
		exp := &FeeGroupStructureResponse{
			List: []FeeGroup{
				{
					GroupName:       "Major Coins",
					WeightingFactor: 1,
					SymbolsNumbers:  4,
					Symbols: []string{
						"ETHUSDT",
						"XRPUSDT",
						"SOLUSDT",
						"BTCUSDT",
					},
					FeeRates: FeeGroupRates{
						Pro: []ProFeeRate{
							{
								Level:        "Pro 1",
								TakerFeeRate: 0.00028,
								MakerFeeRate: 0.0001,
							},
							{
								Level:        "Pro 2",
								TakerFeeRate: 0.00025,
								MakerFeeRate: 0.00005,
							},
						},
						MarketMaker: []MarketMakerFeeRate{
							{
								Level:       "MM 1",
								MakerRebate: -0.00001,
							},
							{
								Level:          "MM 2",
								MakerRebate:    -0.000025,
								RPIMakerRebate: -0.00001,
							},
						},
					},
					UpdateTime: types.Time(time.UnixMilli(1788254100016)),
				},
			},
		}
		assert.Equal(t, exp, result, "GetFeeGroupStructure should decode every field")
		return
	}
	assert.NotEmpty(t, result.List, "GetFeeGroupStructure should return fee groups")
}

func TestGetOptionBaseCoins(t *testing.T) {
	t.Parallel()
	result, err := e.GetOptionBaseCoins(t.Context(), []uint64{0, 2})
	require.NoError(t, err, "GetOptionBaseCoins must not error")
	if mockTests {
		exp := &OptionBaseCoinsResponse{
			List: []OptionBaseCoin{
				{
					BaseCoin:         currency.BTC,
					QuoteCoin:        currency.USDT,
					SettleCoin:       currency.USDT,
					OptionShowName:   "BTC-Options",
					OptionOnlineTime: types.Time(time.UnixMilli(1739952000000)),
					HasSymbol:        true,
				},
				{
					BaseCoin:         currency.NewCode("NVDA"),
					QuoteCoin:        currency.USDT,
					SettleCoin:       currency.USDT,
					OptionShowName:   "NVDA-Options",
					OptionOnlineTime: types.Time(time.UnixMilli(1789675200000)),
					HasSymbol:        true,
					UnderlyingType:   2,
				},
			},
		}
		assert.Equal(t, exp, result, "GetOptionBaseCoins should decode every field")
		return
	}
	assert.NotEmpty(t, result.List, "GetOptionBaseCoins should return base coins")
}

func TestKlineItemUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var k KlineItem
	require.NoError(t, k.UnmarshalJSON([]byte(`["1691905800000","0.000301","0.0003015","0.0002995","0.0003","213303600","64084.7623"]`)), "UnmarshalJSON must not error")
	exp := KlineItem{
		StartTime:  types.Time(time.UnixMilli(1691905800000)),
		OpenPrice:  0.000301,
		HighPrice:  0.0003015,
		LowPrice:   0.0002995,
		ClosePrice: 0.0003,
		Volume:     213303600,
		Turnover:   64084.7623,
	}
	assert.Equal(t, exp, k, "UnmarshalJSON should decode every element")
	assert.Error(t, k.UnmarshalJSON([]byte(`{"startTime":"1691905800000"}`)), "UnmarshalJSON should reject an object")
	assert.Error(t, k.UnmarshalJSON([]byte(`["1691905800000","bad"]`)), "UnmarshalJSON should reject an invalid number")
}

func TestPriceKlineItemUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var k PriceKlineItem
	require.NoError(t, k.UnmarshalJSON([]byte(`["1652486400000","-0.000587","-0.000344","-0.000480","-0.000344"]`)), "UnmarshalJSON must not error")
	exp := PriceKlineItem{
		StartTime:  types.Time(time.UnixMilli(1652486400000)),
		OpenPrice:  -0.000587,
		HighPrice:  -0.000344,
		LowPrice:   -0.00048,
		ClosePrice: -0.000344,
	}
	assert.Equal(t, exp, k, "UnmarshalJSON should decode every element")
	assert.Error(t, k.UnmarshalJSON([]byte(`{"startTime":"1652486400000"}`)), "UnmarshalJSON should reject an object")
	assert.Error(t, k.UnmarshalJSON([]byte(`["bad"]`)), "UnmarshalJSON should reject an invalid timestamp")
}

func TestIntervalToString(t *testing.T) {
	t.Parallel()
	for interval, exp := range map[kline.Interval]string{kline.OneMin: "1", kline.SixHour: "360", kline.TwelveHour: "720", kline.OneDay: "D", kline.OneWeek: "W", kline.OneMonth: "M"} {
		got, err := intervalToString(interval)
		require.NoErrorf(t, err, "intervalToString must not error for %s", interval)
		assert.Equalf(t, exp, got, "intervalToString should convert %s", interval)
	}
	_, err := intervalToString(kline.SevenHour)
	assert.ErrorIs(t, err, kline.ErrUnsupportedInterval, "intervalToString should reject an interval Bybit does not offer")
}

func TestRPIOrderbookLevelUnmarshalJSON(t *testing.T) {
	t.Parallel()
	var l RPIOrderbookLevel
	require.NoError(t, l.UnmarshalJSON([]byte(`["116600.00","4.428","0.29"]`)), "UnmarshalJSON must not error")
	assert.Equal(t, RPIOrderbookLevel{Price: 116600, NonRPISize: 4.428, RPISize: 0.29}, l, "UnmarshalJSON should decode every element")
	assert.Error(t, l.UnmarshalJSON([]byte(`{"price":"116600.00"}`)), "UnmarshalJSON should reject an object")
	assert.Error(t, l.UnmarshalJSON([]byte(`["116600.00","bad","0.29"]`)), "UnmarshalJSON should reject an invalid number")
	assert.ErrorIs(t, l.UnmarshalJSON([]byte(`["116600.00","4.428"]`)), errInvalidRPIOrderbookLevel, "UnmarshalJSON should reject a level without an RPI size")
}

func TestPreListingPhaseTimeUnmarshalJSON(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name, input string
		exp         time.Time
	}{
		{name: "quoted", input: `"1773655200000"`, exp: time.UnixMilli(1773655200000)},
		{name: "bare", input: `1773655200000`, exp: time.UnixMilli(1773655200000)},
		{name: "milliseconds", input: `"1773655200123"`, exp: time.UnixMilli(1773655200123)},
		// 12 digits, which types.Time would take for a 13 digit time missing its last digit
		{name: "12 digits", input: `"946684800001"`, exp: time.UnixMilli(946684800001)},
		{name: "skipped phase", input: `"946684800000"`},
		{name: "bare skipped phase", input: `946684800000`},
		{name: "no end", input: `""`},
		{name: "zero", input: `"0"`},
		{name: "null", input: `null`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var result PreListingPhaseTime
			require.NoError(t, result.UnmarshalJSON([]byte(tc.input)), "UnmarshalJSON must not error")
			assert.Equal(t, PreListingPhaseTime(tc.exp), result, "UnmarshalJSON should decode the time as Unix milliseconds")
		})
	}
	var result PreListingPhaseTime
	assert.ErrorIs(t, result.UnmarshalJSON([]byte(`"2026-03-16"`)), types.ErrInvalidTimestampFormat, "UnmarshalJSON should reject a time that is not in Unix milliseconds")
}

func TestPreListingPhaseTimeMarshalJSON(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		input time.Time
		exp   string
	}{
		{input: time.UnixMilli(1773655200000).UTC(), exp: `"2026-03-16T10:00:00Z"`},
		{exp: `"0001-01-01T00:00:00Z"`},
	} {
		data, err := PreListingPhaseTime(tc.input).MarshalJSON()
		require.NoError(t, err, "MarshalJSON must not error")
		assert.Equal(t, tc.exp, string(data), "MarshalJSON should encode the time in RFC 3339 format")
	}
}

func TestPreListingPhaseTimeTime(t *testing.T) {
	t.Parallel()
	start := time.UnixMilli(1773655200000)
	assert.Equal(t, start, PreListingPhaseTime(start).Time(), "Time should return the phase time")
	assert.True(t, PreListingPhaseTime{}.Time().IsZero(), "Time should return the zero time for a phase without one")
}

func TestExpiryDate(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		expiry time.Time
		exp    string
	}{
		{time.Date(2026, 11, 6, 8, 0, 0, 0, time.UTC), "6NOV26"},
		{time.Date(2026, 12, 25, 8, 0, 0, 0, time.UTC), "25DEC26"},
		{time.Date(2026, 11, 7, 6, 0, 0, 0, time.FixedZone("AEDT", 11*60*60)), "6NOV26"},
	} {
		assert.Equalf(t, tc.exp, expiryDate(tc.expiry), "expiryDate should format %s", tc.expiry)
	}
}

func TestTransformSymbol(t *testing.T) {
	t.Parallel()
	tests := []struct {
		symbol         string
		baseCoin       string
		contractType   string
		item           asset.Item
		expectedSymbol string
	}{
		{
			symbol:         "POPCATUSDT",
			baseCoin:       "POPCAT",
			item:           asset.Spot,
			expectedSymbol: "POPCAT_USDT",
		},
		{
			symbol:         "BTC26SEP25-300000-P",
			item:           asset.Options,
			baseCoin:       "BTC",
			expectedSymbol: "BTC-26SEP25-300000-P",
		},
		{
			symbol:         "1000000BABYDOGEUSDT",
			item:           asset.USDTMarginedFutures,
			baseCoin:       "1000000BABYDOGE",
			expectedSymbol: "1000000BABYDOGE-USDT",
		},
		{
			symbol:         "BTC-06DEC24",
			item:           asset.USDCMarginedFutures,
			expectedSymbol: "BTC-06DEC24",
			contractType:   "LinearFutures",
		},
		{
			symbol:         "1000PEPEPERP",
			baseCoin:       "1000PEPE",
			item:           asset.USDCMarginedFutures,
			expectedSymbol: "1000PEPE-PERP",
		},
		{
			symbol:         "BTCUSD",
			baseCoin:       "BTC",
			item:           asset.CoinMarginedFutures,
			expectedSymbol: "BTC_USD",
		},
		{
			symbol:         "nothingHappens",
			item:           asset.CrossMargin,
			expectedSymbol: "nothingHappens",
		},
	}
	for i := range tests {
		t.Run(tests[i].symbol+" "+tests[i].item.String(), func(t *testing.T) {
			t.Parallel()
			ii := InstrumentInfo{
				Symbol:       tests[i].symbol,
				ContractType: tests[i].contractType,
				BaseCoin:     currency.NewCode(tests[i].baseCoin),
			}
			assert.Equal(t, tests[i].expectedSymbol, ii.transformSymbol(tests[i].item), "transformSymbol should return the expected symbol")
		})
	}
}

// TestTickerVolumes pins which currency each of Bybit's two volume figures carries. An inverse
// contract is worth one unit of its quote currency, so volume24h counts that currency there while
// turnover24h carries the base, the reverse of every other category
func TestTickerVolumes(t *testing.T) {
	t.Parallel()
	// figures from GET /v5/market/tickers, one symbol per category
	for _, tc := range []struct {
		name                string
		asset               asset.Item
		tick                TickerCommon
		wantBase, wantQuote float64
	}{
		{
			name:      "inverse reports the quote currency in volume24h",
			asset:     asset.CoinMarginedFutures,
			tick:      TickerCommon{Volume24Hour: 147122986, Turnover24Hour: 1873.7925},
			wantBase:  1873.7925,
			wantQuote: 147122986,
		},
		{
			name:      "linear reports the base currency in volume24h",
			asset:     asset.USDTMarginedFutures,
			tick:      TickerCommon{Volume24Hour: 52633.7910, Turnover24Hour: 4129941840.8562},
			wantBase:  52633.7910,
			wantQuote: 4129941840.8562,
		},
		{
			name:      "spot reports the base currency in volume24h",
			asset:     asset.Spot,
			tick:      TickerCommon{Volume24Hour: 8125.606399, Turnover24Hour: 638025011.4045},
			wantBase:  8125.606399,
			wantQuote: 638025011.4045,
		},
		{
			name:      "options report the base currency in volume24h",
			asset:     asset.Options,
			tick:      TickerCommon{Volume24Hour: 0.08, Turnover24Hour: 6269.45554303},
			wantBase:  0.08,
			wantQuote: 6269.45554303,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			baseVolume, quoteVolume := tickerVolumes(&tc.tick, tc.asset)
			assert.Equal(t, tc.wantBase, baseVolume, "tickerVolumes should return the base currency volume")
			assert.Equal(t, tc.wantQuote, quoteVolume, "tickerVolumes should return the quote currency volume")
		})
	}
}

func TestCheckTimeWindow(t *testing.T) {
	t.Parallel()
	future := time.Now().Add(time.Hour)
	for _, tc := range []struct {
		name       string
		start, end time.Time
		err        error
	}{
		{name: "ordered", start: marketTestStart, end: marketTestEnd},
		{name: "one millisecond", start: marketTestStart, end: marketTestStart.Add(time.Millisecond)},
		// Bybit answers a window that starts and ends at one time with the record at that time
		{name: "start equals end", start: marketTestStart, end: marketTestStart},
		{name: "start after end", start: marketTestEnd, end: marketTestStart, err: common.ErrStartAfterEnd},
		{name: "no end", start: marketTestStart, err: common.ErrDateUnset},
		{name: "start in the future", start: time.Now().Add(time.Hour), end: time.Now().Add(2 * time.Hour), err: common.ErrStartAfterTimeNow},
		{name: "start equals end in the future", start: future, end: future, err: common.ErrStartAfterTimeNow},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.ErrorIs(t, checkTimeWindow(tc.start, tc.end), tc.err, "checkTimeWindow should return the expected error")
		})
	}
}

// TestTimeWindowEqualBounds sends each windowed market request with its start equal to its end, which Bybit answers
// with the record at that time
func TestTimeWindowEqualBounds(t *testing.T) {
	t.Parallel()
	at := strconv.FormatInt(marketTestStart.UnixMilli(), 10)
	candles := `{"category":"linear","symbol":"BTCUSDT","list":[["` + at + `","81556.2","81756.2","81496.1","81678.4","2395.716","195515205.6596"]]}`
	priceCandles := `{"category":"linear","symbol":"BTCUSDT","list":[["` + at + `","81556.2","81756.2","81500","81678.4"]]}`
	klines := &KlineRequest{Category: cLinear, Symbol: "BTCUSDT", Interval: kline.OneHour, Start: marketTestStart, End: marketTestStart}
	for _, tc := range []struct {
		name             string
		startKey, endKey string
		result           string
		send             func(context.Context, *Exchange) (int, error)
	}{
		{name: "GetKlines", startKey: "start", endKey: "end", result: candles, send: func(ctx context.Context, ex *Exchange) (int, error) {
			resp, err := ex.GetKlines(ctx, klines)
			if err != nil {
				return 0, err
			}
			return len(resp.List), nil
		}},
		{name: "GetMarkPriceKline", startKey: "start", endKey: "end", result: priceCandles, send: func(ctx context.Context, ex *Exchange) (int, error) {
			resp, err := ex.GetMarkPriceKline(ctx, klines)
			if err != nil {
				return 0, err
			}
			return len(resp.List), nil
		}},
		{name: "GetIndexPriceKline", startKey: "start", endKey: "end", result: priceCandles, send: func(ctx context.Context, ex *Exchange) (int, error) {
			resp, err := ex.GetIndexPriceKline(ctx, klines)
			if err != nil {
				return 0, err
			}
			return len(resp.List), nil
		}},
		{name: "GetPremiumIndexPriceKline", startKey: "start", endKey: "end", result: priceCandles, send: func(ctx context.Context, ex *Exchange) (int, error) {
			resp, err := ex.GetPremiumIndexPriceKline(ctx, klines)
			if err != nil {
				return 0, err
			}
			return len(resp.List), nil
		}},
		{
			name: "GetFundingRateHistory", startKey: "startTime", endKey: "endTime",
			result: `{"category":"linear","list":[{"symbol":"BTCUSDT","fundingRate":"0.00005162","fundingRateTimestamp":"` + at + `"}]}`,
			send: func(ctx context.Context, ex *Exchange) (int, error) {
				resp, err := ex.GetFundingRateHistory(ctx, &FundingRateHistoryRequest{Category: cLinear, Symbol: "BTCUSDT", StartTime: marketTestStart, EndTime: marketTestStart})
				if err != nil {
					return 0, err
				}
				return len(resp.List), nil
			},
		},
		{
			name: "GetOpenInterestData", startKey: "startTime", endKey: "endTime",
			result: `{"symbol":"BTCUSDT","category":"linear","list":[{"openInterest":"59338.112","singleOpenInterest":"29669.056","timestamp":"` + at + `"}],"nextPageCursor":""}`,
			send: func(ctx context.Context, ex *Exchange) (int, error) {
				resp, err := ex.GetOpenInterestData(ctx, &OpenInterestRequest{Category: cLinear, Symbol: "BTCUSDT", IntervalTime: "1h", StartTime: marketTestStart, EndTime: marketTestStart})
				if err != nil {
					return 0, err
				}
				return len(resp.List), nil
			},
		},
		{
			name: "GetLongShortRatio", startKey: "startTime", endKey: "endTime",
			result: `{"list":[{"symbol":"BTCUSDT","buyRatio":"0.63","sellRatio":"0.37","timestamp":"` + at + `"}],"nextPageCursor":""}`,
			send: func(ctx context.Context, ex *Exchange) (int, error) {
				resp, err := ex.GetLongShortRatio(ctx, &LongShortRatioRequest{Category: cLinear, Symbol: "BTCUSDT", Period: "1h", StartTime: marketTestStart, EndTime: marketTestStart})
				if err != nil {
					return 0, err
				}
				return len(resp.List), nil
			},
		},
		{
			name: "GetHistoricalVolatility", startKey: "startTime", endKey: "endTime",
			result: `[{"period":7,"value":"0.30540647","time":"` + at + `"}]`,
			send: func(ctx context.Context, ex *Exchange) (int, error) {
				resp, err := ex.GetHistoricalVolatility(ctx, &HistoricalVolatilityRequest{Category: cOption, BaseCoin: currency.BTC, QuoteCoin: currency.USDT, Period: 7, StartTime: marketTestStart, EndTime: marketTestStart})
				return len(resp), err
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var gotQuery url.Values
			ex := newTransportTestExchange(t, func(w http.ResponseWriter, r *http.Request) {
				gotQuery = r.URL.Query()
				_, err := fmt.Fprint(w, `{"retCode":0,"retMsg":"OK","result":`+tc.result+`,"retExtInfo":{},"time":1792139350424}`)
				assert.NoError(t, err, "writing the response should not error")
			})
			records, err := tc.send(t.Context(), ex)
			require.NoErrorf(t, err, "%s must send a window whose start equals its end", tc.name)
			assert.Equalf(t, 1, records, "%s should return the record at that time", tc.name)
			assert.Equalf(t, at, gotQuery.Get(tc.startKey), "%s should send the start", tc.name)
			assert.Equalf(t, at, gotQuery.Get(tc.endKey), "%s should send the end", tc.name)
		})
	}
}
