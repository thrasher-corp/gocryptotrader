package binance

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/common"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/exchange/accounts"
	"github.com/thrasher-corp/gocryptotrader/exchange/order/limits"
	exchange "github.com/thrasher-corp/gocryptotrader/exchanges"
	"github.com/thrasher-corp/gocryptotrader/exchanges/asset"
	"github.com/thrasher-corp/gocryptotrader/exchanges/kline"
	"github.com/thrasher-corp/gocryptotrader/exchanges/order"
	"github.com/thrasher-corp/gocryptotrader/exchanges/orderbook"
	"github.com/thrasher-corp/gocryptotrader/exchanges/request"
	"github.com/thrasher-corp/gocryptotrader/exchanges/sharedtestvalues"
	testexch "github.com/thrasher-corp/gocryptotrader/internal/testing/exchange"
	"github.com/thrasher-corp/gocryptotrader/types"
	"github.com/thrasher-corp/gocryptotrader/types/decimal"
)

// Please supply your own keys here for due diligence testing
const (
	canManipulateRealOrders     = false
	canManipulateAPICredentials = false
	useTestNet                  = false
)

// apiCredentials holds the credentials used for due diligence testing; please supply your own
var apiCredentials = &accounts.Credentials{
	Key:    "",
	Secret: "",
}

var (
	e *Exchange

	// enabled and active tradable pairs used to test endpoints.
	spotTradablePair, marginTradablePair, usdtmTradablePair, coinmTradablePair, optionsTradablePair currency.Pair

	assetToTradablePairMap map[asset.Item]currency.Pair
)

func setFeeBuilder() *exchange.FeeBuilder {
	return &exchange.FeeBuilder{
		Amount:        1,
		FeeType:       exchange.CryptocurrencyTradeFee,
		Pair:          currency.NewPair(currency.BTC, currency.LTC),
		PurchasePrice: 1,
	}
}

// getTime returns the fixed window the mock fixtures record, or the last six days when testing live
func getTime(expanded ...bool) (startTime, endTime time.Time) {
	if mockTests {
		if len(expanded) > 0 && expanded[0] {
			return time.UnixMilli(1744103851944).UTC(), time.UnixMilli(1744190254944).UTC()
		}
		return time.UnixMilli(1744103854944).UTC(), time.UnixMilli(1744190254944).UTC()
	}
	endTime = time.Now().UTC()
	return endTime.Add(-6 * 24 * time.Hour), endTime
}

func TestGetExchangeInfo(t *testing.T) {
	t.Parallel()
	_, err := e.GetExchangeInfo(t.Context(), &ExchangeInfoRequest{Symbols: currency.Pairs{currency.NewBTCUSDT()}, SymbolStatus: "TRADING"})
	require.ErrorIs(t, err, errInvalidParameterCombination, "GetExchangeInfo must reject symbolStatus with symbols")
	_, err = e.GetExchangeInfo(t.Context(), &ExchangeInfoRequest{Symbols: currency.Pairs{currency.NewBTCUSDT()}, Permissions: []string{"SPOT"}})
	require.ErrorIs(t, err, errInvalidParameterCombination, "GetExchangeInfo must reject permissions with symbols")
	_, err = e.GetExchangeInfo(t.Context(), &ExchangeInfoRequest{Symbols: currency.Pairs{currency.EMPTYPAIR}})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetExchangeInfo must reject an empty pair")

	ensureTradablePairs(t)
	result, err := e.GetExchangeInfo(t.Context(), &ExchangeInfoRequest{Symbols: currency.Pairs{spotTradablePair}})
	require.NoError(t, err, "GetExchangeInfo must not error")
	if mockTests {
		exp := &ExchangeInfoResponse{
			Timezone:   "UTC",
			ServerTime: types.Time(time.UnixMilli(1791283064798)),
			RateLimits: []RateLimitItem{
				{
					RateLimitType:  "REQUEST_WEIGHT",
					Interval:       "MINUTE",
					IntervalNumber: 1,
					Limit:          6000,
					Count:          20,
				},
				{
					RateLimitType:  "ORDERS",
					Interval:       "SECOND",
					IntervalNumber: 10,
					Limit:          100,
				},
				{
					RateLimitType:  "ORDERS",
					Interval:       "DAY",
					IntervalNumber: 1,
					Limit:          200000,
				},
				{
					RateLimitType:  "RAW_REQUESTS",
					Interval:       "MINUTE",
					IntervalNumber: 5,
					Limit:          300000,
				},
			},
			ExchangeFilters: []ExchangeFilter{
				{
					FilterType:      "EXCHANGE_MAX_NUM_ORDERS",
					MaxNumberOrders: 1000,
				},
				{
					FilterType:          "EXCHANGE_MAX_NUM_ALGO_ORDERS",
					MaxNumberAlgoOrders: 200,
				},
				{
					FilterType:             "EXCHANGE_MAX_NUM_ICEBERG_ORDERS",
					MaxNumberIcebergOrders: 10,
				},
				{
					FilterType:          "EXCHANGE_MAX_NUM_ORDER_LISTS",
					MaxNumberOrderLists: 20,
				},
			},
			Symbols: []SymbolInfo{
				{
					Symbol:                   "BTCUSDT",
					Status:                   "TRADING",
					BaseAsset:                currency.BTC,
					BaseAssetPrecision:       8,
					QuoteAsset:               currency.USDT,
					QuotePrecision:           8,
					QuoteAssetPrecision:      8,
					BaseCommissionPrecision:  8,
					QuoteCommissionPrecision: 8,
					OrderTypes: []string{
						"LIMIT",
						"LIMIT_MAKER",
						"MARKET",
						"STOP_LOSS",
						"STOP_LOSS_LIMIT",
						"TAKE_PROFIT",
						"TAKE_PROFIT_LIMIT",
					},
					IcebergAllowed:                  true,
					OCOAllowed:                      true,
					OTOAllowed:                      true,
					OPOAllowed:                      true,
					QuoteOrderQuantityMarketAllowed: true,
					AllowTrailingStop:               true,
					CancelReplaceAllowed:            true,
					AmendAllowed:                    true,
					PegInstructionsAllowed:          true,
					IsSpotTradingAllowed:            true,
					IsMarginTradingAllowed:          true,
					Filters: []SymbolFilter{
						{
							FilterType: "PRICE_FILTER",
							MinPrice:   0.01,
							MaxPrice:   1000000,
							TickSize:   0.01,
						},
						{
							FilterType:  "LOT_SIZE",
							MinQuantity: 0.00001,
							MaxQuantity: 9000,
							StepSize:    0.00001,
						},
						{
							FilterType: "ICEBERG_PARTS",
							Limit:      100,
						},
						{
							FilterType:  "MARKET_LOT_SIZE",
							MaxQuantity: 124.7752182,
						},
						{
							FilterType:            "TRAILING_DELTA",
							MinTrailingAboveDelta: 10,
							MaxTrailingAboveDelta: 2000,
							MinTrailingBelowDelta: 10,
							MaxTrailingBelowDelta: 2000,
						},
						{
							FilterType:          "PERCENT_PRICE_BY_SIDE",
							AveragePriceMinutes: 5,
							BidMultiplierUp:     1.2,
							BidMultiplierDown:   0.5,
							AskMultiplierUp:     2,
							AskMultiplierDown:   0.8,
						},
						{
							FilterType:          "NOTIONAL",
							AveragePriceMinutes: 5,
							MinNotional:         5,
							ApplyMinToMarket:    true,
							MaxNotional:         9000000,
							ApplyMaxToMarket:    true,
						},
						{
							FilterType:      "MAX_NUM_ORDERS",
							MaxNumberOrders: 200,
						},
						{
							FilterType:          "MAX_NUM_ORDER_LISTS",
							MaxNumberOrderLists: 20,
						},
						{
							FilterType:          "MAX_NUM_ALGO_ORDERS",
							MaxNumberAlgoOrders: 5,
						},
						{
							FilterType:           "MAX_NUM_ORDER_AMENDS",
							MaxNumberOrderAmends: 10,
						},
						{
							FilterType:          "PERCENT_PRICE",
							MultiplierUp:        5,
							MultiplierDown:      0.2,
							AveragePriceMinutes: 5,
						},
						{
							FilterType:          "MIN_NOTIONAL",
							AveragePriceMinutes: 5,
							MinNotional:         5,
							ApplyToMarket:       true,
						},
						{
							FilterType:             "MAX_NUM_ICEBERG_ORDERS",
							MaxNumberIcebergOrders: 5,
						},
						{
							FilterType:  "MAX_POSITION",
							MaxPosition: 2302,
						},
						{
							FilterType: "T_PLUS_SELL",
							EndTime:    types.Time(time.UnixMilli(1741672924895)),
						},
					},
					Permissions: []string{
						"SPOT",
						"MARGIN",
					},
					PermissionSets: [][]string{
						{
							"SPOT",
							"MARGIN",
							"TRD_GRP_004",
							"TRD_GRP_005",
						},
					},
					DefaultSelfTradePreventionMode: "EXPIRE_MAKER",
					AllowedSelfTradePreventionModes: []string{
						"EXPIRE_TAKER",
						"EXPIRE_MAKER",
						"EXPIRE_BOTH",
						"DECREMENT",
						"TRANSFER",
					},
				},
			},
			SORs: []SORInfo{
				{
					BaseAsset: currency.BTC,
					Symbols: []string{
						"BTCUSDT",
						"BTCUSDC",
					},
				},
			},
		}
		assert.Equal(t, exp, result, "GetExchangeInfo should decode every field")
		return
	}
	require.Len(t, result.Symbols, 1, "GetExchangeInfo must return the requested symbol")
	assert.NotEmpty(t, result.Symbols[0].Filters, "GetExchangeInfo should return the symbol's filters")
	assert.WithinRange(t, result.ServerTime.Time(), time.Now().Add(-24*time.Hour), time.Now().Add(24*time.Hour), "ServerTime should be within a day of now")
}

func TestGetExchangeInfoFilters(t *testing.T) {
	t.Parallel()
	result, err := e.GetExchangeInfo(t.Context(), &ExchangeInfoRequest{
		Permissions:        []string{"SPOT", "MARGIN"},
		ShowPermissionSets: new(false),
		SymbolStatus:       "HALT",
	})
	require.NoError(t, err, "GetExchangeInfo must not error")
	assert.NotEmpty(t, result.RateLimits, "GetExchangeInfo should return the rate limits")
}

func TestGetExchangeServerTime(t *testing.T) {
	t.Parallel()
	result, err := e.GetExchangeServerTime(t.Context())
	require.NoError(t, err, "GetExchangeServerTime must not error")
	if mockTests {
		exp := &ServerTimeResponse{
			ServerTime: types.Time(time.UnixMilli(1775571261666)),
		}
		assert.Equal(t, exp, result, "GetExchangeServerTime should decode every field")
		return
	}
	assert.WithinRange(t, result.ServerTime.Time(), time.Now().Add(-time.Hour), time.Now().Add(time.Hour), "ServerTime should be within an hour of now")
}

func TestGetAggregatedTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetAggregatedTrades(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetAggregatedTrades must reject a nil request")
	_, err = e.GetAggregatedTrades(t.Context(), &AggregatedTradeRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetAggregatedTrades must reject an empty pair")
	startTime, err := time.Parse(time.RFC3339, "2020-01-02T15:04:05Z")
	require.NoError(t, err, "time.Parse must not error")
	_, err = e.GetAggregatedTrades(t.Context(), &AggregatedTradeRequest{Symbol: currency.NewBTCUSDT(), StartTime: startTime, EndTime: startTime.Add(75 * time.Minute), FromID: 2})
	require.ErrorIs(t, err, errInvalidParameterCombination, "GetAggregatedTrades must reject fromId with a time window")
	_, err = e.GetAggregatedTrades(t.Context(), &AggregatedTradeRequest{Symbol: currency.NewBTCUSDT(), StartTime: startTime, EndTime: startTime.Add(-time.Minute)})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetAggregatedTrades must reject a start after the end")
	for _, limit := range []uint64{1001, 5000} {
		_, err = e.GetAggregatedTrades(t.Context(), &AggregatedTradeRequest{Symbol: currency.NewBTCUSDT(), Limit: limit})
		require.ErrorIsf(t, err, errAggregatedTradesStartRequired, "GetAggregatedTrades must reject a limit of %d without fromId or startTime", limit)
	}

	ensureTradablePairs(t)
	result, err := e.GetAggregatedTrades(t.Context(), &AggregatedTradeRequest{Symbol: spotTradablePair, Limit: 5})
	require.NoError(t, err, "GetAggregatedTrades must not error")
	if mockTests {
		exp := []AggregatedTrade{
			{
				AggregateTradeID: 4081863688,
				Price:            86168.13,
				Quantity:         0.06453,
				FirstTradeID:     6739441151,
				LastTradeID:      6739441151,
				Timestamp:        types.Time(time.UnixMilli(1791284810512)),
				IsBuyerMaker:     true,
				IsBestMatch:      true,
			},
			{
				AggregateTradeID: 4081863689,
				Price:            86168.14,
				Quantity:         0.0032,
				FirstTradeID:     6739441152,
				LastTradeID:      6739441152,
				Timestamp:        types.Time(time.UnixMilli(1791284810933)),
				IsBestMatch:      true,
			},
			{
				AggregateTradeID: 4081863690,
				Price:            86168.13,
				Quantity:         0.00116,
				FirstTradeID:     6739441153,
				LastTradeID:      6739441153,
				Timestamp:        types.Time(time.UnixMilli(1791284810941)),
				IsBuyerMaker:     true,
				IsBestMatch:      true,
			},
			{
				AggregateTradeID: 4081863691,
				Price:            86168.13,
				Quantity:         0.00167,
				FirstTradeID:     6739441154,
				LastTradeID:      6739441154,
				Timestamp:        types.Time(time.UnixMilli(1791284811009)),
				IsBuyerMaker:     true,
				IsBestMatch:      true,
			},
			{
				AggregateTradeID: 4081863692,
				Price:            86168.13,
				Quantity:         0.0099,
				FirstTradeID:     6739441155,
				LastTradeID:      6739441155,
				Timestamp:        types.Time(time.UnixMilli(1791284811765)),
				IsBuyerMaker:     true,
				IsBestMatch:      true,
			},
		}
		assert.Equal(t, exp, result, "GetAggregatedTrades should decode every field")
		return
	}
	assert.Len(t, result, 5, "GetAggregatedTrades should return the limit of trades")
}

// TestGetAggregatedTradesBatched covers collecting more trades than one request returns, where each page after the first
// starts at, and so repeats, the last trade collected
func TestGetAggregatedTradesBatched(t *testing.T) {
	t.Parallel()
	if !mockTests {
		ensureTradablePairs(t)
		result, err := e.GetAggregatedTrades(t.Context(), &AggregatedTradeRequest{Symbol: spotTradablePair, StartTime: time.Now().Add(-2 * time.Hour), Limit: 1500})
		require.NoError(t, err, "GetAggregatedTrades must not error")
		require.Len(t, result, 1500, "GetAggregatedTrades must collect the limit over two pages")
		for i := 1; i < len(result); i++ {
			require.Equalf(t, result[i-1].AggregateTradeID+1, result[i].AggregateTradeID, "GetAggregatedTrades must not repeat or skip trade %d", i)
		}
		return
	}
	trades := []AggregatedTrade{
		{
			AggregateTradeID: 4081863688,
			Price:            86168.13,
			Quantity:         0.06453,
			FirstTradeID:     6739441151,
			LastTradeID:      6739441151,
			Timestamp:        types.Time(time.UnixMilli(1791284810512)),
			IsBuyerMaker:     true,
			IsBestMatch:      true,
		},
		{
			AggregateTradeID: 4081863689,
			Price:            86168.14,
			Quantity:         0.0032,
			FirstTradeID:     6739441152,
			LastTradeID:      6739441152,
			Timestamp:        types.Time(time.UnixMilli(1791284810933)),
			IsBestMatch:      true,
		},
		{
			AggregateTradeID: 4081863690,
			Price:            86168.13,
			Quantity:         0.00116,
			FirstTradeID:     6739441153,
			LastTradeID:      6739441153,
			Timestamp:        types.Time(time.UnixMilli(1791284810941)),
			IsBuyerMaker:     true,
			IsBestMatch:      true,
		},
		{
			AggregateTradeID: 4081863691,
			Price:            86168.13,
			Quantity:         0.00167,
			FirstTradeID:     6739441154,
			LastTradeID:      6739441154,
			Timestamp:        types.Time(time.UnixMilli(1791284811009)),
			IsBuyerMaker:     true,
			IsBestMatch:      true,
		},
		{
			AggregateTradeID: 4081863692,
			Price:            86168.13,
			Quantity:         0.0099,
			FirstTradeID:     6739441155,
			LastTradeID:      6739441155,
			Timestamp:        types.Time(time.UnixMilli(1791284811765)),
			IsBuyerMaker:     true,
			IsBestMatch:      true,
		},
	}
	for _, tc := range []struct {
		name    string
		endTime time.Time
		exp     []AggregatedTrade
	}{
		{name: "until a page has no new trades", exp: trades},
		{name: "until a trade after the end time", endTime: time.UnixMilli(1791284811500), exp: trades[:4]},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.GetAggregatedTrades(t.Context(), &AggregatedTradeRequest{Symbol: currency.NewBTCUSDT(), StartTime: time.UnixMilli(1791284810000), EndTime: tc.endTime, Limit: 1001})
			require.NoError(t, err, "GetAggregatedTrades must not error")
			assert.Equal(t, tc.exp, result, "GetAggregatedTrades should collect the trades of every page")
		})
	}
}

