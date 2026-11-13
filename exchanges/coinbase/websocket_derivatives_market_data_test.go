package coinbase

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/currency"
	"github.com/thrasher-corp/gocryptotrader/exchange/websocket"
)

func TestWebsocketAPIGetBookSummaryByCurrency(t *testing.T) {
	t.Parallel()
	ex, _ := newDerivativesWebsocketTestExchange(t)
	resp, err := ex.WebsocketAPI().GetBookSummaryByCurrency(t.Context(), &DerivativesBookSummaryByCurrencyRequest{Currency: currency.USDC, Kind: DerivativesKindFuture})
	require.NoError(t, err, "WebsocketAPI.GetBookSummaryByCurrency must not error")
	if mockTests {
		assert.Equal(t, mockMarketDataFutureSummaries(), resp, "WebsocketAPI.GetBookSummaryByCurrency should decode every field")
	} else {
		assert.NotEmpty(t, resp, "WebsocketAPI.GetBookSummaryByCurrency should return USDC's futures")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetBookSummaryByCurrency(t.Context(), &DerivativesBookSummaryByCurrencyRequest{Currency: currency.USDC})
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.GetBookSummaryByCurrency should error without a connection")
}

func TestWebsocketAPIGetBookSummaryByInstrument(t *testing.T) {
	t.Parallel()
	ex, _ := newDerivativesWebsocketTestExchange(t)
	resp, err := ex.WebsocketAPI().GetBookSummaryByInstrument(t.Context(), testMarketDataPerpetual)
	require.NoError(t, err, "WebsocketAPI.GetBookSummaryByInstrument must not error")
	if mockTests {
		assert.Equal(t, mockMarketDataPerpetualSummary(), resp, "WebsocketAPI.GetBookSummaryByInstrument should decode every field")
	} else {
		require.Len(t, resp, 1, "WebsocketAPI.GetBookSummaryByInstrument must return one summary")
		assert.Equal(t, testMarketDataPerpetual, resp[0].InstrumentName, "WebsocketAPI.GetBookSummaryByInstrument should return the instrument requested")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetBookSummaryByInstrument(t.Context(), testMarketDataPerpetual)
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.GetBookSummaryByInstrument should error without a connection")
}

func TestWebsocketAPIGetContractSize(t *testing.T) {
	t.Parallel()
	ex, _ := newDerivativesWebsocketTestExchange(t)
	resp, err := ex.WebsocketAPI().GetContractSize(t.Context(), testMarketDataPerpetual)
	require.NoError(t, err, "WebsocketAPI.GetContractSize must not error")
	assert.Equal(t, &DerivativesContractSizeResponse{ContractSize: 0.0001}, resp, "WebsocketAPI.GetContractSize should decode BTC_USDC-PERPETUAL's contract size of 0.0001 BTC")

	_, err = newTestExchange(t).WebsocketAPI().GetContractSize(t.Context(), testMarketDataPerpetual)
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.GetContractSize should error without a connection")
}

func TestWebsocketAPIGetCurrencies(t *testing.T) {
	t.Parallel()
	ex, _ := newDerivativesWebsocketTestExchange(t)
	resp, err := ex.WebsocketAPI().GetCurrencies(t.Context())
	require.NoError(t, err, "WebsocketAPI.GetCurrencies must not error")
	if mockTests {
		assert.Equal(t, mockMarketDataCurrencies(), resp, "WebsocketAPI.GetCurrencies should decode every field")
	} else {
		assert.NotEmpty(t, resp, "WebsocketAPI.GetCurrencies should return the gateway's currencies")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetCurrencies(t.Context())
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.GetCurrencies should error without a connection")
}

func TestWebsocketAPIGetDeliveryPrices(t *testing.T) {
	t.Parallel()
	ex, _ := newDerivativesWebsocketTestExchange(t)
	resp, err := ex.WebsocketAPI().GetDeliveryPrices(t.Context(), testMarketDataDeliveryPricesRequest())
	require.NoError(t, err, "WebsocketAPI.GetDeliveryPrices must not error")
	if mockTests {
		assert.Equal(t, mockMarketDataDeliveryPrices(), resp, "WebsocketAPI.GetDeliveryPrices should decode every field")
	} else {
		assert.Len(t, resp.Data, 5, "WebsocketAPI.GetDeliveryPrices should return the prices requested")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetDeliveryPrices(t.Context(), testMarketDataDeliveryPricesRequest())
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.GetDeliveryPrices should error without a connection")
}

func TestWebsocketAPIGetExpirations(t *testing.T) {
	t.Parallel()
	ex, _ := newDerivativesWebsocketTestExchange(t)
	resp, err := ex.WebsocketAPI().GetExpirations(t.Context(), &DerivativesExpirationsRequest{Currency: currency.USDC, Kind: DerivativesKindFuture})
	require.NoError(t, err, "WebsocketAPI.GetExpirations must not error")
	if mockTests {
		assert.Equal(t, mockMarketDataUSDCExpirations(), resp, "WebsocketAPI.GetExpirations should decode the labels keyed by currency")
	} else {
		assert.Contains(t, resp.ByCurrency["usdc"].Future, "PERPETUAL", "WebsocketAPI.GetExpirations should return USDC's perpetuals' label")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetExpirations(t.Context(), &DerivativesExpirationsRequest{})
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.GetExpirations should error without a connection")
}

func TestWebsocketAPIGetFundingChartData(t *testing.T) {
	t.Parallel()
	ex, _ := newDerivativesWebsocketTestExchange(t)
	arg := &DerivativesFundingChartDataRequest{InstrumentName: testMarketDataPerpetual, Length: DerivativesFundingChartLength8Hour}
	resp, err := ex.WebsocketAPI().GetFundingChartData(t.Context(), arg)
	require.NoError(t, err, "WebsocketAPI.GetFundingChartData must not error")
	if mockTests {
		assert.Equal(t, mockMarketDataFundingChart(), resp, "WebsocketAPI.GetFundingChartData should decode every field")
	} else {
		assert.NotEmpty(t, resp.Data, "WebsocketAPI.GetFundingChartData should return the funding chart")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetFundingChartData(t.Context(), arg)
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.GetFundingChartData should error without a connection")
}

func TestWebsocketAPIGetFundingRateHistory(t *testing.T) {
	t.Parallel()
	ex, _ := newDerivativesWebsocketTestExchange(t)
	resp, err := ex.WebsocketAPI().GetFundingRateHistory(t.Context(), testMarketDataFundingRequest())
	require.NoError(t, err, "WebsocketAPI.GetFundingRateHistory must not error")
	if mockTests {
		assert.Equal(t, mockMarketDataFundingRateHistory(), resp, "WebsocketAPI.GetFundingRateHistory should decode every field")
	} else {
		assert.Len(t, resp, 4, "WebsocketAPI.GetFundingRateHistory should return an hour for each hour ending in the range")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetFundingRateHistory(t.Context(), testMarketDataFundingRequest())
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.GetFundingRateHistory should error without a connection")
}

func TestWebsocketAPIGetFundingRateValue(t *testing.T) {
	t.Parallel()
	ex, _ := newDerivativesWebsocketTestExchange(t)
	resp, err := ex.WebsocketAPI().GetFundingRateValue(t.Context(), testMarketDataFundingRequest())
	require.NoError(t, err, "WebsocketAPI.GetFundingRateValue must not error")
	assert.Equal(t, 0.00003480116782906063, resp, "WebsocketAPI.GetFundingRateValue should decode the funding over the range")

	_, err = newTestExchange(t).WebsocketAPI().GetFundingRateValue(t.Context(), testMarketDataFundingRequest())
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.GetFundingRateValue should error without a connection")
}

func TestWebsocketAPIGetHistoricalVolatility(t *testing.T) {
	t.Parallel()
	ex, _ := newDerivativesWebsocketTestExchange(t)
	resp, err := ex.WebsocketAPI().GetHistoricalVolatility(t.Context(), currency.BTC)
	require.NoError(t, err, "WebsocketAPI.GetHistoricalVolatility must not error")
	if mockTests {
		assert.Equal(t, mockMarketDataHistoricalVolatility(), resp, "WebsocketAPI.GetHistoricalVolatility should decode every point")
	} else {
		assert.NotEmpty(t, resp, "WebsocketAPI.GetHistoricalVolatility should return BTC's realised volatility")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetHistoricalVolatility(t.Context(), currency.BTC)
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.GetHistoricalVolatility should error without a connection")
}

func TestWebsocketAPIGetIndexChartData(t *testing.T) {
	t.Parallel()
	ex, _ := newDerivativesWebsocketTestExchange(t)
	arg := &DerivativesIndexChartDataRequest{IndexName: testMarketDataIndexName, Range: DerivativesIndexChartRangeOneHour}
	resp, err := ex.WebsocketAPI().GetIndexChartData(t.Context(), arg)
	require.NoError(t, err, "WebsocketAPI.GetIndexChartData must not error")
	if mockTests {
		assert.Equal(t, mockMarketDataIndexChart(), resp, "WebsocketAPI.GetIndexChartData should decode every point")
	} else {
		assert.NotEmpty(t, resp, "WebsocketAPI.GetIndexChartData should return the index chart")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetIndexChartData(t.Context(), arg)
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.GetIndexChartData should error without a connection")
}

func TestWebsocketAPIGetIndexPrice(t *testing.T) {
	t.Parallel()
	ex, _ := newDerivativesWebsocketTestExchange(t)
	resp, err := ex.WebsocketAPI().GetIndexPrice(t.Context(), testMarketDataIndexName)
	require.NoError(t, err, "WebsocketAPI.GetIndexPrice must not error")
	if mockTests {
		assert.Equal(t, &DerivativesIndexPriceResponse{IndexPrice: 82405.01, EstimatedDeliveryPrice: 82405.01}, resp, "WebsocketAPI.GetIndexPrice should decode every field")
	} else {
		assert.Positive(t, resp.IndexPrice, "WebsocketAPI.GetIndexPrice should return the index price")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetIndexPrice(t.Context(), testMarketDataIndexName)
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.GetIndexPrice should error without a connection")
}

func TestWebsocketAPIGetIndexPriceNames(t *testing.T) {
	t.Parallel()
	ex, _ := newDerivativesWebsocketTestExchange(t)
	resp, err := ex.WebsocketAPI().GetIndexPriceNames(t.Context(), false)
	require.NoError(t, err, "WebsocketAPI.GetIndexPriceNames must not error")
	if mockTests {
		assert.Equal(t, mockMarketDataIndexPriceNames(), resp, "WebsocketAPI.GetIndexPriceNames should decode every name")
	} else {
		assert.Contains(t, resp, DerivativesIndexPriceNameResponse{Name: testMarketDataIndexName}, "WebsocketAPI.GetIndexPriceNames should return the btc_usdc index")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetIndexPriceNames(t.Context(), false)
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.GetIndexPriceNames should error without a connection")
}

func TestWebsocketAPIGetInstrument(t *testing.T) {
	t.Parallel()
	ex, _ := newDerivativesWebsocketTestExchange(t)
	resp, err := ex.WebsocketAPI().GetInstrument(t.Context(), testMarketDataPerpetual)
	require.NoError(t, err, "WebsocketAPI.GetInstrument must not error")
	if mockTests {
		assert.Equal(t, mockMarketDataInstruments()[testMarketDataPerpetual], resp, "WebsocketAPI.GetInstrument should decode every field")
	} else {
		assert.Equal(t, testMarketDataPerpetual, resp.InstrumentName, "WebsocketAPI.GetInstrument should return the instrument requested")
	}

	_, err = ex.WebsocketAPI().GetInstrument(t.Context(), testMarketDataUnknownInstrument)
	derivativesErr, ok := errors.AsType[*DerivativesError](err)
	require.True(t, ok, "WebsocketAPI.GetInstrument must return the gateway's error for an unknown instrument")
	assert.Equal(t, int64(-32602), derivativesErr.Code, "WebsocketAPI.GetInstrument should return the gateway's invalid params error")

	_, err = newTestExchange(t).WebsocketAPI().GetInstrument(t.Context(), testMarketDataPerpetual)
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.GetInstrument should error without a connection")
}

func TestWebsocketAPIGetInstruments(t *testing.T) {
	t.Parallel()
	ex, _ := newDerivativesWebsocketTestExchange(t)
	resp, err := ex.WebsocketAPI().GetInstruments(t.Context(), testMarketDataInstrumentsRequest())
	require.NoError(t, err, "WebsocketAPI.GetInstruments must not error")
	if mockTests {
		assert.Equal(t, mockMarketDataFutures(), resp, "WebsocketAPI.GetInstruments should decode every field")
	} else {
		assert.NotEmpty(t, resp, "WebsocketAPI.GetInstruments should return USDC's futures")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetInstruments(t.Context(), testMarketDataInstrumentsRequest())
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.GetInstruments should error without a connection")
}

func TestWebsocketAPIGetLastSettlementsByCurrency(t *testing.T) {
	t.Parallel()
	ex, _ := newDerivativesWebsocketTestExchange(t)
	resp, err := ex.WebsocketAPI().GetLastSettlementsByCurrency(t.Context(), testMarketDataLastSettlementsByCurrencyRequest())
	require.NoError(t, err, "WebsocketAPI.GetLastSettlementsByCurrency must not error")
	if mockTests {
		assert.Equal(t, mockMarketDataCurrencySettlements(), resp, "WebsocketAPI.GetLastSettlementsByCurrency should decode every field")
	} else {
		assert.Len(t, resp.Settlements, 3, "WebsocketAPI.GetLastSettlementsByCurrency should return the settlements requested")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetLastSettlementsByCurrency(t.Context(), testMarketDataLastSettlementsByCurrencyRequest())
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.GetLastSettlementsByCurrency should error without a connection")
}

func TestWebsocketAPIGetLastSettlementsByInstrument(t *testing.T) {
	t.Parallel()
	ex, _ := newDerivativesWebsocketTestExchange(t)
	resp, err := ex.WebsocketAPI().GetLastSettlementsByInstrument(t.Context(), testMarketDataLastSettlementsByInstrumentRequest())
	require.NoError(t, err, "WebsocketAPI.GetLastSettlementsByInstrument must not error")
	if mockTests {
		assert.Equal(t, mockMarketDataPerpetualSettlements(), resp, "WebsocketAPI.GetLastSettlementsByInstrument should decode every field")
	} else {
		assert.Len(t, resp.Settlements, 2, "WebsocketAPI.GetLastSettlementsByInstrument should return the settlements requested")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetLastSettlementsByInstrument(t.Context(), testMarketDataLastSettlementsByInstrumentRequest())
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.GetLastSettlementsByInstrument should error without a connection")
}

func TestWebsocketAPIGetLastTradesByCurrency(t *testing.T) {
	t.Parallel()
	ex, _ := newDerivativesWebsocketTestExchange(t)
	resp, err := ex.WebsocketAPI().GetLastTradesByCurrency(t.Context(), testMarketDataLastTradesByCurrencyRequest())
	require.NoError(t, err, "WebsocketAPI.GetLastTradesByCurrency must not error")
	if mockTests {
		assert.Equal(t, mockMarketDataFutureTrades(), resp, "WebsocketAPI.GetLastTradesByCurrency should decode every field")
	} else {
		assert.NotEmpty(t, resp.Trades, "WebsocketAPI.GetLastTradesByCurrency should return USDC's latest futures trades")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetLastTradesByCurrency(t.Context(), testMarketDataLastTradesByCurrencyRequest())
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.GetLastTradesByCurrency should error without a connection")
}

func TestWebsocketAPIGetLastTradesByCurrencyAndTime(t *testing.T) {
	t.Parallel()
	ex, _ := newDerivativesWebsocketTestExchange(t)
	resp, err := ex.WebsocketAPI().GetLastTradesByCurrencyAndTime(t.Context(), testMarketDataLastTradesByCurrencyAndTimeRequest())
	require.NoError(t, err, "WebsocketAPI.GetLastTradesByCurrencyAndTime must not error")
	if mockTests {
		assert.Equal(t, mockMarketDataFutureTradesByTime(), resp, "WebsocketAPI.GetLastTradesByCurrencyAndTime should decode every field")
	} else {
		assert.NotEmpty(t, resp.Trades, "WebsocketAPI.GetLastTradesByCurrencyAndTime should return USDC's futures trades of the last hour")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetLastTradesByCurrencyAndTime(t.Context(), testMarketDataLastTradesByCurrencyAndTimeRequest())
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.GetLastTradesByCurrencyAndTime should error without a connection")
}

func TestWebsocketAPIGetLastTradesByInstrument(t *testing.T) {
	t.Parallel()
	ex, _ := newDerivativesWebsocketTestExchange(t)
	resp, err := ex.WebsocketAPI().GetLastTradesByInstrument(t.Context(), testMarketDataLastTradesByInstrumentRequest())
	require.NoError(t, err, "WebsocketAPI.GetLastTradesByInstrument must not error")
	if mockTests {
		assert.Equal(t, mockMarketDataPerpetualTrades(), resp, "WebsocketAPI.GetLastTradesByInstrument should decode every field")
	} else {
		assert.Len(t, resp.Trades, 3, "WebsocketAPI.GetLastTradesByInstrument should return the trades requested")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetLastTradesByInstrument(t.Context(), testMarketDataLastTradesByInstrumentRequest())
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.GetLastTradesByInstrument should error without a connection")
}

func TestWebsocketAPIGetLastTradesByInstrumentAndTime(t *testing.T) {
	t.Parallel()
	ex, _ := newDerivativesWebsocketTestExchange(t)
	resp, err := ex.WebsocketAPI().GetLastTradesByInstrumentAndTime(t.Context(), testMarketDataLastTradesByInstrumentAndTimeRequest())
	require.NoError(t, err, "WebsocketAPI.GetLastTradesByInstrumentAndTime must not error")
	if mockTests {
		assert.Equal(t, mockMarketDataPerpetualTradesByTime(), resp, "WebsocketAPI.GetLastTradesByInstrumentAndTime should decode every field")
	} else {
		assert.Len(t, resp.Trades, 2, "WebsocketAPI.GetLastTradesByInstrumentAndTime should return the trades requested")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetLastTradesByInstrumentAndTime(t.Context(), testMarketDataLastTradesByInstrumentAndTimeRequest())
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.GetLastTradesByInstrumentAndTime should error without a connection")
}

func TestWebsocketAPIGetMarkPriceHistory(t *testing.T) {
	t.Parallel()
	ex, _ := newDerivativesWebsocketTestExchange(t)
	arg := testMarketDataMarkPriceRequest(t)
	resp, err := ex.WebsocketAPI().GetMarkPriceHistory(t.Context(), arg)
	require.NoError(t, err, "WebsocketAPI.GetMarkPriceHistory must not error")
	if mockTests {
		assert.Equal(t, mockMarketDataMarkPrices(), resp, "WebsocketAPI.GetMarkPriceHistory should decode every point")
	} else {
		for i := range resp {
			assert.WithinRangef(t, resp[i].Timestamp.Time(), arg.StartTime, arg.EndTime, "WebsocketAPI.GetMarkPriceHistory should return mark price %d within the range", i)
		}
	}

	_, err = newTestExchange(t).WebsocketAPI().GetMarkPriceHistory(t.Context(), arg)
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.GetMarkPriceHistory should error without a connection")
}

func TestWebsocketAPIGetOrderBook(t *testing.T) {
	t.Parallel()
	ex, _ := newDerivativesWebsocketTestExchange(t)
	arg := &DerivativesOrderBookRequest{InstrumentName: testMarketDataPerpetual, Depth: 5}
	resp, err := ex.WebsocketAPI().GetOrderBook(t.Context(), arg)
	require.NoError(t, err, "WebsocketAPI.GetOrderBook must not error")
	if mockTests {
		assert.Equal(t, mockMarketDataOrderBook(), resp, "WebsocketAPI.GetOrderBook should decode every field")
	} else {
		assert.Len(t, resp.Bids, 5, "WebsocketAPI.GetOrderBook should return the depth requested")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetOrderBook(t.Context(), arg)
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.GetOrderBook should error without a connection")
}

func TestWebsocketAPIGetOrderBookByInstrumentID(t *testing.T) {
	t.Parallel()
	ex, _ := newDerivativesWebsocketTestExchange(t)
	arg := &DerivativesOrderBookByInstrumentIDRequest{InstrumentID: testMarketDataPerpetualID, Depth: 1}
	resp, err := ex.WebsocketAPI().GetOrderBookByInstrumentID(t.Context(), arg)
	require.NoError(t, err, "WebsocketAPI.GetOrderBookByInstrumentID must not error")
	if mockTests {
		assert.Equal(t, mockMarketDataOrderBookByID(), resp, "WebsocketAPI.GetOrderBookByInstrumentID should decode every field")
	} else {
		assert.Equal(t, testMarketDataPerpetual, resp.InstrumentName, "WebsocketAPI.GetOrderBookByInstrumentID should return the instrument requested")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetOrderBookByInstrumentID(t.Context(), arg)
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.GetOrderBookByInstrumentID should error without a connection")
}

func TestWebsocketAPIGetSupportedIndexNames(t *testing.T) {
	t.Parallel()
	ex, _ := newDerivativesWebsocketTestExchange(t)
	resp, err := ex.WebsocketAPI().GetSupportedIndexNames(t.Context(), DerivativesIndexTypeSpot)
	require.NoError(t, err, "WebsocketAPI.GetSupportedIndexNames must not error")
	if mockTests {
		assert.Equal(t, []string{"aave_usdc", "ada_usdc", "aero_usdc", "algo_usdc", "apt_usdc"}, resp, "WebsocketAPI.GetSupportedIndexNames should decode every name")
	} else {
		assert.Contains(t, resp, testMarketDataIndexName, "WebsocketAPI.GetSupportedIndexNames should return the btc_usdc spot index")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetSupportedIndexNames(t.Context(), DerivativesIndexTypeSpot)
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.GetSupportedIndexNames should error without a connection")
}

func TestWebsocketAPIGetTradeVolumes(t *testing.T) {
	t.Parallel()
	ex, _ := newDerivativesWebsocketTestExchange(t)
	resp, err := ex.WebsocketAPI().GetTradeVolumes(t.Context(), true)
	require.NoError(t, err, "WebsocketAPI.GetTradeVolumes must not error")
	if mockTests {
		assert.Equal(t, mockMarketDataTradeVolumes(), resp, "WebsocketAPI.GetTradeVolumes should decode every field")
	} else {
		assert.NotEmpty(t, resp, "WebsocketAPI.GetTradeVolumes should return each currency's volumes")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetTradeVolumes(t.Context(), true)
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.GetTradeVolumes should error without a connection")
}

func TestWebsocketAPIGetTradingViewChartData(t *testing.T) {
	t.Parallel()
	ex, venue := newDerivativesWebsocketTestExchange(t)
	resp, err := ex.WebsocketAPI().GetTradingViewChartData(t.Context(), testMarketDataTradingViewChartDataRequest())
	require.NoError(t, err, "WebsocketAPI.GetTradingViewChartData must not error")
	if mockTests {
		assert.Equal(t, mockMarketDataCandles(), resp, "WebsocketAPI.GetTradingViewChartData should decode every field")
		requests := venue.requestsFor("public/get_tradingview_chart_data")
		require.Len(t, requests, 1, "WebsocketAPI.GetTradingViewChartData must send one request")
		assert.JSONEq(t, `{"instrument_name":"BTC_USDC-PERPETUAL","start_timestamp":1792281600000,"end_timestamp":1792296000000,"resolution":"60"}`, string(requests[0].Params), "WebsocketAPI.GetTradingViewChartData should send its params as a JSON-RPC request's, not a query string")
	} else {
		assert.Len(t, resp.Ticks, 5, "WebsocketAPI.GetTradingViewChartData should return each hour's candle")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetTradingViewChartData(t.Context(), testMarketDataTradingViewChartDataRequest())
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.GetTradingViewChartData should error without a connection")
}

func TestWebsocketAPIGetVolatilityIndexData(t *testing.T) {
	t.Parallel()
	ex, _ := newDerivativesWebsocketTestExchange(t)
	resp, err := ex.WebsocketAPI().GetVolatilityIndexData(t.Context(), testMarketDataVolatilityIndexDataRequest())
	require.NoError(t, err, "WebsocketAPI.GetVolatilityIndexData must not error")
	if mockTests {
		assert.Equal(t, mockMarketDataVolatilityIndexCandles(), resp, "WebsocketAPI.GetVolatilityIndexData should decode every field")
	} else {
		assert.Len(t, resp.Data, 5, "WebsocketAPI.GetVolatilityIndexData should return each hour's candle")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetVolatilityIndexData(t.Context(), testMarketDataVolatilityIndexDataRequest())
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.GetVolatilityIndexData should error without a connection")
}

func TestWebsocketAPIGetTicker(t *testing.T) {
	t.Parallel()
	ex, _ := newDerivativesWebsocketTestExchange(t)
	resp, err := ex.WebsocketAPI().GetTicker(t.Context(), marketDataOption(t))
	require.NoError(t, err, "WebsocketAPI.GetTicker must not error")
	if mockTests {
		assert.Equal(t, mockMarketDataTickers()[mockMarketDataOption], resp, "WebsocketAPI.GetTicker should decode every field of an option")
	} else {
		assert.NotNil(t, resp.Greeks, "WebsocketAPI.GetTicker should return an option's greeks")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetTicker(t.Context(), testMarketDataPerpetual)
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.GetTicker should error without a connection")
}

func TestWebsocketAPIGetTickersByCurrency(t *testing.T) {
	t.Parallel()
	ex, _ := newDerivativesWebsocketTestExchange(t)
	resp, err := ex.WebsocketAPI().GetTickersByCurrency(t.Context(), testMarketDataTickersByCurrencyRequest())
	require.NoError(t, err, "WebsocketAPI.GetTickersByCurrency must not error")
	if mockTests {
		assert.Equal(t, mockMarketDataTickersByCurrency(), resp, "WebsocketAPI.GetTickersByCurrency should decode every field")
	} else {
		assert.NotEmpty(t, resp, "WebsocketAPI.GetTickersByCurrency should return USDC's futures")
	}

	_, err = newTestExchange(t).WebsocketAPI().GetTickersByCurrency(t.Context(), testMarketDataTickersByCurrencyRequest())
	assert.ErrorIs(t, err, websocket.ErrNotConnected, "WebsocketAPI.GetTickersByCurrency should error without a connection")
}
