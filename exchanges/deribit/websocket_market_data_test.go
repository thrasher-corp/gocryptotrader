package deribit

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
)

func TestWebsocketAPIGetAPRHistory(t *testing.T) {
	t.Parallel()
	ex, _ := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().GetAPRHistory(t.Context(), testAPRHistoryRequest())
	require.NoError(t, err, "WebsocketAPI.GetAPRHistory must not error")
	if mockTests {
		assert.Equal(t, testAPRHistory(), result, "WebsocketAPI.GetAPRHistory should decode every field")
	} else {
		assert.Len(t, result.Data, 5, "WebsocketAPI.GetAPRHistory should return the days requested")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetAPRHistory(t.Context(), testAPRHistoryRequest())
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.GetAPRHistory should error without a websocket connection")
}

func TestWebsocketAPIGetBasketIndexComposition(t *testing.T) {
	t.Parallel()
	ex, _ := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().GetBasketIndexComposition(t.Context(), marketDataBasketIndexName)
	require.NoError(t, err, "WebsocketAPI.GetBasketIndexComposition must not error")
	if mockTests {
		assert.Equal(t, testBasketIndexComposition(), result, "WebsocketAPI.GetBasketIndexComposition should decode every field")
	} else {
		assert.Equal(t, marketDataBasketIndexName, result.ProductID, "WebsocketAPI.GetBasketIndexComposition should return the index requested")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetBasketIndexComposition(t.Context(), marketDataBasketIndexName)
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.GetBasketIndexComposition should error without a websocket connection")
}

func TestWebsocketAPIGetBookSummaryByCurrency(t *testing.T) {
	t.Parallel()
	ex, _ := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().GetBookSummaryByCurrency(t.Context(), &BookSummaryByCurrencyRequest{Currency: currency.BTC, Kind: KindOption})
	require.NoError(t, err, "WebsocketAPI.GetBookSummaryByCurrency must not error")
	if mockTests {
		assert.Equal(t, testOptionBookSummaries(), result, "WebsocketAPI.GetBookSummaryByCurrency should decode every field")
	} else {
		assert.NotEmpty(t, result, "WebsocketAPI.GetBookSummaryByCurrency should return BTC's options")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetBookSummaryByCurrency(t.Context(), &BookSummaryByCurrencyRequest{Currency: currency.BTC})
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.GetBookSummaryByCurrency should error without a websocket connection")
}

func TestWebsocketAPIGetBookSummaryByInstrument(t *testing.T) {
	t.Parallel()
	ex, _ := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().GetBookSummaryByInstrument(t.Context(), marketDataPerpetual)
	require.NoError(t, err, "WebsocketAPI.GetBookSummaryByInstrument must not error")
	if mockTests {
		assert.Equal(t, testPerpetualBookSummary(), result, "WebsocketAPI.GetBookSummaryByInstrument should decode every field")
	} else {
		require.Len(t, result, 1, "WebsocketAPI.GetBookSummaryByInstrument must return one summary")
		assert.Equal(t, marketDataPerpetual, result[0].InstrumentName, "WebsocketAPI.GetBookSummaryByInstrument should return the instrument requested")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetBookSummaryByInstrument(t.Context(), marketDataPerpetual)
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.GetBookSummaryByInstrument should error without a websocket connection")
}

func TestWebsocketAPIGetContractSize(t *testing.T) {
	t.Parallel()
	ex, _ := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().GetContractSize(t.Context(), marketDataPerpetual)
	require.NoError(t, err, "WebsocketAPI.GetContractSize must not error")
	assert.Equal(t, &ContractSizeResponse{ContractSize: 10}, result, "WebsocketAPI.GetContractSize should decode BTC-PERPETUAL's contract size of 10 USD")

	_, err = newTestExchange(t).WebsocketAPI().GetContractSize(t.Context(), marketDataPerpetual)
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.GetContractSize should error without a websocket connection")
}

func TestWebsocketAPIGetCurrencies(t *testing.T) {
	t.Parallel()
	ex, _ := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().GetCurrencies(t.Context())
	require.NoError(t, err, "WebsocketAPI.GetCurrencies must not error")
	if mockTests {
		assert.Equal(t, testPublicCurrencies(), result, "WebsocketAPI.GetCurrencies should decode every field")
	} else {
		assert.NotEmpty(t, result, "WebsocketAPI.GetCurrencies should return Deribit's currencies")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetCurrencies(t.Context())
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.GetCurrencies should error without a websocket connection")
}

func TestWebsocketAPIGetDeliveryPrices(t *testing.T) {
	t.Parallel()
	ex, _ := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().GetDeliveryPrices(t.Context(), testDeliveryPricesRequest())
	require.NoError(t, err, "WebsocketAPI.GetDeliveryPrices must not error")
	if mockTests {
		assert.Equal(t, testDeliveryPrices(), result, "WebsocketAPI.GetDeliveryPrices should decode every field")
	} else {
		assert.Len(t, result.Data, 5, "WebsocketAPI.GetDeliveryPrices should return the prices requested")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetDeliveryPrices(t.Context(), testDeliveryPricesRequest())
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.GetDeliveryPrices should error without a websocket connection")
}

func TestWebsocketAPIGetExpirations(t *testing.T) {
	t.Parallel()
	ex, _ := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().GetExpirations(t.Context(), &ExpirationsRequest{Currency: currency.BTC, Kind: KindOption})
	require.NoError(t, err, "WebsocketAPI.GetExpirations must not error")
	if mockTests {
		exp := &ExpirationsResponse{ByCurrency: map[string]Expirations{"btc": {Option: []string{"16OCT26", "17OCT26", "18OCT26", "19OCT26", "23OCT26"}}}}
		assert.Equal(t, exp, result, "WebsocketAPI.GetExpirations should decode the labels keyed by currency")
	} else {
		assert.NotEmpty(t, result.ByCurrency["btc"].Option, "WebsocketAPI.GetExpirations should return BTC's option labels")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetExpirations(t.Context(), &ExpirationsRequest{})
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.GetExpirations should error without a websocket connection")
}

func TestWebsocketAPIGetFundingChartData(t *testing.T) {
	t.Parallel()
	ex, _ := newWebsocketTestExchange(t)
	arg := &FundingChartDataRequest{InstrumentName: marketDataPerpetual, Length: FundingChartLength8Hour}
	result, err := ex.WebsocketAPI().GetFundingChartData(t.Context(), arg)
	require.NoError(t, err, "WebsocketAPI.GetFundingChartData must not error")
	if mockTests {
		assert.Equal(t, testFundingChartData(), result, "WebsocketAPI.GetFundingChartData should decode every field")
	} else {
		assert.NotEmpty(t, result.Data, "WebsocketAPI.GetFundingChartData should return the funding chart")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetFundingChartData(t.Context(), arg)
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.GetFundingChartData should error without a websocket connection")
}

func TestWebsocketAPIGetFundingRateHistory(t *testing.T) {
	t.Parallel()
	ex, _ := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().GetFundingRateHistory(t.Context(), testFundingRangeRequest())
	require.NoError(t, err, "WebsocketAPI.GetFundingRateHistory must not error")
	if mockTests {
		assert.Equal(t, testFundingRateHistory(), result, "WebsocketAPI.GetFundingRateHistory should decode every field")
	} else {
		assert.Len(t, result, 4, "WebsocketAPI.GetFundingRateHistory should return an hour for each hour ending in the range")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetFundingRateHistory(t.Context(), testFundingRangeRequest())
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.GetFundingRateHistory should error without a websocket connection")
}

func TestWebsocketAPIGetFundingRateValue(t *testing.T) {
	t.Parallel()
	ex, _ := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().GetFundingRateValue(t.Context(), testFundingRangeRequest())
	require.NoError(t, err, "WebsocketAPI.GetFundingRateValue must not error")
	if mockTests {
		assert.Equal(t, 4.989342119697176e-05, result, "WebsocketAPI.GetFundingRateValue should decode the funding rate")
	} else {
		assert.NotZero(t, result, "WebsocketAPI.GetFundingRateValue should return the funding rate")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetFundingRateValue(t.Context(), testFundingRangeRequest())
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.GetFundingRateValue should error without a websocket connection")
}

func TestWebsocketAPIGetHistoricalVolatility(t *testing.T) {
	t.Parallel()
	ex, _ := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().GetHistoricalVolatility(t.Context(), currency.BTC)
	require.NoError(t, err, "WebsocketAPI.GetHistoricalVolatility must not error")
	if mockTests {
		assert.Equal(t, testHistoricalVolatility(), result, "WebsocketAPI.GetHistoricalVolatility should decode every point")
	} else {
		assert.NotEmpty(t, result, "WebsocketAPI.GetHistoricalVolatility should return BTC's realised volatility")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetHistoricalVolatility(t.Context(), currency.BTC)
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.GetHistoricalVolatility should error without a websocket connection")
}

func TestWebsocketAPIGetIndexChartData(t *testing.T) {
	t.Parallel()
	ex, _ := newWebsocketTestExchange(t)
	arg := &IndexChartDataRequest{IndexName: marketDataIndexName, Range: IndexChartRangeOneHour}
	result, err := ex.WebsocketAPI().GetIndexChartData(t.Context(), arg)
	require.NoError(t, err, "WebsocketAPI.GetIndexChartData must not error")
	if mockTests {
		assert.Equal(t, testIndexChart(), result, "WebsocketAPI.GetIndexChartData should decode every point")
	} else {
		assert.NotEmpty(t, result, "WebsocketAPI.GetIndexChartData should return the index chart")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetIndexChartData(t.Context(), arg)
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.GetIndexChartData should error without a websocket connection")
}

func TestWebsocketAPIGetIndexPrice(t *testing.T) {
	t.Parallel()
	ex, _ := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().GetIndexPrice(t.Context(), marketDataIndexName)
	require.NoError(t, err, "WebsocketAPI.GetIndexPrice must not error")
	if mockTests {
		assert.Equal(t, &IndexPriceResponse{IndexPrice: 82285.99, EstimatedDeliveryPrice: 82285.99}, result, "WebsocketAPI.GetIndexPrice should decode every field")
	} else {
		assert.Positive(t, result.IndexPrice, "WebsocketAPI.GetIndexPrice should return the index price")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetIndexPrice(t.Context(), marketDataIndexName)
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.GetIndexPrice should error without a websocket connection")
}

func TestWebsocketAPIGetIndexPriceNames(t *testing.T) {
	t.Parallel()
	ex, _ := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().GetIndexPriceNames(t.Context(), false)
	require.NoError(t, err, "WebsocketAPI.GetIndexPriceNames must not error")
	if mockTests {
		assert.Equal(t, testIndexPriceNames(), result, "WebsocketAPI.GetIndexPriceNames should decode every name")
	} else {
		assert.Contains(t, result, IndexPriceName{Name: marketDataIndexName}, "WebsocketAPI.GetIndexPriceNames should return the btc_usd index")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetIndexPriceNames(t.Context(), false)
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.GetIndexPriceNames should error without a websocket connection")
}

func TestWebsocketAPIGetInstrument(t *testing.T) {
	t.Parallel()
	ex, _ := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().GetInstrument(t.Context(), marketDataOption(t))
	require.NoError(t, err, "WebsocketAPI.GetInstrument must not error")
	if mockTests {
		assert.Equal(t, testOptionInstrument(), result, "WebsocketAPI.GetInstrument should decode every field of an option")
	} else {
		assert.Equal(t, KindOption, result.Kind, "WebsocketAPI.GetInstrument should return an option")
	}

	_, err = ex.WebsocketAPI().GetInstrument(t.Context(), marketDataUnknownInstrument)
	var apiErr *APIError
	require.ErrorAs(t, err, &apiErr, "WebsocketAPI.GetInstrument must return Deribit's error for an unknown instrument")
	assert.Equal(t, int64(-32602), apiErr.Code, "WebsocketAPI.GetInstrument should return Deribit's invalid params error")

	_, err = newTestExchange(t).WebsocketAPI().GetInstrument(t.Context(), marketDataPerpetual)
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.GetInstrument should error without a websocket connection")
}

func TestWebsocketAPIGetInstruments(t *testing.T) {
	t.Parallel()
	ex, _ := newWebsocketTestExchange(t)
	arg := &InstrumentsRequest{Currency: currency.BTC, Kind: KindFuture, Extended: true}
	result, err := ex.WebsocketAPI().GetInstruments(t.Context(), arg)
	require.NoError(t, err, "WebsocketAPI.GetInstruments must not error")
	if mockTests {
		assert.Equal(t, testFutureInstruments(), result, "WebsocketAPI.GetInstruments should decode every field")
	} else {
		require.NotEmpty(t, result, "WebsocketAPI.GetInstruments must return BTC's futures")
		assert.Equal(t, marketDataPerpetual, result[0].InstrumentName, "WebsocketAPI.GetInstruments should list the perpetual first")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetInstruments(t.Context(), arg)
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.GetInstruments should error without a websocket connection")
}

func TestWebsocketAPIGetLastSettlementsByCurrency(t *testing.T) {
	t.Parallel()
	ex, _ := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().GetLastSettlementsByCurrency(t.Context(), testLastSettlementsByCurrencyRequest())
	require.NoError(t, err, "WebsocketAPI.GetLastSettlementsByCurrency must not error")
	if mockTests {
		assert.Equal(t, testLastDeliveries(), result, "WebsocketAPI.GetLastSettlementsByCurrency should decode every field")
	} else {
		assert.Len(t, result.Settlements, 3, "WebsocketAPI.GetLastSettlementsByCurrency should return the deliveries requested")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetLastSettlementsByCurrency(t.Context(), testLastSettlementsByCurrencyRequest())
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.GetLastSettlementsByCurrency should error without a websocket connection")
}

func TestWebsocketAPIGetLastSettlementsByInstrument(t *testing.T) {
	t.Parallel()
	ex, _ := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().GetLastSettlementsByInstrument(t.Context(), testLastSettlementsByInstrumentRequest())
	require.NoError(t, err, "WebsocketAPI.GetLastSettlementsByInstrument must not error")
	if mockTests {
		assert.Equal(t, testPerpetualLastSettlements(), result, "WebsocketAPI.GetLastSettlementsByInstrument should decode every field")
	} else {
		assert.Len(t, result.Settlements, 2, "WebsocketAPI.GetLastSettlementsByInstrument should return the settlements requested")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetLastSettlementsByInstrument(t.Context(), testLastSettlementsByInstrumentRequest())
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.GetLastSettlementsByInstrument should error without a websocket connection")
}

func TestWebsocketAPIGetLastTradesByCurrency(t *testing.T) {
	t.Parallel()
	ex, _ := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().GetLastTradesByCurrency(t.Context(), testLastTradesByCurrencyRequest())
	require.NoError(t, err, "WebsocketAPI.GetLastTradesByCurrency must not error")
	if mockTests {
		assert.Equal(t, testLastFutureTrades(), result, "WebsocketAPI.GetLastTradesByCurrency should decode every field")
	} else {
		assert.NotEmpty(t, result.Trades, "WebsocketAPI.GetLastTradesByCurrency should return BTC's latest futures trades")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetLastTradesByCurrency(t.Context(), testLastTradesByCurrencyRequest())
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.GetLastTradesByCurrency should error without a websocket connection")
}

func TestWebsocketAPIGetLastTradesByCurrencyAndTime(t *testing.T) {
	t.Parallel()
	ex, _ := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().GetLastTradesByCurrencyAndTime(t.Context(), testLastTradesByCurrencyAndTimeRequest())
	require.NoError(t, err, "WebsocketAPI.GetLastTradesByCurrencyAndTime must not error")
	if mockTests {
		assert.Equal(t, testLastFutureTradesByTime(), result, "WebsocketAPI.GetLastTradesByCurrencyAndTime should decode every field")
	} else {
		assert.NotEmpty(t, result.Trades, "WebsocketAPI.GetLastTradesByCurrencyAndTime should return BTC's futures trades of the last hour")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetLastTradesByCurrencyAndTime(t.Context(), testLastTradesByCurrencyAndTimeRequest())
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.GetLastTradesByCurrencyAndTime should error without a websocket connection")
}

func TestWebsocketAPIGetLastTradesByInstrument(t *testing.T) {
	t.Parallel()
	ex, _ := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().GetLastTradesByInstrument(t.Context(), testLastTradesByInstrumentRequest())
	require.NoError(t, err, "WebsocketAPI.GetLastTradesByInstrument must not error")
	if mockTests {
		assert.Equal(t, testLastPerpetualTrades(), result, "WebsocketAPI.GetLastTradesByInstrument should decode every field")
	} else {
		assert.Len(t, result.Trades, 3, "WebsocketAPI.GetLastTradesByInstrument should return the trades requested")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetLastTradesByInstrument(t.Context(), testLastTradesByInstrumentRequest())
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.GetLastTradesByInstrument should error without a websocket connection")
}

func TestWebsocketAPIGetLastTradesByInstrumentAndTime(t *testing.T) {
	t.Parallel()
	ex, _ := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().GetLastTradesByInstrumentAndTime(t.Context(), testLastTradesByInstrumentAndTimeRequest())
	require.NoError(t, err, "WebsocketAPI.GetLastTradesByInstrumentAndTime must not error")
	if mockTests {
		assert.Equal(t, testLastPerpetualTradesByTime(), result, "WebsocketAPI.GetLastTradesByInstrumentAndTime should decode every field")
	} else {
		assert.Len(t, result.Trades, 2, "WebsocketAPI.GetLastTradesByInstrumentAndTime should return the trades requested")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetLastTradesByInstrumentAndTime(t.Context(), testLastTradesByInstrumentAndTimeRequest())
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.GetLastTradesByInstrumentAndTime should error without a websocket connection")
}

func TestWebsocketAPIGetMarkPriceHistory(t *testing.T) {
	t.Parallel()
	ex, _ := newWebsocketTestExchange(t)
	arg := testMarkPriceRequest(t)
	result, err := ex.WebsocketAPI().GetMarkPriceHistory(t.Context(), arg)
	require.NoError(t, err, "WebsocketAPI.GetMarkPriceHistory must not error")
	if mockTests {
		assert.Equal(t, testMarkPrices(), result, "WebsocketAPI.GetMarkPriceHistory should decode every point")
	} else {
		for i := range result {
			assert.WithinRangef(t, result[i].Timestamp.Time(), arg.StartTime, arg.EndTime, "WebsocketAPI.GetMarkPriceHistory should return mark price %d within the range", i)
		}
	}

	_, err = newTestExchange(t).WebsocketAPI().GetMarkPriceHistory(t.Context(), arg)
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.GetMarkPriceHistory should error without a websocket connection")
}

func TestWebsocketAPIGetOrderBook(t *testing.T) {
	t.Parallel()
	ex, _ := newWebsocketTestExchange(t)
	arg := &OrderBookRequest{InstrumentName: marketDataPerpetual, Depth: 5}
	result, err := ex.WebsocketAPI().GetOrderBook(t.Context(), arg)
	require.NoError(t, err, "WebsocketAPI.GetOrderBook must not error")
	if mockTests {
		assert.Equal(t, testPerpetualOrderBook(), result, "WebsocketAPI.GetOrderBook should decode every field")
	} else {
		assert.Equal(t, marketDataPerpetual, result.InstrumentName, "WebsocketAPI.GetOrderBook should return the instrument requested")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetOrderBook(t.Context(), arg)
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.GetOrderBook should error without a websocket connection")
}

func TestWebsocketAPIGetOrderBookByInstrumentID(t *testing.T) {
	t.Parallel()
	ex, _ := newWebsocketTestExchange(t)
	arg := &OrderBookByInstrumentIDRequest{InstrumentID: marketDataPerpetualID, Depth: 1}
	result, err := ex.WebsocketAPI().GetOrderBookByInstrumentID(t.Context(), arg)
	require.NoError(t, err, "WebsocketAPI.GetOrderBookByInstrumentID must not error")
	if mockTests {
		assert.Equal(t, testPerpetualOrderBookByID(), result, "WebsocketAPI.GetOrderBookByInstrumentID should decode every field")
	} else {
		assert.Equal(t, marketDataPerpetual, result.InstrumentName, "WebsocketAPI.GetOrderBookByInstrumentID should return the instrument requested")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetOrderBookByInstrumentID(t.Context(), arg)
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.GetOrderBookByInstrumentID should error without a websocket connection")
}

func TestWebsocketAPIGetSupportedIndexNames(t *testing.T) {
	t.Parallel()
	ex, _ := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().GetSupportedIndexNames(t.Context(), IndexTypeSpot)
	require.NoError(t, err, "WebsocketAPI.GetSupportedIndexNames must not error")
	if mockTests {
		assert.Equal(t, []string{"aave_usdc", "ada_usdc", "aero_usdc", "algo_usdc", "apt_usdc"}, result, "WebsocketAPI.GetSupportedIndexNames should decode every name")
	} else {
		assert.Contains(t, result, "btc_usdc", "WebsocketAPI.GetSupportedIndexNames should return the btc_usdc spot index")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetSupportedIndexNames(t.Context(), IndexTypeSpot)
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.GetSupportedIndexNames should error without a websocket connection")
}

func TestWebsocketAPIGetTradeVolumes(t *testing.T) {
	t.Parallel()
	ex, _ := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().GetTradeVolumes(t.Context(), true)
	require.NoError(t, err, "WebsocketAPI.GetTradeVolumes must not error")
	if mockTests {
		assert.Equal(t, testTradeVolumes(), result, "WebsocketAPI.GetTradeVolumes should decode every field")
	} else {
		assert.NotEmpty(t, result, "WebsocketAPI.GetTradeVolumes should return each currency's volumes")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetTradeVolumes(t.Context(), true)
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.GetTradeVolumes should error without a websocket connection")
}

func TestWebsocketAPIGetTradingViewChartData(t *testing.T) {
	t.Parallel()
	ex, venue := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().GetTradingViewChartData(t.Context(), testTradingViewChartDataRequest())
	require.NoError(t, err, "WebsocketAPI.GetTradingViewChartData must not error")
	if mockTests {
		assert.Equal(t, testPerpetualCandles(), result, "WebsocketAPI.GetTradingViewChartData should decode every field")
		requests := venue.received("public/get_tradingview_chart_data")
		require.Len(t, requests, 1, "WebsocketAPI.GetTradingViewChartData must send one request")
		assert.JSONEq(t, `{"instrument_name":"BTC-PERPETUAL","start_timestamp":1792022400000,"end_timestamp":1792036800000,"resolution":"60"}`, string(requests[0].Params), "WebsocketAPI.GetTradingViewChartData should send the time range in Unix milliseconds and Deribit's resolution")
	} else {
		assert.Len(t, result.Ticks, 5, "WebsocketAPI.GetTradingViewChartData should return each hour's candle")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetTradingViewChartData(t.Context(), testTradingViewChartDataRequest())
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.GetTradingViewChartData should error without a websocket connection")
}

func TestWebsocketAPIGetVolatilityIndexData(t *testing.T) {
	t.Parallel()
	ex, _ := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().GetVolatilityIndexData(t.Context(), testVolatilityIndexDataRequest())
	require.NoError(t, err, "WebsocketAPI.GetVolatilityIndexData must not error")
	if mockTests {
		assert.Equal(t, testVolatilityIndexCandles(), result, "WebsocketAPI.GetVolatilityIndexData should decode every field")
	} else {
		assert.Len(t, result.Data, 5, "WebsocketAPI.GetVolatilityIndexData should return each hour's candle")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetVolatilityIndexData(t.Context(), testVolatilityIndexDataRequest())
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.GetVolatilityIndexData should error without a websocket connection")
}

func TestWebsocketAPIGetTicker(t *testing.T) {
	t.Parallel()
	ex, _ := newWebsocketTestExchange(t)
	result, err := ex.WebsocketAPI().GetTicker(t.Context(), marketDataOption(t))
	require.NoError(t, err, "WebsocketAPI.GetTicker must not error")
	if mockTests {
		assert.Equal(t, testInstrumentTickers()[mockMarketDataOption], result, "WebsocketAPI.GetTicker should decode every field of an option")
	} else {
		assert.NotNil(t, result.Greeks, "WebsocketAPI.GetTicker should return an option's greeks")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetTicker(t.Context(), marketDataPerpetual)
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.GetTicker should error without a websocket connection")
}