func TestGetAveragePrice(t *testing.T) {
	t.Parallel()
	_, err := e.GetAveragePrice(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetAveragePrice must reject an empty pair")

	ensureTradablePairs(t)
	result, err := e.GetAveragePrice(t.Context(), spotTradablePair)
	require.NoError(t, err, "GetAveragePrice must not error")
	if mockTests {
		exp := &AveragePriceResponse{
			Minutes:   5,
			Price:     74098.46947828,
			CloseTime: types.Time(time.UnixMilli(1773689051946)),
		}
		assert.Equal(t, exp, result, "GetAveragePrice should decode every field")
		return
	}
	assert.Positive(t, result.Price.Float64(), "GetAveragePrice should return a price")
}

func TestGetOrderBook(t *testing.T) {
	t.Parallel()
	_, err := e.GetOrderBook(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetOrderBook must reject a nil request")
	_, err = e.GetOrderBook(t.Context(), &OrderBookRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetOrderBook must reject an empty pair")

	ensureTradablePairs(t)
	result, err := e.GetOrderBook(t.Context(), &OrderBookRequest{Symbol: spotTradablePair, Limit: 5, SymbolStatus: "TRADING"})
	require.NoError(t, err, "GetOrderBook must not error")
	if mockTests {
		exp := &OrderBookResponse{
			LastUpdateID: 101096656554,
			Bids: orderbook.LevelsArrayPriceAmount{
				{
					Amount: 6.67567,
					Price:  86168.13,
				},
				{
					Amount: 0.00042,
					Price:  86168.12,
				},
				{
					Amount: 0.00014,
					Price:  86168.11,
				},
				{
					Amount: 0.05006,
					Price:  86168.1,
				},
				{
					Amount: 0.00014,
					Price:  86168.06,
				},
			},
			Asks: orderbook.LevelsArrayPriceAmount{
				{
					Amount: 4.38681,
					Price:  86168.14,
				},
				{
					Amount: 0.00048,
					Price:  86168.15,
				},
				{
					Amount: 0.00012,
					Price:  86168.45,
				},
				{
					Amount: 0.00007,
					Price:  86168.46,
				},
				{
					Amount: 0.00006,
					Price:  86169.13,
				},
			},
		}
		assert.Equal(t, exp, result, "GetOrderBook should decode every field")
		return
	}
	assert.Len(t, result.Bids, 5, "GetOrderBook should return the limit of bids")
	assert.Len(t, result.Asks, 5, "GetOrderBook should return the limit of asks")
}

func TestGetMostRecentTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetMostRecentTrades(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetMostRecentTrades must reject a nil request")
	_, err = e.GetMostRecentTrades(t.Context(), &RecentTradeRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetMostRecentTrades must reject an empty pair")

	ensureTradablePairs(t)
	result, err := e.GetMostRecentTrades(t.Context(), &RecentTradeRequest{Symbol: spotTradablePair, Limit: 15})
	require.NoError(t, err, "GetMostRecentTrades must not error")
	if mockTests {
		exp := []RecentTrade{
			{
				ID:            6739441146,
				Price:         86168.14,
				Quantity:      0.00349,
				QuoteQuantity: 300.7268086,
				Time:          types.Time(time.UnixMilli(1791284808990)),
				IsBestMatch:   true,
			},
			{
				ID:            6739441147,
				Price:         86168.13,
				Quantity:      0.00297,
				QuoteQuantity: 255.9193461,
				Time:          types.Time(time.UnixMilli(1791284809188)),
				IsBuyerMaker:  true,
				IsBestMatch:   true,
			},
			{
				ID:            6739441148,
				Price:         86168.14,
				Quantity:      0.00167,
				QuoteQuantity: 143.9007938,
				Time:          types.Time(time.UnixMilli(1791284809388)),
				IsBestMatch:   true,
			},
			{
				ID:            6739441149,
				Price:         86168.13,
				Quantity:      0.00151,
				QuoteQuantity: 130.1138763,
				Time:          types.Time(time.UnixMilli(1791284810259)),
				IsBuyerMaker:  true,
				IsBestMatch:   true,
			},
			{
				ID:            6739441150,
				Price:         86168.13,
				Quantity:      0.00349,
				QuoteQuantity: 300.7267737,
				Time:          types.Time(time.UnixMilli(1791284810342)),
				IsBuyerMaker:  true,
				IsBestMatch:   true,
			},
			{
				ID:            6739441151,
				Price:         86168.13,
				Quantity:      0.06453,
				QuoteQuantity: 5560.4294289,
				Time:          types.Time(time.UnixMilli(1791284810512)),
				IsBuyerMaker:  true,
				IsBestMatch:   true,
			},
			{
				ID:            6739441152,
				Price:         86168.14,
				Quantity:      0.0032,
				QuoteQuantity: 275.738048,
				Time:          types.Time(time.UnixMilli(1791284810933)),
				IsBestMatch:   true,
			},
			{
				ID:            6739441153,
				Price:         86168.13,
				Quantity:      0.00116,
				QuoteQuantity: 99.9550308,
				Time:          types.Time(time.UnixMilli(1791284810941)),
				IsBuyerMaker:  true,
				IsBestMatch:   true,
			},
			{
				ID:            6739441154,
				Price:         86168.13,
				Quantity:      0.00167,
				QuoteQuantity: 143.9007771,
				Time:          types.Time(time.UnixMilli(1791284811009)),
				IsBuyerMaker:  true,
				IsBestMatch:   true,
			},
			{
				ID:            6739441155,
				Price:         86168.13,
				Quantity:      0.0099,
				QuoteQuantity: 853.064487,
				Time:          types.Time(time.UnixMilli(1791284811765)),
				IsBuyerMaker:  true,
				IsBestMatch:   true,
			},
			{
				ID:            6739441156,
				Price:         86168.14,
				Quantity:      0.00274,
				QuoteQuantity: 236.1007036,
				Time:          types.Time(time.UnixMilli(1791284812050)),
				IsBestMatch:   true,
			},
			{
				ID:            6739441157,
				Price:         86168.14,
				Quantity:      0.0003,
				QuoteQuantity: 25.850442,
				Time:          types.Time(time.UnixMilli(1791284812234)),
				IsBestMatch:   true,
			},
			{
				ID:            6739441158,
				Price:         86168.14,
				Quantity:      0.00117,
				QuoteQuantity: 100.8167238,
				Time:          types.Time(time.UnixMilli(1791284812464)),
				IsBestMatch:   true,
			},
			{
				ID:            6739441159,
				Price:         86168.14,
				Quantity:      0.06453,
				QuoteQuantity: 5560.4300742,
				Time:          types.Time(time.UnixMilli(1791284812659)),
				IsBestMatch:   true,
			},
			{
				ID:            6739441160,
				Price:         86168.14,
				Quantity:      0.00167,
				QuoteQuantity: 143.9007938,
				Time:          types.Time(time.UnixMilli(1791284812770)),
				IsBestMatch:   true,
			},
		}
		assert.Equal(t, exp, result, "GetMostRecentTrades should decode every field")
		return
	}
	assert.Len(t, result, 15, "GetMostRecentTrades should return the limit of trades")
}

func TestGetHistoricalTrades(t *testing.T) {
	t.Parallel()
	_, err := e.GetHistoricalTrades(t.Context(), currency.EMPTYPAIR, 5, 0)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetHistoricalTrades must reject an empty pair")

	ensureTradablePairs(t)
	result, err := e.GetHistoricalTrades(t.Context(), spotTradablePair, 5, 0)
	require.NoError(t, err, "GetHistoricalTrades must not error")
	if mockTests {
		exp := []RecentTrade{
			{
				ID:            6739441160,
				Price:         86168.14,
				Quantity:      0.00167,
				QuoteQuantity: 143.9007938,
				Time:          types.Time(time.UnixMilli(1791284812770)),
				IsBestMatch:   true,
			},
			{
				ID:            6739441161,
				Price:         86168.13,
				Quantity:      0.00117,
				QuoteQuantity: 100.8167121,
				Time:          types.Time(time.UnixMilli(1791284813360)),
				IsBuyerMaker:  true,
				IsBestMatch:   true,
			},
			{
				ID:            6739441162,
				Price:         86168.14,
				Quantity:      0.00151,
				QuoteQuantity: 130.1138914,
				Time:          types.Time(time.UnixMilli(1791284813540)),
				IsBestMatch:   true,
			},
			{
				ID:            6739441163,
				Price:         86168.14,
				Quantity:      0.00007,
				QuoteQuantity: 6.0317698,
				Time:          types.Time(time.UnixMilli(1791284813661)),
				IsBestMatch:   true,
			},
			{
				ID:            6739441164,
				Price:         86168.13,
				Quantity:      0.02331,
				QuoteQuantity: 2008.5791103,
				Time:          types.Time(time.UnixMilli(1791284813734)),
				IsBuyerMaker:  true,
				IsBestMatch:   true,
			},
		}
		assert.Equal(t, exp, result, "GetHistoricalTrades should decode every field")
		return
	}
	require.Len(t, result, 5, "GetHistoricalTrades must return the limit of trades")

	older, err := e.GetHistoricalTrades(t.Context(), spotTradablePair, 5, result[0].ID-5)
	require.NoError(t, err, "GetHistoricalTrades must not error")
	require.NotEmpty(t, older, "GetHistoricalTrades must return trades from fromID")
	assert.Equal(t, result[0].ID-5, older[0].ID, "GetHistoricalTrades should start at fromID")
}

func TestGetSpotKline(t *testing.T) {
	t.Parallel()
	_, err := e.GetSpotKline(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetSpotKline must reject a nil request")
	_, err = e.GetSpotKline(t.Context(), &KlinesRequest{Interval: kline.FiveMin.Short()})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetSpotKline must reject an empty pair")
	_, err = e.GetSpotKline(t.Context(), &KlinesRequest{Symbol: currency.NewBTCUSDT()})
	require.ErrorIs(t, err, kline.ErrInvalidInterval, "GetSpotKline must reject an empty interval")
	startTime, endTime := getTime()
	_, err = e.GetSpotKline(t.Context(), &KlinesRequest{Symbol: currency.NewBTCUSDT(), Interval: kline.FiveMin.Short(), Limit: 24, StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetSpotKline must reject a start after the end")

	ensureTradablePairs(t)
	result, err := e.GetSpotKline(t.Context(), &KlinesRequest{Symbol: spotTradablePair, Interval: kline.FiveMin.Short(), Limit: 24, StartTime: startTime, EndTime: endTime})
	require.NoError(t, err, "GetSpotKline must not error")
	if mockTests {
		exp := []CandleStick{
			{
				OpenTime:                 types.Time(time.Unix(1744104000, 0)),
				OpenPrice:                79005.01,
				HighPrice:                79098.99,
				LowPrice:                 78992.01,
				ClosePrice:               79059.91,
				Volume:                   105.16611,
				CloseTime:                types.Time(time.UnixMilli(1744104299999)),
				QuoteAssetVolume:         8313439.0205757,
				NumberOfTrades:           13417,
				TakerBuyBaseAssetVolume:  29.27482,
				TakerBuyQuoteAssetVolume: 2314177.4476047,
			},
			{
				OpenTime:                 types.Time(time.Unix(1744104300, 0)),
				OpenPrice:                79059.9,
				HighPrice:                79059.9,
				LowPrice:                 78880.76,
				ClosePrice:               78880.76,
				Volume:                   98.53652,
				CloseTime:                types.Time(time.UnixMilli(1744104599999)),
				QuoteAssetVolume:         7779192.2943793,
				NumberOfTrades:           14277,
				TakerBuyBaseAssetVolume:  18.56935,
				TakerBuyQuoteAssetVolume: 1465573.9860208,
			},
			{
				OpenTime:                 types.Time(time.Unix(1744104600, 0)),
				OpenPrice:                78880.76,
				HighPrice:                78928,
				LowPrice:                 78869.56,
				ClosePrice:               78892.48,
				Volume:                   82.53537,
				CloseTime:                types.Time(time.UnixMilli(1744104899999)),
				QuoteAssetVolume:         6511708.0264591,
				NumberOfTrades:           9182,
				TakerBuyBaseAssetVolume:  31.3041,
				TakerBuyQuoteAssetVolume: 2469717.8651206,
			},
			{
				OpenTime:                 types.Time(time.Unix(1744104900, 0)),
				OpenPrice:                78892.47,
				HighPrice:                79074,
				LowPrice:                 78860.49,
				ClosePrice:               79019.99,
				Volume:                   86.40483,
				CloseTime:                types.Time(time.UnixMilli(1744105199999)),
				QuoteAssetVolume:         6822587.0856071,
				NumberOfTrades:           14754,
				TakerBuyBaseAssetVolume:  45.90395,
				TakerBuyQuoteAssetVolume: 3624698.2552309,
			},
			{
				OpenTime:                 types.Time(time.Unix(1744105200, 0)),
				OpenPrice:                79020,
				HighPrice:                79154,
				LowPrice:                 79004,
				ClosePrice:               79121.95,
				Volume:                   45.35898,
				CloseTime:                types.Time(time.UnixMilli(1744105499999)),
				QuoteAssetVolume:         3587835.473838,
				NumberOfTrades:           11410,
				TakerBuyBaseAssetVolume:  24.73357,
				TakerBuyQuoteAssetVolume: 1956355.5907622,
			},
		}
		assert.Equal(t, exp, result, "GetSpotKline should decode every field")
		return
	}
	assert.Len(t, result, 24, "GetSpotKline should return the limit of klines")
}

func TestGetTickerData(t *testing.T) {
	t.Parallel()
	_, err := e.GetTickerData(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetTickerData must reject a nil request")
	_, err = e.GetTickerData(t.Context(), &RollingWindowTickerRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairsEmpty, "GetTickerData must reject no pairs")
	_, err = e.GetTickerData(t.Context(), &RollingWindowTickerRequest{Symbols: currency.Pairs{currency.EMPTYPAIR}})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetTickerData must reject an empty pair")
	_, err = e.GetTickerData(t.Context(), &RollingWindowTickerRequest{Symbols: currency.Pairs{currency.NewBTCUSDT()}, WindowSize: 30 * time.Second})
	require.ErrorIs(t, err, errInvalidWindowSize, "GetTickerData must reject a window size that is not whole minutes")

	ensureTradablePairs(t)
	result, err := e.GetTickerData(t.Context(), &RollingWindowTickerRequest{Symbols: currency.Pairs{spotTradablePair}, WindowSize: 20 * time.Minute, Type: "FULL"})
	require.NoError(t, err, "GetTickerData must not error")
	if mockTests {
		exp := []TickerStatistics{
			{
				Symbol:               "BTCUSDT",
				PriceChange:          180.73,
				PriceChangePercent:   0.244,
				WeightedAveragePrice: 74160.10308976,
				OpenPrice:            73919.2,
				HighPrice:            74418.6,
				LowPrice:             73781.54,
				LastPrice:            74099.93,
				Volume:               481.53543,
				QuoteVolume:          35710717.130173,
				OpenTime:             types.Time(time.Unix(1773687840, 0)),
				CloseTime:            types.Time(time.UnixMilli(1773689091957)),
				FirstID:              6113862543,
				LastID:               6113989633,
				Count:                127091,
			},
		}
		assert.Equal(t, exp, result, "GetTickerData should decode every field")
		return
	}
	require.Len(t, result, 1, "GetTickerData must return the requested symbol")
	assert.Equal(t, 20*time.Minute, result[0].CloseTime.Time().Sub(result[0].OpenTime.Time()).Truncate(time.Minute), "GetTickerData should return statistics over the window size")
}

func TestFormatWindowSize(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		windowSize time.Duration
		exp        string
	}{
		{time.Minute, "1m"},
		{20 * time.Minute, "20m"},
		{90 * time.Minute, "90m"},
		{time.Hour, "1h"},
		{25 * time.Hour, "25h"},
		{24 * time.Hour, "1d"},
		{7 * 24 * time.Hour, "7d"},
	} {
		got, err := formatWindowSize(tc.windowSize)
		require.NoErrorf(t, err, "formatWindowSize must not error for %s", tc.windowSize)
		assert.Equalf(t, tc.exp, got, "formatWindowSize should render %s", tc.windowSize)
	}
	for _, windowSize := range []time.Duration{-time.Minute, 30 * time.Second, 90 * time.Second} {
		_, err := formatWindowSize(windowSize)
		assert.ErrorIsf(t, err, errInvalidWindowSize, "formatWindowSize should reject %s", windowSize)
	}
}

func TestGetPriceChangeStats(t *testing.T) {
	t.Parallel()
	_, err := e.GetPriceChangeStats(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetPriceChangeStats must reject a nil request")
	_, err = e.GetPriceChangeStats(t.Context(), &PriceChangeStatsRequest{Symbols: currency.Pairs{currency.EMPTYPAIR}})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetPriceChangeStats must reject an empty pair")

	ensureTradablePairs(t)
	result, err := e.GetPriceChangeStats(t.Context(), &PriceChangeStatsRequest{Symbols: currency.Pairs{spotTradablePair}})
	require.NoError(t, err, "GetPriceChangeStats must not error")
	if mockTests {
		exp := []PriceChangeStats{
			{
				Symbol:               "BTCUSDT",
				PriceChange:          -1641.24,
				PriceChangePercent:   -2.364,
				WeightedAveragePrice: 69011.84878802,
				PreviousClosePrice:   69415.23,
				LastPrice:            67773.99,
				LastQuantity:         0.00904,
				BidPrice:             67773.99,
				BidQuantity:          3.21885,
				AskPrice:             67774,
				AskQuantity:          3.28593,
				OpenPrice:            69415.23,
				HighPrice:            70351.46,
				LowPrice:             67763.64,
				Volume:               16084.48849,
				QuoteVolume:          1110020287.5044677,
				OpenTime:             types.Time(time.UnixMilli(1775485163002)),
				CloseTime:            types.Time(time.UnixMilli(1775571563002)),
				FirstID:              6189028482,
				LastID:               6193332825,
				Count:                4304344,
			},
		}
		assert.Equal(t, exp, result, "GetPriceChangeStats should decode every field")
	} else {
		require.Len(t, result, 1, "GetPriceChangeStats must return the requested symbol")
		assert.Positive(t, result[0].LastPrice.Float64(), "GetPriceChangeStats should return the last price")
	}

	result, err = e.GetPriceChangeStats(t.Context(), &PriceChangeStatsRequest{Symbols: currency.Pairs{currency.NewBTCUSDT(), currency.NewPair(currency.ETH, currency.USDT)}, Type: "MINI", SymbolStatus: "TRADING"})
	require.NoError(t, err, "GetPriceChangeStats must not error")
	if mockTests {
		exp := []PriceChangeStats{
			{
				Symbol:      "BTCUSDT",
				LastPrice:   86168.14,
				OpenPrice:   85998,
				HighPrice:   86725.12,
				LowPrice:    84972.01,
				Volume:      13647.67701,
				QuoteVolume: 1170268285.6561973,
				OpenTime:    types.Time(time.UnixMilli(1791198454002)),
				CloseTime:   types.Time(time.UnixMilli(1791284854002)),
				FirstID:     6736714475,
				LastID:      6739441316,
				Count:       2726842,
			},
			{
				Symbol:      "ETHUSDT",
				LastPrice:   2713.26,
				OpenPrice:   2713.62,
				HighPrice:   2730.28,
				LowPrice:    2679.63,
				Volume:      239165.2595,
				QuoteVolume: 646850757.573201,
				OpenTime:    types.Time(time.UnixMilli(1791198453983)),
				CloseTime:   types.Time(time.UnixMilli(1791284853983)),
				FirstID:     4408736202,
				LastID:      4410794683,
				Count:       2058482,
			},
		}
		assert.Equal(t, exp, result, "GetPriceChangeStats should decode every field")
		return
	}
	require.Len(t, result, 2, "GetPriceChangeStats must return both symbols")
	assert.Zero(t, result[0].BidPrice, "GetPriceChangeStats should leave out the bid price in the MINI type")
}

func TestGetBestPrice(t *testing.T) {
	t.Parallel()
	_, err := e.GetBestPrice(t.Context(), currency.Pairs{currency.EMPTYPAIR}, "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetBestPrice must reject an empty pair")

	ensureTradablePairs(t)
	result, err := e.GetBestPrice(t.Context(), currency.Pairs{spotTradablePair}, "")
	require.NoError(t, err, "GetBestPrice must not error")
	if mockTests {
		exp := []BookTicker{
			{
				Symbol:      "BTCUSDT",
				BidPrice:    67773.99,
				BidQuantity: 3.21877,
				AskPrice:    67774,
				AskQuantity: 3.28593,
			},
		}
		assert.Equal(t, exp, result, "GetBestPrice should decode every field")
	} else {
		require.Len(t, result, 1, "GetBestPrice must return the requested symbol")
		assert.Positive(t, result[0].BidPrice.Float64(), "GetBestPrice should return the bid price")
	}

	result, err = e.GetBestPrice(t.Context(), currency.Pairs{currency.NewBTCUSDT(), currency.NewPair(currency.ETH, currency.USDT)}, "")
	require.NoError(t, err, "GetBestPrice must not error")
	if mockTests {
		exp := []BookTicker{
			{
				Symbol:      "BTCUSDT",
				BidPrice:    86168.13,
				BidQuantity: 5.32827,
				AskPrice:    86168.14,
				AskQuantity: 5.40498,
			},
			{
				Symbol:      "ETHUSDT",
				BidPrice:    2713.25,
				BidQuantity: 133.0409,
				AskPrice:    2713.26,
				AskQuantity: 14.9436,
			},
		}
		assert.Equal(t, exp, result, "GetBestPrice should decode every field")
		return
	}
	assert.Len(t, result, 2, "GetBestPrice should return both symbols")
}

func TestGetLatestSpotPrice(t *testing.T) {
	t.Parallel()
	_, err := e.GetLatestSpotPrice(t.Context(), currency.Pairs{currency.EMPTYPAIR}, "")
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetLatestSpotPrice must reject an empty pair")

	ensureTradablePairs(t)
	result, err := e.GetLatestSpotPrice(t.Context(), currency.Pairs{spotTradablePair}, "")
	require.NoError(t, err, "GetLatestSpotPrice must not error")
	if mockTests {
		exp := []SymbolPrice{
			{
				Symbol: "BTCUSDT",
				Price:  67773.99,
			},
		}
		assert.Equal(t, exp, result, "GetLatestSpotPrice should decode every field")
	} else {
		require.Len(t, result, 1, "GetLatestSpotPrice must return the requested symbol")
		assert.Positive(t, result[0].Price.Float64(), "GetLatestSpotPrice should return the price")
	}

	result, err = e.GetLatestSpotPrice(t.Context(), currency.Pairs{currency.NewBTCUSDT(), currency.NewPair(currency.ETH, currency.USDT)}, "")
	require.NoError(t, err, "GetLatestSpotPrice must not error")
	if mockTests {
		exp := []SymbolPrice{
			{
				Symbol: "BTCUSDT",
				Price:  86168.13,
			},
			{
				Symbol: "ETHUSDT",
				Price:  2713.26,
			},
		}
		assert.Equal(t, exp, result, "GetLatestSpotPrice should decode every field")
		return
	}
	assert.Len(t, result, 2, "GetLatestSpotPrice should return both symbols")
}

func TestGetTradingDayTicker(t *testing.T) {
	t.Parallel()
	_, err := e.GetTradingDayTicker(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetTradingDayTicker must reject a nil request")
	_, err = e.GetTradingDayTicker(t.Context(), &TradingDayTickerRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairsEmpty, "GetTradingDayTicker must reject no pairs")
	_, err = e.GetTradingDayTicker(t.Context(), &TradingDayTickerRequest{Symbols: currency.Pairs{currency.EMPTYPAIR}})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetTradingDayTicker must reject an empty pair")

	ensureTradablePairs(t)
	result, err := e.GetTradingDayTicker(t.Context(), &TradingDayTickerRequest{Symbols: currency.Pairs{spotTradablePair}})
	require.NoError(t, err, "GetTradingDayTicker must not error")
	if mockTests {
		exp := []TickerStatistics{
			{
				Symbol:               "BTCUSDT",
				PriceChange:          -1079.67,
				PriceChangePercent:   -1.568,
				WeightedAveragePrice: 68571.2931115,
				OpenPrice:            68853.66,
				HighPrice:            69247.9,
				LowPrice:             67763.64,
				LastPrice:            67773.99,
				Volume:               8780.18902,
				QuoteVolume:          602068914.864819,
				OpenTime:             types.Time(time.Unix(1775520000, 0)),
				CloseTime:            types.Time(time.UnixMilli(1775606399999)),
				FirstID:              6191107400,
				LastID:               6193332824,
				Count:                2225425,
			},
		}
		assert.Equal(t, exp, result, "GetTradingDayTicker should decode every field")
	} else {
		require.Len(t, result, 1, "GetTradingDayTicker must return the requested symbol")
		assert.Positive(t, result[0].LastPrice.Float64(), "GetTradingDayTicker should return the last price")
	}

	result, err = e.GetTradingDayTicker(t.Context(), &TradingDayTickerRequest{
		Symbols:      currency.Pairs{currency.NewBTCUSDT(), currency.NewPair(currency.ETH, currency.USDT)},
		TimeZone:     "8",
		Type:         "MINI",
		SymbolStatus: "TRADING",
	})
	require.NoError(t, err, "GetTradingDayTicker must not error")
	if mockTests {
		exp := []TickerStatistics{
			{
				Symbol:      "BTCUSDT",
				OpenPrice:   85269.79,
				HighPrice:   86265.09,
				LowPrice:    84972.01,
				LastPrice:   86168.14,
				Volume:      8793.2133,
				QuoteVolume: 753326334.876306,
				OpenTime:    types.Time(time.Unix(1791216000, 0)),
				CloseTime:   types.Time(time.UnixMilli(1791302399999)),
				FirstID:     6737691985,
				LastID:      6739441312,
				Count:       1749328,
			},
			{
				Symbol:      "ETHUSDT",
				OpenPrice:   2697.23,
				HighPrice:   2723.56,
				LowPrice:    2679.63,
				LastPrice:   2713.26,
				Volume:      199158.4844,
				QuoteVolume: 538363687.412439,
				OpenTime:    types.Time(time.Unix(1791216000, 0)),
				CloseTime:   types.Time(time.UnixMilli(1791302399999)),
				FirstID:     4409421081,
				LastID:      4410794682,
				Count:       1373602,
			},
		}
		assert.Equal(t, exp, result, "GetTradingDayTicker should decode every field")
		return
	}
	require.Len(t, result, 2, "GetTradingDayTicker must return both symbols")
	assert.Zero(t, result[0].PriceChange, "GetTradingDayTicker should leave out the price change in the MINI type")
}

func TestGetUIKline(t *testing.T) {
	t.Parallel()
	_, err := e.GetUIKline(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetUIKline must reject a nil request")
	startTime, endTime := getTime()
	_, err = e.GetUIKline(t.Context(), &KlinesRequest{Symbol: currency.NewBTCUSDT(), Interval: kline.FiveMin.Short(), Limit: 3, StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetUIKline must reject a start after the end")

	ensureTradablePairs(t)
	result, err := e.GetUIKline(t.Context(), &KlinesRequest{Symbol: spotTradablePair, Interval: kline.FiveMin.Short(), Limit: 3, StartTime: startTime, EndTime: endTime, TimeZone: "8"})
	require.NoError(t, err, "GetUIKline must not error")
	if mockTests {
		exp := []CandleStick{
			{
				OpenTime:                 types.Time(time.Unix(1744104000, 0)),
				OpenPrice:                79005.01,
				HighPrice:                79098.99,
				LowPrice:                 78992.01,
				ClosePrice:               79059.91,
				Volume:                   105.16611,
				CloseTime:                types.Time(time.UnixMilli(1744104299999)),
				QuoteAssetVolume:         8313439.0205757,
				NumberOfTrades:           13417,
				TakerBuyBaseAssetVolume:  29.27482,
				TakerBuyQuoteAssetVolume: 2314177.4476047,
			},
			{
				OpenTime:                 types.Time(time.Unix(1744104300, 0)),
				OpenPrice:                79059.9,
				HighPrice:                79059.9,
				LowPrice:                 78880.76,
				ClosePrice:               78880.76,
				Volume:                   98.53652,
				CloseTime:                types.Time(time.UnixMilli(1744104599999)),
				QuoteAssetVolume:         7779192.2943793,
				NumberOfTrades:           14277,
				TakerBuyBaseAssetVolume:  18.56935,
				TakerBuyQuoteAssetVolume: 1465573.9860208,
			},
			{
				OpenTime:                 types.Time(time.Unix(1744104600, 0)),
				OpenPrice:                78880.76,
				HighPrice:                78928,
				LowPrice:                 78869.56,
				ClosePrice:               78892.48,
				Volume:                   82.53537,
				CloseTime:                types.Time(time.UnixMilli(1744104899999)),
				QuoteAssetVolume:         6511708.0264591,
				NumberOfTrades:           9182,
				TakerBuyBaseAssetVolume:  31.3041,
				TakerBuyQuoteAssetVolume: 2469717.8651206,
			},
		}
		assert.Equal(t, exp, result, "GetUIKline should decode every field")
		return
	}
	assert.Len(t, result, 3, "GetUIKline should return the limit of klines")
}

func TestCancelAllOpenOrderOnSymbol(t *testing.T) {
	t.Parallel()
	_, err := e.CancelAllOpenOrderOnSymbol(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "CancelAllOpenOrderOnSymbol must reject an empty pair")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
		ensureTradablePairs(t)
	}
	result, err := e.CancelAllOpenOrderOnSymbol(t.Context(), spotTradablePair)
	require.NoError(t, err, "CancelAllOpenOrderOnSymbol must not error")
	if mockTests {
		exp := []CancelledOpenOrder{
			{
				Symbol:                   "BTCUSDT",
				OriginalClientOrderID:    "E6APeyTJvkMvLMYMqu1KQ4",
				OrderID:                  11,
				OrderListID:              -1,
				ClientOrderID:            "pXLV6Hz6mprAcVYpVMTGgx",
				TransactTime:             types.Time(time.UnixMilli(1791284811765)),
				Price:                    85990,
				OriginalQuantity:         0.005,
				ExecutedQuantity:         0.001,
				CummulativeQuoteQuantity: 85.99,
				Status:                   "CANCELED",
				TimeInForce:              "GTC",
				Type:                     "LIMIT",
				Side:                     "BUY",
				SelfTradePreventionMode:  "DECREMENT",
				PreventedMatchID:         1,
				PreventedQuantity:        0.00058,
				StrategyID:               37463720,
				StrategyType:             1000000,
				UsedSOR:                  true,
				WorkingFloor:             "SOR",
			},
			{
				Symbol:                  "BTCUSDT",
				OriginalClientOrderID:   "A3EF2HCwxgZPFMrfwbgrhv",
				OrderID:                 13,
				OrderListID:             -1,
				ClientOrderID:           "pXLV6Hz6mprAcVYpVMTGgx",
				TransactTime:            types.Time(time.UnixMilli(1791284811765)),
				Price:                   86168.13,
				OriginalQuantity:        0.01,
				Status:                  "CANCELED",
				TimeInForce:             "GTC",
				Type:                    "STOP_LOSS_LIMIT",
				Side:                    "SELL",
				SelfTradePreventionMode: "EXPIRE_MAKER",
				IcebergQuantity:         0.002,
				StopPrice:               85000,
				TrailingDelta:           200,
				TrailingTime:            -1,
				PegPriceType:            "PRIMARY_PEG",
				PegOffsetType:           "PRICE_LEVEL",
				PegOffsetValue:          5,
				PeggedPrice:             86168.13,
			},
			{
				Symbol:            "BTCUSDT",
				OrderListID:       1929,
				ContingencyType:   "OCO",
				ListStatusType:    "ALL_DONE",
				ListOrderStatus:   "ALL_DONE",
				ListClientOrderID: "2inzWQdDvZLHbbAmAozX2N",
				TransactionTime:   types.Time(time.UnixMilli(1791284811766)),
				Orders: []OrderListOrder{
					{
						Symbol:        "BTCUSDT",
						OrderID:       20,
						ClientOrderID: "CwOOIPHSmYywx6jZX77TdL",
					},
					{
						Symbol:        "BTCUSDT",
						OrderID:       21,
						ClientOrderID: "461cPg51vQjV3zIMOXNz39",
					},
				},
				OrderReports: []OrderResponse{
					{
						Symbol:                  "BTCUSDT",
						OriginalClientOrderID:   "CwOOIPHSmYywx6jZX77TdL",
						OrderID:                 20,
						OrderListID:             1929,
						ClientOrderID:           "pXLV6Hz6mprAcVYpVMTGgx",
						TransactTime:            types.Time(time.UnixMilli(1791284811766)),
						Price:                   84000,
						OriginalQuantity:        0.002,
						Status:                  "CANCELED",
						TimeInForce:             order.GoodTillCancel,
						Type:                    "STOP_LOSS_LIMIT",
						Side:                    "SELL",
						SelfTradePreventionMode: "EXPIRE_MAKER",
						IcebergQuantity:         0.001,
						StopPrice:               84500,
						StrategyID:              1,
						StrategyType:            1000000,
						TrailingDelta:           100,
						TrailingTime:            1791284800000,
					},
					{
						Symbol:                   "BTCUSDT",
						OriginalClientOrderID:    "461cPg51vQjV3zIMOXNz39",
						OrderID:                  21,
						OrderListID:              1929,
						ClientOrderID:            "pXLV6Hz6mprAcVYpVMTGgx",
						TransactTime:             types.Time(time.UnixMilli(1791284811766)),
						Price:                    88000,
						OriginalQuantity:         0.002,
						ExecutedQuantity:         0.0005,
						CummulativeQuoteQuantity: 44,
						Status:                   "CANCELED",
						TimeInForce:              order.GoodTillCancel,
						Type:                     "LIMIT_MAKER",
						Side:                     "SELL",
						SelfTradePreventionMode:  "DECREMENT",
						IcebergQuantity:          0.001,
						PreventedMatchID:         3,
						PreventedQuantity:        0.0002,
						PegPriceType:             "MARKET_PEG",
						PegOffsetType:            "PRICE_LEVEL",
						PegOffsetValue:           1,
						PeggedPrice:              86168.14,
					},
				},
			},
		}
		assert.Equal(t, exp, result, "CancelAllOpenOrderOnSymbol should decode every field")
		return
	}
	assert.NotNil(t, result, "CancelAllOpenOrderOnSymbol should return the cancelled orders")
}

func TestNewOrder(t *testing.T) {
	t.Parallel()
	_, err := e.NewOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "NewOrder must reject a nil request")
	_, err = e.NewOrder(t.Context(), &NewOrderRequest{Side: order.Buy.String(), Type: order.Limit.String()})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "NewOrder must reject an empty pair")
	_, err = e.NewOrder(t.Context(), &NewOrderRequest{Symbol: currency.NewBTCUSDT(), Type: order.Limit.String()})
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "NewOrder must reject an empty side")
	_, err = e.NewOrder(t.Context(), &NewOrderRequest{Symbol: currency.NewBTCUSDT(), Side: order.Buy.String()})
	require.ErrorIs(t, err, order.ErrTypeIsInvalid, "NewOrder must reject an empty type")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tc := range []struct {
		name string
		req  *NewOrderRequest
		exp  *NewOrderResponse
	}{
		{
			name: "market order by quote quantity",
			req: &NewOrderRequest{
				Symbol:               currency.NewBTCUSDT(),
				Side:                 order.Buy.String(),
				Type:                 order.Market.String(),
				QuoteOrderQuantity:   100,
				NewOrderResponseType: "RESULT",
			},
			exp: &NewOrderResponse{
				Symbol:                     "BTCUSDT",
				OrderID:                    28,
				OrderListID:                -1,
				ClientOrderID:              "6gCrw2kRUAF9CvJDGP16IP",
				TransactTime:               types.Time(time.UnixMilli(1791284810512)),
				OriginalQuantity:           0.00116,
				ExecutedQuantity:           0.00058,
				OriginalQuoteOrderQuantity: 100,
				CummulativeQuoteQuantity:   49.9775154,
				Status:                     "EXPIRED_IN_MATCH",
				TimeInForce:                "GTC",
				Type:                       "MARKET",
				Side:                       "BUY",
				WorkingTime:                1791284810512,
				SelfTradePreventionMode:    "EXPIRE_TAKER",
				PreventedMatchID:           1,
				PreventedQuantity:          0.00058,
			},
		},
		{
			name: "limit order filled at once",
			req: &NewOrderRequest{
				Symbol:                  currency.NewBTCUSDT(),
				Side:                    order.Sell.String(),
				Type:                    order.Limit.String(),
				TimeInForce:             order.ImmediateOrCancel.String(),
				Quantity:                0.003,
				Price:                   86000,
				NewClientOrderID:        "gct-limit-1",
				StrategyID:              37463720,
				StrategyType:            1000000,
				NewOrderResponseType:    "FULL",
				SelfTradePreventionMode: "EXPIRE_MAKER",
			},
			exp: &NewOrderResponse{
				Symbol:                   "BTCUSDT",
				OrderID:                  29,
				OrderListID:              -1,
				ClientOrderID:            "gct-limit-1",
				TransactTime:             types.Time(time.UnixMilli(1791284810933)),
				Price:                    86000,
				OriginalQuantity:         0.003,
				ExecutedQuantity:         0.002,
				CummulativeQuoteQuantity: 172.336275,
				Status:                   "EXPIRED",
				TimeInForce:              "IOC",
				Type:                     "LIMIT",
				Side:                     "SELL",
				WorkingTime:              1791284810933,
				SelfTradePreventionMode:  "EXPIRE_MAKER",
				Fills: []OrderFill{
					{
						Price:           86168.14,
						Quantity:        0.0015,
						Commission:      0.12925221,
						CommissionAsset: currency.USDT,
						TradeID:         6739441152,
					},
					{
						Price:           86168.13,
						Quantity:        0.0005,
						Commission:      0.04308407,
						CommissionAsset: currency.USDT,
						TradeID:         6739441153,
					},
				},
				StrategyID:   37463720,
				StrategyType: 1000000,
				ExpiryReason: "UNFILLED_IOC_QUANTITY_EXPIRED",
			},
		},
		{
			name: "pegged trailing stop limit order",
			req: &NewOrderRequest{
				Symbol:               currency.NewBTCUSDT(),
				Side:                 order.Sell.String(),
				Type:                 "STOP_LOSS_LIMIT",
				TimeInForce:          order.GoodTillCancel.String(),
				Quantity:             0.01,
				StopPrice:            85000,
				TrailingDelta:        200,
				IcebergQuantity:      0.002,
				NewOrderResponseType: "RESULT",
				PegPriceType:         "PRIMARY_PEG",
				PegOffsetValue:       5,
				PegOffsetType:        "PRICE_LEVEL",
			},
			exp: &NewOrderResponse{
				Symbol:                  "BTCUSDT",
				OrderID:                 30,
				OrderListID:             -1,
				ClientOrderID:           "Rq3xvEJgCWrhxyHmmCp7I2",
				TransactTime:            types.Time(time.UnixMilli(1791284811009)),
				Price:                   86168.13,
				OriginalQuantity:        0.01,
				Status:                  "NEW",
				TimeInForce:             "GTC",
				Type:                    "STOP_LOSS_LIMIT",
				Side:                    "SELL",
				WorkingTime:             -1,
				SelfTradePreventionMode: "EXPIRE_MAKER",
				IcebergQuantity:         0.002,
				StopPrice:               85000,
				TrailingDelta:           200,
				TrailingTime:            -1,
				PegPriceType:            "PRIMARY_PEG",
				PegOffsetType:           "PRICE_LEVEL",
				PegOffsetValue:          5,
				PeggedPrice:             86168.13,
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.NewOrder(t.Context(), tc.req)
			require.NoError(t, err, "NewOrder must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "NewOrder should decode every field")
				return
			}
			assert.NotZero(t, result.OrderID, "NewOrder should return the order ID")
		})
	}
}

func TestCancelExistingOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelExistingOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CancelExistingOrder must reject a nil request")
	_, err = e.CancelExistingOrder(t.Context(), &CancelOrderRequest{OrderID: 28})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "CancelExistingOrder must reject an empty pair")
	_, err = e.CancelExistingOrder(t.Context(), &CancelOrderRequest{Symbol: currency.NewBTCUSDT()})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelExistingOrder must reject a request without an order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tc := range []struct {
		name string
		req  *CancelOrderRequest
		exp  *CancelOrderResponse
	}{
		{
			name: "by order ID",
			req: &CancelOrderRequest{
				Symbol:                currency.NewBTCUSDT(),
				OrderID:               28,
				OriginalClientOrderID: "myOrder1",
				NewClientOrderID:      "cancelMyOrder1",
				CancelRestrictions:    "ONLY_NEW",
			},
			exp: &CancelOrderResponse{
				Symbol:                  "BTCUSDT",
				OriginalClientOrderID:   "myOrder1",
				OrderID:                 28,
				OrderListID:             -1,
				ClientOrderID:           "cancelMyOrder1",
				TransactTime:            types.Time(time.UnixMilli(1791284811765)),
				Price:                   86168.13,
				OriginalQuantity:        0.01,
				Status:                  "CANCELED",
				TimeInForce:             "GTC",
				Type:                    "STOP_LOSS_LIMIT",
				Side:                    "SELL",
				SelfTradePreventionMode: "EXPIRE_MAKER",
				IcebergQuantity:         0.002,
				StopPrice:               85000,
				StrategyID:              37463720,
				StrategyType:            1000000,
				TrailingDelta:           200,
				TrailingTime:            -1,
				PegPriceType:            "PRIMARY_PEG",
				PegOffsetType:           "PRICE_LEVEL",
				PegOffsetValue:          5,
				PeggedPrice:             86168.13,
			},
		},
		{
			name: "by client order ID",
			req:  &CancelOrderRequest{Symbol: currency.NewBTCUSDT(), OriginalClientOrderID: "sBI1KM6nNtOfj5tccZSKly"},
			exp: &CancelOrderResponse{
				Symbol:                   "BTCUSDT",
				OriginalClientOrderID:    "sBI1KM6nNtOfj5tccZSKly",
				OrderID:                  31,
				OrderListID:              -1,
				ClientOrderID:            "xFvtHGmPVkAJNdr8LjSq2C",
				TransactTime:             types.Time(time.UnixMilli(1791284811765)),
				Price:                    85990,
				OriginalQuantity:         0.005,
				ExecutedQuantity:         0.001,
				CummulativeQuoteQuantity: 85.99,
				Status:                   "CANCELED",
				TimeInForce:              "GTC",
				Type:                     "LIMIT",
				Side:                     "BUY",
				SelfTradePreventionMode:  "DECREMENT",
				PreventedMatchID:         1,
				PreventedQuantity:        0.00058,
				UsedSOR:                  true,
				WorkingFloor:             "SOR",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.CancelExistingOrder(t.Context(), tc.req)
			require.NoError(t, err, "CancelExistingOrder must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "CancelExistingOrder should decode every field")
				return
			}
			assert.Equal(t, "CANCELED", result.Status, "CancelExistingOrder should cancel the order")
		})
	}
}

func TestCancelOCOOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelOCOOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CancelOCOOrder must reject a nil request")
	_, err = e.CancelOCOOrder(t.Context(), &CancelOrderListRequest{ListClientOrderID: "newderID"})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "CancelOCOOrder must reject an empty pair")
	_, err = e.CancelOCOOrder(t.Context(), &CancelOrderListRequest{Symbol: currency.NewBTCUSDT()})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelOCOOrder must reject a request without an order list ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.CancelOCOOrder(t.Context(), &CancelOrderListRequest{
		Symbol:            currency.NewPair(currency.LTC, currency.BTC),
		OrderListID:       1929,
		ListClientOrderID: "C3wyj4WVEktd7u9aVBRXcN",
		NewClientOrderID:  "cancelMyOrderList1",
	})
	require.NoError(t, err, "CancelOCOOrder must not error")
	if mockTests {
		exp := &OCOOrderResponse{
			OrderListID:       1929,
			ContingencyType:   "OCO",
			ListStatusType:    "ALL_DONE",
			ListOrderStatus:   "ALL_DONE",
			ListClientOrderID: "C3wyj4WVEktd7u9aVBRXcN",
			TransactionTime:   types.Time(time.UnixMilli(1791284811766)),
			Symbol:            "LTCBTC",
			Orders: []OrderListOrder{
				{
					Symbol:        "LTCBTC",
					OrderID:       2,
					ClientOrderID: "pO9ufTiFGg3nw2fOdgeOXa",
				},
				{
					Symbol:        "LTCBTC",
					OrderID:       3,
					ClientOrderID: "Kk7sqHb9J6mJWTMDVW7Vos",
				},
			},
			OrderReports: []OrderResponse{
				{
					Symbol:                  "LTCBTC",
					OriginalClientOrderID:   "pO9ufTiFGg3nw2fOdgeOXa",
					OrderID:                 2,
					OrderListID:             1929,
					ClientOrderID:           "unfWT8ig8i0uj6lPuYLez6",
					TransactTime:            types.Time(time.UnixMilli(1791284811766)),
					Price:                   0.0009,
					OriginalQuantity:        5,
					Status:                  "CANCELED",
					TimeInForce:             order.GoodTillCancel,
					Type:                    "STOP_LOSS_LIMIT",
					Side:                    "SELL",
					SelfTradePreventionMode: "EXPIRE_MAKER",
					IcebergQuantity:         1,
					StopPrice:               0.00095,
					StrategyID:              2,
					StrategyType:            1000001,
					TrailingDelta:           100,
					TrailingTime:            -1,
				},
				{
					Symbol:                   "LTCBTC",
					OriginalClientOrderID:    "Kk7sqHb9J6mJWTMDVW7Vos",
					OrderID:                  3,
					OrderListID:              1929,
					ClientOrderID:            "unfWT8ig8i0uj6lPuYLez6",
					TransactTime:             types.Time(time.UnixMilli(1791284811766)),
					Price:                    0.0012,
					OriginalQuantity:         5,
					ExecutedQuantity:         1,
					CummulativeQuoteQuantity: 0.0012,
					Status:                   "CANCELED",
					TimeInForce:              order.GoodTillCancel,
					Type:                     "LIMIT_MAKER",
					Side:                     "SELL",
					SelfTradePreventionMode:  "DECREMENT",
					PreventedMatchID:         2,
					PreventedQuantity:        1,
					PegPriceType:             "PRIMARY_PEG",
					PegOffsetType:            "PRICE_LEVEL",
					PegOffsetValue:           5,
					PeggedPrice:              0.0012,
				},
			},
		}
		assert.Equal(t, exp, result, "CancelOCOOrder should decode every field")
		return
	}
	assert.Equal(t, uint64(1929), result.OrderListID, "CancelOCOOrder should return the cancelled order list")
}

func TestCancelExistingOrderAndSendNewOrder(t *testing.T) {
	t.Parallel()
	_, err := e.CancelExistingOrderAndSendNewOrder(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "CancelExistingOrderAndSendNewOrder must reject a nil request")
	req := &CancelReplaceOrderRequest{TimeInForce: order.GoodTillCancel.String()}
	_, err = e.CancelExistingOrderAndSendNewOrder(t.Context(), req)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "CancelExistingOrderAndSendNewOrder must reject an empty pair")
	req.Symbol = currency.NewBTCUSDT()
	_, err = e.CancelExistingOrderAndSendNewOrder(t.Context(), req)
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "CancelExistingOrderAndSendNewOrder must reject an empty side")
	req.Side = order.Buy.String()
	_, err = e.CancelExistingOrderAndSendNewOrder(t.Context(), req)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid, "CancelExistingOrderAndSendNewOrder must reject an empty type")
	req.Type = order.Limit.String()
	_, err = e.CancelExistingOrderAndSendNewOrder(t.Context(), req)
	require.ErrorIs(t, err, errCancelReplaceModeRequired, "CancelExistingOrderAndSendNewOrder must reject an empty cancel replace mode")
	req.CancelReplaceMode = "STOP_ON_FAILURE"
	_, err = e.CancelExistingOrderAndSendNewOrder(t.Context(), req)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "CancelExistingOrderAndSendNewOrder must reject a request without the order to cancel")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tc := range []struct {
		name string
		req  *CancelReplaceOrderRequest
		exp  *CancelAndReplaceResponse
	}{
		{
			name: "limit order with every cancellation parameter",
			req: &CancelReplaceOrderRequest{
				Symbol:                      currency.NewBTCUSDT(),
				Side:                        order.Buy.String(),
				Type:                        order.Limit.String(),
				CancelReplaceMode:           "STOP_ON_FAILURE",
				TimeInForce:                 order.GoodTillCancel.String(),
				Quantity:                    0.04,
				Price:                       84000,
				CancelNewClientOrderID:      "osxN3JXAtJvKvCqGeMWMVR",
				CancelOriginalClientOrderID: "DnLo3vTAQcjha43lAZhZ0y",
				CancelOrderID:               9,
				NewClientOrderID:            "wOceeeOzNORyLiQfw7jd8S",
				StrategyID:                  1,
				StrategyType:                1000000,
				IcebergQuantity:             0.01,
				NewOrderResponseType:        "FULL",
				SelfTradePreventionMode:     "EXPIRE_MAKER",
				CancelRestrictions:          "ONLY_NEW",
				OrderRateLimitExceededMode:  "CANCEL_ONLY",
			},
			exp: &CancelAndReplaceResponse{
				CancelResult:   "SUCCESS",
				NewOrderResult: "SUCCESS",
				CancelResponse: CancelReplaceCancelResponse{
					Symbol:                  "BTCUSDT",
					OriginalClientOrderID:   "DnLo3vTAQcjha43lAZhZ0y",
					OrderID:                 9,
					OrderListID:             -1,
					ClientOrderID:           "osxN3JXAtJvKvCqGeMWMVR",
					TransactTime:            types.Time(time.UnixMilli(1791284811765)),
					Price:                   83000,
					OriginalQuantity:        0.04,
					Status:                  "CANCELED",
					TimeInForce:             "GTC",
					Type:                    "LIMIT",
					Side:                    "BUY",
					SelfTradePreventionMode: "EXPIRE_MAKER",
					OrderConditionalFields: OrderConditionalFields{
						IcebergQuantity: 0.01,
						StrategyID:      1,
						StrategyType:    1000000,
					},
				},
				NewOrderResponse: CancelReplaceNewOrderResponse{
					Symbol:                  "BTCUSDT",
					OrderID:                 10,
					OrderListID:             -1,
					ClientOrderID:           "wOceeeOzNORyLiQfw7jd8S",
					TransactTime:            types.Time(time.UnixMilli(1791284811765)),
					Price:                   84000,
					OriginalQuantity:        0.04,
					Status:                  "NEW",
					TimeInForce:             "GTC",
					Type:                    "LIMIT",
					Side:                    "BUY",
					WorkingTime:             1791284811765,
					SelfTradePreventionMode: "EXPIRE_MAKER",
					Fills:                   []OrderFill{},
					OrderConditionalFields: OrderConditionalFields{
						IcebergQuantity: 0.01,
						StrategyID:      1,
						StrategyType:    1000000,
					},
				},
			},
		},
		{
			name: "pegged trailing stop limit order",
			req: &CancelReplaceOrderRequest{
				Symbol:               currency.NewBTCUSDT(),
				Side:                 order.Sell.String(),
				Type:                 "STOP_LOSS_LIMIT",
				CancelReplaceMode:    "ALLOW_FAILURE",
				TimeInForce:          order.GoodTillCancel.String(),
				Quantity:             0.04,
				StopPrice:            85000,
				TrailingDelta:        200,
				CancelOrderID:        10,
				NewOrderResponseType: "RESULT",
				PegPriceType:         "PRIMARY_PEG",
				PegOffsetValue:       5,
				PegOffsetType:        "PRICE_LEVEL",
			},
			exp: &CancelAndReplaceResponse{
				CancelResult:   "SUCCESS",
				NewOrderResult: "SUCCESS",
				CancelResponse: CancelReplaceCancelResponse{
					Symbol:                   "BTCUSDT",
					OriginalClientOrderID:    "wOceeeOzNORyLiQfw7jd8S",
					OrderID:                  10,
					OrderListID:              -1,
					ClientOrderID:            "q5rJyJ7Ql1n6VztmYWbmnF",
					TransactTime:             types.Time(time.UnixMilli(1791284811765)),
					Price:                    85990,
					OriginalQuantity:         0.04,
					ExecutedQuantity:         0.001,
					CummulativeQuoteQuantity: 85.99,
					Status:                   "CANCELED",
					TimeInForce:              "GTC",
					Type:                     "LIMIT",
					Side:                     "BUY",
					SelfTradePreventionMode:  "DECREMENT",
					OrderConditionalFields: OrderConditionalFields{
						PreventedMatchID:  1,
						PreventedQuantity: 0.00058,
						UsedSOR:           true,
						WorkingFloor:      "SOR",
					},
				},
				NewOrderResponse: CancelReplaceNewOrderResponse{
					Symbol:                  "BTCUSDT",
					OrderID:                 11,
					OrderListID:             -1,
					ClientOrderID:           "Rq3xvEJgCWrhxyHmmCp7I2",
					TransactTime:            types.Time(time.UnixMilli(1791284811765)),
					Price:                   86168.13,
					OriginalQuantity:        0.04,
					Status:                  "NEW",
					TimeInForce:             "GTC",
					Type:                    "STOP_LOSS_LIMIT",
					Side:                    "SELL",
					WorkingTime:             -1,
					SelfTradePreventionMode: "EXPIRE_MAKER",
					OrderConditionalFields: OrderConditionalFields{
						StopPrice:      85000,
						TrailingDelta:  200,
						TrailingTime:   -1,
						PegPriceType:   "PRIMARY_PEG",
						PegOffsetType:  "PRICE_LEVEL",
						PegOffsetValue: 5,
						PeggedPrice:    86168.13,
					},
				},
			},
		},
		{
			name: "market order by quote quantity",
			req: &CancelReplaceOrderRequest{
				Symbol:                      currency.NewBTCUSDT(),
				Side:                        order.Buy.String(),
				Type:                        order.Market.String(),
				CancelReplaceMode:           "STOP_ON_FAILURE",
				QuoteOrderQuantity:          100,
				CancelOriginalClientOrderID: "DnLo3vTAQcjha43lAZhZ0y",
			},
			exp: &CancelAndReplaceResponse{
				CancelResult:   "SUCCESS",
				NewOrderResult: "SUCCESS",
				CancelResponse: CancelReplaceCancelResponse{
					Symbol:                  "BTCUSDT",
					OriginalClientOrderID:   "DnLo3vTAQcjha43lAZhZ0y",
					OrderID:                 12,
					OrderListID:             -1,
					ClientOrderID:           "Gv3LmZbmuxWxTDpYvYNBxQ",
					TransactTime:            types.Time(time.UnixMilli(1791284811765)),
					Price:                   86168.13,
					OriginalQuantity:        0.04,
					Status:                  "CANCELED",
					TimeInForce:             "GTC",
					Type:                    "STOP_LOSS_LIMIT",
					Side:                    "SELL",
					SelfTradePreventionMode: "EXPIRE_MAKER",
					OrderConditionalFields: OrderConditionalFields{
						IcebergQuantity: 0.002,
						StopPrice:       85000,
						TrailingDelta:   200,
						TrailingTime:    -1,
						PegPriceType:    "PRIMARY_PEG",
						PegOffsetType:   "PRICE_LEVEL",
						PegOffsetValue:  5,
						PeggedPrice:     86168.13,
					},
				},
				NewOrderResponse: CancelReplaceNewOrderResponse{
					Symbol:                     "BTCUSDT",
					OrderID:                    13,
					OrderListID:                -1,
					ClientOrderID:              "kXbYrb3mWVHjcRqTdW1Lw2",
					TransactTime:               types.Time(time.UnixMilli(1791284811765)),
					OriginalQuantity:           0.00116,
					ExecutedQuantity:           0.00058,
					OriginalQuoteOrderQuantity: 100,
					CummulativeQuoteQuantity:   49.9775154,
					Status:                     "EXPIRED",
					TimeInForce:                "GTC",
					Type:                       "MARKET",
					Side:                       "BUY",
					WorkingTime:                1791284811765,
					SelfTradePreventionMode:    "DECREMENT",
					Fills: []OrderFill{
						{
							Price:           86168.14,
							Quantity:        0.00058,
							Commission:      0.00000058,
							CommissionAsset: currency.BTC,
							TradeID:         6739441154,
						},
					},
					OrderConditionalFields: OrderConditionalFields{
						PreventedMatchID:  1,
						PreventedQuantity: 0.00058,
						ExpiryReason:      "INSUFFICIENT_LIQUIDITY",
					},
				},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.CancelExistingOrderAndSendNewOrder(t.Context(), tc.req)
			require.NoError(t, err, "CancelExistingOrderAndSendNewOrder must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "CancelExistingOrderAndSendNewOrder should decode every field")
				return
			}
			assert.Equal(t, "SUCCESS", result.CancelResult, "CancelExistingOrderAndSendNewOrder should cancel the order")
		})
	}

	for _, tc := range []struct {
		name       string
		statusCode int
		body       string
		code       int64
		exp        *CancelAndReplaceResponse
	}{
		{
			name:       "cancellation failed and new order placed",
			statusCode: http.StatusConflict,
			body:       `{"code":-2021,"msg":"Order cancel-replace partially failed.","data":{"cancelResult":"FAILURE","newOrderResult":"SUCCESS","cancelResponse":{"code":-2011,"msg":"Unknown order sent."},"newOrderResponse":{"symbol":"BTCUSDT","orderId":11,"orderListId":-1,"clientOrderId":"pfojJMg6IMNDKuJqDxvoxN","transactTime":1648540168818}}}`,
			code:       -2021,
			exp: &CancelAndReplaceResponse{
				CancelResult:     "FAILURE",
				NewOrderResult:   "SUCCESS",
				CancelResponse:   CancelReplaceCancelResponse{Code: -2011, Message: "Unknown order sent."},
				NewOrderResponse: CancelReplaceNewOrderResponse{Symbol: "BTCUSDT", OrderID: 11, OrderListID: -1, ClientOrderID: "pfojJMg6IMNDKuJqDxvoxN", TransactTime: types.Time(time.UnixMilli(1648540168818))},
			},
		},
		{
			name:       "cancellation failed and new order not attempted",
			statusCode: http.StatusBadRequest,
			body:       `{"code":-2022,"msg":"Order cancel-replace failed.","data":{"cancelResult":"FAILURE","newOrderResult":"NOT_ATTEMPTED","cancelResponse":{"code":-2011,"msg":"Unknown order sent."},"newOrderResponse":null}}`,
			code:       -2022,
			exp: &CancelAndReplaceResponse{
				CancelResult:   "FAILURE",
				NewOrderResult: "NOT_ATTEMPTED",
				CancelResponse: CancelReplaceCancelResponse{Code: -2011, Message: "Unknown order sent."},
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := newTestServerExchange(t, func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.statusCode)
				_, err := w.Write([]byte(tc.body))
				assert.NoError(t, err, "Write should not error")
			})
			result, err := ex.CancelExistingOrderAndSendNewOrder(t.Context(), &CancelReplaceOrderRequest{Symbol: currency.NewBTCUSDT(), Side: order.Buy.String(), Type: order.Market.String(), CancelReplaceMode: "ALLOW_FAILURE", CancelOrderID: 1, Quantity: 1})
			require.ErrorIs(t, err, errAPIResponse, "CancelExistingOrderAndSendNewOrder must return the API's error")
			var apiErr *APIError
			require.ErrorAs(t, err, &apiErr, "CancelExistingOrderAndSendNewOrder must return an APIError")
			assert.Equal(t, tc.code, apiErr.Code, "CancelExistingOrderAndSendNewOrder should return the API's error code")
			assert.Equal(t, tc.exp, result, "CancelExistingOrderAndSendNewOrder should return the outcome of both halves")
		})
	}
}

func TestCancelReplaceOutcome(t *testing.T) {
	t.Parallel()
	resp := &CancelAndReplaceResponse{CancelResult: "SUCCESS"}
	got, err := cancelReplaceOutcome(resp, nil)
	require.NoError(t, err, "cancelReplaceOutcome must not error without an error")
	assert.Same(t, resp, got, "cancelReplaceOutcome should return the reply without an error")

	got, err = cancelReplaceOutcome[CancelAndReplaceResponse](nil, request.ErrBadStatus)
	assert.ErrorIs(t, err, request.ErrBadStatus, "cancelReplaceOutcome should return an error that is not Binance's")
	assert.Nil(t, got, "cancelReplaceOutcome should return no outcome for an error that is not Binance's")

	got, err = cancelReplaceOutcome[CancelAndReplaceResponse](nil, &APIError{Code: -1013, Message: "Invalid quantity."})
	assert.ErrorIs(t, err, errAPIResponse, "cancelReplaceOutcome should return an error without data")
	assert.Nil(t, got, "cancelReplaceOutcome should return no outcome for an error without data")

	got, err = cancelReplaceOutcome[CancelAndReplaceResponse](nil, &APIError{Code: -2022, Message: "Order cancel-replace failed.", Data: json.RawMessage(`{"cancelResult":"FAILURE","newOrderResult":"NOT_ATTEMPTED"}`)})
	assert.ErrorIs(t, err, errAPIResponse, "cancelReplaceOutcome should return the error with its data")
	assert.Equal(t, &CancelAndReplaceResponse{CancelResult: "FAILURE", NewOrderResult: "NOT_ATTEMPTED"}, got, "cancelReplaceOutcome should decode the outcome from the data")

	got, err = cancelReplaceOutcome[CancelAndReplaceResponse](nil, &APIError{Code: -2022, Data: json.RawMessage(`[1]`)})
	assert.ErrorIs(t, err, errAPIResponse, "cancelReplaceOutcome should return the API's error when the data does not decode")
	assert.ErrorContains(t, err, "error decoding cancel-replace outcome", "cancelReplaceOutcome should report the data that does not decode")
	assert.Nil(t, got, "cancelReplaceOutcome should return no outcome when the data does not decode")
}

func TestAPIError(t *testing.T) {
	t.Parallel()
	err := &APIError{Code: -2013, Message: "Order does not exist."}
	assert.Equal(t, "API error response: order not found: code -2013 msg Order does not exist.", err.Error(), "Error should name the mapped error, code and message")
	assert.ErrorIs(t, err, errAPIResponse, "APIError should match errAPIResponse")
	assert.ErrorIs(t, err, order.ErrOrderNotFound, "APIError should match the error its code maps to")

	err = &APIError{Code: -1003, Message: "Way too much request weight used.", Data: json.RawMessage(`{"retryAfter":1659146400000}`)}
	assert.Equal(t, `API error response: code -1003 msg Way too much request weight used. data {"retryAfter":1659146400000}`, err.Error(), "Error should include the data")
	assert.ErrorIs(t, err, errAPIResponse, "APIError should match errAPIResponse without a mapped error")
	assert.NotErrorIs(t, err, order.ErrOrderNotFound, "APIError should not match an unmapped error")
}

func TestNewOCOOrderList(t *testing.T) {
	t.Parallel()
	_, err := e.NewOCOOrderList(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "NewOCOOrderList must reject a nil request")
	req := &OCOOrderListRequest{Above: OCOOrderListLeg{TimeInForce: order.GoodTillCancel.String()}}
	_, err = e.NewOCOOrderList(t.Context(), req)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "NewOCOOrderList must reject an empty pair")
	req.Symbol = currency.NewPair(currency.LTC, currency.BTC)
	_, err = e.NewOCOOrderList(t.Context(), req)
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "NewOCOOrderList must reject an empty side")
	req.Side = order.Sell.String()
	_, err = e.NewOCOOrderList(t.Context(), req)
	require.ErrorIs(t, err, limits.ErrAmountBelowMin, "NewOCOOrderList must reject a zero quantity")
	req.Quantity = 1
	_, err = e.NewOCOOrderList(t.Context(), req)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid, "NewOCOOrderList must reject an empty above type")
	req.Above.Type = "LIMIT_MAKER"
	_, err = e.NewOCOOrderList(t.Context(), req)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid, "NewOCOOrderList must reject an empty below type")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	result, err := e.NewOCOOrderList(t.Context(), &OCOOrderListRequest{
		Symbol:            currency.NewPair(currency.LTC, currency.BTC),
		ListClientOrderID: "lH1YDkuQKWiXVXHPSKYEIp",
		Side:              order.Sell.String(),
		Quantity:          5,
		Above: OCOOrderListLeg{
			Type:           "LIMIT_MAKER",
			ClientOrderID:  "44nZvqpemY7sVYgPYbvPih",
			Price:          0.0012,
			StrategyID:     1,
			StrategyType:   1000000,
			PegPriceType:   "PRIMARY_PEG",
			PegOffsetType:  "PRICE_LEVEL",
			PegOffsetValue: 5,
		},
		Below: OCOOrderListLeg{
			Type:            "STOP_LOSS_LIMIT",
			ClientOrderID:   "NuMp0nVYnciDiFmVqfpBqK",
			IcebergQuantity: 1,
			Price:           0.0009,
			StopPrice:       0.00095,
			TrailingDelta:   100,
			TimeInForce:     order.GoodTillCancel.String(),
			StrategyID:      2,
			StrategyType:    1000001,
		},
		NewOrderResponseType:    "FULL",
		SelfTradePreventionMode: "EXPIRE_MAKER",
	})
	require.NoError(t, err, "NewOCOOrderList must not error")
	if mockTests {
		exp := &OCOOrderResponse{
			OrderListID:       1,
			ContingencyType:   "OCO",
			ListStatusType:    "EXEC_STARTED",
			ListOrderStatus:   "EXECUTING",
			ListClientOrderID: "lH1YDkuQKWiXVXHPSKYEIp",
			TransactionTime:   types.Time(time.UnixMilli(1791284811009)),
			Symbol:            "LTCBTC",
			Orders: []OrderListOrder{
				{
					Symbol:        "LTCBTC",
					OrderID:       10,
					ClientOrderID: "NuMp0nVYnciDiFmVqfpBqK",
				},
				{
					Symbol:        "LTCBTC",
					OrderID:       11,
					ClientOrderID: "44nZvqpemY7sVYgPYbvPih",
				},
			},
			OrderReports: []OrderResponse{
				{
					Symbol:                  "LTCBTC",
					OrderID:                 10,
					OrderListID:             1,
					ClientOrderID:           "NuMp0nVYnciDiFmVqfpBqK",
					TransactTime:            types.Time(time.UnixMilli(1791284811009)),
					Price:                   0.0009,
					OriginalQuantity:        5,
					Status:                  "NEW",
					TimeInForce:             order.GoodTillCancel,
					Type:                    "STOP_LOSS_LIMIT",
					Side:                    "SELL",
					WorkingTime:             -1,
					SelfTradePreventionMode: "EXPIRE_MAKER",
					Fills:                   []OrderFill{},
					IcebergQuantity:         1,
					StopPrice:               0.00095,
					StrategyID:              2,
					StrategyType:            1000001,
					TrailingDelta:           100,
					TrailingTime:            -1,
				},
				{
					Symbol:                  "LTCBTC",
					OrderID:                 11,
					OrderListID:             1,
					ClientOrderID:           "44nZvqpemY7sVYgPYbvPih",
					TransactTime:            types.Time(time.UnixMilli(1791284811009)),
					Price:                   0.0012,
					OriginalQuantity:        5,
					Status:                  "NEW",
					TimeInForce:             order.GoodTillCancel,
					Type:                    "LIMIT_MAKER",
					Side:                    "SELL",
					WorkingTime:             1791284811009,
					SelfTradePreventionMode: "EXPIRE_MAKER",
					Fills:                   []OrderFill{},
					StrategyID:              1,
					StrategyType:            1000000,
					PegPriceType:            "PRIMARY_PEG",
					PegOffsetType:           "PRICE_LEVEL",
					PegOffsetValue:          5,
					PeggedPrice:             0.0012,
				},
			},
		}
		assert.Equal(t, exp, result, "NewOCOOrderList should decode every field")
		return
	}
	assert.Len(t, result.Orders, 2, "NewOCOOrderList should return both orders")
}

func TestNewOrderTest(t *testing.T) {
	t.Parallel()
	_, err := e.NewOrderTest(t.Context(), nil, false)
	require.ErrorIs(t, err, common.ErrNilPointer, "NewOrderTest must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.NewOrderTest(t.Context(), &NewOrderRequest{
		Symbol:      currency.NewPair(currency.LTC, currency.BTC),
		Side:        order.Buy.String(),
		Type:        order.Limit.String(),
		Price:       0.0025,
		Quantity:    100000,
		TimeInForce: order.GoodTillCancel.String(),
	}, false)
	require.NoError(t, err, "NewOrderTest must not error")
	assert.Equal(t, &OrderTestResponse{}, result, "NewOrderTest should return no commission rates unless they are requested")

	result, err = e.NewOrderTest(t.Context(), &NewOrderRequest{
		Symbol:             currency.NewPair(currency.LTC, currency.BTC),
		Side:               order.Sell.String(),
		Type:               order.Market.String(),
		QuoteOrderQuantity: 10,
	}, true)
	require.NoError(t, err, "NewOrderTest must not error")
	if mockTests {
		exp := &OrderTestResponse{
			StandardCommissionForOrder: OrderCommissionRates{
				Maker: 0.00000112,
				Taker: 0.00000114,
			},
			SpecialCommissionForOrder: OrderCommissionRates{
				Maker: 0.05,
				Taker: 0.06,
			},
			TaxCommissionForOrder: OrderCommissionRates{
				Maker: 0.00000112,
				Taker: 0.00000114,
			},
			Discount: CommissionDiscount{
				EnabledForAccount: true,
				EnabledForSymbol:  true,
				DiscountAsset:     currency.BNB,
				Discount:          0.25,
			},
		}
		assert.Equal(t, exp, result, "NewOrderTest should decode every field")
		return
	}
	assert.NotZero(t, result.StandardCommissionForOrder, "NewOrderTest should return the commission rates")
}

func TestNewOrderUsingSOR(t *testing.T) {
	t.Parallel()
	_, err := e.NewOrderUsingSOR(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "NewOrderUsingSOR must reject a nil request")
	req := &SOROrderRequest{TimeInForce: order.GoodTillCancel.String()}
	_, err = e.NewOrderUsingSOR(t.Context(), req)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "NewOrderUsingSOR must reject an empty pair")
	req.Symbol = currency.NewBTCUSDT()
	_, err = e.NewOrderUsingSOR(t.Context(), req)
	require.ErrorIs(t, err, order.ErrSideIsInvalid, "NewOrderUsingSOR must reject an empty side")
	req.Side = order.Sell.String()
	_, err = e.NewOrderUsingSOR(t.Context(), req)
	require.ErrorIs(t, err, order.ErrTypeIsInvalid, "NewOrderUsingSOR must reject an empty type")
	req.Type = order.Limit.String()
	_, err = e.NewOrderUsingSOR(t.Context(), req)
	require.ErrorIs(t, err, order.ErrAmountIsInvalid, "NewOrderUsingSOR must reject a zero quantity")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e, canManipulateRealOrders)
	}
	for _, tc := range []struct {
		name string
		req  *SOROrderRequest
		exp  *SOROrderResponse
	}{
		{
			name: "limit order",
			req: &SOROrderRequest{
				Symbol:                  currency.NewBTCUSDT(),
				Side:                    order.Buy.String(),
				Type:                    order.Limit.String(),
				TimeInForce:             order.GoodTillCancel.String(),
				Quantity:                0.5,
				Price:                   86200,
				NewClientOrderID:        "sBI1KM6nNtOfj5tccZSKly",
				StrategyID:              1,
				StrategyType:            1000000,
				IcebergQuantity:         0.1,
				NewOrderResponseType:    "FULL",
				SelfTradePreventionMode: "EXPIRE_MAKER",
			},
			exp: &SOROrderResponse{
				Symbol:                   "BTCUSDT",
				OrderID:                  2,
				OrderListID:              -1,
				ClientOrderID:            "sBI1KM6nNtOfj5tccZSKly",
				TransactTime:             types.Time(time.UnixMilli(1791284811009)),
				Price:                    86200,
				OriginalQuantity:         0.5,
				ExecutedQuantity:         0.5,
				CummulativeQuoteQuantity: 43084.07,
				Status:                   "FILLED",
				TimeInForce:              "GTC",
				Type:                     "LIMIT",
				Side:                     "BUY",
				WorkingTime:              1791284811009,
				Fills: []SORFill{
					{
						MatchType:       "ONE_PARTY_TRADE_REPORT",
						Price:           86168.14,
						Quantity:        0.4,
						Commission:      0.0004,
						CommissionAsset: currency.BTC,
						TradeID:         -1,
						AllocationID:    7,
					},
					{
						MatchType:       "ONE_PARTY_TRADE_REPORT",
						Price:           86168.14,
						Quantity:        0.1,
						Commission:      0.0001,
						CommissionAsset: currency.BTC,
						TradeID:         -1,
						AllocationID:    8,
					},
				},
				SelfTradePreventionMode: "EXPIRE_MAKER",
				IcebergQuantity:         0.1,
				StrategyID:              37463720,
				StrategyType:            1000000,
				UsedSOR:                 true,
				WorkingFloor:            "SOR",
			},
		},
		{
			name: "market order",
			req:  &SOROrderRequest{Symbol: currency.NewBTCUSDT(), Side: order.Sell.String(), Type: order.Market.String(), Quantity: 0.2, NewOrderResponseType: "RESULT"},
			exp: &SOROrderResponse{
				Symbol:                   "BTCUSDT",
				OrderID:                  3,
				OrderListID:              -1,
				ClientOrderID:            "H8pYXhQnHg7bQwB7ZGlvKA",
				TransactTime:             types.Time(time.UnixMilli(1791284811009)),
				OriginalQuantity:         0.2,
				ExecutedQuantity:         0.19942,
				CummulativeQuoteQuantity: 17183.648956,
				Status:                   "EXPIRED",
				TimeInForce:              "GTC",
				Type:                     "MARKET",
				Side:                     "SELL",
				WorkingTime:              1791284811009,
				SelfTradePreventionMode:  "DECREMENT",
				PreventedMatchID:         1,
				PreventedQuantity:        0.00058,
				UsedSOR:                  true,
				WorkingFloor:             "SOR",
				ExpiryReason:             "INSUFFICIENT_LIQUIDITY",
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result, err := e.NewOrderUsingSOR(t.Context(), tc.req)
			require.NoError(t, err, "NewOrderUsingSOR must not error")
			if mockTests {
				assert.Equal(t, tc.exp, result, "NewOrderUsingSOR should decode every field")
				return
			}
			assert.True(t, result.UsedSOR, "NewOrderUsingSOR should use smart order routing")
		})
	}
}

func TestNewOrderUsingSORTest(t *testing.T) {
	t.Parallel()
	_, err := e.NewOrderUsingSORTest(t.Context(), nil, false)
	require.ErrorIs(t, err, common.ErrNilPointer, "NewOrderUsingSORTest must reject a nil request")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	req := &SOROrderRequest{
		Symbol:      currency.NewBTCUSDT(),
		Side:        order.Buy.String(),
		Type:        order.Limit.String(),
		TimeInForce: order.GoodTillCancel.String(),
		Quantity:    0.001,
		Price:       31000,
	}
	result, err := e.NewOrderUsingSORTest(t.Context(), req, false)
	require.NoError(t, err, "NewOrderUsingSORTest must not error")
	assert.Equal(t, &OrderTestResponse{}, result, "NewOrderUsingSORTest should return no commission rates unless they are requested")

	result, err = e.NewOrderUsingSORTest(t.Context(), req, true)
	require.NoError(t, err, "NewOrderUsingSORTest must not error")
	if mockTests {
		exp := &OrderTestResponse{
			StandardCommissionForOrder: OrderCommissionRates{
				Maker: 0.00000112,
				Taker: 0.00000114,
			},
			SpecialCommissionForOrder: OrderCommissionRates{
				Maker: 0.05,
				Taker: 0.06,
			},
			TaxCommissionForOrder: OrderCommissionRates{
				Maker: 0.00000112,
				Taker: 0.00000114,
			},
			Discount: CommissionDiscount{
				EnabledForAccount: true,
				EnabledForSymbol:  true,
				DiscountAsset:     currency.BNB,
				Discount:          0.25,
			},
		}
		assert.Equal(t, exp, result, "NewOrderUsingSORTest should decode every field")
		return
	}
	assert.NotZero(t, result.StandardCommissionForOrder, "NewOrderUsingSORTest should return the commission rates")
}

func TestGetCommissionRates(t *testing.T) {
	t.Parallel()
	_, err := e.GetCommissionRates(t.Context(), currency.EMPTYPAIR)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetCommissionRates must reject an empty pair")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		ensureTradablePairs(t)
	}
	result, err := e.GetCommissionRates(t.Context(), spotTradablePair)
	require.NoError(t, err, "GetCommissionRates must not error")
	if mockTests {
		exp := &AccountCommissionResponse{
			Symbol: "BTCUSDT",
			StandardCommission: CommissionRates{
				Maker:  0.0000001,
				Taker:  0.0000002,
				Buyer:  0.0000003,
				Seller: 0.0000004,
			},
			SpecialCommission: CommissionRates{
				Maker:  0.01,
				Taker:  0.02,
				Buyer:  0.03,
				Seller: 0.04,
			},
			TaxCommission: CommissionRates{
				Maker:  0.00000112,
				Taker:  0.00000114,
				Buyer:  0.00000118,
				Seller: 0.00000116,
			},
			Discount: CommissionDiscount{
				EnabledForAccount: true,
				EnabledForSymbol:  true,
				DiscountAsset:     currency.BNB,
				Discount:          0.75,
			},
		}
		assert.Equal(t, exp, result, "GetCommissionRates should decode every field")
		return
	}
	assert.NotEmpty(t, result.Symbol, "GetCommissionRates should return the symbol")
}

func TestGetAllOCOOrders(t *testing.T) {
	t.Parallel()
	startTime, endTime := getTime()
	_, err := e.GetAllOCOOrders(t.Context(), &AllOrderListsRequest{StartTime: endTime, EndTime: startTime, Limit: 10})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetAllOCOOrders must reject a start after the end")
	_, err = e.GetAllOCOOrders(t.Context(), &AllOrderListsRequest{FromID: 1, StartTime: startTime})
	require.ErrorIs(t, err, errInvalidParameterCombination, "GetAllOCOOrders must reject fromId with a time window")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAllOCOOrders(t.Context(), &AllOrderListsRequest{StartTime: startTime, EndTime: endTime, Limit: 10})
	require.NoError(t, err, "GetAllOCOOrders must not error")
	if mockTests {
		exp := []OCOOrderResponse{
			{
				OrderListID:       29,
				ContingencyType:   "OCO",
				ListStatusType:    "EXEC_STARTED",
				ListOrderStatus:   "EXECUTING",
				ListClientOrderID: "amEEAXryFzFwYF1FeRpUoZ",
				TransactionTime:   types.Time(time.UnixMilli(1744105913483)),
				Symbol:            "LTCBTC",
				Orders: []OrderListOrder{
					{
						Symbol:        "LTCBTC",
						OrderID:       4,
						ClientOrderID: "oD7aesZqjEGlZrbtRpy5zB",
					},
					{
						Symbol:        "LTCBTC",
						OrderID:       5,
						ClientOrderID: "Jr1h6xirOxgeJOUuYQS7V3",
					},
				},
			},
			{
				OrderListID:       28,
				ContingencyType:   "OCO",
				ListStatusType:    "ALL_DONE",
				ListOrderStatus:   "ALL_DONE",
				ListClientOrderID: "hG7hFNxJV6cZy3Ze4AUT4d",
				TransactionTime:   types.Time(time.UnixMilli(1744105913407)),
				Symbol:            "LTCBTC",
				Orders: []OrderListOrder{
					{
						Symbol:        "LTCBTC",
						OrderID:       2,
						ClientOrderID: "j6lFOfbmFMRjTYA7rRJ0LP",
					},
					{
						Symbol:        "LTCBTC",
						OrderID:       3,
						ClientOrderID: "z0KCjOdditiLS5ekAFtK81",
					},
				},
			},
		}
		assert.Equal(t, exp, result, "GetAllOCOOrders should decode every field")
	} else {
		assert.NotNil(t, result, "GetAllOCOOrders should return the order lists")
	}

	result, err = e.GetAllOCOOrders(t.Context(), &AllOrderListsRequest{FromID: 29})
	require.NoError(t, err, "GetAllOCOOrders must not error")
	if mockTests {
		exp := []OCOOrderResponse{
			{
				OrderListID:       29,
				ContingencyType:   "OCO",
				ListStatusType:    "EXEC_STARTED",
				ListOrderStatus:   "EXECUTING",
				ListClientOrderID: "amEEAXryFzFwYF1FeRpUoZ",
				TransactionTime:   types.Time(time.UnixMilli(1744105913483)),
				Symbol:            "LTCBTC",
				Orders: []OrderListOrder{
					{
						Symbol:        "LTCBTC",
						OrderID:       4,
						ClientOrderID: "oD7aesZqjEGlZrbtRpy5zB",
					},
					{
						Symbol:        "LTCBTC",
						OrderID:       5,
						ClientOrderID: "Jr1h6xirOxgeJOUuYQS7V3",
					},
				},
			},
		}
		assert.Equal(t, exp, result, "GetAllOCOOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "GetAllOCOOrders should return the order lists")
}

func TestAllOrders(t *testing.T) {
	t.Parallel()
	_, err := e.AllOrders(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "AllOrders must reject a nil request")
	_, err = e.AllOrders(t.Context(), &AllOrdersRequest{})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "AllOrders must reject an empty pair")
	startTime, endTime := getTime()
	_, err = e.AllOrders(t.Context(), &AllOrdersRequest{Symbol: currency.NewBTCUSDT(), StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "AllOrders must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		ensureTradablePairs(t)
	}
	result, err := e.AllOrders(t.Context(), &AllOrdersRequest{Symbol: spotTradablePair, OrderID: 1, StartTime: startTime, EndTime: endTime, Limit: 10})
	require.NoError(t, err, "AllOrders must not error")
	if mockTests {
		exp := []TradeOrderResponse{
			{
				Symbol:                   "BTCUSDT",
				OrderID:                  1,
				OrderListID:              -1,
				ClientOrderID:            "myOrder1",
				Price:                    86000,
				OriginalQuantity:         0.01,
				ExecutedQuantity:         0.01,
				CummulativeQuoteQuantity: 860,
				Status:                   "FILLED",
				TimeInForce:              order.GoodTillCancel,
				Type:                     "LIMIT",
				Side:                     "BUY",
				Time:                     types.Time(time.Unix(1744105000, 0)),
				UpdateTime:               types.Time(time.Unix(1744105000, 0)),
				IsWorking:                true,
				WorkingTime:              1744105000000,
				SelfTradePreventionMode:  "EXPIRE_MAKER",
				IcebergQuantity:          0.002,
				StrategyID:               37463720,
				StrategyType:             1000000,
				PegPriceType:             "PRIMARY_PEG",
				PegOffsetType:            "PRICE_LEVEL",
				PegOffsetValue:           5,
				PeggedPrice:              86000,
			},
			{
				Symbol:                  "BTCUSDT",
				OrderID:                 2,
				OrderListID:             -1,
				ClientOrderID:           "myOrder2",
				Price:                   84000,
				OriginalQuantity:        0.01,
				Status:                  "NEW",
				TimeInForce:             order.GoodTillCancel,
				Type:                    "STOP_LOSS_LIMIT",
				Side:                    "SELL",
				Time:                    types.Time(time.Unix(1744106000, 0)),
				UpdateTime:              types.Time(time.Unix(1744106000, 0)),
				WorkingTime:             -1,
				SelfTradePreventionMode: "EXPIRE_MAKER",
				StopPrice:               85000,
				TrailingDelta:           200,
				TrailingTime:            -1,
			},
			{
				Symbol:                   "BTCUSDT",
				OrderID:                  3,
				OrderListID:              -1,
				ClientOrderID:            "myOrder3",
				Price:                    86000,
				OriginalQuantity:         0.01,
				ExecutedQuantity:         0.004,
				CummulativeQuoteQuantity: 344,
				Status:                   "EXPIRED",
				TimeInForce:              order.ImmediateOrCancel,
				Type:                     "LIMIT",
				Side:                     "BUY",
				Time:                     types.Time(time.Unix(1744107000, 0)),
				UpdateTime:               types.Time(time.Unix(1744107000, 0)),
				IsWorking:                true,
				WorkingTime:              1744107000000,
				SelfTradePreventionMode:  "DECREMENT",
				PreventedMatchID:         1,
				PreventedQuantity:        0.00058,
				UsedSOR:                  true,
				WorkingFloor:             "SOR",
				ExpiryReason:             "UNFILLED_IOC_QUANTITY_EXPIRED",
			},
			{
				Symbol:                     "BTCUSDT",
				OrderID:                    6,
				OrderListID:                -1,
				ClientOrderID:              "myOrder6",
				OriginalQuantity:           0.00116,
				ExecutedQuantity:           0.00058,
				OriginalQuoteOrderQuantity: 100,
				CummulativeQuoteQuantity:   49.9775154,
				Status:                     "EXPIRED",
				TimeInForce:                order.GoodTillCancel,
				Type:                       "MARKET",
				Side:                       "BUY",
				Time:                       types.Time(time.Unix(1744108000, 0)),
				UpdateTime:                 types.Time(time.Unix(1744108000, 0)),
				IsWorking:                  true,
				WorkingTime:                1744108000000,
				SelfTradePreventionMode:    "DECREMENT",
				PreventedMatchID:           1,
				PreventedQuantity:          0.00058,
				StrategyID:                 37463720,
				StrategyType:               1000000,
				ExpiryReason:               "INSUFFICIENT_LIQUIDITY",
			},
		}
		assert.Equal(t, exp, result, "AllOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "AllOrders should return the orders")
}

func TestGetAccount(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetAccount(t.Context(), true)
	require.NoError(t, err, "GetAccount must not error")
	if mockTests {
		exp := &AccountResponse{
			MakerCommission:  9,
			TakerCommission:  10,
			BuyerCommission:  2,
			SellerCommission: 3,
			CommissionRates: CommissionRates{
				Maker:  0.0009,
				Taker:  0.001,
				Buyer:  0.0002,
				Seller: 0.0003,
			},
			CanTrade:                   true,
			CanWithdraw:                true,
			CanDeposit:                 true,
			Brokered:                   true,
			RequireSelfTradePrevention: true,
			PreventSOR:                 true,
			UpdateTime:                 types.Time(time.UnixMilli(1791284811009)),
			AccountType:                "SPOT",
			Balances: []Balance{
				{
					Asset:  currency.BTC,
					Free:   decimal.MustFromString("1.25000000"),
					Locked: decimal.MustFromString("0.01000000"),
				},
				{
					Asset:  currency.USDT,
					Free:   decimal.MustFromString("86168.13000000"),
					Locked: decimal.MustFromString("120.50000000"),
				},
			},
			Permissions: []string{
				"SPOT",
			},
			UID: 354937868,
		}
		assert.Equal(t, exp, result, "GetAccount should decode every field")
		return
	}
	assert.NotEmpty(t, result.AccountType, "GetAccount should return the account type")
}

func TestOpenOrders(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		ensureTradablePairs(t)
	}
	result, err := e.OpenOrders(t.Context(), currency.EMPTYPAIR)
	require.NoError(t, err, "OpenOrders must not error")
	if mockTests {
		exp := []TradeOrderResponse{
			{
				Symbol:                  "BTCUSDT",
				OrderID:                 2,
				OrderListID:             -1,
				ClientOrderID:           "myOrder2",
				Price:                   84000,
				OriginalQuantity:        0.01,
				Status:                  "NEW",
				TimeInForce:             order.GoodTillCancel,
				Type:                    "STOP_LOSS_LIMIT",
				Side:                    "SELL",
				Time:                    types.Time(time.Unix(1744106000, 0)),
				UpdateTime:              types.Time(time.Unix(1744106000, 0)),
				WorkingTime:             -1,
				SelfTradePreventionMode: "EXPIRE_MAKER",
				IcebergQuantity:         0.002,
				StopPrice:               85000,
				StrategyID:              37463720,
				StrategyType:            1000000,
				TrailingDelta:           200,
				TrailingTime:            -1,
				PegPriceType:            "MARKET_PEG",
				PegOffsetType:           "PRICE_LEVEL",
				PegOffsetValue:          1,
				PeggedPrice:             84000,
			},
			{
				Symbol:                   "ETHUSDT",
				OrderID:                  4,
				OrderListID:              -1,
				ClientOrderID:            "myOrder4",
				Price:                    2700,
				OriginalQuantity:         0.01,
				ExecutedQuantity:         0.001,
				CummulativeQuoteQuantity: 2.7,
				Status:                   "PARTIALLY_FILLED",
				TimeInForce:              order.GoodTillCancel,
				Type:                     "LIMIT",
				Side:                     "BUY",
				Time:                     types.Time(time.Unix(1744108000, 0)),
				UpdateTime:               types.Time(time.Unix(1744108000, 0)),
				IsWorking:                true,
				WorkingTime:              1744108000000,
				SelfTradePreventionMode:  "DECREMENT",
				PreventedMatchID:         1,
				PreventedQuantity:        0.00058,
				UsedSOR:                  true,
				WorkingFloor:             "SOR",
			},
		}
		assert.Equal(t, exp, result, "OpenOrders should decode every field")
	} else {
		assert.NotNil(t, result, "OpenOrders should return the orders")
	}

	result, err = e.OpenOrders(t.Context(), spotTradablePair)
	require.NoError(t, err, "OpenOrders must not error")
	if mockTests {
		exp := []TradeOrderResponse{
			{
				Symbol:                  "BTCUSDT",
				OrderID:                 2,
				OrderListID:             -1,
				ClientOrderID:           "myOrder2",
				Price:                   84000,
				OriginalQuantity:        0.01,
				Status:                  "NEW",
				TimeInForce:             order.GoodTillCancel,
				Type:                    "STOP_LOSS_LIMIT",
				Side:                    "SELL",
				Time:                    types.Time(time.Unix(1744106000, 0)),
				UpdateTime:              types.Time(time.Unix(1744106000, 0)),
				WorkingTime:             -1,
				SelfTradePreventionMode: "EXPIRE_MAKER",
				IcebergQuantity:         0.002,
				StopPrice:               85000,
				StrategyID:              37463720,
				StrategyType:            1000000,
				TrailingDelta:           200,
				TrailingTime:            -1,
				PegPriceType:            "MARKET_PEG",
				PegOffsetType:           "PRICE_LEVEL",
				PegOffsetValue:          1,
				PeggedPrice:             84000,
			},
			{
				Symbol:                   "BTCUSDT",
				OrderID:                  5,
				OrderListID:              -1,
				ClientOrderID:            "myOrder5",
				Price:                    86000,
				OriginalQuantity:         0.01,
				ExecutedQuantity:         0.001,
				CummulativeQuoteQuantity: 86,
				Status:                   "PARTIALLY_FILLED",
				TimeInForce:              order.GoodTillCancel,
				Type:                     "LIMIT",
				Side:                     "BUY",
				Time:                     types.Time(time.Unix(1744109000, 0)),
				UpdateTime:               types.Time(time.Unix(1744109000, 0)),
				IsWorking:                true,
				WorkingTime:              1744109000000,
				SelfTradePreventionMode:  "DECREMENT",
				PreventedMatchID:         1,
				PreventedQuantity:        0.00058,
				UsedSOR:                  true,
				WorkingFloor:             "SOR",
			},
		}
		assert.Equal(t, exp, result, "OpenOrders should decode every field")
		return
	}
	assert.NotNil(t, result, "OpenOrders should return the orders")
}

func TestQueryOrder(t *testing.T) {
	t.Parallel()
	_, err := e.QueryOrder(t.Context(), currency.EMPTYPAIR, "", 1337)
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "QueryOrder must reject an empty pair")
	_, err = e.QueryOrder(t.Context(), currency.NewBTCUSDT(), "", 0)
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "QueryOrder must reject a request without an order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		ensureTradablePairs(t)
	}
	for _, tc := range []struct {
		clientOrderID string
		orderID       uint64
		exp           *TradeOrderResponse
	}{
		{
			clientOrderID: "myOrder1",
			orderID:       1337,
			exp: &TradeOrderResponse{
				Symbol:                     "BTCUSDT",
				OrderID:                    1337,
				OrderListID:                -1,
				ClientOrderID:              "myOrder1",
				OriginalQuantity:           0.00116,
				ExecutedQuantity:           0.00058,
				OriginalQuoteOrderQuantity: 100,
				CummulativeQuoteQuantity:   49.9775154,
				Status:                     "EXPIRED",
				TimeInForce:                order.GoodTillCancel,
				Type:                       "MARKET",
				Side:                       "BUY",
				Time:                       types.Time(time.UnixMilli(1791284810512)),
				UpdateTime:                 types.Time(time.UnixMilli(1791284810512)),
				IsWorking:                  true,
				WorkingTime:                1791284810512,
				SelfTradePreventionMode:    "DECREMENT",
				PreventedMatchID:           1,
				PreventedQuantity:          0.00058,
				StrategyID:                 37463720,
				StrategyType:               1000000,
				ExpiryReason:               "INSUFFICIENT_LIQUIDITY",
			},
		},
		{
			clientOrderID: "myOrder2",
			exp: &TradeOrderResponse{
				Symbol:                  "BTCUSDT",
				OrderID:                 2,
				OrderListID:             -1,
				ClientOrderID:           "myOrder2",
				Price:                   84000,
				OriginalQuantity:        0.01,
				Status:                  "NEW",
				TimeInForce:             order.GoodTillCancel,
				Type:                    "STOP_LOSS_LIMIT",
				Side:                    "SELL",
				Time:                    types.Time(time.Unix(1744106000, 0)),
				UpdateTime:              types.Time(time.Unix(1744106000, 0)),
				WorkingTime:             -1,
				SelfTradePreventionMode: "EXPIRE_MAKER",
				IcebergQuantity:         0.002,
				StopPrice:               85000,
				TrailingDelta:           200,
				TrailingTime:            -1,
				PegPriceType:            "MARKET_PEG",
				PegOffsetType:           "PRICE_LEVEL",
				PegOffsetValue:          1,
				PeggedPrice:             84000,
			},
		},
		{
			clientOrderID: "myOrder3",
			exp: &TradeOrderResponse{
				Symbol:                   "BTCUSDT",
				OrderID:                  3,
				OrderListID:              -1,
				ClientOrderID:            "myOrder3",
				Price:                    86000,
				OriginalQuantity:         0.01,
				ExecutedQuantity:         0.01,
				CummulativeQuoteQuantity: 860,
				Status:                   "FILLED",
				TimeInForce:              order.ImmediateOrCancel,
				Type:                     "LIMIT",
				Side:                     "BUY",
				Time:                     types.Time(time.Unix(1744107000, 0)),
				UpdateTime:               types.Time(time.Unix(1744107000, 0)),
				IsWorking:                true,
				WorkingTime:              1744107000000,
				SelfTradePreventionMode:  "DECREMENT",
				IcebergQuantity:          0.002,
				PreventedMatchID:         1,
				PreventedQuantity:        0.00058,
				StrategyID:               37463720,
				StrategyType:             1000000,
				UsedSOR:                  true,
				WorkingFloor:             "SOR",
				PegPriceType:             "PRIMARY_PEG",
				PegOffsetType:            "PRICE_LEVEL",
				PegOffsetValue:           5,
				PeggedPrice:              86000,
				ExpiryReason:             "UNFILLED_IOC_QUANTITY_EXPIRED",
			},
		},
	} {
		result, err := e.QueryOrder(t.Context(), spotTradablePair, tc.clientOrderID, tc.orderID)
		require.NoErrorf(t, err, "QueryOrder must not error for %s", tc.clientOrderID)
		if mockTests {
			assert.Equalf(t, tc.exp, result, "QueryOrder should decode every field for %s", tc.clientOrderID)
			continue
		}
		assert.Equalf(t, tc.clientOrderID, result.ClientOrderID, "QueryOrder should return %s", tc.clientOrderID)
	}
}

func TestGetOCOOrders(t *testing.T) {
	t.Parallel()
	_, err := e.GetOCOOrders(t.Context(), 0, "")
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetOCOOrders must reject a request without an order list ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetOCOOrders(t.Context(), 123456, "h2USkA5YQpaXHPIrkd96xE")
	require.NoError(t, err, "GetOCOOrders must not error")
	if mockTests {
		exp := &OCOOrderResponse{
			OrderListID:       123456,
			ContingencyType:   "OCO",
			ListStatusType:    "EXEC_STARTED",
			ListOrderStatus:   "EXECUTING",
			ListClientOrderID: "h2USkA5YQpaXHPIrkd96xE",
			TransactionTime:   types.Time(time.UnixMilli(1791284811009)),
			Symbol:            "LTCBTC",
			Orders: []OrderListOrder{
				{
					Symbol:        "LTCBTC",
					OrderID:       4,
					ClientOrderID: "qD1gy3kc3Gx0rihm9Y3xwS",
				},
				{
					Symbol:        "LTCBTC",
					OrderID:       5,
					ClientOrderID: "ARzZ9I00CPM8i3NhmU9Ega",
				},
			},
		}
		assert.Equal(t, exp, result, "GetOCOOrders should decode every field")
		return
	}
	assert.Equal(t, uint64(123456), result.OrderListID, "GetOCOOrders should return the order list")
}

func TestGetAllocations(t *testing.T) {
	t.Parallel()
	_, err := e.GetAllocations(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetAllocations must reject a nil request")
	_, err = e.GetAllocations(t.Context(), &AllocationsRequest{FromAllocationID: 10, OrderID: 10})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetAllocations must reject an empty pair")
	startTime, endTime := getTime()
	_, err = e.GetAllocations(t.Context(), &AllocationsRequest{Symbol: currency.NewBTCUSDT(), StartTime: endTime, EndTime: startTime})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetAllocations must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		ensureTradablePairs(t)
	}
	result, err := e.GetAllocations(t.Context(), &AllocationsRequest{Symbol: spotTradablePair, StartTime: startTime, EndTime: endTime, FromAllocationID: 10, Limit: 10, OrderID: 10})
	require.NoError(t, err, "GetAllocations must not error")
	if mockTests {
		exp := []Allocation{
			{
				Symbol:          "BTCUSDT",
				AllocationID:    10,
				AllocationType:  "SOR",
				OrderID:         10,
				OrderListID:     -1,
				Price:           86168.14,
				Quantity:        0.4,
				QuoteQuantity:   34467.256,
				Commission:      0.0004,
				CommissionAsset: currency.BTC,
				Time:            types.Time(time.Unix(1744105000, 0)),
				IsBuyer:         true,
			},
			{
				Symbol:          "BTCUSDT",
				AllocationID:    11,
				AllocationType:  "SOR",
				OrderID:         10,
				OrderListID:     -1,
				Price:           86168.14,
				Quantity:        0.1,
				QuoteQuantity:   8616.814,
				Commission:      8.616814,
				CommissionAsset: currency.USDT,
				Time:            types.Time(time.UnixMilli(1744105000001)),
				IsMaker:         true,
				IsAllocator:     true,
			},
		}
		assert.Equal(t, exp, result, "GetAllocations should decode every field")
		return
	}
	assert.NotNil(t, result, "GetAllocations should return the allocations")
}

func TestGetPreventedMatches(t *testing.T) {
	t.Parallel()
	_, err := e.GetPreventedMatches(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetPreventedMatches must reject a nil request")
	_, err = e.GetPreventedMatches(t.Context(), &PreventedMatchesRequest{OrderID: 12, Limit: 10})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetPreventedMatches must reject an empty pair")
	_, err = e.GetPreventedMatches(t.Context(), &PreventedMatchesRequest{Symbol: currency.NewBTCUSDT()})
	require.ErrorIs(t, err, order.ErrOrderIDNotSet, "GetPreventedMatches must reject a request without a prevented match or order ID")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		ensureTradablePairs(t)
	}
	result, err := e.GetPreventedMatches(t.Context(), &PreventedMatchesRequest{Symbol: spotTradablePair, OrderID: 12, FromPreventedMatchID: 1, Limit: 10})
	require.NoError(t, err, "GetPreventedMatches must not error")
	if mockTests {
		exp := []PreventedMatch{
			{
				Symbol:                  "BTCUSDT",
				PreventedMatchID:        1,
				TakerOrderID:            12,
				MakerSymbol:             "BTCUSDT",
				MakerOrderID:            3,
				TradeGroupID:            1,
				SelfTradePreventionMode: "EXPIRE_MAKER",
				Price:                   86168.13,
				MakerPreventedQuantity:  0.00058,
				TransactTime:            types.Time(time.UnixMilli(1791284810512)),
			},
		}
		assert.Equal(t, exp, result, "GetPreventedMatches should decode every field")
	} else {
		assert.NotNil(t, result, "GetPreventedMatches should return the prevented matches")
	}

	result, err = e.GetPreventedMatches(t.Context(), &PreventedMatchesRequest{Symbol: spotTradablePair, PreventedMatchID: 1})
	require.NoError(t, err, "GetPreventedMatches must not error")
	if mockTests {
		exp := []PreventedMatch{
			{
				Symbol:                  "BTCUSDT",
				PreventedMatchID:        1,
				TakerOrderID:            12,
				MakerSymbol:             "BTCUSDT",
				MakerOrderID:            3,
				TradeGroupID:            1,
				SelfTradePreventionMode: "EXPIRE_MAKER",
				Price:                   86168.13,
				MakerPreventedQuantity:  0.00058,
				TransactTime:            types.Time(time.UnixMilli(1791284810512)),
			},
		}
		assert.Equal(t, exp, result, "GetPreventedMatches should decode every field")
		return
	}
	assert.NotNil(t, result, "GetPreventedMatches should return the prevented matches")
}

func TestGetAccountTradeList(t *testing.T) {
	t.Parallel()
	_, err := e.GetAccountTradeList(t.Context(), nil)
	require.ErrorIs(t, err, common.ErrNilPointer, "GetAccountTradeList must reject a nil request")
	_, err = e.GetAccountTradeList(t.Context(), &AccountTradeListRequest{Limit: 10})
	require.ErrorIs(t, err, currency.ErrCurrencyPairEmpty, "GetAccountTradeList must reject an empty pair")
	startTime, endTime := getTime()
	_, err = e.GetAccountTradeList(t.Context(), &AccountTradeListRequest{Symbol: currency.NewBTCUSDT(), StartTime: endTime, EndTime: startTime, Limit: 10})
	require.ErrorIs(t, err, common.ErrStartAfterEnd, "GetAccountTradeList must reject a start after the end")

	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
		ensureTradablePairs(t)
	}
	result, err := e.GetAccountTradeList(t.Context(), &AccountTradeListRequest{Symbol: spotTradablePair, StartTime: startTime, EndTime: endTime, Limit: 10})
	require.NoError(t, err, "GetAccountTradeList must not error")
	if mockTests {
		exp := []AccountTrade{
			{
				Symbol:          "BTCUSDT",
				ID:              28457,
				OrderID:         100234,
				OrderListID:     -1,
				Price:           86168.14,
				Quantity:        0.0015,
				QuoteQuantity:   129.25221,
				Commission:      0.12925221,
				CommissionAsset: currency.USDT,
				Time:            types.Time(time.Unix(1744105000, 0)),
				IsBuyer:         true,
				IsMaker:         true,
				IsBestMatch:     true,
			},
			{
				Symbol:          "BTCUSDT",
				ID:              28458,
				OrderID:         100235,
				OrderListID:     29,
				Price:           86168.13,
				Quantity:        0.0005,
				QuoteQuantity:   43.084065,
				Commission:      0.00004213,
				CommissionAsset: currency.BNB,
				Time:            types.Time(time.UnixMilli(1744105000001)),
				IsBestMatch:     true,
			},
		}
		assert.Equal(t, exp, result, "GetAccountTradeList should decode every field")
	} else {
		assert.NotNil(t, result, "GetAccountTradeList should return the trades")
	}

	result, err = e.GetAccountTradeList(t.Context(), &AccountTradeListRequest{Symbol: spotTradablePair, OrderID: 100234, FromID: 28457})
	require.NoError(t, err, "GetAccountTradeList must not error")
	if mockTests {
		exp := []AccountTrade{
			{
				Symbol:          "BTCUSDT",
				ID:              28457,
				OrderID:         100234,
				OrderListID:     -1,
				Price:           86168.14,
				Quantity:        0.0015,
				QuoteQuantity:   129.25221,
				Commission:      0.12925221,
				CommissionAsset: currency.USDT,
				Time:            types.Time(time.Unix(1744105000, 0)),
				IsBuyer:         true,
				IsMaker:         true,
				IsBestMatch:     true,
			},
		}
		assert.Equal(t, exp, result, "GetAccountTradeList should decode every field")
		return
	}
	assert.NotNil(t, result, "GetAccountTradeList should return the trades")
}

func TestGetOpenOCOList(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetOpenOCOList(t.Context())
	require.NoError(t, err, "GetOpenOCOList must not error")
	if mockTests {
		exp := []OCOOrderResponse{
			{
				OrderListID:       31,
				ContingencyType:   "OCO",
				ListStatusType:    "EXEC_STARTED",
				ListOrderStatus:   "EXECUTING",
				ListClientOrderID: "wuB13fmulKj3YjdqWEcsnp",
				TransactionTime:   types.Time(time.UnixMilli(1791284811009)),
				Symbol:            "LTCBTC",
				Orders: []OrderListOrder{
					{
						Symbol:        "LTCBTC",
						OrderID:       4,
						ClientOrderID: "r3EH2N76dHfLoSZWIUw1bT",
					},
					{
						Symbol:        "LTCBTC",
						OrderID:       5,
						ClientOrderID: "Cv1SnyPD3qhqpbjpYEHbd2",
					},
				},
			},
		}
		assert.Equal(t, exp, result, "GetOpenOCOList should decode every field")
		return
	}
	assert.NotNil(t, result, "GetOpenOCOList should return the open order lists")
}

func TestGetCurrentOrderCountUsage(t *testing.T) {
	t.Parallel()
	if !mockTests {
		sharedtestvalues.SkipTestIfCredentialsUnset(t, e)
	}
	result, err := e.GetCurrentOrderCountUsage(t.Context())
	require.NoError(t, err, "GetCurrentOrderCountUsage must not error")
	if mockTests {
		exp := []RateLimitItem{
			{
				RateLimitType:  "ORDERS",
				Interval:       "SECOND",
				IntervalNumber: 10,
				Limit:          50,
				Count:          3,
			},
			{
				RateLimitType:  "ORDERS",
				Interval:       "DAY",
				IntervalNumber: 1,
				Limit:          160000,
				Count:          1290,
			},
		}
		assert.Equal(t, exp, result, "GetCurrentOrderCountUsage should decode every field")
		return
	}
	assert.NotEmpty(t, result, "GetCurrentOrderCountUsage should return the unfilled order counts")
}

func TestFormatWithdrawPermissions(t *testing.T) {
	t.Parallel()
	expectedResult := exchange.AutoWithdrawCryptoText + " & " + exchange.NoFiatWithdrawalsText
	withdrawPermissions := e.FormatWithdrawPermissions()
	require.Equal(t, expectedResult, withdrawPermissions, "FormatWithdrawPermissions must return the withdrawal permissions")
}

// newTestServerExchange returns an exchange with placeholder credentials whose spot REST URL points at a test server
// running handler
func newTestServerExchange(t *testing.T, handler http.HandlerFunc) *Exchange {
	t.Helper()
	ex := new(Exchange)
	require.NoError(t, testexch.Setup(ex), "Setup must not error")
	ex.Name = t.Name()
	ex.API.AuthenticatedSupport = true
	ex.SetCredentials(&accounts.Credentials{Key: "test-api-key", Secret: "test-api-secret"})
	server := httptest.NewTestServer(t, handler)
	require.NoError(t, ex.SetHTTPClient(server.Client()), "SetHTTPClient must not error")
	require.NoError(t, ex.API.Endpoints.SetRunningURL(exchange.RestSpotSupplementary.String(), server.URL), "SetRunningURL must not error")
	return ex
}

func TestSendHTTPRequest(t *testing.T) {
	t.Parallel()
	ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method, "SendHTTPRequest should send a GET request")
		assert.Equal(t, "/api/v3/avgPrice", r.URL.Path, "SendHTTPRequest should request the path")
		assert.Equal(t, "symbol=BTCUSDT", r.URL.RawQuery, "SendHTTPRequest should send the query unchanged")
		assert.Empty(t, r.Header.Get("X-MBX-APIKEY"), "SendHTTPRequest should not send the API key")
		_, err := w.Write([]byte(`{"mins":5,"price":"86164.33298681","closeTime":1791284810512}`))
		assert.NoError(t, err, "Write should not error")
	})
	var result *AveragePriceResponse
	require.NoError(t, ex.SendHTTPRequest(t.Context(), exchange.RestSpotSupplementary, "/api/v3/avgPrice?symbol=BTCUSDT", spotDefaultRate, &result), "SendHTTPRequest must not error")
	exp := &AveragePriceResponse{Minutes: 5, Price: 86164.33298681, CloseTime: types.Time(time.UnixMilli(1791284810512))}
	assert.Equal(t, exp, result, "SendHTTPRequest should decode the response")
}

func TestSendRequestErrorResponse(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name       string
		statusCode int
		body       string
		errs       []error
		notErrs    []error
		msg        string
	}{
		{
			name:       "mapped invalid symbol code",
			statusCode: http.StatusBadRequest,
			body:       `{"code":-1121,"msg":"Invalid symbol."}`,
			errs:       []error{errAPIResponse, currency.ErrPairNotFound, request.ErrBadStatus},
			msg:        "code -1121 msg Invalid symbol.",
		},
		{
			name:       "mapped unauthorised code",
			statusCode: http.StatusUnauthorized,
			body:       `{"code":-1002,"msg":"You are not authorized to execute this request."}`,
			errs:       []error{errAPIResponse, request.ErrAuthRequestFailed, request.ErrBadStatus},
			msg:        "code -1002 msg You are not authorized to execute this request.",
		},
		{
			name:       "mapped unknown order code",
			statusCode: http.StatusBadRequest,
			body:       `{"code":-2013,"msg":"Order does not exist."}`,
			errs:       []error{errAPIResponse, order.ErrOrderNotFound, request.ErrBadStatus},
			msg:        "code -2013 msg Order does not exist.",
		},
		{
			name:       "unmapped code",
			statusCode: http.StatusBadRequest,
			body:       `{"code":-1100,"msg":"Illegal characters found in parameter 'symbols'."}`,
			errs:       []error{errAPIResponse, request.ErrBadStatus},
			notErrs:    []error{currency.ErrPairNotFound, request.ErrAuthRequestFailed, order.ErrOrderNotFound},
			msg:        "code -1100 msg Illegal characters found in parameter 'symbols'.",
		},
		{
			name:       "negative code with a success status",
			statusCode: http.StatusOK,
			body:       `{"code":-2011,"msg":"Unknown order sent."}`,
			errs:       []error{errAPIResponse},
			notErrs:    []error{request.ErrBadStatus},
			msg:        "code -2011 msg Unknown order sent.",
		},
		{
			name:       "error status without an error object",
			statusCode: http.StatusInternalServerError,
			body:       `{"someField":"value"}`,
			errs:       []error{request.ErrBadStatus},
			notErrs:    []error{errAPIResponse},
		},
		{
			name:       "error status without a JSON body",
			statusCode: http.StatusBadGateway,
			body:       `<html><body>Bad Gateway</body></html>`,
			errs:       []error{request.ErrBadStatus},
			notErrs:    []error{errAPIResponse},
		},
		{
			name:       "null body",
			statusCode: http.StatusOK,
			body:       `null`,
			errs:       []error{common.ErrNoResponse},
			notErrs:    []error{errAPIResponse},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := newTestServerExchange(t, func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.statusCode)
				_, err := w.Write([]byte(tc.body))
				assert.NoError(t, err, "Write should not error")
			})
			var result *ErrResponse
			err := ex.SendHTTPRequest(t.Context(), exchange.RestSpotSupplementary, "/api/v3/time", spotDefaultRate, &result)
			for _, want := range tc.errs {
				require.ErrorIs(t, err, want, "SendHTTPRequest must return the expected error")
			}
			for _, unwanted := range tc.notErrs {
				assert.NotErrorIs(t, err, unwanted, "SendHTTPRequest should not return an unrelated error")
			}
			if tc.msg != "" {
				assert.ErrorContains(t, err, tc.msg, "SendHTTPRequest should report the code and message")
			}
		})
	}
}

func TestSendRequestSuccessMessage(t *testing.T) {
	t.Parallel()
	for _, body := range []string{`{"code":200,"msg":"success"}`, `{"code":0,"msg":"success"}`} {
		t.Run(body, func(t *testing.T) {
			t.Parallel()
			ex := newTestServerExchange(t, func(w http.ResponseWriter, _ *http.Request) {
				_, err := w.Write([]byte(body))
				assert.NoError(t, err, "Write should not error")
			})
			var result *ErrResponse
			require.NoError(t, ex.SendHTTPRequest(t.Context(), exchange.RestSpotSupplementary, "/sapi/v1/test", spotDefaultRate, &result), "SendHTTPRequest must not error")
			require.NotNil(t, result, "SendHTTPRequest must decode the response")
			assert.Equal(t, "success", result.Message, "SendHTTPRequest should decode the message")
		})
	}
}

func TestSendAuthHTTPRequest(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name       string
		method     string
		recvWindow string
		exp        string
	}{
		{name: "default receive window", method: http.MethodDelete, exp: "5000"},
		{name: "receive window set by the caller", method: http.MethodPost, recvWindow: "10000", exp: "10000"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
				assert.Equal(t, tc.method, r.Method, "SendAuthHTTPRequest should send the method")
				assert.Equal(t, "/api/v3/order", r.URL.Path, "SendAuthHTTPRequest should request the path")
				assert.Equal(t, "test-api-key", r.Header.Get("X-MBX-APIKEY"), "SendAuthHTTPRequest should send the API key")
				payload, signature, found := strings.Cut(r.URL.RawQuery, "&signature=")
				assert.True(t, found, "SendAuthHTTPRequest should send the signature last")
				mac := hmac.New(sha256.New, []byte("test-api-secret"))
				_, err := mac.Write([]byte(payload))
				assert.NoError(t, err, "Write should not error")
				assert.Equal(t, hex.EncodeToString(mac.Sum(nil)), signature, "SendAuthHTTPRequest should sign the encoded parameters with HMAC-SHA256")
				assert.Contains(t, payload, url.Values{"symbols": {`["BTCUSDT","ETHUSDT"]`}}.Encode(), "SendAuthHTTPRequest should sign and send the percent-encoded parameters")
				query := r.URL.Query()
				assert.Equal(t, `["BTCUSDT","ETHUSDT"]`, query.Get("symbols"), "SendAuthHTTPRequest should send the parameters")
				assert.Equal(t, tc.exp, query.Get("recvWindow"), "SendAuthHTTPRequest should send the receive window")
				timestamp, err := strconv.ParseInt(query.Get("timestamp"), 10, 64)
				assert.NoError(t, err, "ParseInt should not error")
				assert.WithinDuration(t, time.Now(), time.UnixMilli(timestamp), time.Minute, "SendAuthHTTPRequest should send the current timestamp")
				_, err = w.Write([]byte(`{"code":200,"msg":"success"}`))
				assert.NoError(t, err, "Write should not error")
			})
			params := url.Values{}
			params.Set("symbols", `["BTCUSDT","ETHUSDT"]`)
			if tc.recvWindow != "" {
				params.Set("recvWindow", tc.recvWindow)
			}
			var result *ErrResponse
			require.NoError(t, ex.SendAuthHTTPRequest(t.Context(), exchange.RestSpotSupplementary, tc.method, "/api/v3/order", params, spotDefaultRate, &result), "SendAuthHTTPRequest must not error")
			assert.Equal(t, &ErrResponse{Code: 200, Message: "success"}, result, "SendAuthHTTPRequest should decode the response")
		})
	}
}

func TestSendAPIKeyHTTPRequest(t *testing.T) {
	t.Parallel()
	ex := newTestServerExchange(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPut, r.Method, "SendAPIKeyHTTPRequest should send the method")
		assert.Equal(t, "test-api-key", r.Header.Get("X-MBX-APIKEY"), "SendAPIKeyHTTPRequest should send the API key")
		assert.Equal(t, "listenKey=pqia91ma19a5s61cv6a81va65sdf19v8a65a1a5s61cv6a81va65sdf19v8a65a1", r.URL.RawQuery, "SendAPIKeyHTTPRequest should send neither a timestamp nor a signature")
		_, err := w.Write([]byte(`{}`))
		assert.NoError(t, err, "Write should not error")
	})
	var result *ErrResponse
	require.NoError(t, ex.SendAPIKeyHTTPRequest(t.Context(), exchange.RestSpotSupplementary, http.MethodPut, "/sapi/v1/userDataStream?listenKey=pqia91ma19a5s61cv6a81va65sdf19v8a65a1a5s61cv6a81va65sdf19v8a65a1", spotDefaultRate, &result), "SendAPIKeyHTTPRequest must not error")
	assert.Equal(t, &ErrResponse{}, result, "SendAPIKeyHTTPRequest should decode the response")
}

func TestObjectOrArrayUnmarshalJSON(t *testing.T) {
	t.Parallel()
	exp := objectOrArray[SymbolPrice]{{Symbol: "BTCUSDT", Price: 86168.13}, {Symbol: "ETHUSDT", Price: 2713.26}}
	var result objectOrArray[SymbolPrice]
	require.NoError(t, json.Unmarshal([]byte(`[{"symbol":"BTCUSDT","price":"86168.13000000"},{"symbol":"ETHUSDT","price":"2713.26000000"}]`), &result), "Unmarshal must not error for an array")
	assert.Equal(t, exp, result, "Unmarshal should decode every element of an array")

	result = nil
	require.NoError(t, json.Unmarshal([]byte(` {"symbol":"BTCUSDT","price":"86168.13000000"}`), &result), "Unmarshal must not error for an object")
	assert.Equal(t, exp[:1], result, "Unmarshal should decode an object into one element")

	assert.Error(t, json.Unmarshal([]byte(`{"symbol":1}`), &result), "Unmarshal should error for an invalid object")
	assert.Error(t, json.Unmarshal([]byte(`"BTCUSDT"`), &result), "Unmarshal should error for neither an object nor an array")
	assert.Error(t, json.Unmarshal([]byte(`[1]`), &result), "Unmarshal should error for an array of non-objects")
}

func TestCandleStickUnmarshalJSON(t *testing.T) {
	t.Parallel()
	exp := CandleStick{
		OpenTime:                 types.Time(time.UnixMilli(1499040000000)),
		OpenPrice:                0.0163479,
		HighPrice:                0.8,
		LowPrice:                 0.015758,
		ClosePrice:               0.015771,
		Volume:                   148976.11427815,
		CloseTime:                types.Time(time.UnixMilli(1499644799999)),
		QuoteAssetVolume:         2434.19055334,
		NumberOfTrades:           308,
		TakerBuyBaseAssetVolume:  1756.87402397,
		TakerBuyQuoteAssetVolume: 28.46694368,
	}
	var result CandleStick
	require.NoError(t, json.Unmarshal([]byte(`[1499040000000,"0.01634790","0.80000000","0.01575800","0.01577100","148976.11427815",1499644799999,"2434.19055334",308,"1756.87402397","28.46694368","0"]`), &result), "Unmarshal must not error")
	assert.Equal(t, exp, result, "Unmarshal should decode every element")

	assert.Error(t, json.Unmarshal([]byte(`{"openTime":1499040000000}`), &result), "Unmarshal should error for an object")
}

func TestSetTimeRangeParams(t *testing.T) {
	t.Parallel()
	start, end := time.UnixMilli(1744103854944), time.UnixMilli(1744190254944)
	params := url.Values{}
	require.ErrorIs(t, setTimeRangeParams(params, end, start), common.ErrStartAfterEnd, "setTimeRangeParams must reject a start after the end")
	assert.Empty(t, params, "setTimeRangeParams should not set parameters for a rejected range")
	for _, tc := range []struct {
		name       string
		start, end time.Time
		exp        url.Values
	}{
		{"both bounds", start, end, url.Values{"startTime": {"1744103854944"}, "endTime": {"1744190254944"}}},
		{"start only", start, time.Time{}, url.Values{"startTime": {"1744103854944"}}},
		{"end only", time.Time{}, end, url.Values{"endTime": {"1744190254944"}}},
		{"neither", time.Time{}, time.Time{}, url.Values{}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			params := url.Values{}
			require.NoError(t, setTimeRangeParams(params, tc.start, tc.end), "setTimeRangeParams must not error")
			assert.Equal(t, tc.exp, params, "setTimeRangeParams should set only the given bounds")
		})
	}
}
